package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/straubt1/curling-schedule/pkg/scraper"
)

type SlotOutput struct {
	Time   string `json:"time"`
	Sheets string `json:"sheets"`
}

type DayAvailability struct {
	DayName      string       `json:"dayName"`
	Date         string       `json:"date"`
	Availability []SlotOutput `json:"availability"`
}

type AvailabilityOutput struct {
	LastUpdated  string            `json:"lastUpdated"`
	Timezone     string            `json:"timezone"`
	Availability []DayAvailability `json:"availability"`
}

func main() {
	out := flag.String("out", "", "markdown README output file (optional)")
	jsonOut := flag.String("json-out", "", "JSON output file path")
	days := flag.Int("days", 7, "how many days out to query (inclusive)")
	venue := flag.String("venue", "", "sevenrooms venue id (overrides default)")
	flag.Parse()

	if *jsonOut == "" {
		log.Fatal("--json-out is required")
	}

	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		loc = time.FixedZone("US/Central", -6*60*60)
	}
	now := time.Now().In(loc)

	if *venue != "" {
		scraper.Venue = *venue
	}

	var mdFile *os.File
	if *out != "" {
		mdFile, err = os.Create(*out)
		if err != nil {
			log.Fatalf("create output: %v", err)
		}
		defer mdFile.Close()

		header := fmt.Sprintf(
			"# Curling Schedule\n\nView ice availability at TeeLine: **https://straubt1.github.io/curling-schedule**\n\nLast Update at **%s**\n\n| Day         | Date        | Times       |\n| ----------- | ----------- | ----------- |\n",
			now.Format("01-02-2006 03:04:05 PM"),
		)
		if _, err := mdFile.WriteString(header); err != nil {
			log.Fatalf("write header: %v", err)
		}
	}

	var availability []DayAvailability
	for i := 0; i <= *days; i++ {
		date := now.AddDate(0, 0, i)
		dayName := date.Format("Monday")
		dateStr := date.Format("01-02-2006")

		slots, err := scraper.GetAvailability(dateStr)
		if err != nil {
			log.Printf("warning: failed getting times for %s: %v", dateStr, err)
			slots = nil
		}

		if mdFile != nil {
			timeParts := make([]string, 0, len(slots))
			for _, s := range slots {
				timeParts = append(timeParts, s.Time)
			}
			timesDisplay := strings.Join(timeParts, "<br>")
			line := fmt.Sprintf("|%s|%s|%s|\n", dayName, dateStr, timesDisplay)
			if _, err := mdFile.WriteString(line); err != nil {
				log.Fatalf("write line: %v", err)
			}
		}

		slotOutputs := make([]SlotOutput, 0, len(slots))
		for _, s := range slots {
			slotOutputs = append(slotOutputs, SlotOutput{Time: s.Time, Sheets: s.Sheets})
		}
		availability = append(availability, DayAvailability{
			DayName:      dayName,
			Date:         dateStr,
			Availability: slotOutputs,
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
