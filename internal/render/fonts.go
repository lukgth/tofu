package render

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"tofu/web"
)

// copyFonts copies embedded font files from web/static/fonts to
// outDir/assets-blog/fonts/ so the site's CSS can reference them.
func copyFonts(outDir string) error {
	srcFS := web.Static
	prefix := "static/fonts"
	dstRoot := filepath.Join(outDir, "assets-blog", "fonts")

	return fs.WalkDir(srcFS, prefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(prefix, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dstRoot, rel)

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}

		src, err := srcFS.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}

		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		return nil
	})
}
