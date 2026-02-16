package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

func runAnalyze(dataPath string) error {
	data, err := loadData(dataPath)
	if err != nil {
		return err
	}

	result := analyze(data)
	printReport(result, data)
	return nil
}

func loadData(path string) (*ChannelData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w (run 'fetch' first)", path, err)
	}
	defer f.Close()

	var data ChannelData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &data, nil
}

func analyze(data *ChannelData) *AnalysisResult {
	r := &AnalysisResult{}

	var shorts, longform, live []Video
	for _, v := range data.Videos {
		switch v.VideoType {
		case "short":
			shorts = append(shorts, v)
		case "long-form":
			longform = append(longform, v)
		case "live":
			live = append(live, v)
		}
	}

	r.TotalVideos = len(data.Videos)
	r.Shorts = len(shorts)
	r.LongForm = len(longform)
	r.LiveStreams = len(live)

	// --- Shorts bulk posting impact analysis ---
	r.BulkPostingDays = findBulkPostingDays(shorts)
	bulkDates := map[string]bool{}
	for _, bp := range r.BulkPostingDays {
		bulkDates[bp.Date] = true
	}

	// Expand bulk dates to a ±7 day window
	bulkWindow := map[string]bool{}
	for _, bp := range r.BulkPostingDays {
		t, err := parseDateStr(bp.Date)
		if err != nil {
			continue
		}
		for d := -7; d <= 7; d++ {
			bulkWindow[t.AddDate(0, 0, d).Format("2006-01-02")] = true
		}
	}

	var bulkLF, nonBulkLF []Video
	for _, v := range longform {
		date := v.PublishedAt.Format("2006-01-02")
		if bulkWindow[date] {
			bulkLF = append(bulkLF, v)
		} else {
			nonBulkLF = append(nonBulkLF, v)
		}
	}

	r.BulkPeriodPerf = calcPerformance(bulkLF)
	r.NonBulkPeriodPerf = calcPerformance(nonBulkLF)

	// --- Monthly trends by type ---
	r.ShortsOverTime = monthlyBuckets(shorts)
	r.LongFormOverTime = monthlyBuckets(longform)
	r.LiveOverTime = monthlyBuckets(live)

	// --- Top/bottom videos ---
	r.TopByViews = topN(data.Videos, 10, func(a, b Video) bool { return a.ViewCount > b.ViewCount })
	r.BottomByViews = topN(filterMinViews(longform, 0), 10, func(a, b Video) bool { return a.ViewCount < b.ViewCount })
	r.TopByEngagement = topN(filterMinViews(data.Videos, 100), 10, func(a, b Video) bool {
		return engagementRate(a) > engagementRate(b)
	})

	// --- Title analysis ---
	r.TitleInsights = analyzeTitles(longform)

	// --- Watch time analysis (only if analytics data is available) ---
	if data.HasAnalytics {
		r.HasAnalytics = true
		r.WatchTime = analyzeWatchTime(data.Videos)
		r.ShortsWatchTime = analyzeWatchTime(shorts)
		r.LongFormWatchTime = analyzeWatchTime(longform)
		r.WatchTimeOverTime = watchTimeMonthlyBuckets(data.Videos)
		r.TopByWatchTime = topN(data.Videos, 10, func(a, b Video) bool {
			return a.EstimatedMinutesWatched > b.EstimatedMinutesWatched
		})
		r.TopByRetention = topN(filterMinViews(data.Videos, 100), 10, func(a, b Video) bool {
			return a.AverageViewPercentage > b.AverageViewPercentage
		})
	}

	return r
}

func findBulkPostingDays(shorts []Video) []BulkPostingDay {
	dayCounts := map[string]int{}
	for _, v := range shorts {
		day := v.PublishedAt.Format("2006-01-02")
		dayCounts[day]++
	}

	var bulk []BulkPostingDay
	for day, count := range dayCounts {
		if count >= 3 {
			bulk = append(bulk, BulkPostingDay{Date: day, ShortCount: count})
		}
	}
	sort.Slice(bulk, func(i, j int) bool { return bulk[i].Date < bulk[j].Date })
	return bulk
}

func calcPerformance(videos []Video) PeriodPerformance {
	if len(videos) == 0 {
		return PeriodPerformance{}
	}
	var totalViews, totalLikes, totalComments int64
	var totalEng float64
	engCount := 0
	for _, v := range videos {
		totalViews += v.ViewCount
		totalLikes += v.LikeCount
		totalComments += v.CommentCount
		if v.ViewCount > 0 {
			totalEng += float64(v.LikeCount+v.CommentCount) / float64(v.ViewCount)
			engCount++
		}
	}
	n := float64(len(videos))
	p := PeriodPerformance{
		VideoCount:  len(videos),
		AvgViews:    float64(totalViews) / n,
		AvgLikes:    float64(totalLikes) / n,
		AvgComments: float64(totalComments) / n,
	}
	if engCount > 0 {
		p.AvgEngagement = totalEng / float64(engCount)
	}
	return p
}

func monthlyBuckets(videos []Video) []TimeBucket {
	buckets := map[string]*TimeBucket{}
	for _, v := range videos {
		key := v.PublishedAt.Format("2006-01")
		b, ok := buckets[key]
		if !ok {
			b = &TimeBucket{Period: key}
			buckets[key] = b
		}
		b.VideoCount++
		b.TotalViews += v.ViewCount
		if v.ViewCount > 0 {
			b.AvgEng += float64(v.LikeCount+v.CommentCount) / float64(v.ViewCount)
		}
	}

	var result []TimeBucket
	for _, b := range buckets {
		if b.VideoCount > 0 {
			b.AvgViews = float64(b.TotalViews) / float64(b.VideoCount)
			b.AvgEng = b.AvgEng / float64(b.VideoCount)
		}
		result = append(result, *b)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Period < result[j].Period })
	return result
}

func topN(videos []Video, n int, less func(a, b Video) bool) []Video {
	sorted := make([]Video, len(videos))
	copy(sorted, videos)
	sort.Slice(sorted, func(i, j int) bool { return less(sorted[i], sorted[j]) })
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

func filterMinViews(videos []Video, min int64) []Video {
	var out []Video
	for _, v := range videos {
		if v.ViewCount >= min {
			out = append(out, v)
		}
	}
	return out
}

func engagementRate(v Video) float64 {
	if v.ViewCount == 0 {
		return 0
	}
	return float64(v.LikeCount+v.CommentCount) / float64(v.ViewCount)
}

var numberRe = regexp.MustCompile(`\d+`)

func analyzeTitles(longform []Video) TitleInsights {
	if len(longform) < 6 {
		return TitleInsights{}
	}

	sorted := make([]Video, len(longform))
	copy(sorted, longform)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ViewCount > sorted[j].ViewCount })

	n := int(math.Max(float64(len(sorted)/4), 3))
	top := sorted[:n]
	bottom := sorted[len(sorted)-n:]

	ti := TitleInsights{}

	for _, v := range top {
		ti.AvgLenTop += float64(len(v.Title))
		if strings.Contains(v.Title, "?") {
			ti.QuestionsTop++
		}
		if numberRe.MatchString(v.Title) {
			ti.NumbersTop++
		}
		ti.CapsWordsTop += float64(countCapsWords(v.Title))
	}

	for _, v := range bottom {
		ti.AvgLenBottom += float64(len(v.Title))
		if strings.Contains(v.Title, "?") {
			ti.QuestionsBottom++
		}
		if numberRe.MatchString(v.Title) {
			ti.NumbersBottom++
		}
		ti.CapsWordsBottom += float64(countCapsWords(v.Title))
	}

	nt := float64(len(top))
	nb := float64(len(bottom))
	ti.AvgLenTop /= nt
	ti.AvgLenBottom /= nb
	ti.CapsWordsTop /= nt
	ti.CapsWordsBottom /= nb

	return ti
}

func countCapsWords(s string) int {
	count := 0
	for _, word := range strings.Fields(s) {
		if len(word) >= 2 && word == strings.ToUpper(word) && hasLetter(word) {
			count++
		}
	}
	return count
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func analyzeWatchTime(videos []Video) WatchTimeAnalysis {
	if len(videos) == 0 {
		return WatchTimeAnalysis{}
	}
	w := WatchTimeAnalysis{}
	var totalDuration, totalRetention float64
	durationCount, retentionCount := 0, 0
	for _, v := range videos {
		w.TotalMinutes += v.EstimatedMinutesWatched
		w.SubscribersGained += v.SubscribersGained
		w.SubscribersLost += v.SubscribersLost
		if v.AverageViewDuration > 0 {
			totalDuration += v.AverageViewDuration
			durationCount++
		}
		if v.AverageViewPercentage > 0 {
			totalRetention += v.AverageViewPercentage
			retentionCount++
		}
	}
	w.AvgMinutesPerVideo = w.TotalMinutes / float64(len(videos))
	if durationCount > 0 {
		w.AvgViewDuration = totalDuration / float64(durationCount)
	}
	if retentionCount > 0 {
		w.AvgRetention = totalRetention / float64(retentionCount)
	}
	w.NetSubscribers = w.SubscribersGained - w.SubscribersLost
	return w
}

func watchTimeMonthlyBuckets(videos []Video) []WatchTimeBucket {
	buckets := map[string]*WatchTimeBucket{}
	for _, v := range videos {
		key := v.PublishedAt.Format("2006-01")
		b, ok := buckets[key]
		if !ok {
			b = &WatchTimeBucket{Period: key}
			buckets[key] = b
		}
		b.TotalMinutes += v.EstimatedMinutesWatched
		b.VideoCount++
		if v.AverageViewPercentage > 0 {
			b.AvgRetention += v.AverageViewPercentage
		}
	}

	var result []WatchTimeBucket
	for _, b := range buckets {
		if b.VideoCount > 0 {
			b.AvgMinutesPerVideo = b.TotalMinutes / float64(b.VideoCount)
			b.AvgRetention = b.AvgRetention / float64(b.VideoCount)
		}
		result = append(result, *b)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Period < result[j].Period })
	return result
}

func parseDateStr(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func printReport(r *AnalysisResult, data *ChannelData) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  YouTube Analytics Report: %s\n", data.ChannelTitle)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// --- Overview ---
	fmt.Println("── Overview ──────────────────────────────────────────────")
	fmt.Printf("  Total videos:   %d\n", r.TotalVideos)
	fmt.Printf("  Shorts:         %d\n", r.Shorts)
	fmt.Printf("  Long-form:      %d\n", r.LongForm)
	fmt.Printf("  Live streams:   %d\n", r.LiveStreams)
	fmt.Println()

	// --- Shorts Bulk Posting Impact ---
	fmt.Println("── Shorts Bulk Posting Impact ─────────────────────────────")
	if len(r.BulkPostingDays) == 0 {
		fmt.Println("  No bulk posting days detected (3+ shorts in one day).")
	} else {
		fmt.Printf("  Bulk posting days found: %d\n", len(r.BulkPostingDays))
		for _, bp := range r.BulkPostingDays {
			fmt.Printf("    %s: %d shorts\n", bp.Date, bp.ShortCount)
		}
		fmt.Println()
		fmt.Println("  Long-form performance NEAR bulk posting (±7 days):")
		printPerf("    ", r.BulkPeriodPerf)
		fmt.Println("  Long-form performance AWAY from bulk posting:")
		printPerf("    ", r.NonBulkPeriodPerf)

		if r.BulkPeriodPerf.VideoCount > 0 && r.NonBulkPeriodPerf.VideoCount > 0 {
			viewsDiff := (r.BulkPeriodPerf.AvgViews - r.NonBulkPeriodPerf.AvgViews) / r.NonBulkPeriodPerf.AvgViews * 100
			fmt.Printf("\n  → Views difference: %+.1f%%\n", viewsDiff)
			if viewsDiff < -10 {
				fmt.Println("  ⚠ Long-form videos near bulk shorts posting get FEWER views.")
			} else if viewsDiff > 10 {
				fmt.Println("  ✓ Long-form videos near bulk shorts posting get MORE views.")
			} else {
				fmt.Println("  ~ No significant difference detected.")
			}
		}
	}
	fmt.Println()

	// --- Shorts Trend ---
	fmt.Println("── Shorts Performance Over Time ───────────────────────────")
	if len(r.ShortsOverTime) > 0 {
		fmt.Printf("  %-10s  %5s  %10s  %8s\n", "Month", "Count", "Avg Views", "Avg Eng%")
		for _, b := range r.ShortsOverTime {
			fmt.Printf("  %-10s  %5d  %10.0f  %7.2f%%\n", b.Period, b.VideoCount, b.AvgViews, b.AvgEng*100)
		}

		if len(r.ShortsOverTime) >= 3 {
			first := r.ShortsOverTime[0]
			last := r.ShortsOverTime[len(r.ShortsOverTime)-1]
			if first.AvgViews > 0 {
				trend := (last.AvgViews - first.AvgViews) / first.AvgViews * 100
				fmt.Printf("\n  → Shorts views trend (first→last month): %+.1f%%\n", trend)
			}
		}
	} else {
		fmt.Println("  No shorts found.")
	}
	fmt.Println()

	// --- Long-form Trend ---
	fmt.Println("── Long-Form Performance Over Time ────────────────────────")
	if len(r.LongFormOverTime) > 0 {
		fmt.Printf("  %-10s  %5s  %10s  %8s\n", "Month", "Count", "Avg Views", "Avg Eng%")
		for _, b := range r.LongFormOverTime {
			fmt.Printf("  %-10s  %5d  %10.0f  %7.2f%%\n", b.Period, b.VideoCount, b.AvgViews, b.AvgEng*100)
		}
	}
	fmt.Println()

	// --- Top Videos ---
	fmt.Println("── Top 10 Videos by Views ─────────────────────────────────")
	for i, v := range r.TopByViews {
		fmt.Printf("  %2d. [%s] %s\n", i+1, v.VideoType, truncate(v.Title, 55))
		fmt.Printf("      Views: %-10s  Likes: %-8s  Eng: %.2f%%\n",
			fmtNum(v.ViewCount), fmtNum(v.LikeCount), engagementRate(v)*100)
	}
	fmt.Println()

	// --- Top by Engagement ---
	fmt.Println("── Top 10 Videos by Engagement Rate (min 100 views) ──────")
	for i, v := range r.TopByEngagement {
		fmt.Printf("  %2d. [%s] %s\n", i+1, v.VideoType, truncate(v.Title, 55))
		fmt.Printf("      Views: %-10s  Eng: %.2f%%\n", fmtNum(v.ViewCount), engagementRate(v)*100)
	}
	fmt.Println()

	// --- Title Analysis ---
	fmt.Println("── Title Analysis (Top vs Bottom quartile, long-form) ────")
	ti := r.TitleInsights
	if ti.AvgLenTop > 0 {
		fmt.Printf("  Avg title length:     Top: %.0f chars   Bottom: %.0f chars\n", ti.AvgLenTop, ti.AvgLenBottom)
		fmt.Printf("  Titles with '?':      Top: %d          Bottom: %d\n", ti.QuestionsTop, ti.QuestionsBottom)
		fmt.Printf("  Titles with numbers:  Top: %d          Bottom: %d\n", ti.NumbersTop, ti.NumbersBottom)
		fmt.Printf("  Avg ALL-CAPS words:   Top: %.1f        Bottom: %.1f\n", ti.CapsWordsTop, ti.CapsWordsBottom)
	} else {
		fmt.Println("  Not enough long-form videos for title analysis.")
	}
	fmt.Println()

	// --- Watch Time sections (only if analytics data available) ---
	if r.HasAnalytics {
		fmt.Println("── Watch Time Overview ────────────────────────────────────")
		fmt.Printf("  Total watch time:       %.0f minutes (%.1f hours)\n", r.WatchTime.TotalMinutes, r.WatchTime.TotalMinutes/60)
		fmt.Printf("  Avg per video:          %.1f minutes\n", r.WatchTime.AvgMinutesPerVideo)
		fmt.Printf("  Avg view duration:      %.0f seconds\n", r.WatchTime.AvgViewDuration)
		fmt.Printf("  Avg retention:          %.1f%%\n", r.WatchTime.AvgRetention)
		fmt.Printf("  Net subscribers:        %+d (gained: %d, lost: %d)\n",
			r.WatchTime.NetSubscribers, r.WatchTime.SubscribersGained, r.WatchTime.SubscribersLost)
		fmt.Println()

		fmt.Println("── Watch Time by Content Type ─────────────────────────────")
		fmt.Printf("  %-12s  %12s  %10s  %10s  %10s\n", "Type", "Total Min", "Avg Min", "Avg Dur(s)", "Retention")
		fmt.Printf("  %-12s  %12.0f  %10.1f  %10.0f  %9.1f%%\n", "Shorts",
			r.ShortsWatchTime.TotalMinutes, r.ShortsWatchTime.AvgMinutesPerVideo,
			r.ShortsWatchTime.AvgViewDuration, r.ShortsWatchTime.AvgRetention)
		fmt.Printf("  %-12s  %12.0f  %10.1f  %10.0f  %9.1f%%\n", "Long-Form",
			r.LongFormWatchTime.TotalMinutes, r.LongFormWatchTime.AvgMinutesPerVideo,
			r.LongFormWatchTime.AvgViewDuration, r.LongFormWatchTime.AvgRetention)
		fmt.Println()

		fmt.Println("── Top 10 Videos by Watch Time ────────────────────────────")
		for i, v := range r.TopByWatchTime {
			fmt.Printf("  %2d. [%s] %s\n", i+1, v.VideoType, truncate(v.Title, 55))
			fmt.Printf("      Watch: %.0f min  Views: %-10s  Retention: %.1f%%\n",
				v.EstimatedMinutesWatched, fmtNum(v.ViewCount), v.AverageViewPercentage)
		}
		fmt.Println()

		fmt.Println("── Top 10 Videos by Retention (min 100 views) ────────────")
		for i, v := range r.TopByRetention {
			fmt.Printf("  %2d. [%s] %s\n", i+1, v.VideoType, truncate(v.Title, 55))
			fmt.Printf("      Retention: %.1f%%  Avg Duration: %.0fs  Views: %s\n",
				v.AverageViewPercentage, v.AverageViewDuration, fmtNum(v.ViewCount))
		}
		fmt.Println()

		fmt.Println("── Monthly Watch Time Trend ───────────────────────────────")
		if len(r.WatchTimeOverTime) > 0 {
			fmt.Printf("  %-10s  %5s  %12s  %10s  %10s\n", "Month", "Count", "Total Min", "Avg Min", "Retention")
			for _, b := range r.WatchTimeOverTime {
				fmt.Printf("  %-10s  %5d  %12.0f  %10.1f  %9.1f%%\n",
					b.Period, b.VideoCount, b.TotalMinutes, b.AvgMinutesPerVideo, b.AvgRetention)
			}
		}
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
}

func printPerf(prefix string, p PeriodPerformance) {
	fmt.Printf("%sVideos: %d | Avg views: %.0f | Avg likes: %.0f | Avg engagement: %.2f%%\n",
		prefix, p.VideoCount, p.AvgViews, p.AvgLikes, p.AvgEngagement*100)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func fmtNum(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
