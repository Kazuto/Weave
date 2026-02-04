package weave

import (
	"testing"

	"github.com/Kazuto/Weave/internal/version"
)

func TestHello(t *testing.T) {
	got := Hello("Kazuto")
	want := "Hello, Kazuto!"
	if got != want {
		t.Fatalf("unexpected greeting: got %q want %q", got, want)
	}
}

func TestVersionAvailable(t *testing.T) {
	if version.Version == "" {
		t.Fatalf("version should not be empty")
	}
}
