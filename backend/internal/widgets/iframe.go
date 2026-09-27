package widgets

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nimbus/backend/internal/utils"
)

const (
	defaultIframeHeight = 300
	minIframeHeight     = 100
	maxIframeHeight     = 1200
)

type iframeConfig struct {
	URL string `json:"url"`
	// Height in pixels
	Height int `json:"height"`
}

type iframe struct{}

func init() { Register(iframe{}) }

func (iframe) Type() string { return "iframe" }

func (iframe) Meta() Meta {
	return Meta{
		Name:         "Embed",
		Category:     CategoryGeneral,
		DefaultSize:  "2x2",
		AllowedSizes: allSizes,
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

	if cfg.Height == 0 {
		cfg.Height = defaultIframeHeight
	}
	if cfg.Height < minIframeHeight || cfg.Height > maxIframeHeight {
		return nil, fmt.Errorf("height must be between %d and %d pixels", minIframeHeight, maxIframeHeight)
	}
	return encodeConfig(cfg)
}
