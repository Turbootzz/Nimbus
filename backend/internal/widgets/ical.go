package widgets

import (
	"bufio"
	"bytes"
	"errors"
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
	lines := unfoldICal(data)
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
func unfoldICal(data []byte) []string {
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
	return lines
}

// splitICalLine splits "DTSTART;TZID=Europe/Amsterdam:20261007T090000"
// into its name, parameters and value
func splitICalLine(line string) (name string, params map[string]string, value string) {
	head, value, _ := strings.Cut(line, ":")
	parts := strings.Split(head, ";")
	params = map[string]string{}
	for _, p := range parts[1:] {
		k, v, _ := strings.Cut(p, "=")
		params[strings.ToUpper(k)] = strings.Trim(v, `"`)
	}
	return strings.ToUpper(parts[0]), params, value
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
		if l, err := time.LoadLocation(tzid); err == nil {
			loc = l
		}
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
