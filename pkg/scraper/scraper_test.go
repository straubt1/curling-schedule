package scraper

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestGetAvailabilityTimes_HappyPath(t *testing.T) {
	payload := `{"data":{"availability":{"2025-10-12":[[{"times":[{"type":"book","time":"17:00"},{"type":"book","time":"18:00"}]}]]}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	old := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = old }()

	times, err := GetAvailabilityTimes("10-12-2025")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if times != "17:00 (1); 18:00 (1)" {
		t.Fatalf("unexpected times: %q", times)
	}
}

func TestGetAvailabilityTimes_DedupeAndSort(t *testing.T) {
	payload := `{"data":{"availability":{"x":[[{"times":[{"type":"book","time":"18:00"},{"type":"book","time":"17:00"},{"type":"book","time":"18:00"}]}]]}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	old := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = old }()

	times, err := GetAvailabilityTimes("10-12-2025")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// 18:00 appears twice in the payload, expect counts to reflect that.
	if times != "17:00 (1); 18:00 (2)" {
		t.Fatalf("unexpected times: %q", times)
	}
}

func TestGetAvailabilityTimes_Empty(t *testing.T) {
	payload := `{"data":{"availability":{}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	old := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = old }()

	times, err := GetAvailabilityTimes("10-12-2025")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if times != "" {
		t.Fatalf("expected empty times, got %q", times)
	}

}

func TestParseOrdering(t *testing.T) {
	// This ensures times like "10:00 PM" sort after "1:00 PM" and AM/PM ordering works
	s := []string{"10:00 PM", "1:00 PM", "11:30 AM", "12:15 AM", "07:00"}
	set := map[string]struct{}{}
	for _, v := range s {
		set[v] = struct{}{}
	}

	// reuse collectTimes by wrapping into a structure that mimics JSON
	// but we will just directly build arr similar to GetAvailabilityTimes end
	type ts struct {
		raw string
		min int
		ok  bool
	}
	arr := make([]ts, 0, len(set))
	for k := range set {
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

	got := make([]string, 0, len(arr))
	for _, e := range arr {
		got = append(got, e.raw)
	}

	// expect earliest to latest: 12:15 AM, 07:00, 11:30 AM, 1:00 PM, 10:00 PM
	want := []string{"12:15 AM", "07:00", "11:30 AM", "1:00 PM", "10:00 PM"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordering mismatch at %d: got %v, want %v", i, got, want)
		}
	}
}
