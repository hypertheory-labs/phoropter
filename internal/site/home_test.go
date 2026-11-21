package site

import (
	"strings"
	"testing"
)

// calico: recent is the latest day with a plan and any day within a week of
// it, newest first.
func TestRecentPlansAreAWeekOfTheLatestDay(t *testing.T) {
	s := &Site{}
	for _, d := range []string{"2026-10-01", "2026-11-12", "2026-11-13", "2026-11-20", "2026-11-20"} {
		s.Plans = append(s.Plans, &Plan{Date: d, Name: d})
	}
	s.Plans[4].Name = "later"
	var got []string
	for _, p := range recentPlans(s) {
		got = append(got, p.Name)
	}
	if strings.Join(got, " ") != "later 2026-11-20 2026-11-13" {
		t.Errorf("recent = %v", got)
	}
}
