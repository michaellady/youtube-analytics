package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ClipPlatformRow is one clip's performance on one platform. The browser pass
// (see skill/clip-results) writes these for the no-API platforms (facebook,
// instagram, tiktok, linkedin_org, linkedin_personal); the youtube rows are
// joined in deterministically from data/videos.json by this command.
type ClipPlatformRow struct {
	OpusID   string `json:"opus_id"`
	Title    string `json:"title,omitempty"`
	Platform string `json:"platform"`
	Views    int64  `json:"views"`
	Likes    int64  `json:"likes,omitempty"`
	Comments int64  `json:"comments,omitempty"`
	Follows  int64  `json:"follows,omitempty"` // subs (YouTube) / follows (Meta) gained
}

// ClipResultsDoc is data/opus_clips/<project>-platform-results.json — the
// browser pass appends rows here, then `clip-results report` consolidates.
type ClipResultsDoc struct {
	Project     string            `json:"project"`
	SourceVideo string            `json:"source_video,omitempty"`
	MeasuredAt  string            `json:"measured_at,omitempty"`
	Rows        []ClipPlatformRow `json:"rows"`
}

var opusTagRe = regexp.MustCompile(`\[opus:([A-Za-z0-9_]+)\]`)

// extractOpusID pulls the CLIPID out of the first `[opus:CLIPID]` tag in s.
func extractOpusID(s string) (string, bool) {
	m := opusTagRe.FindStringSubmatch(s)
	if m == nil || m[1] == "" {
		return "", false
	}
	return m[1], true
}

// youtubeRowsForClips returns a youtube row for every video whose description
// carries an [opus:ID] tag. If clipIDs is non-empty, only those IDs are kept.
func youtubeRowsForClips(data *ChannelData, clipIDs []string) []ClipPlatformRow {
	want := map[string]bool{}
	for _, id := range clipIDs {
		want[id] = true
	}
	var rows []ClipPlatformRow
	for _, v := range data.Videos {
		id, ok := extractOpusID(v.Description)
		if !ok {
			continue
		}
		if len(want) > 0 && !want[id] {
			continue
		}
		rows = append(rows, ClipPlatformRow{
			OpusID:   id,
			Title:    v.Title,
			Platform: "youtube",
			Views:    v.ViewCount,
			Likes:    v.LikeCount,
			Comments: v.CommentCount,
			Follows:  v.SubscribersGained,
		})
	}
	return rows
}

// PlatformTotal aggregates one platform across all clips in a batch.
type PlatformTotal struct {
	Platform string
	Posts    int
	Views    int64
	Likes    int64
	Comments int64
	Follows  int64
}

func (p PlatformTotal) AvgViews() float64 {
	if p.Posts == 0 {
		return 0
	}
	return float64(p.Views) / float64(p.Posts)
}

// ClipAcross is one clip's view counts across every platform it ran on.
type ClipAcross struct {
	OpusID string
	Title  string
	ByPlat map[string]int64
	Total  int64
}

// ClipAggregation is the fully-reduced view of a batch.
type ClipAggregation struct {
	Platforms    []PlatformTotal // sorted by views desc
	Clips        []ClipAcross    // sorted by cross-platform total views desc
	GrandViews   int64
	GrandFollows int64
}

func aggregateClips(rows []ClipPlatformRow) ClipAggregation {
	platAgg := map[string]*PlatformTotal{}
	clipAgg := map[string]*ClipAcross{}
	for _, r := range rows {
		p := platAgg[r.Platform]
		if p == nil {
			p = &PlatformTotal{Platform: r.Platform}
			platAgg[r.Platform] = p
		}
		p.Posts++
		p.Views += r.Views
		p.Likes += r.Likes
		p.Comments += r.Comments
		p.Follows += r.Follows

		c := clipAgg[r.OpusID]
		if c == nil {
			c = &ClipAcross{OpusID: r.OpusID, ByPlat: map[string]int64{}}
			clipAgg[r.OpusID] = c
		}
		if r.Title != "" && c.Title == "" {
			c.Title = r.Title
		}
		c.ByPlat[r.Platform] += r.Views
		c.Total += r.Views
	}

	var agg ClipAggregation
	for _, p := range platAgg {
		agg.Platforms = append(agg.Platforms, *p)
		agg.GrandViews += p.Views
		agg.GrandFollows += p.Follows
	}
	sort.Slice(agg.Platforms, func(i, j int) bool {
		if agg.Platforms[i].Views != agg.Platforms[j].Views {
			return agg.Platforms[i].Views > agg.Platforms[j].Views
		}
		return agg.Platforms[i].Platform < agg.Platforms[j].Platform
	})
	for _, c := range clipAgg {
		agg.Clips = append(agg.Clips, *c)
	}
	sort.Slice(agg.Clips, func(i, j int) bool {
		if agg.Clips[i].Total != agg.Clips[j].Total {
			return agg.Clips[i].Total > agg.Clips[j].Total
		}
		return agg.Clips[i].OpusID < agg.Clips[j].OpusID
	})
	return agg
}

// commaInt formats 3345 as "3,345". Kept local so the report's number format is
// stable regardless of fmtNum's (K-suffix) behavior.
func commaInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// renderClipReport produces the consolidated cross-platform markdown report.
func renderClipReport(doc ClipResultsDoc, agg ClipAggregation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — cross-platform clip results\n\n", doc.Project)
	if doc.SourceVideo != "" {
		fmt.Fprintf(&b, "Source video: %s\n", doc.SourceVideo)
	}
	if doc.MeasuredAt != "" {
		fmt.Fprintf(&b, "Measured: %s\n", doc.MeasuredAt)
	}
	b.WriteString("\n> Metrics are NOT apples-to-apples across platforms (YouTube/TikTok \"views\" ≠ IG plays ≈ FB views ≠ LinkedIn impressions). Compare within a platform.\n\n")

	b.WriteString("## Platform totals (ranked by views/impressions)\n\n")
	b.WriteString("| Platform | Posts | Views | Avg/post | Likes | Comments | Follows |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, p := range agg.Platforms {
		fmt.Fprintf(&b, "| %s | %d | %s | %.0f | %d | %d | %d |\n",
			p.Platform, p.Posts, commaInt(p.Views), p.AvgViews(), p.Likes, p.Comments, p.Follows)
	}
	fmt.Fprintf(&b, "\n**GRAND TOTAL: %s views/impressions across all platforms** (follows/subs gained: %d).\n\n",
		commaInt(agg.GrandViews), agg.GrandFollows)

	// Platform-specific breakouts: the top clip per platform.
	b.WriteString("## Top clip per platform\n\n")
	for _, p := range agg.Platforms {
		var bestID, bestTitle string
		var best int64 = -1
		for _, c := range agg.Clips {
			if v, ok := c.ByPlat[p.Platform]; ok && v > best {
				best, bestID, bestTitle = v, c.OpusID, c.Title
			}
		}
		if best >= 0 {
			label := bestTitle
			if label == "" {
				label = bestID
			}
			fmt.Fprintf(&b, "- **%s**: %s (%s views)\n", p.Platform, label, commaInt(best))
		}
	}

	b.WriteString("\n## Clips by cross-platform reach\n\n")
	b.WriteString("| Total | Clip | Per-platform |\n|---|---|---|\n")
	for _, c := range agg.Clips {
		label := c.Title
		if label == "" {
			label = c.OpusID
		}
		var parts []string
		for _, p := range agg.Platforms {
			if v, ok := c.ByPlat[p.Platform]; ok {
				parts = append(parts, fmt.Sprintf("%s %s", p.Platform, commaInt(v)))
			}
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", commaInt(c.Total), label, strings.Join(parts, " · "))
	}
	return b.String()
}

func summaryText(doc ClipResultsDoc, agg ClipAggregation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %d clips, %d platform-posts\n", doc.Project, len(agg.Clips), countRows(agg))
	for _, p := range agg.Platforms {
		fmt.Fprintf(&b, "  %-18s %8s views  (%d posts, avg %.0f)\n", p.Platform, commaInt(p.Views), p.Posts, p.AvgViews())
	}
	fmt.Fprintf(&b, "  %-18s %8s\n", "GRAND TOTAL", commaInt(agg.GrandViews))
	return b.String()
}

func countRows(agg ClipAggregation) int {
	n := 0
	for _, p := range agg.Platforms {
		n += p.Posts
	}
	return n
}

func cmdClipResults(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("clip-results requires a subcommand: report")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "report":
		return runClipResultsReport(rest)
	default:
		return fmt.Errorf("unknown clip-results subcommand: %s", sub)
	}
}

func runClipResultsReport(args []string) error {
	fs := flag.NewFlagSet("clip-results report", flag.ExitOnError)
	resultsPath := fs.String("results", "", "path to <project>-platform-results.json (browser-filled rows)")
	dataPath := fs.String("data", "data/videos.json", "videos.json for the YouTube [opus:]-tag join")
	out := fs.String("out", "", "write the consolidated markdown report here (default: stdout summary only)")
	noYT := fs.Bool("no-youtube-join", false, "skip auto-adding YouTube rows from videos.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *resultsPath == "" {
		return fmt.Errorf("--results is required (the browser-filled <project>-platform-results.json)")
	}
	raw, err := os.ReadFile(*resultsPath)
	if err != nil {
		return err
	}
	var doc ClipResultsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", *resultsPath, err)
	}

	// Deterministic YouTube join (transport): add youtube rows for every opus id
	// present in the doc that isn't already covered.
	if !*noYT {
		ids := map[string]bool{}
		hasYT := map[string]bool{}
		for _, r := range doc.Rows {
			ids[r.OpusID] = true
			if r.Platform == "youtube" {
				hasYT[r.OpusID] = true
			}
		}
		data, err := loadData(*dataPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping YouTube join (%v)\n", err)
		} else {
			var idList []string
			for id := range ids {
				idList = append(idList, id)
			}
			for _, yr := range youtubeRowsForClips(data, idList) {
				if !hasYT[yr.OpusID] {
					doc.Rows = append(doc.Rows, yr)
				}
			}
		}
	}

	agg := aggregateClips(doc.Rows)
	fmt.Print(summaryText(doc, agg))
	if *out != "" {
		if err := os.WriteFile(*out, []byte(renderClipReport(doc, agg)), 0644); err != nil {
			return err
		}
		fmt.Printf("\nWrote %s\n", *out)
	}
	return nil
}
