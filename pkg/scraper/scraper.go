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

// GetAvailabilityTimes queries the SevenRooms API for a given date in MM-DD-YYYY
// and returns a semicolon-separated list of available times (type=="book").
func GetAvailabilityTimes(date string) (string, error) {
	q := url.Values{}
	q.Set("venue", "teeline")
	q.Set("time_slot", "16:00")
	q.Set("party_size", "4")
	q.Set("halo_size_interval", "24")
	q.Set("start_date", date)
	q.Set("num_days", "1")
	q.Set("channel", "SEVENROOMS_WIDGET")
	q.Set("selected_lang_code", "en")

	full := BaseURL + "?" + q.Encode()
	resp, err := http.Get(full)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	// Unmarshal to a generic structure and walk it to extract any "times" arrays.
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
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
		// fallback to searching whole document
		avail = root
	}

	timesSet := map[string]struct{}{}
	collectTimes(avail, timesSet)

	if len(timesSet) == 0 {
		return "", nil
	}

	// Prepare slice and sort by parsed time-of-day when possible.
	type ts struct {
		raw string
		min int  // minutes since midnight
		ok  bool // whether parse succeeded
	}

	arr := make([]ts, 0, len(timesSet))
	for k := range timesSet {
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
			// Parsed times come before unparsed ones
			return a.ok
		}
		// fallback to lexical order
		return a.raw < b.raw
	})

	out := make([]string, 0, len(arr))
	for _, e := range arr {
		out = append(out, e.raw)
	}
	return strings.Join(out, "; "), nil
}

// parseToMinutes tries to parse a time string into minutes since midnight.
// It supports 24-hour (15:04) and 12-hour (3:04 PM) formats. Returns (minutes, true)
// on success, (0, false) on failure.
func parseToMinutes(s string) (int, bool) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ";,|")
	// Try a list of layouts
	layouts := []string{
		"15:04", "15:04:05",
		"3:04 PM", "03:04 PM", "3:04PM", "03:04PM",
		"3PM", "3 PM", "03PM", "03 PM",
	}
	// Normalize multiple spaces
	s = strings.Join(strings.Fields(s), " ")
	// Ensure AM/PM uppercase
	s = strings.ToUpper(s)

	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.Hour()*60 + t.Minute(), true
		}
	}

	// Some inputs might be like "1:00P PM" or odd variants; try to remove stray letters
	cleaned := s
	// remove trailing non-digit/non-colon/non-space/AMPM chars
	// keep only digits, colon, space, A, P, M
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
func collectTimes(v any, set map[string]struct{}) {
	switch x := v.(type) {
	case nil:
		return
	case []any:
		for _, el := range x {
			collectTimes(el, set)
		}
	case map[string]any:
		// If there is a "times" key with an array, process it.
		if timesVal, ok := x["times"]; ok {
			if arr, ok := timesVal.([]any); ok {
				for _, item := range arr {
					if itm, ok := item.(map[string]any); ok {
						typ, _ := itm["type"].(string)
						tm, _ := itm["time"].(string)
						if typ == "book" && tm != "" {
							set[tm] = struct{}{}
						}
					}
				}
			}
		}
		// Recurse into all values in the map
		for _, val := range x {
			collectTimes(val, set)
		}
	default:
		// other primitive types ignored
		return
	}
}
