// Package configfile keeps one admin's dashboard in sync with a file
// (NIMBUS_CONFIG_FILE): a dashboard document in JSON or YAML plus its owner.
// docs/CONFIGURATION.md describes the file.
package configfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/nimbus/backend/internal/dashdoc"
	"go.yaml.in/yaml/v3"
)

// MaxSize is the largest config file that is read: the size limit of an import
const MaxSize = 10 * 1024 * 1024

// File is a config file: a dashboard document and the admin it belongs to
type File struct {
	// Owner is the email address of an existing admin
	Owner string `json:"owner"`
	dashdoc.Document
	// variables are the values ${VAR} was replaced by: the secrets
	variables []string
}

// Mask returns text without what the file's variables hold, for the log.
// ponytail: only the value as it is; one that an error shows escaped
// (a quote, a backslash) is not recognised
func (f *File) Mask(text string) string {
	for _, value := range f.variables {
		text = strings.ReplaceAll(text, value, "***")
	}
	return text
}

// Parse reads a config file. `${VAR}` in a string value is replaced by what
// lookup gives for VAR (os.LookupEnv); `$${VAR}` gives a literal `${VAR}`.
// The file is parsed first and its values are replaced afterwards, so what a
// variable holds can never change the structure of the file.
func Parse(data []byte, lookup func(string) (string, bool)) (*File, error) {
	if len(data) > MaxSize {
		return nil, fmt.Errorf("the file is larger than %d MB", MaxSize/(1024*1024))
	}
	var tree any
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '{' {
		if err := json.Unmarshal(data, &tree); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	} else if err := yaml.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	if _, ok := tree.(map[string]any); !ok {
		return nil, errors.New("the file must hold a mapping with owner, version and the dashboard")
	}

	values := &interpolation{lookup: lookup, missing: map[string]bool{}, used: map[string]bool{}}
	tree = values.apply(tree)
	if len(values.missing) > 0 {
		names := make([]string, 0, len(values.missing))
		for name := range values.missing {
			names = append(names, name)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("environment variable not set or empty: %s", strings.Join(names, ", "))
	}
	if values.nul {
		return nil, errors.New(`a value holds the character \u0000, which can not be stored`)
	}

	// Through JSON, so a YAML file is checked exactly like a JSON one.
	// Unknown fields are refused: a typo would otherwise be dropped silently.
	raw, err := json.Marshal(tree)
	if err != nil {
		return nil, fmt.Errorf("unsupported value: %w", err)
	}
	var file File
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("invalid config file: %s", strings.TrimPrefix(err.Error(), "json: "))
	}
	if file.Owner = strings.TrimSpace(file.Owner); file.Owner == "" {
		return nil, errors.New("owner is required: the email address of the admin the dashboard belongs to")
	}
	// The longest first, so a value that holds another is masked whole
	for value := range values.used {
		file.variables = append(file.variables, value)
	}
	sort.Slice(file.variables, func(i, j int) bool { return len(file.variables[i]) > len(file.variables[j]) })
	return &file, nil
}

// variable matches ${NAME} and its escape $${NAME}
var variable = regexp.MustCompile(`\$?\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// interpolation replaces the variables of a parsed file and notes what is
// wrong with its values
type interpolation struct {
	lookup func(string) (string, bool)
	// missing are the variables that are not set. An empty one counts too:
	// a secret must never silently become empty.
	missing map[string]bool
	// used are the values the variables were replaced by
	used map[string]bool
	// nul: a value holds the character the database can't store
	nul bool
}

// apply replaces the variables in every string value under value
func (in *interpolation) apply(value any) any {
	switch v := value.(type) {
	case string:
		replaced := variable.ReplaceAllStringFunc(v, func(match string) string {
			if strings.HasPrefix(match, "$$") {
				return match[1:]
			}
			name := match[2 : len(match)-1]
			replacement, ok := in.lookup(name)
			if !ok || replacement == "" {
				in.missing[name] = true
				return ""
			}
			in.used[replacement] = true
			return replacement
		})
		in.nul = in.nul || strings.ContainsRune(replaced, 0)
		return replaced
	case map[string]any:
		for key, item := range v {
			v[key] = in.apply(item)
		}
	case map[any]any:
		// YAML with keys that are not strings; JSON has no such thing
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[fmt.Sprint(key)] = in.apply(item)
		}
		return out
	case []any:
		for i, item := range v {
			v[i] = in.apply(item)
		}
	}
	return value
}
