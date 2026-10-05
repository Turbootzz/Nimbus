package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	// redactedHeaderValue stands in for a stored header value in responses
	redactedHeaderValue = "********"
)

// Header values may be keys. They are stored with the widget and never
// returned by the API (see Redact).
type customAPIHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type customAPIField struct {
	Label string `json:"label"` // unique; it keys the value in the payload
	Path  string `json:"path"`  // see parsePath
	Unit  string `json:"unit"`
}

type customAPIConfig struct {
	URL     string            `json:"url"`
	Headers []customAPIHeader `json:"headers"`
	Fields  []customAPIField  `json:"fields"`
	tlsOption
}

// customAPIPayload has the shape of an integration payload, so the browser
// shows it like a KPI row. KPIs is keyed by label, so an edited field list
// never shows an old value under another label.
type customAPIPayload struct {
	KPIs map[string]any `json:"kpis"`
	// Missing lists the labels whose path found nothing
	Missing []string `json:"missing,omitempty"`
}

// The server fetches any URL the user gives and shows what it finds, so it
// can read JSON from every host on the LAN. That is why it is admin only.
type customAPI struct{}

func init() { Register(customAPI{}) }

func (customAPI) Type() string { return "custom_api" }

func (customAPI) Meta() Meta {
	return Meta{
		Name:         "Custom API",
		Category:     CategoryInfo,
		DefaultSize:  "2x1",
		AllowedSizes: serviceSizes,
		AdminOnly:    true,
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
	cfg.tlsOption.normalise()

	// Empty rows are left over from the form; drop them
	headers := []customAPIHeader{}
	for _, h := range cfg.Headers {
		h.Name, h.Value = strings.TrimSpace(h.Name), strings.TrimSpace(h.Value)
		if h.Name != "" || h.Value != "" {
			headers = append(headers, h)
		}
	}
	if len(headers) > maxCustomAPIHeaders {
		return nil, fmt.Errorf("at most %d headers are allowed", maxCustomAPIHeaders)
	}
	for i, h := range headers {
		switch {
		case !httpguts.ValidHeaderFieldName(h.Name):
			return nil, fmt.Errorf("header %d needs a valid name", i+1)
		case strings.EqualFold(h.Name, "Host"):
			return nil, fmt.Errorf("header %d: put the host in the URL instead", i+1)
		case h.Value == redactedHeaderValue:
			return nil, fmt.Errorf("enter the value of header %d again", i+1)
		case len(h.Value) > maxCustomAPIHeaderBytes || !httpguts.ValidHeaderFieldValue(h.Value):
			return nil, fmt.Errorf("header %d has an invalid value", i+1)
		}
	}
	cfg.Headers = headers

	fields := []customAPIField{}
	for _, f := range cfg.Fields {
		f.Label, f.Path, f.Unit = strings.TrimSpace(f.Label), strings.TrimSpace(f.Path), strings.TrimSpace(f.Unit)
		if f.Label != "" || f.Path != "" || f.Unit != "" {
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 || len(fields) > maxCustomAPIFields {
		return nil, fmt.Errorf("add 1 to %d values to show", maxCustomAPIFields)
	}
	labels := map[string]bool{}
	for i, f := range fields {
		switch {
		case f.Label == "":
			return nil, fmt.Errorf("value %d needs a label", i+1)
		case labels[f.Label]:
			return nil, fmt.Errorf("value %d: label %q is used twice", i+1, f.Label)
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
		labels[f.Label] = true
	}
	cfg.Fields = fields
	return encodeConfig(cfg)
}

// Redact masks the header values
func (customAPI) Redact(config json.RawMessage) json.RawMessage {
	cfg, err := decodeConfig[customAPIConfig](config)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	for i := range cfg.Headers {
		if cfg.Headers[i].Value != "" {
			cfg.Headers[i].Value = redactedHeaderValue
		}
	}
	out, err := json.Marshal(cfg)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

// Unredact puts back the stored value of each masked header, matched by
// name. A renamed header has to be entered again (Validate says so).
func (customAPI) Unredact(config, stored json.RawMessage) json.RawMessage {
	cfg, err := decodeConfig[customAPIConfig](config)
	if err != nil {
		return config
	}
	old, err := decodeConfig[customAPIConfig](stored)
	if err != nil {
		return config
	}
	for i := range cfg.Headers {
		h := &cfg.Headers[i]
		if h.Value != redactedHeaderValue {
			continue
		}
		for _, o := range old.Headers {
			if strings.EqualFold(o.Name, strings.TrimSpace(h.Name)) {
				h.Value = o.Value
				break
			}
		}
	}
	out, err := json.Marshal(cfg)
	if err != nil {
		return config
	}
	return out
}

func (customAPI) Fetch(ctx context.Context, req *FetchRequest) (any, error) {
	cfg, err := decodeConfig[customAPIConfig](req.Config)
	if err != nil || cfg.URL == "" {
		return nil, errors.New("widget has no URL")
	}

	header := http.Header{}
	for _, h := range cfg.Headers {
		header.Add(h.Name, h.Value)
	}
	if header.Get("Accept") == "" {
		header.Set("Accept", "application/json")
	}

	client := req.Client
	if len(cfg.Headers) > 0 {
		// Go only drops its own auth headers on a redirect to another host;
		// a key in a custom header would go along, or go out unencrypted
		c := *req.Client
		check := c.CheckRedirect
		c.CheckRedirect = func(r *http.Request, via []*http.Request) error {
			if r.URL.Host != via[0].URL.Host {
				return errors.New("redirected to another host, which would get the headers")
			}
			if via[len(via)-1].URL.Scheme == "https" && r.URL.Scheme == "http" {
				return errors.New("redirected from https to http, which would send the headers unencrypted")
			}
			if check != nil {
				return check(r, via)
			}
			return nil
		}
		client = &c
	}

	body, err := getBody(ctx, client, cfg.URL, header)
	if err != nil {
		return nil, err
	}
	var doc any
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, errors.New("response is not valid JSON")
	}

	payload := customAPIPayload{KPIs: make(map[string]any, len(cfg.Fields))}
	for _, f := range cfg.Fields {
		steps, err := parsePath(f.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", f.Label, err.Error())
		}
		value, ok := lookupPath(doc, steps)
		if !ok {
			// APIs leave out keys that are empty; the other values still show
			payload.Missing = append(payload.Missing, f.Label)
			continue
		}
		switch value.(type) {
		case map[string]any, []any:
			return nil, fmt.Errorf("%s: %s is a list or object, not a value", f.Label, f.Path)
		}
		payload.KPIs[f.Label] = value
	}
	if len(payload.Missing) == len(cfg.Fields) {
		return nil, fmt.Errorf("nothing found at %s", cfg.Fields[0].Path)
	}
	return payload, nil
}
