package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "fetch":
		apiKey := os.Getenv("YOUTUBE_API_KEY")
		handle := os.Getenv("CHANNEL_HANDLE")
		if apiKey == "" || handle == "" {
			fmt.Fprintln(os.Stderr, "Error: YOUTUBE_API_KEY and CHANNEL_HANDLE must be set in .env")
			os.Exit(1)
		}
		err = runFetch(apiKey, handle)
	case "analyze":
		err = runAnalyze("data/videos.json")
	case "dashboard":
		err = runDashboard("data/videos.json", "data/dashboard.html")
	case "fetch-analytics":
		err = runFetchAnalytics("data/videos.json")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: youtube-analytics <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  fetch             Fetch all video data from YouTube API → data/videos.json")
	fmt.Println("  fetch-analytics   Fetch watch time data via OAuth2 → merge into data/videos.json")
	fmt.Println("  analyze           Analyze data and print terminal report")
	fmt.Println("  dashboard         Generate interactive HTML dashboard → data/dashboard.html")
}
