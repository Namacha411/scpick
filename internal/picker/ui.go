package picker

import (
	"fmt"

	"github.com/ktr0731/go-fuzzyfinder"
)

// PickFiles opens an interactive fuzzy-finder over items, letting the user
// tag zero or more entries (Ctrl+Space) before confirming with Enter. It is
// used for file-pick mode, where multiple files may be selected at once.
// Not covered by automated tests; verify manually per SPEC.md.
func PickFiles(items []ListItem) ([]ListItem, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("picker: pick files: no items to show")
	}
	idxs, err := fuzzyfinder.FindMulti(items, func(i int) string { return items[i].Label })
	if err != nil {
		return nil, fmt.Errorf("picker: pick files: %w", err)
	}
	selected := make([]ListItem, len(idxs))
	for i, idx := range idxs {
		selected[i] = items[idx]
	}
	return selected, nil
}

// PickOne opens an interactive fuzzy-finder over items and returns the
// single selected item. Used for dir-pick mode and for host selection.
// Not covered by automated tests; verify manually per SPEC.md.
func PickOne(items []ListItem) (ListItem, error) {
	if len(items) == 0 {
		return ListItem{}, fmt.Errorf("picker: pick one: no items to show")
	}
	idx, err := fuzzyfinder.Find(items, func(i int) string { return items[i].Label })
	if err != nil {
		return ListItem{}, fmt.Errorf("picker: pick one: %w", err)
	}
	return items[idx], nil
}
