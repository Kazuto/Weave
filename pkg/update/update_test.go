package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	oldBinary = "old binary"
	testTag   = "v0.1.0"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"v0.0.11", "v0.0.10", true},
		{"v0.0.10", "v0.0.9", true}, // compared as numbers, not text
		{"v0.1.0", "v0.0.99", true},
		{"v1.0.0", "v0.99.99", true},
		{"v0.0.10", "v0.0.10", false},
		{"v0.0.9", "v0.0.10", false},
		{"v0.1.0", "v1.0.0", false},
		{"", "v0.0.10", false},
		{"v0.0.11", "", false},
		{"v0.0.11", "dev", false},
		{"v0.0.11", "v0.0.10-2-gabc123", false},
		{"v0.0.11-rc1", "v0.0.10", false},
		{"0.0.11", "v0.0.10", false},
		{"v99999999999999999999.0.0", "v0.0.10", false},
		{"\x1b[31mv9.9.9", "v0.0.10", false},
	}

	for _, tt := range tests {
		t.Run(tt.candidate+" vs "+tt.current, func(t *testing.T) {
			if got := IsNewer(tt.candidate, tt.current); got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.candidate, tt.current, got, tt.want)
			}
		})
	}
}

func TestClassifyBuild(t *testing.T) {
	tests := []struct {
		name          string
		stamped       string
		moduleVersion string
		want          Build
	}{
		{"release binary", "v0.0.10", "v0.0.10", Build{Version: "v0.0.10"}},
		{"go install", "dev", "v0.0.10", Build{Version: "v0.0.10", GoInstall: true}},
		{"plain go build", "dev", "(devel)", Build{}},
		{"go build on a dirty tree", "dev", "v0.0.10+dirty", Build{}},
		{"make build between tags", "v0.0.10-2-gabc123", "v0.0.11-0.2026", Build{}},
		{"make build on a dirty tree", "v0.0.10-dirty", "v0.0.10+dirty", Build{}},
		{"no version information", "dev", "", Build{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyBuild(tt.stamped, tt.moduleVersion); got != tt.want {
				t.Errorf("classifyBuild() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestArchiveAndBinaryNamesMatchCI(t *testing.T) {
	// These must match the names ci.yml and the Makefile publish.
	if got := archiveName("v0.0.10", "linux", "amd64"); got != "weave-v0.0.10-linux-amd64.tar.gz" {
		t.Errorf("archiveName() = %s", got)
	}
	if got := archiveName("v0.0.10", "windows", "amd64"); got != "weave-v0.0.10-windows-amd64.zip" {
		t.Errorf("archiveName() = %s", got)
	}
	if got := binaryName("darwin", "arm64"); got != "weave-darwin-arm64" {
		t.Errorf("binaryName() = %s", got)
	}
	if got := binaryName("windows", "amd64"); got != "weave-windows-amd64.exe" {
		t.Errorf("binaryName() = %s", got)
	}
}

func TestChecksumFor(t *testing.T) {
	checksums := "aaa  weave-v0.1.0-linux-amd64.tar.gz\nbbb *weave-v0.1.0-windows-amd64.zip\n"

	if got, err := checksumFor(checksums, "weave-v0.1.0-linux-amd64.tar.gz"); err != nil || got != "aaa" {
		t.Errorf("text mode entry = %q, %v", got, err)
	}
	if got, err := checksumFor(checksums, "weave-v0.1.0-windows-amd64.zip"); err != nil || got != "bbb" {
		t.Errorf("binary mode entry = %q, %v", got, err)
	}
	if _, err := checksumFor(checksums, "weave-v0.1.0-linux"); err == nil {
		t.Error("a partial name should not match")
	}
}

func TestReadLimited(t *testing.T) {
	if got, err := readLimited(strings.NewReader("12345"), 5); err != nil || string(got) != "12345" {
		t.Errorf("at the limit = %q, %v", got, err)
	}
	if _, err := readLimited(strings.NewReader("123456"), 5); err == nil {
		t.Error("over the limit should fail")
	}
}

// fakeReleases serves files by path, like GitHub's release API and download URLs.
func fakeReleases(t *testing.T, files map[string][]byte) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)

	return &Client{
		http:        server.Client(),
		latestURL:   server.URL + "/latest",
		downloadURL: server.URL + "/download",
		goos:        "linux",
		goarch:      "amd64",
	}
}

func TestLatest(t *testing.T) {
	archive := `{"name":"weave-v0.1.0-linux-amd64.tar.gz"}`
	checksums := `{"name":"checksums.txt"}`
	other := `{"name":"weave-v0.1.0-darwin-arm64.tar.gz"}`

	tests := []struct {
		name      string
		body      string
		want      Release
		wantError bool
	}{
		{
			name: "all files uploaded",
			body: fmt.Sprintf(`{"tag_name":"v0.1.0","assets":[%s,%s,%s]}`, other, archive, checksums),
			want: Release{Version: "v0.1.0", Ready: true},
		},
		{
			name: "binaries still being built",
			body: `{"tag_name":"v0.1.0","assets":[]}`,
			want: Release{Version: "v0.1.0"},
		},
		{
			name: "no checksums, like releases up to v0.0.10",
			body: fmt.Sprintf(`{"tag_name":"v0.1.0","assets":[%s]}`, archive),
			want: Release{Version: "v0.1.0"},
		},
		{
			name: "no build for this platform",
			body: fmt.Sprintf(`{"tag_name":"v0.1.0","assets":[%s,%s]}`, other, checksums),
			want: Release{Version: "v0.1.0"},
		},
		{name: "tag isn't a release version", body: `{"tag_name":"nightly","assets":[]}`, wantError: true},
		{name: "not JSON", body: `<html>`, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakeReleases(t, map[string][]byte{"/latest": []byte(tt.body)})

			got, err := client.Latest(context.Background())
			if (err != nil) != tt.wantError {
				t.Fatalf("Latest() error = %v, wantError %v", err, tt.wantError)
			}
			if got != tt.want {
				t.Errorf("Latest() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLatestReportsServerErrors(t *testing.T) {
	client := fakeReleases(t, map[string][]byte{})

	if _, err := client.Latest(context.Background()); err == nil {
		t.Error("Latest() error = nil, want error for a 404")
	}
}

// fakeBinary is a script standing in for a release binary that reports the given version.
func fakeBinary(t *testing.T, reportedVersion string) []byte {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake binaries are shell scripts")
	}
	return []byte("#!/bin/sh\necho \"weave " + reportedVersion + "\"\n")
}

func tarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	writer := tar.NewWriter(gz)
	// Mode 0644 matches the real archives, which lose the executable bit in CI.
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func zipFile(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	file, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// installedBinary creates the "running" binary in a temp directory.
func installedBinary(t *testing.T, name string, mode os.FileMode) string {
	t.Helper()

	exe := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(exe, []byte(oldBinary), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(exe, mode); err != nil {
		t.Fatal(err)
	}

	return exe
}

func assertUnchanged(t *testing.T, exe string) {
	t.Helper()

	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != oldBinary {
		t.Errorf("binary was changed to %q", data)
	}
	assertNoTempFiles(t, exe)
}

func assertNoTempFiles(t *testing.T, exe string) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Dir(exe))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".weave-update-") {
			t.Errorf("left temp file behind: %s", entry.Name())
		}
	}
}

func TestInstallReplacesBinary(t *testing.T) {
	for _, mode := range []os.FileMode{0755, 0750} {
		t.Run(mode.String(), func(t *testing.T) {
			binary := fakeBinary(t, testTag)
			archive := tarGz(t, "weave-linux-amd64", binary)
			client := fakeReleases(t, map[string][]byte{
				"/download/v0.1.0/checksums.txt":                   []byte(sha256Hex(archive) + "  weave-v0.1.0-linux-amd64.tar.gz\n"),
				"/download/v0.1.0/weave-v0.1.0-linux-amd64.tar.gz": archive,
			})
			exe := installedBinary(t, "weave", mode)

			if err := client.installTo(context.Background(), testTag, exe); err != nil {
				t.Fatalf("installTo() error = %v", err)
			}

			data, err := os.ReadFile(exe)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, binary) {
				t.Errorf("binary = %q, want the new one", data)
			}
			info, err := os.Stat(exe)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != mode {
				t.Errorf("mode = %v, want %v", info.Mode().Perm(), mode)
			}
			assertNoTempFiles(t, exe)
		})
	}
}

func TestInstallOnWindowsLayout(t *testing.T) {
	binary := fakeBinary(t, testTag)
	archive := zipFile(t, "weave-windows-amd64.exe", binary)
	client := fakeReleases(t, map[string][]byte{
		"/download/v0.1.0/checksums.txt":                  []byte(sha256Hex(archive) + " *weave-v0.1.0-windows-amd64.zip\n"),
		"/download/v0.1.0/weave-v0.1.0-windows-amd64.zip": archive,
	})
	client.goos = "windows"
	exe := installedBinary(t, "weave.exe", 0755)

	if err := client.installTo(context.Background(), testTag, exe); err != nil {
		t.Fatalf("installTo() error = %v", err)
	}

	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, binary) {
		t.Errorf("binary = %q, want the new one", data)
	}
	old, err := os.ReadFile(exe + ".old")
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != oldBinary {
		t.Errorf("moved-aside binary = %q", old)
	}
	assertNoTempFiles(t, exe)
}

func TestInstallLeavesBinaryAloneOnFailure(t *testing.T) {
	good := tarGz(t, "weave-linux-amd64", fakeBinary(t, testTag))
	archivePath := "/download/v0.1.0/weave-v0.1.0-linux-amd64.tar.gz"
	checksumsPath := "/download/v0.1.0/checksums.txt"
	entry := func(data []byte) []byte {
		return []byte(sha256Hex(data) + "  weave-v0.1.0-linux-amd64.tar.gz\n")
	}

	failing := tarGz(t, "weave-linux-amd64", []byte("#!/bin/sh\nexit 1\n"))
	wrongVersion := tarGz(t, "weave-linux-amd64", fakeBinary(t, "v0.0.9"))
	dirtyVersion := tarGz(t, "weave-linux-amd64", fakeBinary(t, "v0.1.0-dirty"))
	silent := tarGz(t, "weave-linux-amd64", []byte("#!/bin/sh\nexit 0\n"))
	wrongName := tarGz(t, "weave-darwin-arm64", fakeBinary(t, testTag))
	notAnArchive := []byte("not an archive")

	tests := []struct {
		name  string
		files map[string][]byte
		want  string
	}{
		{
			name:  "checksum doesn't match",
			files: map[string][]byte{checksumsPath: entry([]byte("something else")), archivePath: good},
			want:  "checksum mismatch",
		},
		{
			name:  "no checksum for this archive",
			files: map[string][]byte{checksumsPath: []byte("abc  weave-v0.1.0-darwin-arm64.tar.gz\n"), archivePath: good},
			want:  "no checksum published",
		},
		{
			name:  "checksums file missing",
			files: map[string][]byte{archivePath: good},
			want:  "failed to download checksums",
		},
		{
			name:  "archive missing",
			files: map[string][]byte{checksumsPath: entry(good)},
			want:  "failed to download weave-v0.1.0-linux-amd64.tar.gz",
		},
		{
			name:  "archive is corrupt",
			files: map[string][]byte{checksumsPath: entry(notAnArchive), archivePath: notAnArchive},
			want:  "failed to extract",
		},
		{
			name:  "archive has no binary for this platform",
			files: map[string][]byte{checksumsPath: entry(wrongName), archivePath: wrongName},
			want:  "file not found in archive",
		},
		{
			name:  "new binary fails to run",
			files: map[string][]byte{checksumsPath: entry(failing), archivePath: failing},
			want:  "failed to run",
		},
		{
			name:  "new binary reports another version",
			files: map[string][]byte{checksumsPath: entry(wrongVersion), archivePath: wrongVersion},
			want:  "doesn't report version",
		},
		{
			name:  "new binary reports a dirty build",
			files: map[string][]byte{checksumsPath: entry(dirtyVersion), archivePath: dirtyVersion},
			want:  "doesn't report version",
		},
		{
			name:  "new binary reports nothing",
			files: map[string][]byte{checksumsPath: entry(silent), archivePath: silent},
			want:  "doesn't report version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakeReleases(t, tt.files)
			exe := installedBinary(t, "weave", 0755)

			err := client.installTo(context.Background(), testTag, exe)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("installTo() error = %v, want it to contain %q", err, tt.want)
			}
			assertUnchanged(t, exe)
		})
	}
}

func TestInstallRejectsInvalidTag(t *testing.T) {
	client := fakeReleases(t, map[string][]byte{})
	exe := installedBinary(t, "weave", 0755)

	for _, tag := range []string{"", "latest", "v0.1.0/../../evil", "v0.1"} {
		if err := client.installTo(context.Background(), tag, exe); err == nil {
			t.Errorf("installTo(%q) error = nil, want error", tag)
		}
	}
	assertUnchanged(t, exe)
}

func TestInstallReportsReadOnlyDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to read-only directories")
	}

	binary := fakeBinary(t, testTag)
	archive := tarGz(t, "weave-linux-amd64", binary)
	client := fakeReleases(t, map[string][]byte{
		"/download/v0.1.0/checksums.txt":                   []byte(sha256Hex(archive) + "  weave-v0.1.0-linux-amd64.tar.gz\n"),
		"/download/v0.1.0/weave-v0.1.0-linux-amd64.tar.gz": archive,
	})
	exe := installedBinary(t, "weave", 0755)
	dir := filepath.Dir(exe)
	if err := os.Chmod(dir, 0500); err != nil { // #nosec G302 -- test directory must be read-only
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) }) // #nosec G302 -- lets the test clean up its directory

	err := client.installTo(context.Background(), testTag, exe)
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("installTo() error = %v, want a permission error", err)
	}
	assertUnchanged(t, exe)
}

func TestSwapBinaryRestoresOnFailure(t *testing.T) {
	exe := installedBinary(t, "weave.exe", 0755)
	missing := filepath.Join(filepath.Dir(exe), "does-not-exist")

	if err := swapBinary(missing, exe, true); err == nil {
		t.Fatal("swapBinary() error = nil, want error")
	}

	assertUnchanged(t, exe)
	if _, err := os.Stat(exe + ".old"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("moved-aside copy should be gone, got %v", err)
	}
}

func TestSwapBinaryRemovesPreviousOldCopy(t *testing.T) {
	exe := installedBinary(t, "weave.exe", 0755)
	if err := os.WriteFile(exe+".old", []byte("from the last update"), 0600); err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(filepath.Dir(exe), "new")
	if err := os.WriteFile(newPath, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := swapBinary(newPath, exe, true); err != nil {
		t.Fatalf("swapBinary() error = %v", err)
	}

	old, err := os.ReadFile(exe + ".old")
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != oldBinary {
		t.Errorf("moved-aside binary = %q", old)
	}
}
