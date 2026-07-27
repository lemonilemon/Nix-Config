package collect

import (
	"path/filepath"
	"strings"
)

const (
	wallpaperNameMax = 22
	wallpaperColumns = 3
)

// WallpaperItem is one entry of the wallpaper picker grid.
type WallpaperItem struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Thumb    string `json:"thumb"`
	Animated string `json:"animated"`
	Active   string `json:"active"`
}

// WallpaperItems mirrors wallpaper.wallpaper_items.
//
// realPath and thumb are injected because both touch the filesystem; the
// original takes thumb_fn for the same reason. resolve() is what makes the
// active highlight work at all: awww query reports the resolved target, and
// Home Manager deploys the seed as a store symlink, so comparing the literal
// strings never matches for the seeded wallpaper.
func WallpaperItems(
	files []string,
	current string,
	realPath func(string) string,
	thumb func(string) string,
) []WallpaperItem {
	currentReal := realPath(current)
	items := []WallpaperItem{}
	for _, path := range files {
		animated := "false"
		if strings.ToLower(filepath.Ext(path)) == ".gif" {
			animated = "true"
		}
		active := "false"
		if currentReal != "" && realPath(path) == currentReal {
			active = "true"
		}
		items = append(items, WallpaperItem{
			Name:     TruncateText(PathStem(path), wallpaperNameMax),
			Path:     path,
			Thumb:    thumb(path),
			Animated: animated,
			Active:   active,
		})
	}
	return items
}

// PathName is pathlib.PurePath.name.
//
// Not filepath.Base, which disagrees on four shapes: Base("") and Base(".") are
// ".", Base("/") is "/", and Base("/w/.") is "." -- where pathlib gives "", "",
// "" and "w", because it drops "." components during normalisation and has no
// name for a bare root.
func PathName(path string) string {
	name := ""
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." {
			continue
		}
		name = part
	}
	return name
}

// PathStem is pathlib.PurePath.stem.
//
// filepath.Ext is not the suffix rule -- Ext(".bashrc") is ".bashrc", so
// trimming it would leave "". pathlib ignores LEADING dots entirely and then
// splits at the last remaining one, which is why ".bashrc" and "..a" have no
// suffix while ".a.b" has ".b".
//
// Derived by probing CPython rather than read off the source, because the rule
// changed: on 3.14 Path("a.").suffix is "." and the stem is "a", where earlier
// versions treated a trailing dot as no suffix at all and gave "a.". This
// targets the 3.14 the daemon runs on. If the interpreter is ever upgraded, the
// equivalence gate is what will notice.
func PathStem(path string) string {
	name := PathName(path)
	stripped := strings.TrimLeft(name, ".")
	dot := strings.LastIndex(stripped, ".")
	if dot < 0 {
		return name
	}
	suffixLen := len(stripped) - dot
	return name[:len(name)-suffixLen]
}

// RowsFromItems mirrors wallpaper.rows_from_items.
//
// Returns [] rather than nil for an empty input: eww.yuck iterates rows with
// (for row in ...), and a nil slice marshals to null, which eww cannot index.
func RowsFromItems(items []WallpaperItem, columns int) [][]WallpaperItem {
	rows := [][]WallpaperItem{}
	if columns <= 0 {
		return rows
	}
	for start := 0; start < len(items); start += columns {
		end := start + columns
		if end > len(items) {
			end = len(items)
		}
		rows = append(rows, items[start:end])
	}
	return rows
}

// WallpaperColumns is the grid width the popup lays out.
const WallpaperColumns = wallpaperColumns
