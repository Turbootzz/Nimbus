package widgets

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/nimbus/backend/internal/utils"
)

const (
	maxBookmarks         = 50
	maxBookmarkNameRunes = 100
	maxBookmarkIconRunes = 32
)

type bookmark struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	// Icon is an emoji or an image URL; empty shows the site's initial
	Icon string `json:"icon"`
}

type bookmarksConfig struct {
	Items []bookmark `json:"items"`
}

type bookmarks struct{}

func init() { Register(bookmarks{}) }

func (bookmarks) Type() string { return "bookmarks" }

func (bookmarks) Meta() Meta {
	return Meta{
		Name:         "Bookmarks",
		Category:     CategoryGeneral,
		DefaultSize:  "2x2",
		AllowedSizes: allSizes,
		Static:       true,
	}
}

func (bookmarks) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[bookmarksConfig](config)
	if err != nil {
		return nil, err
	}
	if len(cfg.Items) > maxBookmarks {
		return nil, fmt.Errorf("at most %d bookmarks are allowed", maxBookmarks)
	}
	if cfg.Items == nil {
		cfg.Items = []bookmark{}
	}

	for i := range cfg.Items {
		item := &cfg.Items[i]
		item.Name = strings.TrimSpace(item.Name)
		item.URL = strings.TrimSpace(item.URL)
		item.Icon = strings.TrimSpace(item.Icon)

		n := i + 1
		if item.Name == "" || utf8.RuneCountInString(item.Name) > maxBookmarkNameRunes {
			return nil, fmt.Errorf("bookmark %d needs a name of at most %d characters", n, maxBookmarkNameRunes)
		}
		// http(s) only, so a bookmark can never be a javascript: link
		if err := utils.ValidateIconURL(item.URL); err != nil {
			return nil, fmt.Errorf("bookmark %d: invalid URL: %s", n, err.Error())
		}
		if err := validateBookmarkIcon(item.Icon); err != nil {
			return nil, fmt.Errorf("bookmark %d: %s", n, err.Error())
		}
	}
	return encodeConfig(cfg)
}

func validateBookmarkIcon(icon string) error {
	if strings.HasPrefix(icon, "http://") || strings.HasPrefix(icon, "https://") {
		if err := utils.ValidateIconURL(icon); err != nil {
			return fmt.Errorf("invalid icon URL: %s", err.Error())
		}
		return nil
	}
	if utf8.RuneCountInString(icon) > maxBookmarkIconRunes {
		return fmt.Errorf("icon must be an emoji or an image URL")
	}
	return nil
}
