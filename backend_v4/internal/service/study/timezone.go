package study

import "time"

// DayStart returns the start of the current day in the user's timezone, converted to UTC.
func DayStart(now time.Time, tz *time.Location) time.Time {
	userNow := now.In(tz)
	dayStart := time.Date(userNow.Year(), userNow.Month(), userNow.Day(), 0, 0, 0, 0, tz)
	return dayStart.UTC()
}

// NextDayStart returns the start of the next day in the user's timezone, converted to UTC.
func NextDayStart(now time.Time, tz *time.Location) time.Time {
	dayStart := DayStart(now, tz)
	// AddDate handles DST correctly, Add(24h) does not
	nextDay := dayStart.In(tz).AddDate(0, 0, 1)
	return time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, tz).UTC()
}

// ParseTimezone parses a timezone string, returning UTC as fallback.
func ParseTimezone(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}
