//go:build integration

package singbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCheckGoldens validates generated configs with a real sing-box binary.
// Run: go test -tags integration ./internal/core/singbox/
func TestCheckGoldens(t *testing.T) {
	bin, err := FindBinary("")
	if err != nil {
		t.Skip("sing-box not installed")
	}
	files, _ := filepath.Glob(filepath.Join("..", "..", "..", "testdata", "singbox", "*.golden"))
	if len(files) == 0 {
		t.Fatal("no golden files")
	}
	for _, f := range files {
		out, err := exec.Command(bin, "check", "-c", f).CombinedOutput()
		if err != nil {
			t.Errorf("check %s: %v\n%s", filepath.Base(f), err, out)
			continue
		}
		t.Logf("check %s ok", filepath.Base(f))
	}
	_ = os.Getenv("")
}
