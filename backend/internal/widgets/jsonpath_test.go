package widgets

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePathErrors(t *testing.T) {
	for _, path := range []string{"", "$", "$.", "a..b", "a.", "a[", "a[x]", "a[]", "a[0]b", "a[0]]"} {
		_, err := parsePath(path)
		assert.Error(t, err, path)
	}
}

func TestLookupPath(t *testing.T) {
	var doc any
	require.NoError(t, json.Unmarshal([]byte(`{
		"data": {"name": "nimbus", "count": 3, "on": true, "none": null, "length": 7},
		"items": [{"id": 1}, {"id": 2}, {"id": 3}],
		"matrix": [[1, 2], [3, 4]]
	}`), &doc))

	cases := []struct {
		path string
		want any
	}{
		{"data.name", "nimbus"},
		{"$.data.count", float64(3)},
		{"data.on", true},
		{"data.none", nil},
		{"data.length", float64(7)}, // a key wins over the list size
		{"items[0].id", float64(1)},
		{"items[-1].id", float64(3)},
		{"items.length", 3},
		{"matrix[1][0]", float64(3)},
		{"matrix[0].length", 2},
	}
	for _, tc := range cases {
		steps, err := parsePath(tc.path)
		require.NoError(t, err, tc.path)
		got, ok := lookupPath(doc, steps)
		assert.True(t, ok, tc.path)
		assert.Equal(t, tc.want, got, tc.path)
	}

	var list any
	require.NoError(t, json.Unmarshal([]byte(`[{"id": "first"}]`), &list))
	steps, err := parsePath("$[0].id")
	require.NoError(t, err)
	got, ok := lookupPath(list, steps)
	assert.True(t, ok)
	assert.Equal(t, "first", got)

	for _, path := range []string{"missing", "data.name.more", "items[3]", "items[-4]", "items.id", "data[0]", "items.length.x"} {
		steps, err := parsePath(path)
		require.NoError(t, err, path)
		_, ok := lookupPath(doc, steps)
		assert.False(t, ok, path)
	}
}
