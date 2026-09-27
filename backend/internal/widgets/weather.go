package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

// openMeteoURL is a variable so tests can point it at a fake server
var openMeteoURL = "https://api.open-meteo.com"

const (
	weatherUnitsMetric   = "metric"
	weatherUnitsImperial = "imperial"
	weatherForecastDays  = 4
	// Open-Meteo is free and keyless; a 10 minute minimum keeps us polite
	weatherMinRefreshSeconds = 600
	maxFetchBodyBytes        = 1 << 20
)

type weatherConfig struct {
	// Pointers so a missing coordinate is an error, not 0
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	// Location is the name shown on the tile
	Location string `json:"location"`
	Units    string `json:"units"`
}

// weatherPayload is what the browser gets
type weatherPayload struct {
	Temperature     float64      `json:"temperature"`
	FeelsLike       float64      `json:"feels_like"`
	WeatherCode     int          `json:"weather_code"`
	WindSpeed       float64      `json:"wind_speed"`
	IsDay           bool         `json:"is_day"`
	TemperatureUnit string       `json:"temperature_unit"`
	WindUnit        string       `json:"wind_unit"`
	Daily           []weatherDay `json:"daily"`
}

type weatherDay struct {
	Date        string  `json:"date"`
	WeatherCode int     `json:"weather_code"`
	Max         float64 `json:"max"`
	Min         float64 `json:"min"`
}

// openMeteoResponse is the part of the forecast response we use
type openMeteoResponse struct {
	Current struct {
		Temperature float64 `json:"temperature_2m"`
		FeelsLike   float64 `json:"apparent_temperature"`
		WeatherCode int     `json:"weather_code"`
		WindSpeed   float64 `json:"wind_speed_10m"`
		IsDay       int     `json:"is_day"`
	} `json:"current"`
	CurrentUnits struct {
		Temperature string `json:"temperature_2m"`
		WindSpeed   string `json:"wind_speed_10m"`
	} `json:"current_units"`
	Daily struct {
		Time        []string  `json:"time"`
		WeatherCode []int     `json:"weather_code"`
		Max         []float64 `json:"temperature_2m_max"`
		Min         []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

type weather struct{}

func init() { Register(weather{}) }

func (weather) Type() string { return "weather" }

func (weather) Meta() Meta {
	return Meta{
		Name:              "Weather",
		Category:          CategoryInfo,
		DefaultSize:       "2x1",
		AllowedSizes:      serviceSizes,
		MinRefreshSeconds: weatherMinRefreshSeconds,
	}
}

func (weather) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[weatherConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Latitude == nil || cfg.Longitude == nil {
		return nil, errors.New("pick a location")
	}
	if *cfg.Latitude < -90 || *cfg.Latitude > 90 || *cfg.Longitude < -180 || *cfg.Longitude > 180 {
		return nil, errors.New("latitude must be between -90 and 90 and longitude between -180 and 180")
	}

	cfg.Location = strings.TrimSpace(cfg.Location)
	if utf8.RuneCountInString(cfg.Location) > 100 {
		return nil, errors.New("location name must be 100 characters or less")
	}

	switch cfg.Units {
	case "":
		cfg.Units = weatherUnitsMetric
	case weatherUnitsMetric, weatherUnitsImperial:
	default:
		return nil, errors.New("units must be metric or imperial")
	}
	return encodeConfig(cfg)
}

func (weather) Fetch(ctx context.Context, req *FetchRequest) (any, error) {
	cfg, err := decodeConfig[weatherConfig](req.Config)
	if err != nil || cfg.Latitude == nil || cfg.Longitude == nil {
		return nil, errors.New("widget has no location")
	}

	url := fmt.Sprintf("%s/v1/forecast?latitude=%.4f&longitude=%.4f"+
		"&current=temperature_2m,apparent_temperature,weather_code,wind_speed_10m,is_day"+
		"&daily=weather_code,temperature_2m_max,temperature_2m_min&timezone=auto&forecast_days=%d",
		openMeteoURL, *cfg.Latitude, *cfg.Longitude, weatherForecastDays)
	if cfg.Units == weatherUnitsImperial {
		url += "&temperature_unit=fahrenheit&wind_speed_unit=mph"
	}

	var data openMeteoResponse
	if err := getJSON(ctx, req.Client, url, &data); err != nil {
		return nil, fmt.Errorf("weather service: %w", err)
	}

	payload := weatherPayload{
		Temperature:     data.Current.Temperature,
		FeelsLike:       data.Current.FeelsLike,
		WeatherCode:     data.Current.WeatherCode,
		WindSpeed:       data.Current.WindSpeed,
		IsDay:           data.Current.IsDay == 1,
		TemperatureUnit: data.CurrentUnits.Temperature,
		WindUnit:        data.CurrentUnits.WindSpeed,
		Daily:           []weatherDay{},
	}
	d := data.Daily
	for i := range d.Time {
		if i >= len(d.WeatherCode) || i >= len(d.Max) || i >= len(d.Min) {
			break
		}
		payload.Daily = append(payload.Daily, weatherDay{Date: d.Time[i], WeatherCode: d.WeatherCode[i], Max: d.Max[i], Min: d.Min[i]})
	}
	return payload, nil
}

// getJSON GETs url and decodes a JSON body of at most maxFetchBodyBytes
func getJSON(ctx context.Context, client *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBodyBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxFetchBodyBytes {
		return errors.New("response is larger than 1 MB")
	}
	if err := json.Unmarshal(body, v); err != nil {
		return errors.New("response is not valid JSON")
	}
	return nil
}
