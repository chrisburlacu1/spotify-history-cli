package db

import "time"

// RawSpotifyStream matches Spotify's standard export JSON format
type RawSpotifyStream struct {
	EndTime    string `json:"endTime"`
	ArtistName string `json:"artistName"`
	TrackName  string `json:"trackName"`
	MsPlayed   int64  `json:"msPlayed"`
}

// RawExtendedStream matches Spotify's Extended Streaming History format (history.json)
type RawExtendedStream struct {
	Ts                             string `json:"ts"`
	MsPlayed                       int64  `json:"ms_played"`
	MasterMetadataTrackName        *string `json:"master_metadata_track_name"`
	MasterMetadataAlbumArtistName  *string `json:"master_metadata_album_artist_name"`
	MasterMetadataAlbumAlbumName   *string `json:"master_metadata_album_album_name"`
	SpotifyTrackUri                *string `json:"spotify_track_uri"`
	EpisodeName                    *string `json:"episode_name"`
	Skipped                        *bool  `json:"skipped"`
}

// IngestStats summary produced by the ingestion process
type IngestStats struct {
	TotalFiles       int     `json:"totalFiles"`
	TotalStreams     int     `json:"totalStreams"`
	DuplicateStreams int     `json:"duplicateStreams"`
	TotalArtists     int     `json:"totalArtists"`
	TotalAlbums      int     `json:"totalAlbums"`
	TotalTracks      int     `json:"totalTracks"`
	TracksWithAlbums int     `json:"tracksWithAlbums"`
	SkippedStreams   int     `json:"skippedStreams"`
	SkipRatePct      float64 `json:"skipRatePct"`
	TotalHours       float64 `json:"totalHours"`
	TotalDays        float64 `json:"totalDays"`
	StartDate        string  `json:"startDate"`
	EndDate          string  `json:"endDate"`
	ElapsedDuration  string  `json:"elapsedDuration"`
	DatabaseSizeMB   float64 `json:"databaseSizeMb"`
}

// ArtistSummary represents aggregated stats for an artist
type ArtistSummary struct {
	Rank          int     `json:"rank,omitempty"`
	ArtistID      int64   `json:"artistId"`
	ArtistName    string  `json:"artistName"`
	TotalPlays    int     `json:"totalPlays"`
	FullPlays     int     `json:"fullPlays"`
	Skips         int     `json:"skips"`
	SkipRatePct   float64 `json:"skipRatePct"`
	UniqueTracks  int     `json:"uniqueTracks"`
	UniqueAlbums  int     `json:"uniqueAlbums"`
	TotalMinutes  float64 `json:"totalMinutes"`
	TotalHours    float64 `json:"totalHours"`
	FirstListened string  `json:"firstListened"`
	LastListened  string  `json:"lastListened"`
}

// AlbumSummary represents aggregated stats for an album
type AlbumSummary struct {
	Rank          int     `json:"rank,omitempty"`
	AlbumID       int64   `json:"albumId"`
	AlbumTitle    string  `json:"albumTitle"`
	ArtistName    string  `json:"artistName"`
	AlbumType     string  `json:"albumType,omitempty"`
	ReleaseDate   string  `json:"releaseDate,omitempty"`
	ReleaseYear   int     `json:"releaseYear,omitempty"`
	CoverURL      string  `json:"coverUrl,omitempty"`
	TotalPlays    int     `json:"totalPlays"`
	FullPlays     int     `json:"fullPlays"`
	Skips         int     `json:"skips"`
	SkipRatePct   float64 `json:"skipRatePct"`
	TracksPlayed  int     `json:"tracksPlayed"`
	TotalMinutes  float64 `json:"totalMinutes"`
	TotalHours    float64 `json:"totalHours"`
	FirstListened string  `json:"firstListened"`
	LastListened  string  `json:"lastListened"`
}

// TrackSummary represents aggregated stats for a track
type TrackSummary struct {
	Rank         int     `json:"rank,omitempty"`
	TrackID      int64   `json:"trackId"`
	ArtistName   string  `json:"artistName"`
	TrackName    string  `json:"trackName"`
	AlbumName    string  `json:"albumName,omitempty"`
	TrackNumber  int     `json:"trackNumber,omitempty"`
	PlayCount    int     `json:"playCount"`
	FullPlays    int     `json:"fullPlays"`
	Skips        int     `json:"skips"`
	SkipRatePct  float64 `json:"skipRatePct"`
	TotalMinutes float64 `json:"totalMinutes"`
	TotalHours   float64 `json:"totalHours"`
	FirstPlayed  string  `json:"firstPlayed"`
	LastPlayed   string  `json:"lastPlayed"`
}

// AlbumTrackItem represents a track within a specific album detail view
type AlbumTrackItem struct {
	TrackNumber    int     `json:"trackNumber"`
	TrackName      string  `json:"trackName"`
	PlayCount      int     `json:"playCount"`
	FullPlays      int     `json:"fullPlays"`
	Skips          int     `json:"skips"`
	CompletionRate float64 `json:"completionRate"`
	TotalMinutes   float64 `json:"totalMinutes"`
	TotalHours     float64 `json:"totalHours"`
}

// AlbumDetailProfile represents a full breakdown of a single album
type AlbumDetailProfile struct {
	AlbumTitle     string           `json:"albumTitle"`
	ArtistName     string           `json:"artistName"`
	AlbumType      string           `json:"albumType"`
	ReleaseDate    string           `json:"releaseDate"`
	CoverURL       string           `json:"coverUrl"`
	TotalPlays     int              `json:"totalPlays"`
	TotalHours     float64          `json:"totalHours"`
	TracksPlayed   int              `json:"tracksPlayed"`
	FirstListened  string           `json:"firstListened"`
	LastListened   string           `json:"lastListened"`
	Tracks         []AlbumTrackItem `json:"tracks"`
}

// AlbumPlaythrough represents a detected front-to-back sequential album listen
type AlbumPlaythrough struct {
	AlbumTitle      string  `json:"albumTitle"`
	ArtistName      string  `json:"artistName"`
	StartTime       string  `json:"startTime"`
	EndTime         string  `json:"endTime"`
	TracksPlayed    int     `json:"tracksPlayed"`
	StartTrackNum   int     `json:"startTrackNum"`
	EndTrackNum     int     `json:"endTrackNum"`
	IsFullAlbum     bool    `json:"isFullAlbum"`
	DurationMinutes float64 `json:"durationMinutes"`
}

// DecadeStats represents listening metrics grouped by release decade
type DecadeStats struct {
	Decade       string  `json:"decade"`
	TotalStreams int     `json:"totalStreams"`
	TotalHours   float64 `json:"totalHours"`
	UniqueTracks int     `json:"uniqueTracks"`
	TopArtist    string  `json:"topArtist"`
	TopAlbum     string  `json:"topAlbum"`
	Percentage   float64 `json:"percentage"`
}

// MonthlyStats represents monthly listening volume
type MonthlyStats struct {
	Year               int     `json:"year"`
	Month              int     `json:"month"`
	YearMonth          string  `json:"yearMonth"`
	TotalStreams       int     `json:"totalStreams"`
	UniqueArtists      int     `json:"uniqueArtists"`
	UniqueAlbums       int     `json:"uniqueAlbums"`
	UniqueTracks       int     `json:"uniqueTracks"`
	TotalHours         float64 `json:"totalHours"`
	AvgDurationMinutes float64 `json:"avgDurationMinutes"`
}

// HourlyHabit represents listening habits by hour of day (0-23)
type HourlyHabit struct {
	Hour        int     `json:"hour"`
	StreamCount int     `json:"streamCount"`
	TotalHours  float64 `json:"totalHours"`
	Percentage  float64 `json:"percentage"`
}

// DayHabit represents listening habits by day of week
type DayHabit struct {
	DayOfWeek   int     `json:"dayOfWeek"`
	DayName     string  `json:"dayName"`
	StreamCount int     `json:"streamCount"`
	TotalHours  float64 `json:"totalHours"`
	Percentage  float64 `json:"percentage"`
}

// ObsessionTrack represents a single-day repeat binge
type ObsessionTrack struct {
	Date         string  `json:"date"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName,omitempty"`
	PlaysInDay   int     `json:"playsInDay"`
	TotalMinutes float64 `json:"totalMinutes"`
}

// ListeningSession represents a continuous block of listening
type ListeningSession struct {
	StartTime       string  `json:"startTime"`
	EndTime         string  `json:"endTime"`
	DurationMinutes float64 `json:"durationMinutes"`
	TrackCount      int     `json:"trackCount"`
	TopArtist       string  `json:"topArtist"`
	TopAlbum        string  `json:"topAlbum,omitempty"`
}

// DeepArtistProfile gives complete breakdown of a single artist
type DeepArtistProfile struct {
	ArtistName       string           `json:"artistName"`
	TotalPlays       int              `json:"totalPlays"`
	FullPlays        int              `json:"fullPlays"`
	Skips            int              `json:"skips"`
	SkipRatePct      float64          `json:"skipRatePct"`
	TotalMinutes     float64          `json:"totalMinutes"`
	TotalHours       float64          `json:"totalHours"`
	UniqueAlbums     int              `json:"uniqueAlbums"`
	UniqueTracks     int              `json:"uniqueTracks"`
	FirstListened    string           `json:"firstListened"`
	LastListened     string           `json:"lastListened"`
	PeakListeningDay string           `json:"peakListeningDay"`
	PeakHour         int              `json:"peakHour"`
	TopAlbums        []AlbumSummary   `json:"topAlbums"`
	TopTracks        []TrackSummary   `json:"topTracks"`
	MonthlyBreakdown []MonthlyStats   `json:"monthlyBreakdown"`
}

// StreamRecord represents a full stream row for searches and exports
type StreamRecord struct {
	ID            int64     `json:"id"`
	PlayedAt      time.Time `json:"playedAt"`
	PlayedAtStr   string    `json:"playedAtStr"`
	Date          string    `json:"date"`
	Year          int       `json:"year"`
	Month         int       `json:"month"`
	DayOfWeek     int       `json:"dayOfWeek"`
	Hour          int       `json:"hour"`
	ArtistName    string    `json:"artistName"`
	TrackName     string    `json:"trackName"`
	AlbumName     string    `json:"albumName,omitempty"`
	TrackNumber   int       `json:"trackNumber,omitempty"`
	MsPlayed      int64     `json:"msPlayed"`
	SecondsPlayed float64   `json:"secondsPlayed"`
	MinutesPlayed float64   `json:"minutesPlayed"`
	HoursPlayed   float64   `json:"hoursPlayed"`
	IsSkipped     int       `json:"isSkipped"`
}

// WrappedReport contains the full Spotify Wrapped recap data
type WrappedReport struct {
	Year               string            `json:"year"` // "2026", "2025", "all"
	TotalHours         float64           `json:"totalHours"`
	TotalMinutes       float64           `json:"totalMinutes"`
	TotalStreams       int               `json:"totalStreams"`
	UniqueArtists      int               `json:"uniqueArtists"`
	UniqueAlbums       int               `json:"uniqueAlbums"`
	UniqueTracks       int               `json:"uniqueTracks"`
	SkipRatePct        float64           `json:"skipRatePct"`
	ArchetypeTitle     string            `json:"archetypeTitle"`
	ArchetypeDesc      string            `json:"archetypeDescription"`
	TopArtists         []ArtistSummary   `json:"topArtists"`
	TopAlbums          []AlbumSummary    `json:"topAlbums"`
	TopTracks          []TrackSummary    `json:"topTracks"`
	TopMonth           string            `json:"topMonth"`
	TopMonthHours      float64           `json:"topMonthHours"`
	PeakDay            string            `json:"peakDay"`
	PeakDayHours       float64           `json:"peakDayHours"`
	PeakDayStreams     int               `json:"peakDayStreams"`
	PeakListeningTime  string            `json:"peakListeningTime"`
	TopObsession       *ObsessionTrack   `json:"topObsession,omitempty"`
	TopPlaythrough     *AlbumPlaythrough `json:"topPlaythrough,omitempty"`
}
