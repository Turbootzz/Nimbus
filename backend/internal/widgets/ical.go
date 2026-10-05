package widgets

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

// icalEvent is one VEVENT of an iCalendar feed
type icalEvent struct {
	Summary string
	Start   time.Time
	AllDay  bool
}

// parseICal reads the events of an iCalendar (RFC 5545) feed. ponytail:
// recurring events (RRULE) only show on their first date; expanding them
// needs a real recurrence engine.
func parseICal(data []byte) ([]icalEvent, error) {
	lines, err := unfoldICal(data)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 || !strings.EqualFold(lines[0], "BEGIN:VCALENDAR") {
		return nil, errors.New("not an iCalendar feed")
	}

	var events []icalEvent
	var current *icalEvent
	depth := 0 // components nested in the event, like VALARM
	for _, line := range lines {
		name, params, value := splitICalLine(line)
		switch {
		case name == "BEGIN" && strings.EqualFold(value, "VEVENT"):
			current, depth = &icalEvent{}, 0
		case current == nil:
			continue
		case name == "BEGIN":
			depth++
		case name == "END" && depth > 0:
			depth--
		case name == "END" && strings.EqualFold(value, "VEVENT"):
			if !current.Start.IsZero() {
				events = append(events, *current)
			}
			current = nil
		case depth > 0:
			continue
		case name == "SUMMARY":
			current.Summary = unescapeICal(value)
		case name == "DTSTART":
			current.Start, current.AllDay = parseICalTime(value, params)
		}
	}
	return events, nil
}

// unfoldICal splits into lines and joins folded ones (a line that starts
// with a space or tab continues the previous one)
func unfoldICal(data []byte) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), maxFetchBodyBytes)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
			continue
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not read the calendar: %w", err)
	}
	return lines, nil
}

// splitICalLine splits "DTSTART;TZID=Europe/Amsterdam:20261007T090000"
// into its name, parameters and value
func splitICalLine(line string) (name string, params map[string]string, value string) {
	// The value starts at the first colon outside quotes: Outlook quotes
	// TZIDs like "(UTC+01:00) Amsterdam, Berlin"
	head, quoted := line, false
	for i, r := range line {
		if r == '"' {
			quoted = !quoted
		} else if r == ':' && !quoted {
			head, value = line[:i], line[i+1:]
			break
		}
	}
	params = map[string]string{}
	parts := splitOutsideQuotes(head, ';')
	for _, p := range parts[1:] {
		k, v, _ := strings.Cut(p, "=")
		params[strings.ToUpper(k)] = strings.Trim(v, `"`)
	}
	return strings.ToUpper(parts[0]), params, value
}

func splitOutsideQuotes(s string, sep rune) []string {
	var parts []string
	start, quoted := 0, false
	for i, r := range s {
		if r == '"' {
			quoted = !quoted
		} else if r == sep && !quoted {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

// windowsZones maps the zone names Outlook and Exchange write to IANA ones.
// ponytail: the common ones only; add more as feeds need them.
var windowsZones = map[string]string{
	"UTC":                            "UTC",
	"GMT Standard Time":              "Europe/London",
	"W. Europe Standard Time":        "Europe/Berlin",
	"Romance Standard Time":          "Europe/Paris",
	"Central Europe Standard Time":   "Europe/Budapest",
	"Central European Standard Time": "Europe/Warsaw",
	"E. Europe Standard Time":        "Europe/Chisinau",
	"FLE Standard Time":              "Europe/Kiev",
	"GTB Standard Time":              "Europe/Bucharest",
	"Russian Standard Time":          "Europe/Moscow",
	"Eastern Standard Time":          "America/New_York",
	"Central Standard Time":          "America/Chicago",
	"Mountain Standard Time":         "America/Denver",
	"Pacific Standard Time":          "America/Los_Angeles",
	"India Standard Time":            "Asia/Kolkata",
	"China Standard Time":            "Asia/Shanghai",
	"Tokyo Standard Time":            "Asia/Tokyo",
	"AUS Eastern Standard Time":      "Australia/Sydney",
}

// icalLocation resolves a TZID: an IANA name, a Windows name, or an
// Outlook display name like "(UTC+01:00) Amsterdam, Berlin" (its first city
// that is a zone, else the fixed offset without daylight saving). Unknown
// zones are the server's.
func icalLocation(tzid string) *time.Location {
	if l, err := time.LoadLocation(tzid); err == nil && tzid != "" {
		return l
	}
	if iana, ok := windowsZones[tzid]; ok {
		if l, err := time.LoadLocation(iana); err == nil {
			return l
		}
	}
	if _, cities, ok := strings.Cut(tzid, ") "); ok && strings.HasPrefix(tzid, "(UTC") {
		for _, city := range strings.Split(cities, ",") {
			name := strings.ReplaceAll(strings.TrimSpace(city), " ", "_")
			for _, region := range []string{"Europe", "America", "Asia", "Australia", "Africa", "Pacific"} {
				if l, err := time.LoadLocation(region + "/" + name); err == nil {
					return l
				}
			}
		}
	}
	var sign rune
	var hours, minutes int
	if _, err := fmt.Sscanf(tzid, "(UTC%c%d:%d)", &sign, &hours, &minutes); err == nil {
		offset := hours*3600 + minutes*60
		if sign == '-' {
			offset = -offset
		}
		return time.FixedZone(tzid, offset)
	}
	return time.Local
}

// parseICalTime reads a DATE or DATE-TIME: UTC with a Z, in its TZID, or
// floating (the server's zone). A zero time means it couldn't be read.
func parseICalTime(value string, params map[string]string) (time.Time, bool) {
	if params["VALUE"] == "DATE" || len(value) == len("20060102") {
		t, err := time.Parse("20060102", value)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	}
	if t, err := time.Parse("20060102T150405Z", value); err == nil {
		return t, false
	}
	loc := time.Local
	if tzid := params["TZID"]; tzid != "" {
		loc = icalLocation(tzid)
	}
	t, err := time.ParseInLocation("20060102T150405", value, loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, false
}

var icalUnescaper = strings.NewReplacer(`\n`, " ", `\N`, " ", `\,`, ",", `\;`, ";", `\\`, `\`)

func unescapeICal(s string) string {
	return strings.TrimSpace(icalUnescaper.Replace(s))
}
