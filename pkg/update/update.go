// Package update finds new Weave releases and installs them over the running binary.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/Kazuto/weave/pkg/version"
)

const (
	latestReleaseURL   = "https://api.github.com/repos/Kazuto/weave/releases/latest"
	releaseDownloadURL = "https://github.com/Kazuto/weave/releases/download"
	checksumsName      = "checksums.txt"

	// InstallTimeout bounds a full `weave update`, including the download.
	InstallTimeout = 5 * time.Minute

	verifyTimeout = 10 * time.Second

	maxMetadataSize = 5 << 20
	maxArchiveSize  = 100 << 20
	maxBinarySize   = 200 << 20
)

var releasePattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

// parseVersion reads an exact release version. Anything else, such as a build
// described as v0.1.0-2-gabc123 or v0.1.0-dirty, isn't a release.
func parseVersion(v string) ([3]int, bool) {
	match := releasePattern.FindStringSubmatch(v)
	if match == nil {
		return [3]int{}, false
	}

	var parsed [3]int
	for i, part := range match[1:] {
		n, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}, false
		}
		parsed[i] = n
	}

	return parsed, true
}

// IsNewer reports whether candidate is a later release than current.
func IsNewer(candidate, current string) bool {
	a, ok := parseVersion(candidate)
	if !ok {
		return false
	}
	b, ok := parseVersion(current)
	if !ok {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}

	return false
}

// Build describes how the running binary was built.
type Build struct {
	// Version is the release this binary was built from. It's empty for development builds.
	Version string
	// GoInstall is true when `go install` manages the binary, so Weave must not replace it.
	GoInstall bool
}

// CurrentBuild inspects the running binary.
func CurrentBuild() Build {
	moduleVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		moduleVersion = info.Main.Version
	}

	return classifyBuild(version.Version, moduleVersion)
}

func classifyBuild(stamped, moduleVersion string) Build {
	if _, ok := parseVersion(stamped); ok {
		return Build{Version: stamped}
	}

	// `go install` doesn't apply the Makefile's ldflags, but it does record the module version.
	if stamped == "dev" {
		if _, ok := parseVersion(moduleVersion); ok {
			return Build{Version: moduleVersion, GoInstall: true}
		}
	}

	return Build{}
}

// Release describes the latest published release.
type Release struct {
	Version string
	// Ready is false until the files this platform needs have been uploaded. Binaries are
	// built after the release is published, so there's a gap of a few minutes.
	Ready bool
}

// Client talks to GitHub releases.
type Client struct {
	http        *http.Client
	latestURL   string
	downloadURL string
	goos        string
	goarch      string
}

// NewClient returns a client for the official release location and this platform.
func NewClient() *Client {
	return &Client{
		http:        &http.Client{},
		latestURL:   latestReleaseURL,
		downloadURL: releaseDownloadURL,
		goos:        runtime.GOOS,
		goarch:      runtime.GOARCH,
	}
}

func archiveName(tag, goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("weave-%s-%s-%s.zip", tag, goos, goarch)
	}
	return fmt.Sprintf("weave-%s-%s-%s.tar.gz", tag, goos, goarch)
}

func binaryName(goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("weave-%s-%s.exe", goos, goarch)
	}
	return fmt.Sprintf("weave-%s-%s", goos, goarch)
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response is larger than the %d byte limit", limit)
	}
	return data, nil
}

func (c *Client) get(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "weave/"+version.Version)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	return readLimited(resp.Body, limit)
}

// Latest looks up the latest published release.
func (c *Client) Latest(ctx context.Context) (Release, error) {
	data, err := c.get(ctx, c.latestURL, maxMetadataSize)
	if err != nil {
		return Release{}, err
	}

	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return Release{}, fmt.Errorf("failed to read release details: %w", err)
	}

	if _, ok := parseVersion(payload.TagName); !ok {
		return Release{}, fmt.Errorf("latest release has an unexpected version %q", payload.TagName)
	}

	archive := archiveName(payload.TagName, c.goos, c.goarch)
	hasArchive, hasChecksums := false, false
	for _, asset := range payload.Assets {
		hasArchive = hasArchive || asset.Name == archive
		hasChecksums = hasChecksums || asset.Name == checksumsName
	}

	return Release{Version: payload.TagName, Ready: hasArchive && hasChecksums}, nil
}

// Install downloads the given release and replaces the running binary with it.
func (c *Client) Install(ctx context.Context, tag string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate the running binary: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("failed to locate the running binary: %w", err)
	}

	return c.installTo(ctx, tag, exe)
}

func (c *Client) installTo(ctx context.Context, tag, exe string) error {
	if _, ok := parseVersion(tag); !ok {
		return fmt.Errorf("invalid release version %q", tag)
	}

	archive := archiveName(tag, c.goos, c.goarch)
	base := c.downloadURL + "/" + tag + "/"

	checksums, err := c.get(ctx, base+checksumsName, maxMetadataSize)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	want, err := checksumFor(string(checksums), archive)
	if err != nil {
		return err
	}

	data, err := c.get(ctx, base+archive, maxArchiveSize)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", archive, err)
	}

	// Nothing from the archive is read until it has passed this check.
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", archive, want, got)
	}

	name := binaryName(c.goos, c.goarch)
	var binary []byte
	if c.goos == "windows" {
		binary, err = extractFromZip(data, name)
	} else {
		binary, err = extractFromTarGz(data, name)
	}
	if err != nil {
		return fmt.Errorf("failed to extract %s from %s: %w", name, archive, err)
	}

	return replaceBinary(exe, binary, tag, c.goos == "windows")
}

// checksumFor finds a file's entry in sha256sum output.
func checksumFor(checksums, name string) (string, error) {
	for line := range strings.SplitSeq(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum published for %s", name)
}

func extractFromTarGz(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("file not found in archive")
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeReg && header.Name == name {
			return readLimited(reader, maxBinarySize)
		}
	}
}

func extractFromZip(data []byte, name string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return readLimited(rc, maxBinarySize)
	}

	return nil, errors.New("file not found in archive")
}

// replaceBinary swaps the new binary in, keeping the current file's permissions because the
// archives don't preserve them. The current binary is untouched until the new one has run.
func replaceBinary(exe string, binary []byte, tag string, moveAside bool) error {
	info, err := os.Stat(exe)
	if err != nil {
		return err
	}

	// A temp file in the same directory lets the final rename be atomic.
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".weave-update-*"+filepath.Ext(exe))
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w", filepath.Dir(exe), err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(binary); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write the new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write the new binary: %w", err)
	}
	if err := os.Chmod(tmpPath, info.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to set permissions on the new binary: %w", err)
	}

	if err := verifyBinary(tmpPath, tag); err != nil {
		return err
	}

	return swapBinary(tmpPath, exe, moveAside)
}

// verifyBinary runs the new binary before it replaces anything. A binary that doesn't report
// this exact version couldn't update itself later, so it's rejected too.
func verifyBinary(path, tag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), verifyTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, path, "version").Output() // #nosec G204 -- path is the temp file holding the checksum-verified binary
	if err != nil {
		return fmt.Errorf("the new binary failed to run: %w", err)
	}

	fields := strings.Fields(string(out))
	if len(fields) == 0 || fields[len(fields)-1] != tag {
		return fmt.Errorf("the new binary doesn't report version %s", tag)
	}

	return nil
}

// swapBinary renames the new binary into place. Windows can't overwrite a running executable
// but can rename it, so there the current one is moved aside first. That copy can't be deleted
// while it's running, so it stays until the next update removes it.
func swapBinary(newPath, exe string, moveAside bool) error {
	if !moveAside {
		return os.Rename(newPath, exe)
	}

	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(newPath, exe); err != nil {
		_ = os.Rename(old, exe)
		return err
	}

	return nil
}
