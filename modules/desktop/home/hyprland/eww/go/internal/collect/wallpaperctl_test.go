package collect

import (
	"os"
	"path/filepath"
	"testing"
)

// TestListDirFollowsSymlinks pins the bug the shadow-daemon comparison found.
//
// No fixture-driven test can cover this: InstallFixture replaces ListDir
// wholesale, so the real implementation -- where the lstat-versus-stat choice
// lives -- is only exercised against a real filesystem. This repo deploys its
// seed wallpaper as a Home Manager store symlink, so an lstat here silently
// removes a wallpaper from the picker.
func TestListDirFollowsSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "real.png")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plain.png"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "linked.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone.png"), filepath.Join(dir, "broken.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub.png"), 0o755); err != nil {
		t.Fatal(err)
	}

	files := ScanWallpaperFiles(dir)
	got := map[string]bool{}
	for _, path := range files {
		got[filepath.Base(path)] = true
	}

	if !got["linked.png"] {
		t.Error("a symlinked wallpaper was dropped; ListDir is lstat-ing")
	}
	if !got["plain.png"] {
		t.Error("a plain file was dropped")
	}
	if got["broken.png"] {
		t.Error("a broken symlink was listed; is_file() is False for one")
	}
	if got["sub.png"] {
		t.Error("a directory was listed as a wallpaper")
	}
	if len(files) != 2 {
		t.Errorf("expected exactly plain.png and linked.png, got %v", files)
	}
}
