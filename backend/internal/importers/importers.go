// Package importers turns the config of another dashboard (Homepage, Homarr,
// Homer, Dashy) or a docker-compose file into a dashboard document. The
// transformers are pure: they read bytes and return a document plus warnings
// about what they left out, and never touch the network, the environment or
// the database. Secrets in a config (API keys, passwords) are dropped; a
// `${VAR}` is never resolved. The document goes through the normal import
// (services.DashboardTransferService), which validates all of it again.
package importers

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

// MaxSize is the largest config a transformer reads
const MaxSize = 1024 * 1024

// maxWarnings caps the list; a config that produces more is mostly unusable
const maxWarnings = 100

// Formats
const (
	FormatHomepage = "homepage"
	FormatHomarr   = "homarr"
	FormatHomer    = "homer"
	FormatDashy    = "dashy"
	FormatCompose  = "compose"
)

// Result is a converted config
type Result struct {
	Format   string           `json:"format"`
	Document dashdoc.Document `json:"document"`
	// Warnings tell what was skipped or changed, for the user to read
	Warnings []string `json:"warnings"`
}

// Options are what a conversion can't know from the config
type Options struct {
	// Host is where published docker-compose ports are reached: a compose
	// file doesn't say which machine it runs on. Defaults to localhost.
	Host string
}

var hostPattern = regexp.MustCompile(`^([A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?|\[[0-9A-Fa-f:.]{2,45}\])$`)

// Convert turns a config into a document. format is one of the Format
// constants, or "" or "auto" to detect it. Its errors are safe to show.
func Convert(format string, data []byte, opts Options) (*Result, error) {
	if len(data) > MaxSize {
		return nil, fmt.Errorf("the config is larger than %d MB", MaxSize/(1024*1024))
	}
	if opts.Host == "" {
		opts.Host = "localhost"
	}
	if !hostPattern.MatchString(opts.Host) {
		return nil, errors.New("host must be a host name or an IP address, without scheme or port")
	}
	if format == "" || format == "auto" {
		detected, ok := Detect(data)
		if !ok {
			return nil, errors.New("the format was not recognised; choose it yourself")
		}
		format = detected
	}

	var (
		doc      dashdoc.Document
		warnings []string
		err      error
	)
	switch format {
	case FormatHomepage:
		doc, warnings, err = Homepage(data)
	case FormatHomarr:
		doc, warnings, err = Homarr(data)
	case FormatHomer:
		doc, warnings, err = Homer(data)
	case FormatDashy:
		doc, warnings, err = Dashy(data)
	case FormatCompose:
		doc, warnings, err = Compose(data, opts.Host)
	default:
		return nil, fmt.Errorf("unknown format %q; use homepage, homarr, homer, dashy or compose", format)
	}
	if err != nil {
		return nil, err
	}
	if warnings == nil {
		warnings = []string{}
	}
	return &Result{Format: format, Document: doc, Warnings: warnings}, nil
}

// Detect guesses the format of a config from its shape
func Detect(data []byte) (string, bool) {
	tree, err := parse(data)
	if err != nil {
		return "", false
	}
	switch root := tree.(type) {
	case []any:
		// Homepage's services.yaml and bookmarks.yaml are lists of groups
		if len(root) > 0 {
			if _, ok := root[0].(map[string]any); ok {
				return FormatHomepage, true
			}
		}
	case map[string]any:
		if _, ok := root["apps"].([]any); ok {
			return FormatHomarr, true
		}
		if _, ok := root["sections"].([]any); ok {
			return FormatDashy, true
		}
		switch root["services"].(type) {
		case []any:
			return FormatHomer, true
		case map[string]any:
			return FormatCompose, true
		}
	}
	return "", false
}

// parse reads JSON or YAML into maps with string keys, lists and scalars
func parse(data []byte) (any, error) {
	if len(data) > MaxSize {
		return nil, fmt.Errorf("the config is larger than %d MB", MaxSize/(1024*1024))
	}
	var tree any
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if json.Unmarshal(data, &tree) == nil {
			return tree, nil
		}
	}
	if err := yaml.Unmarshal(data, &tree); err != nil {
		return nil, errors.New("the config is not valid YAML or JSON")
	}
	return stringKeys(tree), nil
}

// stringKeys turns the map[any]any that YAML gives for keys like `8080:` or
// `true:` into maps with string keys
func stringKeys(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			v[key] = stringKeys(item)
		}
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[fmt.Sprint(key)] = stringKeys(item)
		}
		return out
	case []any:
		for i, item := range v {
			v[i] = stringKeys(item)
		}
	}
	return value
}

// Helpers to read a parsed config that may hold anything anywhere

func asMap(value any) map[string]any {
	m, _ := value.(map[string]any)
	return m
}

func asList(value any) []any {
	l, _ := value.([]any)
	return l
}

// asString returns a scalar as text; maps, lists and nil give ""
func asString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case nil, map[string]any, []any:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

// sortedKeys gives a map's keys in a stable order
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
