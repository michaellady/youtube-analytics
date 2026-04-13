package main

import (
	"fmt"
	"strings"
)

func runVideo(dataPath, id string) error {
	data, err := loadData(dataPath)
	if err != nil {
		return err
	}
	var found *Video
	for i := range data.Videos {
		if data.Videos[i].ID == id {
			found = &data.Videos[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("video %q not found in %s", id, dataPath)
	}
	v := *found

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  %s\n", v.Title)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  ID:            %s\n", v.ID)
	fmt.Printf("  URL:           https://youtube.com/watch?v=%s\n", v.ID)
	fmt.Printf("  Type:          %s\n", v.VideoType)
	fmt.Printf("  Published:     %s\n", v.PublishedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  Duration:      %ds (%s)\n", v.DurationSeconds, v.Duration)
	fmt.Println()

	fmt.Println("── Engagement ────────────────────────────────────────────")
	fmt.Printf("  Views:         %s\n", fmtNum(v.ViewCount))
	fmt.Printf("  Likes:         %s\n", fmtNum(v.LikeCount))
	fmt.Printf("  Comments:      %s\n", fmtNum(v.CommentCount))
	fmt.Printf("  Engagement:    %.2f%%\n", engagementRate(v)*100)
	fmt.Println()

	if v.EstimatedMinutesWatched > 0 || v.AverageViewPercentage > 0 || v.SubscribersGained > 0 {
		fmt.Println("── Analytics ─────────────────────────────────────────────")
		fmt.Printf("  Watch time:    %.0f minutes (%.1f hours)\n", v.EstimatedMinutesWatched, v.EstimatedMinutesWatched/60)
		fmt.Printf("  Avg duration:  %.0f seconds\n", v.AverageViewDuration)
		fmt.Printf("  Retention:     %.1f%%\n", v.AverageViewPercentage)
		fmt.Printf("  Subs gained:   +%d\n", v.SubscribersGained)
		fmt.Printf("  Subs lost:     -%d\n", v.SubscribersLost)
		fmt.Printf("  Net subs:      %+d\n", v.SubscribersGained-v.SubscribersLost)
		fmt.Printf("  Subs/1K views: %.2f\n", subsPer1K(v))
		fmt.Println()
	}

	if v.EstimatedRevenue > 0 || v.AdImpressions > 0 {
		fmt.Println("── Monetization ──────────────────────────────────────────")
		fmt.Printf("  Revenue:       $%.4f\n", v.EstimatedRevenue)
		fmt.Printf("  CPM:           $%.2f\n", v.CPM)
		fmt.Printf("  Ad impressions:%s\n", fmtNum(v.AdImpressions))
		fmt.Printf("  Monetized plays:%s\n", fmtNum(v.MonetizedPlaybacks))
		fmt.Println()
	}

	fmt.Println("── Tags ──────────────────────────────────────────────────")
	if len(v.Tags) == 0 {
		fmt.Println("  (no tags)")
	} else {
		fmt.Printf("  %s\n", strings.Join(v.Tags, ", "))
	}
	fmt.Println()

	fmt.Println("── Description ───────────────────────────────────────────")
	if v.Description == "" {
		fmt.Println("  (no description)")
	} else {
		for _, line := range strings.Split(v.Description, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
	fmt.Println()

	if v.ThumbnailURL != "" {
		fmt.Printf("Thumbnail: %s\n", v.ThumbnailURL)
	}
	return nil
}
