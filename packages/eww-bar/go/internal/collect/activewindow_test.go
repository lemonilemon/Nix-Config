package collect

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Four entries -- kitty, firefox, spotify, thunar -- shipped for years as a bare
// space, byte-identical to the fallback for an unknown app. Nothing caught it because
// a missing Nerd Font glyph and a space look the same in an editor, a terminal, a
// diff and the bar itself. So this asserts on the bytes: every value must carry a
// code point outside ASCII.
func TestAppIconsHoldRealGlyphs(t *testing.T) {
	for class, icon := range appIcons {
		if icon == appIconFallback {
			t.Errorf("%q maps to the fallback, so the entry does nothing", class)
			continue
		}
		if !strings.HasSuffix(icon, " ") {
			t.Errorf("%q = %q: needs a trailing space to separate it from the class",
				class, icon)
		}
		glyph := strings.TrimSuffix(icon, " ")
		r, size := utf8.DecodeRuneInString(glyph)
		switch {
		case r == utf8.RuneError:
			t.Errorf("%q = %q: not valid UTF-8", class, icon)
		case r < 0x80:
			t.Errorf("%q = %q: starts with ASCII %q, so the glyph was lost",
				class, icon, r)
		case size != len(glyph):
			t.Errorf("%q = %q: expected exactly one glyph, got %d bytes",
				class, icon, len(glyph))
		}
	}
}

// lookupAppIcon folds the class before indexing, so a key that is not already folded
// is unreachable -- and unreachable in the quiet way, where the app just never gets
// its icon.
func TestAppIconKeysAreNormalised(t *testing.T) {
	for class := range appIcons {
		if folded := normalizeClass(class); folded != class {
			t.Errorf("key %q is unreachable: lookupAppIcon would index %q",
				class, folded)
		}
	}
}

func TestNormalizeClass(t *testing.T) {
	cases := []struct {
		class string
		want  string
	}{
		{"kitty", "kitty"},
		// XWayland reports the capitalised X11 WM_CLASS for the same app whose
		// Wayland app_id is lowercase.
		{"Code", "code"},
		{"1Password", "1password"},
		{"org.wireshark.Wireshark", "org.wireshark.wireshark"},
		// NixOS wraps the binary and the app takes its class from argv[0].
		{".blueman-manager-wrapped", "blueman-manager"},
		{".gopeed-wrapped", "gopeed"},
		// A hyphen that is not the wrapper suffix survives.
		{"nwg-displays", "nwg-displays"},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeClass(c.class); got != c.want {
			t.Errorf("normalizeClass(%q) = %q, want %q", c.class, got, c.want)
		}
	}
}

func TestLookupAppIcon(t *testing.T) {
	kitty := appIcons["kitty"]

	cases := []struct {
		name  string
		class string
		want  string
	}{
		{"known class", "kitty", kitty},
		{"capitalised by XWayland", "Kitty", kitty},
		{"wrapped by nix", ".kitty-wrapped", kitty},
		{"unknown class falls back", "some-app-nobody-mapped", appIconFallback},
		{"empty class falls back", "", appIconFallback},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := lookupAppIcon(c.class); got != c.want {
				t.Errorf("lookupAppIcon(%q) = %q, want %q", c.class, got, c.want)
			}
		})
	}
}

func TestActiveWindowStateFrom(t *testing.T) {
	kitty := appIcons["kitty"]

	cases := []struct {
		name string
		json string
		want ActiveWindowState
	}{
		{
			name: "class and title",
			json: `{"class":"kitty","title":"nvim"}`,
			want: ActiveWindowState{
				Text:    kitty + "kitty · nvim",
				Tooltip: "kitty\nnvim",
				Class:   "kitty",
			},
		},
		{
			name: "class only drops the separator",
			json: `{"class":"kitty","title":""}`,
			want: ActiveWindowState{
				Text:    kitty + "kitty",
				Tooltip: "kitty\n",
				Class:   "kitty",
			},
		},
		{
			// Nothing focused. hyprctl answers with an empty object between
			// closing one window and focusing the next, and the bar has to show
			// an empty label rather than a lone icon.
			name: "empty object is the zero state",
			json: `{}`,
			want: ActiveWindowState{},
		},
		{
			// A dead hyprctl, a timeout, or a socket that closed early all
			// arrive here as an empty string.
			name: "unparseable input is the zero state",
			json: ``,
			want: ActiveWindowState{},
		},
		{
			name: "unknown class still renders, without an icon",
			json: `{"class":"nobody-mapped-this","title":"x"}`,
			want: ActiveWindowState{
				Text:    appIconFallback + "nobody-mapped-this · x",
				Tooltip: "nobody-mapped-this\nx",
				Class:   "nobody-mapped-this",
			},
		},
		{
			// The tooltip is deliberately NOT truncated: it is the place to
			// read a title the bar had to cut.
			name: "long title is truncated in the label only",
			json: `{"class":"kitty","title":"` + strings.Repeat("a", 70) + `"}`,
			want: ActiveWindowState{
				Text:    kitty + "kitty · " + strings.Repeat("a", 57) + "...",
				Tooltip: "kitty\n" + strings.Repeat("a", 70),
				Class:   "kitty",
			},
		},
		{
			// Titles carry whatever the app put there. Counting runes rather
			// than bytes is what keeps a CJK title from being cut mid-character.
			name: "multibyte title truncates by rune",
			json: `{"class":"kitty","title":"` + strings.Repeat("界", 70) + `"}`,
			want: ActiveWindowState{
				Text:    kitty + "kitty · " + strings.Repeat("界", 57) + "...",
				Tooltip: "kitty\n" + strings.Repeat("界", 70),
				Class:   "kitty",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ActiveWindowStateFrom(c.json)
			if got != c.want {
				t.Errorf("ActiveWindowStateFrom(%q):\n got %#v\nwant %#v",
					c.json, got, c.want)
			}
		})
	}
}
