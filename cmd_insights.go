package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"
)

func cmdInsights(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("insights requires a subcommand: list|pending|grade|new")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "list":
		return runInsightsList(rest)
	case "pending":
		return runInsightsPending(rest)
	case "grade":
		return runInsightsGrade(rest)
	case "new":
		return runInsightsNew(rest)
	default:
		return fmt.Errorf("unknown insights subcommand: %s", sub)
	}
}

func runInsightsList(args []string) error {
	fs := flag.NewFlagSet("insights list", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	files, err := listInsightFiles(insightsDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Printf("No insight files in %s. Use `insights new <date>` to create one.\n", insightsDir)
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tHYPOTHESIS_ID\tCOHORT\tEVAL_AFTER\tVERDICT\tPREDICTION")
	for _, f := range files {
		date := f.Date.Format("2006-01-02")
		if f.Date.IsZero() {
			date = filepath.Base(f.Path)
		}
		if len(f.Hypotheses) == 0 {
			fmt.Fprintf(w, "%s\t(no hypotheses)\t\t\t\t\n", date)
			continue
		}
		for _, h := range f.Hypotheses {
			verdict := h.Verdict
			if verdict == "" {
				verdict = "(pending)"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				date, h.ID, h.Cohort,
				h.EvaluateAfter.Format("2006-01-02"),
				verdict, truncate(h.Prediction, 60))
		}
	}
	return w.Flush()
}

// runInsightsPending lists hypotheses past their evaluate_after with no verdict,
// AND emits the current metrics for each cited video so the model can grade
// them without re-running analyze. Pure transport — no judgment.
func runInsightsPending(args []string) error {
	fs := flag.NewFlagSet("insights pending", flag.ExitOnError)
	asOf := fs.String("as-of", "", "evaluate as of this date (default: today)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	now := time.Now()
	if *asOf != "" {
		t, err := time.Parse("2006-01-02", *asOf)
		if err != nil {
			return fmt.Errorf("--as-of: %w", err)
		}
		now = t
	}

	pending, err := pendingHypotheses(insightsDir, now)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		fmt.Println("No pending hypotheses.")
		return nil
	}

	data, err := loadData(videosJSONPath)
	if err != nil {
		return err
	}
	byID := make(map[string]Video, len(data.Videos))
	for _, v := range data.Videos {
		byID[v.ID] = v
	}

	for _, h := range pending {
		fmt.Printf("─── %s ───────────────────────────────────────────\n", h.ID)
		fmt.Printf("  Cohort:        %s\n", h.Cohort)
		fmt.Printf("  Prediction:    %s\n", h.Prediction)
		if h.Metric != "" {
			fmt.Printf("  Metric:        %s (direction: %s)\n", h.Metric, h.Direction)
		}
		fmt.Printf("  Evaluate after: %s\n", h.EvaluateAfter.Format("2006-01-02"))
		if len(h.EvidenceVideoIDs) > 0 {
			fmt.Println("  Evidence videos (current metrics):")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "    \tVIDEO_ID\tVIEWS\tWATCH_MIN\tRET%\tNET_SUBS\tTITLE")
			for _, vid := range h.EvidenceVideoIDs {
				v, ok := byID[vid]
				if !ok {
					fmt.Fprintf(w, "    \t%s\t(not found)\t\t\t\t\n", vid)
					continue
				}
				fmt.Fprintf(w, "    \t%s\t%d\t%.0f\t%.1f\t%+d\t%s\n",
					v.ID, v.ViewCount, v.EstimatedMinutesWatched,
					v.AverageViewPercentage,
					v.SubscribersGained-v.SubscribersLost,
					truncate(v.Title, 50))
			}
			w.Flush()
		}
		fmt.Println()
	}
	fmt.Printf("%d pending hypothesis(es). Grade with: insights grade <id> --verdict <v> --outcome \"<text>\"\n", len(pending))
	return nil
}

func runInsightsGrade(args []string) error {
	fs := flag.NewFlagSet("insights grade", flag.ExitOnError)
	verdict := fs.String("verdict", "", "confirm | refute | inconclusive")
	outcome := fs.String("outcome", "", "free-form outcome description (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: insights grade <hypothesis-id> --verdict <v> --outcome \"<text>\"")
	}
	id := fs.Arg(0)
	switch *verdict {
	case "confirm", "refute", "inconclusive":
	default:
		return fmt.Errorf("--verdict must be confirm|refute|inconclusive, got %q", *verdict)
	}
	if *outcome == "" {
		return fmt.Errorf("--outcome is required (describe what actually happened)")
	}
	if err := gradeHypothesis(insightsDir, id, *verdict, *outcome); err != nil {
		return err
	}
	fmt.Printf("Graded %s: %s — %s\n", id, *verdict, *outcome)
	return nil
}

func runInsightsNew(args []string) error {
	fs := flag.NewFlagSet("insights new", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	dateStr := ""
	if fs.NArg() > 0 {
		dateStr = fs.Arg(0)
	}
	var date time.Time
	if dateStr == "" {
		date = mostRecentMonday(time.Now())
	} else {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date %q (want YYYY-MM-DD): %w", dateStr, err)
		}
		date = t
	}

	path := filepath.Join(insightsDir, date.Format("2006-01-02")+".md")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists; edit it directly", path)
	}

	body := fmt.Sprintf("# Insights — %s\n\nNotes / hypotheses for the week starting %s.\n",
		date.Format("2006-01-02"), date.Format("2006-01-02"))

	if err := writeInsightFile(path, &InsightFile{
		Date: date,
		Body: body,
	}); err != nil {
		return err
	}
	fmt.Printf("Created %s — edit it to add hypotheses with frontmatter.\n", path)
	return nil
}

// mostRecentMonday returns the most recent Monday on or before now.
func mostRecentMonday(now time.Time) time.Time {
	day := int(now.Weekday())
	if day == 0 {
		day = 7
	}
	offset := day - 1
	monday := now.AddDate(0, 0, -offset)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}
