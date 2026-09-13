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

// WallpaperItems injects realPath and thumb because both touch the filesystem.
// resolve() is what makes the active highlight work: awww query reports the resolved
// target, and Home Manager deploys the seed as a store symlink, so comparing literal
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

// PathName is pathlib.PurePath.name, not filepath.Base, which disagrees on four
// shapes: Base(""), Base("."), Base("/") and Base("/w/.") give ".", ".", "/" and
// "." where pathlib gives "", "", "" and "w".
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

// PathStem is pathlib.PurePath.stem. filepath.Ext is not the suffix rule --
// Ext(".bashrc") is ".bashrc" -- while pathlib ignores LEADING dots entirely and
// then splits at the last remaining one, so ".bashrc" and "..a" have no suffix
// while ".a.b" has ".b".
//
// Targets CPython 3.14, where Path("a.").suffix is "." and the stem is "a"; earlier
// versions gave "a.".
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

// RowsFromItems returns [] rather than nil for an empty input: eww.yuck iterates
// rows with (for row in ...), and a nil slice marshals to null, which eww cannot
// index.
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
