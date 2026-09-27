package integrations

import (
	"context"
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
	assert.Equal(t, "Alpha", kinds[0].Name, "kinds are sorted by name")
	assert.Equal(t, "alpha", kinds[0].Kind, "registry fills in Kind")
	assert.Equal(t, "zeta", kinds[len(kinds)-1].Kind)
	for i := 1; i < len(kinds); i++ {
		assert.LessOrEqual(t, kinds[i-1].Name, kinds[i].Name)
	}
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
