package widgets

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

const maxMarkdownRunes = 10000

type markdownConfig struct {
	// Content is rendered in the browser without raw HTML
	Content string `json:"content"`
}

type markdown struct{}

func init() { Register(markdown{}) }

func (markdown) Type() string { return "markdown" }

func (markdown) Meta() Meta {
	return Meta{
		Name:         "Note",
		Category:     CategoryGeneral,
		DefaultSize:  "2x2",
		AllowedSizes: allSizes,
		Static:       true,
	}
}

func (markdown) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[markdownConfig](config)
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(cfg.Content) > maxMarkdownRunes {
		return nil, fmt.Errorf("note must be %d characters or less", maxMarkdownRunes)
	}
	return encodeConfig(cfg)
}
