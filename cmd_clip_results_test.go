package main

import (
	"strings"
	"testing"
	"time"
)

func TestExtractOpusID(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"... #EnterpriseVibeCode [opus:La4Wghg6IX] Comment", "La4Wghg6IX", true},
		{"bonus variant [opus:La4Wghg6IX_bonus]", "La4Wghg6IX_bonus", true},
		{"no tag here", "", false},
		{"empty [opus:]", "", false},
	}
	for _, c := range cases {
		got, ok := extractOpusID(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("extractOpusID(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestYoutubeRowsForClips(t *testing.T) {
	data := &ChannelData{Videos: []Video{
		{ID: "ytA", Title: "A", Description: "x [opus:CLIP1] y", ViewCount: 100, LikeCount: 3, CommentCount: 1, SubscribersGained: 2},
		{ID: "ytB", Title: "B", Description: "no tag", ViewCount: 999},
		{ID: "ytC", Title: "C", Description: "[opus:CLIP2]", ViewCount: 50},
	}}
	// nil clipIDs => all opus-tagged videos
	all := youtubeRowsForClips(data, nil)
	if len(all) != 2 {
		t.Fatalf("want 2 youtube rows, got %d", len(all))
	}
	// filtered to CLIP1 only
	rows := youtubeRowsForClips(data, []string{"CLIP1"})
	if len(rows) != 1 || rows[0].OpusID != "CLIP1" || rows[0].Platform != "youtube" ||
		rows[0].Views != 100 || rows[0].Follows != 2 {
		t.Fatalf("unexpected CLIP1 row: %+v", rows)
	}
}

func TestAggregateClips(t *testing.T) {
	rows := []ClipPlatformRow{
		{OpusID: "C1", Title: "one", Platform: "youtube", Views: 2813, Follows: 2},
		{OpusID: "C1", Platform: "facebook", Views: 532},
		{OpusID: "C2", Title: "two", Platform: "youtube", Views: 42},
		{OpusID: "C2", Platform: "facebook", Views: 1377},
	}
	agg := aggregateClips(rows)

	// platform ranking by views: facebook (1909) > youtube (2855)? no — youtube 2855 > facebook 1909
	if len(agg.Platforms) != 2 || agg.Platforms[0].Platform != "youtube" {
		t.Fatalf("platform ranking wrong: %+v", agg.Platforms)
	}
	if agg.Platforms[0].Views != 2855 || agg.Platforms[0].Posts != 2 {
		t.Errorf("youtube total wrong: %+v", agg.Platforms[0])
	}
	if agg.GrandViews != 4764 || agg.GrandFollows != 2 {
		t.Errorf("grand totals wrong: views=%d follows=%d", agg.GrandViews, agg.GrandFollows)
	}
	// clip ranking by cross-platform total: C1 (3345) > C2 (1419)
	if len(agg.Clips) != 2 || agg.Clips[0].OpusID != "C1" || agg.Clips[0].Total != 3345 {
		t.Fatalf("clip ranking wrong: %+v", agg.Clips)
	}
	if agg.Clips[0].ByPlat["facebook"] != 532 {
		t.Errorf("C1 facebook views wrong: %d", agg.Clips[0].ByPlat["facebook"])
	}
}

func TestRenderClipReport(t *testing.T) {
	doc := ClipResultsDoc{Project: "P123", MeasuredAt: time.Now().Format("2006-01-02"), Rows: []ClipPlatformRow{
		{OpusID: "C1", Title: "hero", Platform: "youtube", Views: 2813},
		{OpusID: "C1", Platform: "facebook", Views: 532},
	}}
	agg := aggregateClips(doc.Rows)
	md := renderClipReport(doc, agg)
	for _, want := range []string{"P123", "youtube", "facebook", "GRAND TOTAL", "3,345"} {
		if !strings.Contains(md, want) {
			t.Errorf("report missing %q\n---\n%s", want, md)
		}
	}
}
