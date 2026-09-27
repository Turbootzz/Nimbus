package integrations

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

type stubKind struct {
	kind string
	meta Meta
}

func (s stubKind) Kind() string                                   { return s.kind }
func (s stubKind) Meta() Meta                                     { return s.meta }
func (s stubKind) Test(context.Context, *Conn) error              { return nil }
func (s stubKind) Fetch(context.Context, *Conn) (*Payload, error) { return &Payload{}, nil }

// register adds a kind for one test and removes it afterwards
func register(t *testing.T, i Integration) {
	t.Helper()
	Register(i)
	t.Cleanup(func() { delete(registry, i.Kind()) })
}

func TestRegister_GetAndKinds(t *testing.T) {
	register(t, stubKind{kind: "zeta", meta: Meta{Name: "Zeta", AuthTypes: []string{models.IntegrationAuthNone}}})
	register(t, stubKind{kind: "alpha", meta: Meta{Name: "Alpha", AuthTypes: []string{models.IntegrationAuthAPIKey}}})

	got, ok := Get("alpha")
	assert.True(t, ok)
	assert.Equal(t, "alpha", got.Kind())

	_, ok = Get("missing")
	assert.False(t, ok)

	kinds := Kinds()
	byKind := map[string]Meta{}
	for i, meta := range kinds {
		byKind[meta.Kind] = meta
		if i > 0 {
			assert.LessOrEqual(t, strings.ToLower(kinds[i-1].Name), strings.ToLower(meta.Name), "kinds are sorted by name")
		}
	}
	assert.Equal(t, "Alpha", byKind["alpha"].Name, "registry fills in Kind")
	assert.Contains(t, byKind, "zeta")
}

func TestKinds_SameNameSortsByKind(t *testing.T) {
	meta := Meta{Name: "Same Name", AuthTypes: []string{models.IntegrationAuthNone}}
	register(t, stubKind{kind: "same-b", meta: meta})
	register(t, stubKind{kind: "same-a", meta: meta})

	var order []string
	for _, k := range Kinds() {
		if k.Name == "Same Name" {
			order = append(order, k.Kind)
		}
	}
	assert.Equal(t, []string{"same-a", "same-b"}, order)
}

func TestKinds_EmptyIsNotNil(t *testing.T) {
	assert.NotNil(t, Kinds(), "empty registry must encode as [] not null")
}

func TestRegister_PanicsOnProgrammerErrors(t *testing.T) {
	valid := Meta{Name: "Dup", AuthTypes: []string{models.IntegrationAuthNone}}
	register(t, stubKind{kind: "dup", meta: valid})

	cases := map[string]Integration{
		"empty kind":        stubKind{kind: "", meta: valid},
		"duplicate kind":    stubKind{kind: "dup", meta: valid},
		"no auth types":     stubKind{kind: "noauth", meta: Meta{Name: "No auth"}},
		"unknown auth type": stubKind{kind: "badauth", meta: Meta{Name: "Bad", AuthTypes: []string{"oauth"}}},
	}
	for name, i := range cases {
		assert.Panics(t, func() { Register(i) }, name)
	}
}

func TestState_ConcurrentUse(t *testing.T) {
	var s State
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Set("sid", "abc")
			s.Get("sid")
		}()
	}
	wg.Wait()

	v, ok := s.Get("sid")
	assert.True(t, ok)
	assert.Equal(t, "abc", v)

	s.Delete("sid")
	_, ok = s.Get("sid")
	assert.False(t, ok)
}
