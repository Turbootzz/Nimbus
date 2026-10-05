package widgets

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// pathStep is one key or array index of a JSON path
type pathStep struct {
	key     string
	index   int
	isIndex bool
}

// parsePath parses a path like "data.items[0].name". A leading "$" is
// allowed, negative indexes count from the end and "length" on a list gives
// its size. ponytail: keys with dots or brackets can't be written; add
// quoting when someone needs it.
func parsePath(path string) ([]pathStep, error) {
	rest := strings.TrimPrefix(strings.TrimPrefix(path, "$"), ".")
	if rest == "" {
		return nil, errors.New("path is empty")
	}

	var steps []pathStep
	for _, part := range strings.Split(rest, ".") {
		key, indexes, hasIndex := strings.Cut(part, "[")
		if key == "" && !hasIndex {
			return nil, fmt.Errorf("path %q has an empty part", path)
		}
		if key != "" {
			steps = append(steps, pathStep{key: key})
		}
		if !hasIndex {
			continue
		}
		// indexes is what follows the first "[", e.g. "0]" or "0][-1]"
		if !strings.HasSuffix(indexes, "]") {
			return nil, fmt.Errorf("path %q has an unclosed [", path)
		}
		for _, index := range strings.Split(strings.TrimSuffix(indexes, "]"), "][") {
			n, err := strconv.Atoi(index)
			if err != nil {
				return nil, fmt.Errorf("path %q has a bad index [%s]", path, index)
			}
			steps = append(steps, pathStep{index: n, isIndex: true})
		}
	}
	return steps, nil
}

// lookupPath walks a decoded JSON document; false when a step is missing
func lookupPath(doc any, steps []pathStep) (any, bool) {
	cur := doc
	for _, step := range steps {
		switch v := cur.(type) {
		case map[string]any:
			next, ok := v[step.key]
			if step.isIndex || !ok {
				return nil, false
			}
			cur = next
		case []any:
			if !step.isIndex {
				if step.key != "length" {
					return nil, false
				}
				cur = len(v)
				continue
			}
			i := step.index
			if i < 0 {
				i += len(v)
			}
			if i < 0 || i >= len(v) {
				return nil, false
			}
			cur = v[i]
		default:
			return nil, false
		}
	}
	return cur, true
}
