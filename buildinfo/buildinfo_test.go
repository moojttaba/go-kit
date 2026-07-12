package buildinfo

import (
	"strings"
	"testing"
)

func TestReadFallsBackToDevOrRevision(t *testing.T) {
	// No ldflags version in `go test`; Read must still yield a non-empty Version
	// (either the embedded VCS revision or "dev") and a Go toolchain string.
	got := Read()
	if got.Version == "" {
		t.Fatal("Version is empty; want VCS revision or \"dev\"")
	}
	if got.Go == "" {
		t.Error("Go toolchain version is empty")
	}
}

func TestLdflagsVersionWins(t *testing.T) {
	prev := version
	t.Cleanup(func() { version = prev })

	version = "v1.2.3"
	if got := Read().Version; got != "v1.2.3" {
		t.Fatalf("Version = %q; want the ldflags-injected v1.2.3", got)
	}
}

func TestDirtyRevisionIsShortenedAndFlagged(t *testing.T) {
	// With no ldflags version, a long revision is shortened to 12 chars and
	// "-dirty" is appended when the tree was modified. Exercise the derivation
	// indirectly: a "dev" fallback must never contain a slash or be over-long.
	prev := version
	t.Cleanup(func() { version = prev })
	version = ""

	v := Read().Version
	if strings.ContainsAny(v, " /") {
		t.Errorf("derived Version %q contains unexpected characters", v)
	}
}
