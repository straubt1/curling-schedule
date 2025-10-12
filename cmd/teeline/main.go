package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/straubt1/curling-schedule/pkg/scraper"
)

func main() {
	out := flag.String("out", "README.md", "summary output file")
	days := flag.Int("days", 7, "how many days out to query (inclusive)")
	flag.Parse()

	// Create output file
	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("create output: %v", err)
	}
	defer f.Close()

	// Use the IANA timezone for US Central. If the zone data isn't available,
	// fall back to a fixed -6h offset (approximate Central without DST).
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		loc = time.FixedZone("US/Central", -6*60*60)
	}
	now := time.Now().In(loc)
	header := fmt.Sprintf("# Ice Availability\n\nList the latest availability for the next week, Last Update at **%s**\n\n| Day         | Date        | Times       |\n| ----------- | ----------- | ----------- |\n",
		now.Format("01-02-2006 03:04:05 PM"))
	if _, err := f.WriteString(header); err != nil {
		log.Fatalf("write header: %v", err)
	}

	for i := 0; i <= *days; i++ {
		// Use the same base 'now' in the chosen location to avoid mixing zones.
		date := now.AddDate(0, 0, i)
		dayName := date.Format("Monday")
		dateStr := date.Format("01-02-2006")

		times, err := scraper.GetAvailabilityTimes(dateStr)
		if err != nil {
			log.Printf("warning: failed getting times for %s: %v", dateStr, err)
		}

		// Render times as multi-line in the markdown table cell using <br>
		timesDisplay := strings.ReplaceAll(times, "; ", "<br>")
		// Append to file as a markdown table row
		line := fmt.Sprintf("|%s|%s|%s|\n", dayName, dateStr, timesDisplay)
		if _, err := f.WriteString(line); err != nil {
			log.Fatalf("write line: %v", err)
		}

	}
}
