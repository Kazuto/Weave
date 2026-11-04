package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	checkInterval = 24 * time.Hour
	checkTimeout  = 2 * time.Second

	optOutEnv = "WEAVE_NO_UPDATE_CHECK"
)

// checkState remembers the last check so that most runs need no network request.
type checkState struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

// Check is an update check that may still be running in the background.
type Check struct {
	build  Build
	done   chan struct{}
	latest string
}

// StartCheck begins a background check when one is appropriate. It returns nil when no check
// should run, which is safe to call Notice on.
func StartCheck() *Check {
	if !checksAllowed(os.Getenv) || !stderrIsTerminal() {
		return nil
	}

	build := CurrentBuild()
	if build.Version == "" {
		return nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil
	}

	return startCheck(NewClient(), filepath.Join(cacheDir, "weave", "update-check.json"), build)
}

func checksAllowed(getenv func(string) string) bool {
	return getenv("CI") == "" && getenv(optOutEnv) == ""
}

func stderrIsTerminal() bool {
	info, err := os.Stderr.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func startCheck(client *Client, statePath string, build Build) *Check {
	saved := loadState(statePath)
	check := &Check{build: build, done: make(chan struct{}), latest: saved.Latest}

	if time.Since(saved.CheckedAt) < checkInterval {
		close(check.done)
		return check
	}

	go func() {
		defer close(check.done)

		ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
		defer cancel()

		// A failed check counts too, so being offline doesn't slow down every run.
		release, err := client.Latest(ctx)
		if err == nil && release.Ready {
			saved.Latest = release.Version
			check.latest = release.Version
		}
		saved.CheckedAt = time.Now()
		saveState(statePath, saved)
	}()

	return check
}

// Notice waits for the check, which is bounded by checkTimeout, and returns a message if a newer
// release is available. It returns an empty string otherwise.
func (c *Check) Notice() string {
	if c == nil {
		return ""
	}

	<-c.done

	if !IsNewer(c.latest, c.build.Version) {
		return ""
	}

	command := "weave update"
	if c.build.GoInstall {
		command = "go install github.com/Kazuto/Weave@latest"
	}

	return fmt.Sprintf("New version available: %s → %s. Run `%s` to install it.", c.build.Version, c.latest, command)
}

// loadState treats a missing or unreadable file as never having checked.
func loadState(path string) checkState {
	var state checkState

	data, err := os.ReadFile(path) // #nosec G304 -- path is a fixed file in the user's cache directory
	if err != nil {
		return state
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return checkState{}
	}

	return state
}

// saveState is best effort, since the worst outcome of a failed write is an extra check.
func saveState(path string, state checkState) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0600)
}
