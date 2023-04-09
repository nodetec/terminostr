package main

import (
	"fmt"
	"strconv"
	"time"
)

func getRelativeTime(timestamp string) string {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return "Invalid timestamp"
	}

	relativeTime := time.Since(time.Unix(ts, 0))

	minutes := int(relativeTime.Minutes())
	hours := int(relativeTime.Hours())
	days := int(relativeTime.Hours() / 24)
	weeks := int(relativeTime.Hours() / (24 * 7))
	months := int(relativeTime.Hours() / (24 * 30))
	years := int(relativeTime.Hours() / (24 * 365))

	if years > 0 {
		return fmt.Sprintf("%d year", years) + pluralize(years)
	} else if months > 0 {
		return fmt.Sprintf("%d month", months) + pluralize(months)
	} else if weeks > 0 {
		return fmt.Sprintf("%d week", weeks) + pluralize(weeks)
	} else if days > 0 {
		return fmt.Sprintf("%d day", days) + pluralize(days)
	} else if hours > 0 {
		return fmt.Sprintf("%d hour", hours) + pluralize(hours)
	} else {
		return fmt.Sprintf("%d minute", minutes) + pluralize(minutes)
	}
}

func pluralize(n int) string {
	if n != 1 {
		return "s ago"
	}
	return " ago"
}
