# YouTube Analytics CLI

A Go CLI tool that fetches your YouTube channel data and generates detailed analytics reports and interactive dashboards. Includes watch time, retention, subscriber impact, and posting pattern analysis.

## Features

- **Video metadata** via YouTube Data API v3 (views, likes, comments, duration, tags)
- **Watch time analytics** via YouTube Analytics API v2 with OAuth2 (estimated minutes watched, average view duration, audience retention, subscriber gains/losses)
- **Terminal report** with posting patterns, engagement trends, title analysis, and watch time breakdowns
- **Interactive HTML dashboard** with Chart.js visualizations

## Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- A [Google Cloud](https://console.cloud.google.com) project with:
  - **YouTube Data API v3** enabled
  - **YouTube Analytics API** enabled (for watch time data)

## Setup

### 1. Get a YouTube Data API key

1. Go to [Google Cloud Console](https://console.cloud.google.com) > APIs & Services > Credentials
2. Click **Create Credentials** > **API Key**
3. (Recommended) Restrict the key to YouTube Data API v3

### 2. Create a `.env` file in the project root

```
YOUTUBE_API_KEY=your-api-key-here
CHANNEL_HANDLE=@YourChannelHandle
```

### 3. (Optional) Set up OAuth2 for watch time data

The `fetch-analytics` command requires OAuth2 authentication as the channel owner.

1. In Google Cloud Console > APIs & Services > Credentials, click **Create Credentials** > **OAuth client ID**
2. Application type: **Desktop app**
3. Download the JSON and save it as `client_secret.json` in the project root
4. Go to **OAuth consent screen** and add your Google account email as a **test user**
5. Under the client's settings, add `http://localhost:8089/callback` as an **Authorized redirect URI**

## Usage

```bash
# Install dependencies
go mod download

# Fetch video metadata (uses API key)
go run . fetch

# Fetch watch time analytics (opens browser for OAuth2)
go run . fetch-analytics

# Print terminal report
go run . analyze

# Generate interactive HTML dashboard
go run . dashboard
```

### Commands

| Command | Description |
|---|---|
| `fetch` | Fetch all video metadata from YouTube Data API v3. Saves to `data/videos.json` and snapshots to `data/snapshots/` (keeps last 30) |
| `fetch-analytics` | Fetch watch time data via OAuth2. Merges into `data/videos.json` |
| `analyze` | Print a detailed terminal report. Supports `--diff <snapshot>` and `--compare-periods A..B C..D` modes |
| `dashboard` | Generate an interactive HTML dashboard at `data/dashboard.html` |
| `find-duplicates` | Detect near-duplicate uploads (same title, similar duration, close timestamps) |
| `data-quality` | Report videos missing tags, with short or templated descriptions, or missing thumbnails |
| `titles` | Title pattern analysis with keyword lift for top performers |
| `video <id>` | Deep-dive on a single video with all analytics fields |
| `export` | Flatten video data to CSV or TSV |

### Filter flags

`analyze`, `dashboard`, `find-duplicates`, `data-quality`, `titles`, and `export` all accept:

| Flag | Description |
|---|---|
| `--since YYYY-MM-DD` | Videos published on or after this date |
| `--until YYYY-MM-DD` | Videos published on or before this date |
| `--type <short\|long-form\|live>` | Filter by video type (repeatable) |
| `--duration-min N` | Minimum duration in seconds |
| `--duration-max N` | Maximum duration in seconds |
| `--exclude <ID>` | Exclude a specific video ID (repeatable) |

Examples:

```bash
# Recent 3-10 min long-form only
go run . analyze --since 2026-04-01 --type long-form --duration-min 180 --duration-max 600

# Title pattern for top live streams
go run . titles --top 10 --by views --type live

# Export recent videos to CSV
go run . export --format csv --since 2026-04-01 --output recent.csv

# Delta since a snapshot
go run . analyze --diff data/snapshots/videos-20260401T000000Z.json

# Compare two months side-by-side
go run . analyze --compare-periods 2026-01-01..2026-01-31 2026-03-01..2026-03-31
```

## What You Get

### Terminal Report (`analyze`)

- Video count breakdown (shorts, long-form, live)
- Shorts bulk posting impact on long-form performance
- Monthly performance trends by content type
- Top/bottom videos by views and engagement
- Title analysis (top vs bottom quartile)
- Watch time overview, by content type, and monthly trends
- Top videos by watch time and retention

### HTML Dashboard (`dashboard`)

- Stat cards (total videos, shorts, long-form, live, watch time, retention, subscribers)
- Video performance timeline (scatter chart)
- Monthly trends by content type (bar + engagement line charts)
- Shorts posting frequency vs long-form performance (dual-axis)
- Monthly watch time and retention (dual-axis)
- Watch time by content type (doughnut)
- Subscriber impact by type (grouped bar)
- Sortable tables for top videos

All watch time sections gracefully degrade if `fetch-analytics` hasn't been run.

## Project Structure

```
.
├── main.go              # CLI entry point, flag.FlagSet dispatch, shared filter flags
├── models.go            # Data types (Video, ChannelData, VideoFilter, analysis types)
├── fetch.go             # YouTube Data API v3 video fetcher
├── oauth.go             # OAuth2 flow (local callback server, token management)
├── fetch_analytics.go   # YouTube Analytics API v2 queries
├── snapshot.go          # Per-fetch snapshots of videos.json
├── analyze.go           # Analysis logic and terminal report
├── dashboard.go         # HTML dashboard generation (Chart.js)
├── diff.go              # analyze --diff + --compare-periods
├── duplicates.go        # find-duplicates subcommand
├── data_quality.go      # data-quality subcommand
├── titles.go            # titles subcommand
├── video.go             # video <id> subcommand
├── export.go            # export subcommand
├── go.mod
├── go.sum
├── .env                 # API key and channel handle (not committed)
├── client_secret.json   # OAuth2 credentials (not committed)
└── data/                # Output directory (not committed)
    ├── videos.json      # Fetched video + analytics data
    ├── token.json       # Saved OAuth2 token
    ├── dashboard.html   # Generated dashboard
    └── snapshots/       # Auto-saved videos.json snapshots (keeps last 30)
```
