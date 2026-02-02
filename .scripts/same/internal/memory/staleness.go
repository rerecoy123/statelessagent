package memory

import (
	"fmt"
	"strings"
	"time"

	"github.com/sgxdev/same/internal/store"
)

// StaleNote represents a note past its review-by date.
type StaleNote struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	ReviewBy    string `json:"review_by"`
	DaysOverdue int    `json:"days_overdue"`
	ContentType string `json:"content_type"`
}

// FindStaleNotes queries the index for notes with review_by dates.
func FindStaleNotes(db *store.DB, maxResults int, overdueOnly bool) []StaleNote {
	notes, err := db.GetStaleNotes(maxResults*2, overdueOnly)
	if err != nil || len(notes) == 0 {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)
	var results []StaleNote
	seen := make(map[string]bool)

	for _, n := range notes {
		if seen[n.Path] {
			continue
		}
		seen[n.Path] = true

		reviewByStr := strings.TrimSpace(n.ReviewBy)
		if reviewByStr == "" {
			continue
		}

		reviewDate, err := parseDate(reviewByStr)
		if err != nil {
			continue
		}

		daysOverdue := int(today.Sub(reviewDate).Hours() / 24)

		if overdueOnly && daysOverdue < 0 {
			continue
		}

		results = append(results, StaleNote{
			Path:        n.Path,
			Title:       n.Title,
			ReviewBy:    reviewByStr,
			DaysOverdue: daysOverdue,
			ContentType: n.ContentType,
		})

		if len(results) >= maxResults {
			break
		}
	}

	// Sort by most overdue first (already roughly sorted by review_by ASC from query)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].DaysOverdue > results[i].DaysOverdue {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// FormatStaleNotesContext formats stale notes for injection as context.
func FormatStaleNotesContext(staleNotes []StaleNote) string {
	if len(staleNotes) == 0 {
		return ""
	}

	lines := []string{"Notes past their review-by date:"}
	limit := 5
	if len(staleNotes) < limit {
		limit = len(staleNotes)
	}

	for _, note := range staleNotes[:limit] {
		urgency := "upcoming"
		if note.DaysOverdue > 0 {
			urgency = "OVERDUE"
		} else if note.DaysOverdue == 0 {
			urgency = "due today"
		}

		line := fmt.Sprintf("- [%s](%s) — %s", note.Title, note.Path, urgency)
		if note.DaysOverdue > 0 {
			line += fmt.Sprintf(" by %d days", note.DaysOverdue)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func parseDate(s string) (time.Time, error) {
	// Try ISO datetime first
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t.Truncate(24 * time.Hour), nil
	}

	// Try date-only formats
	for _, layout := range []string{"2006-01-02", "2006/01/02"} {
		t, err = time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unparseable date: %s", s)
}
