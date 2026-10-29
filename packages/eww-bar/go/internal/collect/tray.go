package collect

import "strings"

// TrayItemsFromText reads the same `busctl get-property ... RegisteredStatusNotifierItems`
// output as TrayCountFromText and returns each item's address, a bus name
// followed by an object path.
func TrayItemsFromText(text string) []string {
	parts := SplitWhitespaceN(text, -1)
	if len(parts) < 2 || parts[0] != "as" {
		return nil
	}
	var items []string
	for _, part := range parts[2:] {
		if item := strings.Trim(part, `"`); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// TrayItemAddress splits an item address into the bus name and object path to
// query. A bare bus name gets the specification's default path.
func TrayItemAddress(item string) (string, string) {
	if name, path, ok := strings.Cut(item, "/"); ok {
		return name, "/" + path
	}
	return item, "/StatusNotifierItem"
}

// TrayItemShownFromText reads `busctl get-property ... Status` for one item.
// eww's tray hides only a Passive item, so an unreadable status counts as
// shown.
func TrayItemShownFromText(text string) bool {
	parts := SplitWhitespaceN(text, -1)
	return len(parts) != 2 || parts[0] != "s" || parts[1] != `"Passive"`
}
