package platform

import (
	"fmt"
	"time"
)

// dateToDays converts a YYYY-MM-DD date string to days since January 1, 1970.
func DateToDays(dateStr string) (int64, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid date %q, want YYYY-MM-DD", dateStr)
	}
	return t.UTC().Unix() / (24 * 60 * 60), nil
}

// localDate is today's date (YYYY-MM-DD) in the server's local time zone.
func localDate(now time.Time) string {
	return now.Format("2006-01-02")
}

// optionalDateToDays defaults a missing date to local today
// (FatSecret's own "today" is in its time zone and shifts late entries).
func OptionalDateToDays(dateStr string) (int64, error) {
	if dateStr == "" {
		dateStr = localDate(time.Now())
	}
	return DateToDays(dateStr)
}
