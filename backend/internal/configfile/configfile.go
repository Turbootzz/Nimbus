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
	"strconv"
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
	// values are the string values of the file and what its variables held:
	// what the log must not repeat
	values map[string]bool
}

// minLoggedSecret is the length from which a value of the file counts as
// something the log must not repeat; shorter ones ("1x1", a unit) are in
// half the messages and are no secrets
const minLoggedSecret = 4

// Reason describes why applying the file failed, for the log, without
// repeating a value of the file: a message that quotes one (a URL that is
// not valid, say) is left out and only where it is wrong remains.
func (f *File) Reason(err error) string {
	var invalid *dashdoc.ValidationError
	if !errors.As(err, &invalid) {
		return f.safe(err.Error())
	}
	parts := make([]string, len(invalid.Issues))
	for i, issue := range invalid.Issues {
		parts[i] = issue.Path + ": " + f.safe(issue.Message)
	}
	return "invalid: " + strings.Join(parts, "; ")
}

// safe returns text, unless it holds a value of the file
func (f *File) safe(text string) string {
	for value := range f.values {
		if strings.Contains(text, value) {
			return "(not shown: the message quotes a value of the file)"
		}
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
			return nil, fmt.Errorf("invalid JSON: %s", jsonProblem(data, err))
		}
	} else if err := yaml.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("invalid YAML: %s", yamlProblem(err))
	}
	if _, ok := tree.(map[string]any); !ok {
		return nil, errors.New("the file must hold a mapping with owner, version and the dashboard")
	}

	values := &interpolation{lookup: lookup, missing: map[string]bool{}, seen: map[string]bool{}}
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
	file.values = values.seen
	return &file, nil
}

// The parsers can quote the file in their messages (an alias name, a value
// with a tag, a key, a character), and the file holds secrets: the log gets
// the line and a description that quotes nothing.

// yamlMessage splits a YAML error into its line, when it has one, and its
// description
var yamlMessage = regexp.MustCompile(`^yaml: (?:unmarshal errors:\n  )?(?:line (\d+): )?(.*)`)

// plainText quotes nothing of the file: words, and at most a single quoted
// punctuation mark like ']'
var plainText = regexp.MustCompile(`^(?:[A-Za-z0-9 ,:<>-]|'[^\sA-Za-z0-9]')+$`)

// parseProblem is what the log may tell about a parse error
func parseProblem(line int, description string) string {
	if !plainText.MatchString(description) {
		description = "the parser's description is not shown, it quotes the file"
	}
	if line > 0 {
		return fmt.Sprintf("line %d: %s", line, description)
	}
	return description
}

func yamlProblem(err error) string {
	match := yamlMessage.FindStringSubmatch(err.Error())
	if match == nil {
		return parseProblem(0, "")
	}
	line, _ := strconv.Atoi(match[1])
	return parseProblem(line, match[2])
}

func jsonProblem(data []byte, err error) string {
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		return parseProblem(0, "")
	}
	read := data[:min(max(syntax.Offset, 0), int64(len(data)))]
	return parseProblem(1+bytes.Count(read, []byte("\n")), syntax.Error())
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
	// seen are the string values and what the variables held (File.values)
	seen map[string]bool
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
			in.seen[replacement] = true
			return replacement
		})
		if len(replaced) >= minLoggedSecret {
			in.seen[replaced] = true
		}
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
