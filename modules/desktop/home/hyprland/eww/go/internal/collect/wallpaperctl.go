package collect

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The impure half of the wallpaper picker: scanning the directory, building
// thumbnails, and asking awww to change the wallpaper.

// wallpaperExtensions is awww-decodable only, and notably does NOT include
// .jxl. Adding one it cannot decode puts a broken tile in the picker.
var wallpaperExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".bmp": true, ".tiff": true,
}

// awwwTransition are calibration values: grow-from-top-right matches where the
// picker popup sits, so the new wallpaper appears to expand out of it.
var awwwTransition = []string{
	"--transition-type", "grow",
	"--transition-pos", "top-right",
	"--transition-duration", "0.8",
	"--transition-fps", "60",
}

const awwwTimeout = 5 * time.Second

// magickTimeout is per IMAGE, and a cold cache pays it once per wallpaper.
const magickTimeout = 15 * time.Second

func wallpaperDir() string { return expandUser("~/Pictures/wallpapers") }
func thumbDir() string     { return expandUser("~/.cache/eww-bar/wallpaper-thumbs") }

// DirEntryInfo is the part of a directory listing these collectors use.
type DirEntryInfo struct {
	Path    string
	IsFile  bool
	MtimeNS int64
	Size    int64
}

// The remaining filesystem seams. ListDir and ResolvePath are separate from
// ReadTextFile because the picker needs metadata and symlink resolution.
var (
	ListDir = func(dir string) ([]DirEntryInfo, bool) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, false
		}
		listing := make([]DirEntryInfo, 0, len(entries))
		for _, entry := range entries {
			full := filepath.Join(dir, entry.Name())
			// os.Stat, NOT entry.Info(): Info is an lstat, so a symlink reports as
			// not-a-regular-file and drops out of the picker. pathlib's is_file() and
			// stat() both FOLLOW links, and the seed wallpaper this repo deploys is a
			// Home Manager store symlink. No unit test could see this -- the fixture
			// stubs ListDir.
			info, err := os.Stat(full)
			if err != nil {
				// A broken symlink, excluded in Python too.
				continue
			}
			listing = append(listing, DirEntryInfo{
				Path:    full,
				IsFile:  info.Mode().IsRegular(),
				MtimeNS: info.ModTime().UnixNano(),
				Size:    info.Size(),
			})
		}
		return listing, true
	}

	ResolvePath = func(path string) (string, bool) {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", false
		}
		absolute, err := filepath.Abs(resolved)
		if err != nil {
			return "", false
		}
		return absolute, true
	}

	FileExists = func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}
)

// ScanWallpaperFiles sorts, because Path.iterdir does not and the picker grid would
// otherwise reshuffle on every rescan. The sort is on the full path.
func ScanWallpaperFiles(dir string) []string {
	if dir == "" {
		dir = wallpaperDir()
	}
	entries, ok := ListDir(dir)
	if !ok {
		return []string{}
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].Path < entries[b].Path })

	files := []string{}
	for _, entry := range entries {
		if !entry.IsFile {
			continue
		}
		if wallpaperExtensions[strings.ToLower(filepath.Ext(entry.Path))] {
			files = append(files, entry.Path)
		}
	}
	return files
}

// ThumbCachePath's digest input is EXACTLY "<path>:<mtime_ns>:<size>". Getting any
// part wrong -- the separator, the field order, nanoseconds versus seconds --
// invalidates every cached thumbnail at once, at up to 15 s each to rebuild.
func ThumbCachePath(path string, mtimeNS, size int64) string {
	digest := sha1.Sum([]byte(
		path + ":" + strconv.FormatInt(mtimeNS, 10) + ":" + strconv.FormatInt(size, 10)))
	return filepath.Join(thumbDir(), hex.EncodeToString(digest[:])+".png")
}

// EnsureThumbnail returns "" on any failure, so the picker renders a tile with no
// image rather than breaking.
func EnsureThumbnail(path string) string {
	info, ok := statFor(path)
	if !ok {
		return ""
	}
	thumb := ThumbCachePath(path, info.MtimeNS, info.Size)
	if FileExists(thumb) {
		return thumb
	}

	// "[0]" takes the first frame, which is what makes an animated GIF produce
	// a still rather than a multi-frame PNG.
	RunStatus(magickTimeout, "magick", path+"[0]",
		"-thumbnail", "320x200^", "-gravity", "center", "-extent", "320x200", thumb)

	if !FileExists(thumb) {
		return ""
	}
	return thumb
}

func statFor(path string) (DirEntryInfo, bool) {
	entries, ok := ListDir(filepath.Dir(path))
	if !ok {
		return DirEntryInfo{}, false
	}
	for _, entry := range entries {
		if entry.Path == path {
			return entry, true
		}
	}
	return DirEntryInfo{}, false
}

// realPath is the resolved target, or "" for an empty input or an unresolvable
// path. The empty-string guard is redundant and kept: it states that "no current
// wallpaper" is not a path lookup, at the point it applies.
func realPath(path string) string {
	if path == "" {
		return ""
	}
	resolved, ok := ResolvePath(path)
	if !ok {
		return ""
	}
	return resolved
}

type Wallpaper struct {
	Current string            `json:"current"`
	Count   int               `json:"count"`
	Rows    [][]WallpaperItem `json:"rows"`
}

func CollectWallpaper() Wallpaper {
	current := ParseAwwwQuery(RunText(defaultTimeout, "awww", "query"))
	items := WallpaperItems(ScanWallpaperFiles(""), current, realPath, EnsureThumbnail)
	return Wallpaper{
		Current: current,
		Count:   len(items),
		Rows:    RowsFromItems(items, wallpaperColumns),
	}
}

// PersistCurrentWallpaper records the wallpaper now on screen for awww-init to
// redraw at the next login: awww-daemon runs with --no-cache, so this file, not the
// daemon's own cache, is what carries a pick across logins.
//
// Called from the control serve loop, NOT from SetWallpaper: the golden replay pins
// SetWallpaper's and ControlHandle's journals byte-for-byte.
func PersistCurrentWallpaper(current string) {
	if current == "" {
		return
	}
	// Trailing newline so $(cat ...) in awww-init strips it cleanly.
	_ = WriteTextFile(expandUser("~/.local/state/awww/current-wallpaper"), current+"\n")
}

// SetWallpaper swallows the awww failure: the picker re-reads the state afterwards
// either way, so a failed change shows as the old wallpaper still being active.
func SetWallpaper(path string) (Wallpaper, error) {
	if path == "" {
		return Wallpaper{}, errValue("wallpaper set requires a path")
	}
	RunStatus(awwwTimeout, "awww", append([]string{"img", path}, awwwTransition...)...)
	return CollectWallpaper(), nil
}
