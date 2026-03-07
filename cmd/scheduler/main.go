package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/straubt1/curling-schedule/pkg/scraper"
)

type DayAvailability struct {
	DayName string   `json:"dayName"`
	Date    string   `json:"date"`
	Times   []string `json:"times"`
}

type AvailabilityOutput struct {
	LastUpdated  string            `json:"lastUpdated"`
	Timezone     string            `json:"timezone"`
	Availability []DayAvailability `json:"availability"`
}

func main() {
	jsonOut := flag.String("json-out", "", "JSON output file path")
	days := flag.Int("days", 7, "how many days out to query (inclusive)")
	venue := flag.String("venue", "", "sevenrooms venue id (overrides default)")
	flag.Parse()

	if *jsonOut == "" {
		log.Fatal("--json-out is required")
	}

	// Use the IANA timezone for US Central. If the zone data isn't available,
	// fall back to a fixed -6h offset (approximate Central without DST).
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		loc = time.FixedZone("US/Central", -6*60*60)
	}
	now := time.Now().In(loc)

	// If a venue was supplied via flag, override the scraper package default.
	if *venue != "" {
		scraper.Venue = *venue
	}

	var availability []DayAvailability
	for i := 0; i <= *days; i++ {
		date := now.AddDate(0, 0, i)
		dayName := date.Format("Monday")
		dateStr := date.Format("01-02-2006")

		times, err := scraper.GetAvailabilityTimes(dateStr)
		if err != nil {
			log.Printf("warning: failed getting times for %s: %v", dateStr, err)
		}

		var timesList []string
		if times != "" {
			timesList = strings.Split(times, "; ")
		} else {
			timesList = []string{}
		}
		availability = append(availability, DayAvailability{
			DayName: dayName,
			Date:    dateStr,
			Times:   timesList,
		})
	}

	payload := AvailabilityOutput{
		LastUpdated:  now.Format(time.RFC3339),
		Timezone:     "America/Chicago",
		Availability: availability,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Fatalf("marshal json: %v", err)
	}
	if err := os.WriteFile(*jsonOut, data, 0644); err != nil {
		log.Fatalf("write json output: %v", err)
	}
	log.Printf("wrote JSON to %s", *jsonOut)
}
