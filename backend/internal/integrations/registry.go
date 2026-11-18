package integrations

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

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
	seen := map[string]bool{}
	for _, option := range meta.Options {
		if option.Type != "" && option.Type != OptionNumber {
			panic(fmt.Sprintf("integrations: kind %q option %q has unknown type %q", kind, option.Key, option.Type))
		}
		if option.Key == "" || option.Label == "" || seen[option.Key] {
			panic(fmt.Sprintf("integrations: kind %q has an option without a unique key and a label", kind))
		}
		seen[option.Key] = true
		if _, err := regexp.Compile(option.Pattern); err != nil {
			panic(fmt.Sprintf("integrations: kind %q option %q has a bad pattern: %v", kind, option.Key, err))
		}
	}
	for _, kpi := range meta.KPIs {
		for _, option := range []string{kpi.Option, kpi.Without} {
			if option != "" && !seen[option] {
				panic(fmt.Sprintf("integrations: kind %q KPI %q names unknown option %q", kind, kpi.Key, option))
			}
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
	sort.Slice(metas, func(a, b int) bool {
		nameA, nameB := strings.ToLower(metas[a].Name), strings.ToLower(metas[b].Name)
		if nameA == nameB {
			return metas[a].Kind < metas[b].Kind
		}
		return nameA < nameB
	})
	return metas
}
