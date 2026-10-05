package widgets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/nimbus/backend/internal/utils"
	"golang.org/x/net/http/httpguts"
)

const (
	maxCustomAPIHeaders     = 10
	maxCustomAPIHeaderBytes = 4096
	maxCustomAPIFields      = 4
	maxCustomAPILabelRunes  = 40
	maxCustomAPIPathRunes   = 200
	maxCustomAPIUnitRunes   = 10
)

// Header values are stored with the widget like the rest of its config.
// Apps with a native integration keep their keys encrypted there instead.
type customAPIHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type customAPIField struct {
	Label string `json:"label"`
	Path  string `json:"path"` // see parsePath
	Unit  string `json:"unit"`
}

type customAPIConfig struct {
	URL     string            `json:"url"`
	Headers []customAPIHeader `json:"headers"`
	Fields  []customAPIField  `json:"fields"`
}

// customAPIPayload has the shape of an integration payload, so the browser
// shows it like a KPI row. KPIs is keyed by field index.
type customAPIPayload struct {
	KPIs map[string]any `json:"kpis"`
}

type customAPI struct{}

func init() { Register(customAPI{}) }

func (customAPI) Type() string { return "custom_api" }

func (customAPI) Meta() Meta {
	return Meta{
		Name:         "Custom API",
		Category:     CategoryInfo,
		DefaultSize:  "2x1",
		AllowedSizes: serviceSizes,
	}
}

func (customAPI) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[customAPIConfig](config)
	if err != nil {
		return nil, err
	}

	// The server fetches it, so it gets the same check as integration URLs
	cfg.URL = strings.TrimSpace(cfg.URL)
	if err := utils.ValidateWebhookURL(cfg.URL); err != nil {
		return nil, fmt.Errorf("invalid URL: %s", err.Error())
	}

	if len(cfg.Headers) > maxCustomAPIHeaders {
		return nil, fmt.Errorf("at most %d headers are allowed", maxCustomAPIHeaders)
	}
	if cfg.Headers == nil {
		cfg.Headers = []customAPIHeader{}
	}
	for i := range cfg.Headers {
		h := &cfg.Headers[i]
		h.Name = strings.TrimSpace(h.Name)
		h.Value = strings.TrimSpace(h.Value)
		if !httpguts.ValidHeaderFieldName(h.Name) {
			return nil, fmt.Errorf("header %d needs a valid name", i+1)
		}
		if len(h.Value) > maxCustomAPIHeaderBytes || !httpguts.ValidHeaderFieldValue(h.Value) {
			return nil, fmt.Errorf("header %d has an invalid value", i+1)
		}
	}

	if len(cfg.Fields) == 0 || len(cfg.Fields) > maxCustomAPIFields {
		return nil, fmt.Errorf("add 1 to %d values to show", maxCustomAPIFields)
	}
	for i := range cfg.Fields {
		f := &cfg.Fields[i]
		f.Label = strings.TrimSpace(f.Label)
		f.Path = strings.TrimSpace(f.Path)
		f.Unit = strings.TrimSpace(f.Unit)
		switch {
		case f.Label == "":
			return nil, fmt.Errorf("value %d needs a label", i+1)
		case utf8.RuneCountInString(f.Label) > maxCustomAPILabelRunes:
			return nil, fmt.Errorf("value %d: label must be %d characters or less", i+1, maxCustomAPILabelRunes)
		case utf8.RuneCountInString(f.Path) > maxCustomAPIPathRunes:
			return nil, fmt.Errorf("value %d: path must be %d characters or less", i+1, maxCustomAPIPathRunes)
		case utf8.RuneCountInString(f.Unit) > maxCustomAPIUnitRunes:
			return nil, fmt.Errorf("value %d: unit must be %d characters or less", i+1, maxCustomAPIUnitRunes)
		}
		if _, err := parsePath(f.Path); err != nil {
			return nil, fmt.Errorf("value %d: %s", i+1, err.Error())
		}
	}
	return encodeConfig(cfg)
}

func (customAPI) Fetch(ctx context.Context, req *FetchRequest) (any, error) {
	cfg, err := decodeConfig[customAPIConfig](req.Config)
	if err != nil || cfg.URL == "" {
		return nil, errors.New("widget has no URL")
	}

	header := http.Header{"Accept": {"application/json"}}
	for _, h := range cfg.Headers {
		header.Add(h.Name, h.Value)
	}
	body, err := getBody(ctx, req.Client, cfg.URL, header)
	if err != nil {
		return nil, err
	}

	// UseNumber keeps big integers exact
	var doc any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return nil, errors.New("response is not valid JSON")
	}

	kpis := make(map[string]any, len(cfg.Fields))
	for i, f := range cfg.Fields {
		steps, err := parsePath(f.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", f.Label, err.Error())
		}
		value, ok := lookupPath(doc, steps)
		if !ok {
			return nil, fmt.Errorf("%s: nothing found at %s", f.Label, f.Path)
		}
		switch value.(type) {
		case map[string]any, []any:
			return nil, fmt.Errorf("%s: %s is a list or object, not a value", f.Label, f.Path)
		}
		kpis[strconv.Itoa(i)] = value
	}
	return customAPIPayload{KPIs: kpis}, nil
}
