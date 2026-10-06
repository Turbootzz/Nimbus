package widgets

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/utils"
)

// The embed fills its card, so there is no height setting
type iframeConfig struct {
	URL string `json:"url"`
}

type iframe struct{}

func init() { Register(iframe{}) }

func (iframe) Type() string { return "iframe" }

func (iframe) Meta() Meta {
	return Meta{
		Name:        "Embed",
		Category:    CategoryGeneral,
		DefaultSize: "2x2",
		// A page needs height; one row is too short to use
		AllowedSizes: []string{models.CardSize1x2, models.CardSize2x2},
		Static:       true,
	}
}

func (iframe) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[iframeConfig](config)
	if err != nil {
		return nil, err
	}

	// The browser loads the page, not the server, so private hosts are fine.
	// http(s) only keeps javascript: and data: URLs out.
	cfg.URL = strings.TrimSpace(cfg.URL)
	if err := utils.ValidateIconURL(cfg.URL); err != nil {
		return nil, fmt.Errorf("invalid URL: %s", err.Error())
	}

	return encodeConfig(cfg)
}
