package widgets

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const icalFixture = "BEGIN:VCALENDAR\r\n" +
	"VERSION:2.0\r\n" +
	"BEGIN:VEVENT\r\n" +
	"SUMMARY:Bin day\\, green\r\n" +
	"DTSTART;VALUE=DATE:20261009\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\n" +
	"SUMMARY:Dentist appointment with a long name that is fol\r\n" +
	" ded over two lines\r\n" +
	"DTSTART;TZID=Europe/Amsterdam:20261007T090000\r\n" +
	"BEGIN:VALARM\r\n" +
	"SUMMARY:Alarm, not the event\r\n" +
	"DTSTART:20991231T000000Z\r\n" +
	"END:VALARM\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\n" +
	"SUMMARY:Standup\r\n" +
	"DTSTART:20261008T073000Z\r\n" +
	"RRULE:FREQ=DAILY\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\n" +
	"SUMMARY:No start\r\n" +
	"END:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func TestParseICal(t *testing.T) {
	events, err := parseICal([]byte(icalFixture))
	require.NoError(t, err)
	amsterdam, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)

	require.Len(t, events, 3, "the event without a start is left out")
	assert.Equal(t, icalEvent{Summary: "Bin day, green", Start: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), AllDay: true}, events[0])
	assert.Equal(t, "Dentist appointment with a long name that is folded over two lines", events[1].Summary)
	assert.True(t, events[1].Start.Equal(time.Date(2026, 10, 7, 9, 0, 0, 0, amsterdam)), "TZID is honoured, the alarm ignored")
	assert.Equal(t, time.Date(2026, 10, 8, 7, 30, 0, 0, time.UTC), events[2].Start)
}

func TestParseICalRejects(t *testing.T) {
	for _, bad := range []string{"", "<html>", "BEGIN:VEVENT\nEND:VEVENT"} {
		_, err := parseICal([]byte(bad))
		assert.Error(t, err, bad)
	}
	// A broken date leaves the event out instead of failing the feed
	events, err := parseICal([]byte(strings.Replace(icalFixture, "20261009", "2026-10-09", 1)))
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestParseICalOutlookZones(t *testing.T) {
	feed := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\nSUMMARY:Quoted\r\nDTSTART;TZID=\"(UTC+01:00) Amsterdam, Berlin, Bern, Rome, Stockholm, Vienna\":20261007T090000\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nSUMMARY:Windows\r\nDTSTART;TZID=W. Europe Standard Time:20261007T090000\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"
	events, err := parseICal([]byte(feed))
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC), events[0].Start.UTC(), "fixed +01:00 from the display name")
	assert.Equal(t, time.Date(2026, 10, 7, 7, 0, 0, 0, time.UTC), events[1].Start.UTC(), "Europe/Berlin is +02:00 in October")
}
