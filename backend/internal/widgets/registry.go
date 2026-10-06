package widgets

import (
	"fmt"
	"slices"
	"sort"

	"github.com/nimbus/backend/internal/models"
)

// registry is written only from init(), so reads need no lock
var registry = map[string]WidgetType{}

// Register adds a type. Call it from the type's init(); it panics on
// programmer errors so a broken type fails at startup, not at runtime.
func Register(w WidgetType) {
	typ := w.Type()
	if typ == "" {
		panic("widgets: type must not be empty")
	}
	if _, dup := registry[typ]; dup {
		panic(fmt.Sprintf("widgets: type %q registered twice", typ))
	}
	meta := w.Meta()
	if meta.Name == "" || meta.Category == "" {
		panic(fmt.Sprintf("widgets: type %q needs a name and a category", typ))
	}
	if len(meta.AllowedSizes) == 0 || !slices.Contains(meta.AllowedSizes, meta.DefaultSize) {
		panic(fmt.Sprintf("widgets: type %q default size must be one of its allowed sizes", typ))
	}
	if _, fetches := w.(Fetcher); fetches == meta.Static {
		panic(fmt.Sprintf("widgets: type %q must be static or implement Fetcher, not both or neither", typ))
	}
	for _, size := range meta.AllowedSizes {
		if !models.IsValidWidgetCardSize(size) {
			panic(fmt.Sprintf("widgets: type %q has unknown size %q", typ, size))
		}
	}
	registry[typ] = w
}

// Get returns the type with the given id
func Get(typ string) (WidgetType, bool) {
	w, ok := registry[typ]
	return w, ok
}

// Types returns the metadata of all registered types, sorted by name
func Types() []Meta {
	metas := make([]Meta, 0, len(registry))
	for typ, w := range registry {
		meta := w.Meta()
		meta.Type = typ
		metas = append(metas, meta)
	}
	sort.Slice(metas, func(a, b int) bool { return metas[a].Name < metas[b].Name })
	return metas
}
