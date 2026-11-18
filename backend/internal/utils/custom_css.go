package utils

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxCustomCSSBytes caps a user's custom CSS
const MaxCustomCSSBytes = 20 * 1024

// Never valid in custom CSS: other ways to load a resource, the script hooks
// of old browsers, and the end of the <style> element it is shown in
var forbiddenCSS = []struct{ token, message string }{
	{"@import", "@import is not allowed"},
	{"image-set(", "image-set() is not allowed; use url() with a data: image"},
	{"src(", "src() is not allowed"},
	{"expression(", "expression() is not allowed"},
	{"javascript:", "javascript: is not allowed"},
	{"-moz-binding", "-moz-binding is not allowed"},
	{"</style", "</style is not allowed"},
}

// ValidateCustomCSS rejects CSS that could load anything from another host
// or leave its <style> element. The CSS only styles its owner's pages today,
// but a theme can be shared as a file, so it must not be able to send data
// anywhere: no @import, and url() only with an inline data: image. Without
// a request there is also no way to leak what a selector matched.
// It rejects instead of removing: removing can join the rest into a new
// token, and the user should see what was refused.
func ValidateCustomCSS(css string) error {
	if len(css) > MaxCustomCSSBytes {
		return errors.New("custom CSS is larger than 20 KB")
	}
	if !utf8.ValidString(css) || strings.ContainsRune(css, 0) {
		return errors.New("custom CSS must be plain text")
	}
	// Browsers read \75rl( as url( and ignore case, so check what they see.
	// Comments are checked both kept and removed: stricter than a browser.
	decoded := strings.ToLower(decodeCSSEscapes(css))
	for _, text := range []string{decoded, stripCSSComments(decoded)} {
		for _, forbidden := range forbiddenCSS {
			if strings.Contains(text, forbidden.token) {
				return errors.New(forbidden.message)
			}
		}
		for rest := text; ; {
			i := strings.Index(rest, "url(")
			if i < 0 {
				break
			}
			rest = rest[i+len("url("):]
			// At most one quote: a second one would be part of the URL,
			// which then is a relative one
			target := strings.TrimLeft(rest, " \t\n")
			if strings.HasPrefix(target, `"`) || strings.HasPrefix(target, `'`) {
				target = target[1:]
			}
			if !strings.HasPrefix(target, "data:image/") {
				return errors.New("url() may only hold a data: image")
			}
		}
	}
	return nil
}

// decodeCSSEscapes resolves backslash escapes the way a CSS parser does:
// up to six hex digits and one optional space are a code point, a backslash
// before a newline is nothing, any other escaped character is itself.
// An escaped quote is a character, not the start of a string, so it comes
// out as U+FFFD and can't pass for the quote around a url().
func decodeCSSEscapes(css string) string {
	// A parser first turns every kind of newline into \n
	css = cssNewlines.Replace(css)
	if !strings.Contains(css, `\`) {
		return css
	}
	var out strings.Builder
	runes := []rune(css)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '\\' {
			out.WriteRune(runes[i])
			continue
		}
		i++
		if i == len(runes) {
			break
		}
		end := i
		for end < len(runes) && end < i+6 && isHexDigit(runes[end]) {
			end++
		}
		if end == i {
			if runes[i] != '\n' {
				out.WriteRune(unquote(runes[i]))
			}
			continue
		}
		code, _ := strconv.ParseInt(string(runes[i:end]), 16, 32)
		if code == 0 || code > utf8.MaxRune {
			code = utf8.RuneError
		}
		out.WriteRune(unquote(rune(code)))
		// One whitespace character after the digits belongs to the escape
		if end < len(runes) && (runes[end] == ' ' || runes[end] == '\t' || runes[end] == '\n') {
			end++
		}
		i = end - 1
	}
	return out.String()
}

func unquote(r rune) rune {
	if r == '"' || r == '\'' {
		return utf8.RuneError
	}
	return r
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

var cssNewlines = strings.NewReplacer("\r\n", "\n", "\r", "\n", "\f", "\n")

// stripCSSComments removes /* ... */; an open comment runs to the end
func stripCSSComments(css string) string {
	var out strings.Builder
	for {
		start := strings.Index(css, "/*")
		if start < 0 {
			return out.String() + css
		}
		out.WriteString(css[:start])
		end := strings.Index(css[start+2:], "*/")
		if end < 0 {
			return out.String()
		}
		css = css[start+2+end+2:]
	}
}
