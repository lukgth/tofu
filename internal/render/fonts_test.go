package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFonts(t *testing.T) {
	outDir := t.TempDir()

	if err := copyFonts(outDir); err != nil {
		t.Fatalf("copyFonts: %v", err)
	}

	expected := []string{
		"rubik-regular.ttf",
		"rubik-italic.ttf",
		"rubik-light.ttf",
		"rubik-medium.ttf",
		"rubik-bold.ttf",
	}

	for _, name := range expected {
		dst := filepath.Join(outDir, "assets-blog", "fonts", name)
		fi, err := os.Stat(dst)
		if err != nil {
			t.Errorf("missing font %s: %v", name, err)
			continue
		}
		if fi.Size() < 100*1024 {
			t.Errorf("font %s is only %d bytes (expected >= 100KB)", name, fi.Size())
		}
	}
}