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

	ioskeley := []string{
		"ioskeley-regular.ttf",
		"ioskeley-bold.ttf",
		"ioskeley-italic.ttf",
	}
	gelasio := []string{
		"gelasio-regular.ttf",
		"gelasio-bold.ttf",
		"gelasio-italic.ttf",
	}
	for _, name := range gelasio {
		dst := filepath.Join(outDir, "assets-blog", "fonts", name)
		fi, err := os.Stat(dst)
		if err != nil {
			t.Errorf("missing font %s: %v", name, err)
			continue
		}
		if fi.Size() < 10*1024 {
			t.Errorf("font %s is only %d bytes (expected >= 10KB)", name, fi.Size())
		}
	}
	for _, name := range ioskeley {
		dst := filepath.Join(outDir, "assets-blog", "fonts", name)
		fi, err := os.Stat(dst)
		if err != nil {
			t.Errorf("missing font %s: %v", name, err)
			continue
		}
		if fi.Size() < 10*1024 {
			t.Errorf("font %s is only %d bytes (expected >= 10KB)", name, fi.Size())
		}
	}
}
