package widgets

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubType struct {
	typ  string
	meta Meta
}

func (s stubType) Type() string { return s.typ }
func (s stubType) Meta() Meta   { return s.meta }
func (s stubType) Validate(c json.RawMessage) (json.RawMessage, error) {
	return c, nil
}

// register adds a type for one test and removes it afterwards
func register(t *testing.T, w WidgetType) {
	t.Helper()
	Register(w)
	t.Cleanup(func() { delete(registry, w.Type()) })
}

func validMeta(name string) Meta {
	return Meta{Name: name, Category: CategoryGeneral, DefaultSize: "2x1", AllowedSizes: allSizes}
}

func TestRegister_GetAndTypes(t *testing.T) {
	register(t, stubType{typ: "zz-stub", meta: validMeta("ZZ Stub")})

	got, ok := Get("zz-stub")
	assert.True(t, ok)
	assert.Equal(t, "zz-stub", got.Type())

	_, ok = Get("missing")
	assert.False(t, ok)

	types := Types()
	assert.Equal(t, "zz-stub", types[len(types)-1].Type, "registry fills in Type, sorted by name")
	for i := 1; i < len(types); i++ {
		assert.LessOrEqual(t, types[i-1].Name, types[i].Name)
	}
}

func TestBuiltinStaticTypes(t *testing.T) {
	for _, typ := range []string{"clock", "markdown", "bookmarks", "iframe"} {
		w, ok := Get(typ)
		if assert.True(t, ok, typ) {
			assert.True(t, w.Meta().Static, typ)
			assert.Empty(t, w.Meta().IntegrationKinds, typ)
		}
	}
}

func TestRegister_PanicsOnProgrammerErrors(t *testing.T) {
	register(t, stubType{typ: "dup", meta: validMeta("Dup")})

	noCategory := validMeta("No category")
	noCategory.Category = ""
	badDefault := validMeta("Bad default")
	badDefault.AllowedSizes = []string{"1x1"}
	badSize := validMeta("Bad size")
	badSize.AllowedSizes = []string{"2x1", "3x3"}

	cases := map[string]WidgetType{
		"empty type":      stubType{typ: "", meta: validMeta("Empty")},
		"duplicate type":  stubType{typ: "dup", meta: validMeta("Dup")},
		"no name":         stubType{typ: "noname", meta: validMeta("")},
		"no category":     stubType{typ: "nocat", meta: noCategory},
		"default size":    stubType{typ: "baddefault", meta: badDefault},
		"unknown size":    stubType{typ: "badsize", meta: badSize},
		"no sizes at all": stubType{typ: "nosizes", meta: Meta{Name: "N", Category: CategoryGeneral}},
	}
	for name, w := range cases {
		assert.Panics(t, func() { Register(w) }, name)
	}
}
