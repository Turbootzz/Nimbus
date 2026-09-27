package widgets

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Clock date formats
const (
	clockDateNone  = "none"
	clockDateShort = "short"
	clockDateLong  = "long"
)

type clockConfig struct {
	// Timezone is an IANA name; empty uses the browser's zone
	Timezone    string `json:"timezone"`
	Hour12      bool   `json:"hour12"`
	ShowSeconds bool   `json:"show_seconds"`
	DateFormat  string `json:"date_format"`
}

type clock struct{}

func init() { Register(clock{}) }

func (clock) Type() string { return "clock" }

func (clock) Meta() Meta {
	return Meta{
		Name:         "Clock",
		Category:     CategoryGeneral,
		DefaultSize:  "2x1",
		AllowedSizes: serviceSizes,
		Static:       true,
	}
}

func (clock) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[clockConfig](config)
	if err != nil {
		return nil, err
	}

	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if cfg.Timezone != "" {
		// "Local" means the server's zone, which the browser can't know
		if _, err := time.LoadLocation(cfg.Timezone); err != nil || cfg.Timezone == "Local" {
			return nil, errors.New("unknown timezone")
		}
	}

	switch cfg.DateFormat {
	case "":
		cfg.DateFormat = clockDateShort
	case clockDateNone, clockDateShort, clockDateLong:
	default:
		return nil, errors.New("date format must be none, short or long")
	}
	return encodeConfig(cfg)
}
