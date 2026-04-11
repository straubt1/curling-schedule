package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// BaseURL is the API endpoint base and is replaceable in tests.
var BaseURL = "https://www.sevenrooms.com/api-yoa/availability/widget/range"

// Venue is the venue identifier used in the query. It defaults to the
// placeholder value "scheduler" but can be overridden by callers (for
// example, from a flag or environment variable in the CLI).
var Venue = "teeline"

// TimeSlot represents a single available time slot and how many sheets
// are available ("1" or "2+").
type TimeSlot struct {
	Time   string
	Sheets string // "1" or "2+"
}

// GetAvailability queries the SevenRooms API twice for a given date (MM-DD-YYYY):
// once with party_size=4 and once with party_size=20. It returns a sorted slice
// of TimeSlot values. Slots found in both responses get Sheets="2+"; slots found
// only in the party_size=4 response get Sheets="1".
func GetAvailability(date string) ([]TimeSlot, error) {
	setSmall, err := fetchTimesSet(date, "4")
	if err != nil {
		return nil, fmt.Errorf("party_size=4 query: %w", err)
	}

	setLarge, err := fetchTimesSet(date, "20")
	if err != nil {
		return nil, fmt.Errorf("party_size=20 query: %w", err)
	}

	if len(setSmall) == 0 {
		return []TimeSlot{}, nil
	}

	type ts struct {
		raw string
		min int
		ok  bool
	}

	arr := make([]ts, 0, len(setSmall))
	for k := range setSmall {
		if m, ok := parseToMinutes(k); ok {
			arr = append(arr, ts{raw: k, min: m, ok: true})
		} else {
			arr = append(arr, ts{raw: k, ok: false})
		}
	}

	sort.Slice(arr, func(i, j int) bool {
		a, b := arr[i], arr[j]
		if a.ok && b.ok {
			return a.min < b.min
		}
		if a.ok != b.ok {
			return a.ok
		}
		return a.raw < b.raw
	})

	out := make([]TimeSlot, 0, len(arr))
	for _, e := range arr {
		sheets := "1"
		if setLarge[e.raw] {
			sheets = "2+"
		}
		out = append(out, TimeSlot{Time: e.raw, Sheets: sheets})
	}
	return out, nil
}

// fetchTimesSet queries the SevenRooms API for the given date and party size,
// returning a set of available time strings (type=="book").
func fetchTimesSet(date, partySize string) (map[string]bool, error) {
	q := url.Values{}
	q.Set("venue", Venue)
	q.Set("time_slot", "16:00")
	q.Set("party_size", partySize)
	q.Set("halo_size_interval", "32")
	q.Set("start_date", date)
	q.Set("num_days", "1")
	q.Set("channel", "SEVENROOMS_WIDGET")
	q.Set("selected_lang_code", "en")

	full := BaseURL + "?" + q.Encode()
	resp, err := http.Get(full)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 512 {
			snippet = snippet[:512]
		}
		return nil, fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, full, snippet)
	}

	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	// Navigate to root.data.availability if present; otherwise search whole document.
	var avail any
	if m, ok := root.(map[string]any); ok {
		if d, ok := m["data"]; ok {
			if dm, ok := d.(map[string]any); ok {
				avail = dm["availability"]
			}
		}
	}
	if avail == nil {
		avail = root
	}

	counts := map[string]int{}
	collectTimes(avail, counts)

	set := make(map[string]bool, len(counts))
	for k := range counts {
		set[k] = true
	}
	return set, nil
}

// parseToMinutes tries to parse a time string into minutes since midnight.
// It supports 24-hour (15:04) and 12-hour (3:04 PM) formats. Returns (minutes, true)
// on success, (0, false) on failure.
func parseToMinutes(s string) (int, bool) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ";,|")
	layouts := []string{
		"15:04", "15:04:05",
		"3:04 PM", "03:04 PM", "3:04PM", "03:04PM",
		"3PM", "3 PM", "03PM", "03 PM",
	}
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ToUpper(s)

	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.Hour()*60 + t.Minute(), true
		}
	}

	cleaned := s
	var b strings.Builder
	for _, r := range cleaned {
		if (r >= '0' && r <= '9') || r == ':' || r == ' ' || r == 'A' || r == 'P' || r == 'M' {
			b.WriteRune(r)
		}
	}
	cleaned = strings.TrimSpace(b.String())
	for _, l := range layouts {
		if t, err := time.Parse(l, cleaned); err == nil {
			return t.Hour()*60 + t.Minute(), true
		}
	}
	return 0, false
}

// collectTimes recursively walks nested maps/slices looking for objects with a
// "times" key containing an array of {type,time} objects.
func collectTimes(v any, counts map[string]int) {
	switch x := v.(type) {
	case nil:
		return
	case []any:
		for _, el := range x {
			collectTimes(el, counts)
		}
	case map[string]any:
		if timesVal, ok := x["times"]; ok {
			if arr, ok := timesVal.([]any); ok {
				for _, item := range arr {
					if itm, ok := item.(map[string]any); ok {
						typ, _ := itm["type"].(string)
						tm, _ := itm["time"].(string)
						if typ == "book" && tm != "" {
							counts[tm]++
						}
					}
				}
			}
		}
		for k, val := range x {
			if k == "times" {
				continue
			}
			collectTimes(val, counts)
		}
	default:
		return
	}
}
