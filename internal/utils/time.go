package utils

import (
	"fmt"
	"time"
)

func FormatDuration(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func FormatTime(t time.Time, timezone string) string {
	if timezone == "" {
		timezone = DefaultTimeZone
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	return t.In(loc).Format("2006-01-02 15:04:05")
}

func FormatTimeISO(t time.Time) string {
	return t.Format(time.RFC3339)
}

func ParseTimeISO(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeStr)
}

func StartOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func EndOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 999999999, t.Location())
}

func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday as 7
	}
	return StartOfDay(t.AddDate(0, 0, -weekday+1))
}

func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

func StartOfMonth(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}

func EndOfMonth(t time.Time) time.Time {
	return EndOfDay(StartOfMonth(t).AddDate(0, 1, -1))
}

func TimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / (24 * 7))
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / (24 * 30))
		return fmt.Sprintf("%d months ago", months)
	}

	years := int(duration.Hours() / (24 * 365))
	return fmt.Sprintf("%d years ago", years)
}

func IsBusinessHours(t time.Time, timezone string) bool {
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err == nil {
			t = t.In(loc)
		}
	}

	hour := t.Hour()
	weekday := t.Weekday()

	// Business hours: Monday-Friday, 9 AM - 6 PM
	return weekday >= time.Monday && weekday <= time.Friday && hour >= 9 && hour < 18
}

func NextBusinessDay(t time.Time) time.Time {
	for {
		t = t.AddDate(0, 0, 1)
		if t.Weekday() >= time.Monday && t.Weekday() <= time.Friday {
			return StartOfDay(t).Add(9 * time.Hour) // 9 AM
		}
	}
}
