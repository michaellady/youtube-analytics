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
| `fetch` | Fetch all video metadata from YouTube Data API v3. Saves to `data/videos.json` |
| `fetch-analytics` | Fetch watch time data via OAuth2. Merges into `data/videos.json` |
| `analyze` | Print a detailed terminal report with trends and insights |
| `dashboard` | Generate an interactive HTML dashboard at `data/dashboard.html` |

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
├── main.go              # CLI entry point and subcommand routing
├── models.go            # Data types (Video, ChannelData, analysis types)
├── fetch.go             # YouTube Data API v3 video fetcher
├── oauth.go             # OAuth2 flow (local callback server, token management)
├── fetch_analytics.go   # YouTube Analytics API v2 queries
├── analyze.go           # Analysis logic and terminal report
├── dashboard.go         # HTML dashboard generation (Chart.js)
├── go.mod
├── go.sum
├── .env                 # API key and channel handle (not committed)
├── client_secret.json   # OAuth2 credentials (not committed)
└── data/                # Output directory (not committed)
    ├── videos.json      # Fetched video + analytics data
    ├── token.json       # Saved OAuth2 token
    └── dashboard.html   # Generated dashboard
```
