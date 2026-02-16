package main

import "time"

// Video represents a single YouTube video with all metadata and statistics.
type Video struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	PublishedAt     time.Time `json:"published_at"`
	ThumbnailURL    string    `json:"thumbnail_url"`
	Duration        string    `json:"duration"`
	DurationSeconds int       `json:"duration_seconds"`
	ViewCount       int64     `json:"view_count"`
	LikeCount       int64     `json:"like_count"`
	CommentCount    int64     `json:"comment_count"`
	Tags            []string  `json:"tags,omitempty"`
	VideoType       string    `json:"video_type"` // "short", "long-form", "live"

	// Analytics API fields (only populated after fetch-analytics)
	EstimatedMinutesWatched float64 `json:"estimated_minutes_watched,omitempty"`
	AverageViewDuration     float64 `json:"average_view_duration,omitempty"`     // seconds
	AverageViewPercentage   float64 `json:"average_view_percentage,omitempty"`   // 0-100
	SubscribersGained       int64   `json:"subscribers_gained,omitempty"`
	SubscribersLost         int64   `json:"subscribers_lost,omitempty"`
}

// ChannelData is the top-level structure saved to data/videos.json.
type ChannelData struct {
	ChannelID          string    `json:"channel_id"`
	ChannelTitle       string    `json:"channel_title"`
	Handle             string    `json:"handle"`
	FetchedAt          time.Time `json:"fetched_at"`
	HasAnalytics       bool      `json:"has_analytics,omitempty"`
	AnalyticsFetchedAt time.Time `json:"analytics_fetched_at,omitempty"`
	Videos             []Video   `json:"videos"`
}

// AnalysisResult holds all computed analysis data.
type AnalysisResult struct {
	TotalVideos int
	Shorts      int
	LongForm    int
	LiveStreams  int

	BulkPostingDays   []BulkPostingDay
	BulkPeriodPerf    PeriodPerformance
	NonBulkPeriodPerf PeriodPerformance

	ShortsOverTime   []TimeBucket
	LongFormOverTime []TimeBucket
	LiveOverTime     []TimeBucket

	TopByViews      []Video
	BottomByViews   []Video
	TopByEngagement []Video

	TitleInsights TitleInsights

	// Watch time analytics (only populated when HasAnalytics is true)
	HasAnalytics       bool
	WatchTime          WatchTimeAnalysis
	ShortsWatchTime    WatchTimeAnalysis
	LongFormWatchTime  WatchTimeAnalysis
	WatchTimeOverTime  []WatchTimeBucket
	TopByWatchTime     []Video
	TopByRetention     []Video
}

// WatchTimeAnalysis holds aggregated watch time metrics.
type WatchTimeAnalysis struct {
	TotalMinutes       float64
	AvgMinutesPerVideo float64
	AvgViewDuration    float64 // seconds
	AvgRetention       float64 // 0-100
	SubscribersGained  int64
	SubscribersLost    int64
	NetSubscribers     int64
}

// WatchTimeBucket groups watch time data by month.
type WatchTimeBucket struct {
	Period             string
	TotalMinutes       float64
	AvgMinutesPerVideo float64
	AvgRetention       float64
	VideoCount         int
}

// BulkPostingDay records a date with 3+ shorts uploaded.
type BulkPostingDay struct {
	Date       string
	ShortCount int
}

// PeriodPerformance holds aggregated metrics for a time period.
type PeriodPerformance struct {
	VideoCount    int
	AvgViews      float64
	AvgLikes      float64
	AvgComments   float64
	AvgEngagement float64 // (likes+comments)/views
}

// TimeBucket groups video performance by a time period (week/month).
type TimeBucket struct {
	Period     string
	VideoCount int
	AvgViews   float64
	TotalViews int64
	AvgEng     float64
}

// TitleInsights compares title characteristics of top vs bottom performers.
type TitleInsights struct {
	AvgLenTop       float64
	AvgLenBottom    float64
	QuestionsTop    int
	QuestionsBottom int
	NumbersTop      int
	NumbersBottom   int
	CapsWordsTop    float64
	CapsWordsBottom float64
}
