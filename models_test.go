package main

import "testing"

func TestVideoFilter_Cohorts(t *testing.T) {
	videos := []Video{
		{ID: "a", Title: "A", VideoType: "long-form"},
		{ID: "b", Title: "B", VideoType: "long-form"},
		{ID: "c", Title: "C", VideoType: "live"},
	}
	assignments := map[string][]string{
		"a": {"gastown-series"},
		"b": {"evergreen"},
		"c": {"gastown-series", "live-deep-dives"},
	}

	tests := []struct {
		name    string
		filter  VideoFilter
		wantIDs []string
	}{
		{
			name:    "single cohort",
			filter:  VideoFilter{Cohorts: []string{"gastown-series"}, CohortAssignments: assignments},
			wantIDs: []string{"a", "c"},
		},
		{
			name:    "multiple cohorts is OR",
			filter:  VideoFilter{Cohorts: []string{"gastown-series", "evergreen"}, CohortAssignments: assignments},
			wantIDs: []string{"a", "b", "c"},
		},
		{
			name:    "no cohort filter passes all",
			filter:  VideoFilter{},
			wantIDs: []string{"a", "b", "c"},
		},
		{
			name:    "unknown cohort matches nothing",
			filter:  VideoFilter{Cohorts: []string{"made-up"}, CohortAssignments: assignments},
			wantIDs: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.filter.Apply(videos)
			gotIDs := make([]string, 0, len(got))
			for _, v := range got {
				gotIDs = append(gotIDs, v.ID)
			}
			if !sliceEq(gotIDs, tc.wantIDs) {
				t.Errorf("got %v, want %v", gotIDs, tc.wantIDs)
			}
		})
	}
}

func TestVideoFilter_CohortsCombineWithType(t *testing.T) {
	videos := []Video{
		{ID: "a", VideoType: "long-form"},
		{ID: "b", VideoType: "live"},
	}
	assignments := map[string][]string{
		"a": {"gastown-series"},
		"b": {"gastown-series"},
	}
	f := VideoFilter{
		Cohorts:           []string{"gastown-series"},
		CohortAssignments: assignments,
		Types:             []string{"live"},
	}
	got := f.Apply(videos)
	if len(got) != 1 || got[0].ID != "b" {
		t.Errorf("expected [b], got %+v", got)
	}
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
