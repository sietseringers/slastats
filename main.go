package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/apognu/gocal"
)

type times map[string]time.Duration

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage:   slastats <url> <start> <end>")
		fmt.Println("Example: slastats http://example.com/example.ics 2020-01-01 2020-12-31")
		os.Exit(0)
	}

	start, end, err := parseTimeRange(os.Args[2], os.Args[3])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cal, err := getCal(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	process(cal, start, end)
}

func parseTimeRange(startstring, endstring string) (time.Time, time.Time, error) {
	layout := "2006-01-02"
	start, err := time.Parse(layout, startstring)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to parse start time in format yyyy-mm-dd: %w", err)
	}
	end, err := time.Parse(layout, endstring)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to parse end time in format yyyy-mm-dd: %w", err)
	}
	if start.After(end) {
		return time.Time{}, time.Time{}, errors.New("start time after end time")
	}
	return start, end, nil
}

func getCal(url string) (*gocal.Gocal, error) {
	body, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to GET iCal feed: %w", err)
	}

	c := gocal.NewParser(body.Body)
	c.SkipBounds = true
	if c.Parse() != nil {
		return nil, fmt.Errorf("failed to parse iCal feed: %w", err)
	}

	return c, nil
}

func process(cal *gocal.Gocal, start, end time.Time) {
	inside, outside, first, last := aggregate(cal, start, end)

	keys := sortKeys(inside)
	for _, name := range keys {
		fmt.Printf("%8v: %.0f, %.0f\n", name, inside[name].Hours(), outside[name].Hours())
	}

	fmt.Println()
	fmt.Println("Start:", first)
	fmt.Println("End:  ", last)
}

func aggregate(cal *gocal.Gocal, start, end time.Time) (times, times, time.Time, time.Time) {
	inside, outside := make(times), make(times)
	var first, last time.Time

	for _, e := range cal.Events {
		if e.Start.Before(start) || e.End.After(end) {
			continue
		}

		if first.IsZero() {
			first = *e.Start
		}
		last = *e.End

		i, o := split(*e.Start, *e.End)
		inside[e.Summary] += i
		outside[e.Summary] += o
	}

	return inside, outside, first, last
}

func split(start, end time.Time) (time.Duration, time.Duration) {
	var inside, outside time.Duration

	for start.Before(end) {
		if start.Hour() >= 9 && start.Hour() < 17 {
			inside += time.Hour
		} else {
			outside += time.Hour
		}
		start = start.Add(time.Hour)
	}

	return inside, outside
}

func sortKeys(t times) []string {
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
