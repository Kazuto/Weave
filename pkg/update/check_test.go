package update

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var releaseBuild = Build{Version: "v0.0.10"}

const readyRelease = `{"tag_name":"v0.1.0","assets":[{"name":"weave-v0.1.0-linux-amd64.tar.gz"},{"name":"checksums.txt"}]}`

// countingServer reports how many requests it received, so tests can assert when no check ran.
func countingServer(t *testing.T, status int, body string) (*Client, *atomic.Int32) {
	t.Helper()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return &Client{
		http:      server.Client(),
		latestURL: server.URL,
		goos:      "linux",
		goarch:    "amd64",
	}, &requests
}

func statePathIn(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "cache", "weave", "update-check.json")
}

func TestCheckQueriesWhenThereIsNoState(t *testing.T) {
	client, requests := countingServer(t, http.StatusOK, readyRelease)
	path := statePathIn(t)

	notice := startCheck(client, path, releaseBuild).Notice()

	if requests.Load() != 1 {
		t.Errorf("requests = %d, want 1", requests.Load())
	}
	if !strings.Contains(notice, "v0.0.10 → v0.1.0") || !strings.Contains(notice, "`weave update`") {
		t.Errorf("Notice() = %q", notice)
	}

	saved := loadState(path)
	if saved.Latest != "v0.1.0" {
		t.Errorf("saved latest = %q", saved.Latest)
	}
	if time.Since(saved.CheckedAt) > time.Minute {
		t.Errorf("saved time = %v", saved.CheckedAt)
	}
}

func TestCheckUsesRecentStateWithoutNetwork(t *testing.T) {
	client, requests := countingServer(t, http.StatusOK, readyRelease)
	path := statePathIn(t)
	saveState(path, checkState{CheckedAt: time.Now().Add(-time.Hour), Latest: "v0.0.11"})

	notice := startCheck(client, path, releaseBuild).Notice()

	if requests.Load() != 0 {
		t.Errorf("requests = %d, want 0", requests.Load())
	}
	if !strings.Contains(notice, "v0.0.11") {
		t.Errorf("Notice() = %q, want the saved version", notice)
	}
}

func TestCheckQueriesAgainAfterTheInterval(t *testing.T) {
	client, requests := countingServer(t, http.StatusOK, readyRelease)
	path := statePathIn(t)
	saveState(path, checkState{CheckedAt: time.Now().Add(-25 * time.Hour), Latest: "v0.0.11"})

	notice := startCheck(client, path, releaseBuild).Notice()

	if requests.Load() != 1 {
		t.Errorf("requests = %d, want 1", requests.Load())
	}
	if !strings.Contains(notice, "v0.1.0") {
		t.Errorf("Notice() = %q, want the fresh version", notice)
	}
}

func TestCheckRunsOnlyOnceADay(t *testing.T) {
	client, requests := countingServer(t, http.StatusOK, readyRelease)
	path := statePathIn(t)

	for range 3 {
		startCheck(client, path, releaseBuild).Notice()
	}

	if requests.Load() != 1 {
		t.Errorf("requests = %d, want 1", requests.Load())
	}
}

func TestFailedCheckKeepsLastVersionAndDoesNotRetryImmediately(t *testing.T) {
	client, requests := countingServer(t, http.StatusInternalServerError, "")
	path := statePathIn(t)
	saveState(path, checkState{CheckedAt: time.Now().Add(-25 * time.Hour), Latest: "v0.0.11"})

	notice := startCheck(client, path, releaseBuild).Notice()
	if !strings.Contains(notice, "v0.0.11") {
		t.Errorf("Notice() = %q, want the last known version", notice)
	}

	startCheck(client, path, releaseBuild).Notice()
	if requests.Load() != 1 {
		t.Errorf("requests = %d, want 1", requests.Load())
	}
}

func TestCheckIgnoresReleaseStillBeingBuilt(t *testing.T) {
	client, _ := countingServer(t, http.StatusOK, `{"tag_name":"v0.1.0","assets":[]}`)

	if notice := startCheck(client, statePathIn(t), releaseBuild).Notice(); notice != "" {
		t.Errorf("Notice() = %q, want none until the files are uploaded", notice)
	}
}

func TestCheckGivesUpOnASlowServer(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })
	client := &Client{http: server.Client(), latestURL: server.URL, goos: "linux", goarch: "amd64"}

	start := time.Now()
	notice := startCheck(client, statePathIn(t), releaseBuild).Notice()

	if notice != "" {
		t.Errorf("Notice() = %q", notice)
	}
	if elapsed := time.Since(start); elapsed > checkTimeout+time.Second {
		t.Errorf("Notice() took %v", elapsed)
	}
}

func TestNoNoticeWithoutANewerRelease(t *testing.T) {
	tests := []struct {
		name   string
		latest string
	}{
		{"same version", "v0.0.10"},
		{"older version", "v0.0.9"},
		{"nothing known yet", ""},
		{"file was edited", "\x1b[31mv9.9.9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := countingServer(t, http.StatusOK, readyRelease)
			path := statePathIn(t)
			saveState(path, checkState{CheckedAt: time.Now(), Latest: tt.latest})

			if notice := startCheck(client, path, releaseBuild).Notice(); notice != "" {
				t.Errorf("Notice() = %q, want none", notice)
			}
		})
	}
}

func TestNoticeForGoInstallUsesGoInstall(t *testing.T) {
	client, _ := countingServer(t, http.StatusOK, readyRelease)
	build := Build{Version: "v0.0.10", GoInstall: true}

	notice := startCheck(client, statePathIn(t), build).Notice()

	if !strings.Contains(notice, "`go install github.com/Kazuto/weave@latest`") {
		t.Errorf("Notice() = %q", notice)
	}
	if strings.Contains(notice, "weave update") {
		t.Errorf("Notice() = %q, should not suggest weave update", notice)
	}
}

func TestUnreadableStateCountsAsNeverChecked(t *testing.T) {
	client, requests := countingServer(t, http.StatusOK, readyRelease)
	path := statePathIn(t)
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}

	startCheck(client, path, releaseBuild).Notice()

	if requests.Load() != 1 {
		t.Errorf("requests = %d, want 1", requests.Load())
	}
}

func TestUnwritableStateStillGivesANotice(t *testing.T) {
	client, _ := countingServer(t, http.StatusOK, readyRelease)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0600); err != nil {
		t.Fatal(err)
	}

	notice := startCheck(client, filepath.Join(blocker, "update-check.json"), releaseBuild).Notice()

	if !strings.Contains(notice, "v0.1.0") {
		t.Errorf("Notice() = %q", notice)
	}
}

func TestNilCheckHasNoNotice(t *testing.T) {
	var check *Check
	if notice := check.Notice(); notice != "" {
		t.Errorf("Notice() = %q", notice)
	}
}

func TestChecksAllowed(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"default", map[string]string{}, true},
		{"CI", map[string]string{"CI": "true"}, false},
		{"opted out", map[string]string{optOutEnv: "1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			if got := checksAllowed(getenv); got != tt.want {
				t.Errorf("checksAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
