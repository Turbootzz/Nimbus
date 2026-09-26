package integrations

import (
	"fmt"
	"sort"

	"github.com/nimbus/backend/internal/models"
)

// registry is written only from init(), so reads need no lock.
var registry = map[string]Integration{}

// Register adds a kind. Call it from the kind's init(); it panics on
// programmer errors so a broken kind fails at startup, not at runtime.
func Register(i Integration) {
	kind := i.Kind()
	if kind == "" {
		panic("integrations: kind must not be empty")
	}
	if _, dup := registry[kind]; dup {
		panic(fmt.Sprintf("integrations: kind %q registered twice", kind))
	}
	meta := i.Meta()
	if len(meta.AuthTypes) == 0 {
		panic(fmt.Sprintf("integrations: kind %q has no auth types", kind))
	}
	for _, authType := range meta.AuthTypes {
		if !models.IsValidIntegrationAuthType(authType) {
			panic(fmt.Sprintf("integrations: kind %q has unknown auth type %q", kind, authType))
		}
	}
	registry[kind] = i
}

// Get returns the kind with the given id
func Get(kind string) (Integration, bool) {
	i, ok := registry[kind]
	return i, ok
}

// Kinds returns the metadata of all registered kinds, sorted by name
func Kinds() []Meta {
	metas := make([]Meta, 0, len(registry))
	for kind, i := range registry {
		meta := i.Meta()
		meta.Kind = kind
		metas = append(metas, meta)
	}
	sort.Slice(metas, func(a, b int) bool { return metas[a].Name < metas[b].Name })
	return metas
}
