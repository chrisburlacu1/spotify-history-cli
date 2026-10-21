package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	conn *sql.DB
	path string
}

func Open(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-64000)&_pragma=temp_store(MEMORY)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	conn.SetMaxOpenConns(1)

	return &Database{
		conn: conn,
		path: dbPath,
	}, nil
}

func (d *Database) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

func (d *Database) Conn() *sql.DB {
	return d.conn
}

func (d *Database) InitSchema() error {
	schema := `
	PRAGMA foreign_keys = ON;

	DROP VIEW IF EXISTS v_decades;
	DROP VIEW IF EXISTS v_top_albums;
	DROP VIEW IF EXISTS v_top_artists;
	DROP VIEW IF EXISTS v_top_tracks;
	DROP VIEW IF EXISTS v_monthly_stats;
	DROP VIEW IF EXISTS v_hourly_patterns;
	DROP VIEW IF EXISTS v_daily_stats;
	DROP VIEW IF EXISTS v_streams;
	DROP TABLE IF EXISTS streams_fts;
	DROP TABLE IF EXISTS streams;
	DROP TABLE IF EXISTS track_artists;
	DROP TABLE IF EXISTS tracks;
	DROP TABLE IF EXISTS albums;
	DROP TABLE IF EXISTS artists;

	CREATE TABLE artists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		spotify_uri TEXT
	);

	CREATE TABLE albums (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		artist_id INTEGER NOT NULL REFERENCES artists(id),
		title TEXT NOT NULL,
		album_type TEXT,            -- 'album', 'single', 'compilation'
		release_date TEXT,          -- 'YYYY-MM-DD'
		release_year INTEGER,
		cover_url TEXT,
		spotify_uri TEXT,
		UNIQUE(artist_id, title)
	);

	CREATE TABLE tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		artist_id INTEGER NOT NULL REFERENCES artists(id),
		album_id INTEGER REFERENCES albums(id),
		title TEXT NOT NULL,
		track_number INTEGER,       -- 1, 2, 3...
		duration_ms INTEGER,
		spotify_uri TEXT,
		UNIQUE(artist_id, title)
	);

	CREATE TABLE track_artists (
		track_id INTEGER NOT NULL REFERENCES tracks(id),
		artist_id INTEGER NOT NULL REFERENCES artists(id),
		PRIMARY KEY(track_id, artist_id)
	);

	CREATE TABLE streams (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		track_id INTEGER NOT NULL REFERENCES tracks(id),
		played_at DATETIME NOT NULL,
		date TEXT NOT NULL,
		year INTEGER NOT NULL,
		month INTEGER NOT NULL,
		day INTEGER NOT NULL,
		day_of_week INTEGER NOT NULL,  -- 0=Monday, 6=Sunday
		day_name TEXT NOT NULL,
		hour INTEGER NOT NULL,
		ms_played INTEGER NOT NULL,
		seconds_played REAL NOT NULL,
		minutes_played REAL NOT NULL,
		hours_played REAL NOT NULL,
		is_skipped INTEGER NOT NULL,
		UNIQUE(played_at, track_id, ms_played)
	);

	CREATE INDEX idx_streams_played_at ON streams(played_at);
	CREATE INDEX idx_streams_date ON streams(date);
	CREATE INDEX idx_streams_year ON streams(year);
	CREATE INDEX idx_streams_year_month ON streams(year, month);
	CREATE INDEX idx_streams_track_id ON streams(track_id);
	CREATE INDEX idx_streams_hour ON streams(hour);
	CREATE INDEX idx_streams_skipped ON streams(is_skipped);
	CREATE INDEX idx_tracks_album_id ON tracks(album_id);
	CREATE INDEX idx_tracks_artist_id ON tracks(artist_id);
	CREATE INDEX idx_albums_artist_id ON albums(artist_id);
	CREATE INDEX idx_albums_release_year ON albums(release_year);

	CREATE VIRTUAL TABLE streams_fts USING fts5(
		artist_name,
		track_name,
		album_name,
		content='v_streams',
		content_rowid='stream_id'
	);

	-- Master Relational View
	CREATE VIEW v_streams AS
	SELECT 
		s.id AS stream_id,
		s.played_at,
		s.date,
		s.year,
		s.month,
		s.day,
		s.day_of_week,
		s.day_name,
		s.hour,
		a.name AS artist_name,
		a.id AS artist_id,
		t.title AS track_name,
		t.id AS track_id,
		t.track_number,
		t.duration_ms AS track_duration_ms,
		t.spotify_uri AS track_spotify_uri,
		al.title AS album_name,
		al.id AS album_id,
		al.album_type,
		al.release_date AS album_release_date,
		al.release_year AS album_release_year,
		al.cover_url AS album_cover_url,
		s.ms_played,
		s.seconds_played,
		s.minutes_played,
		s.hours_played,
		s.is_skipped
	FROM streams s
	JOIN tracks t ON s.track_id = t.id
	JOIN artists a ON t.artist_id = a.id
	LEFT JOIN albums al ON t.album_id = al.id;

	-- Top Artists View
	CREATE VIEW v_top_artists AS
	SELECT 
		a.id AS artist_id,
		a.name AS artist_name,
		COUNT(*) AS total_plays,
		SUM(CASE WHEN s.is_skipped = 0 THEN 1 ELSE 0 END) AS full_plays,
		SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) AS skips,
		ROUND(100.0 * SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) / COUNT(*), 1) AS skip_rate_pct,
		COUNT(DISTINCT t.id) AS unique_tracks,
		COUNT(DISTINCT t.album_id) AS unique_albums,
		ROUND(SUM(s.minutes_played), 1) AS total_minutes,
		ROUND(SUM(s.hours_played), 2) AS total_hours,
		MIN(s.date) AS first_listened,
		MAX(s.date) AS last_listened
	FROM streams s
	JOIN tracks t ON s.track_id = t.id
	JOIN artists a ON t.artist_id = a.id
	GROUP BY a.id, a.name
	ORDER BY total_minutes DESC;

	-- Top Albums View
	CREATE VIEW v_top_albums AS
	SELECT 
		al.id AS album_id,
		al.title AS album_title,
		a.name AS artist_name,
		al.album_type,
		al.release_date,
		al.release_year,
		al.cover_url,
		COUNT(*) AS total_plays,
		SUM(CASE WHEN s.is_skipped = 0 THEN 1 ELSE 0 END) AS full_plays,
		SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) AS skips,
		ROUND(100.0 * SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) / COUNT(*), 1) AS skip_rate_pct,
		COUNT(DISTINCT t.id) AS tracks_played,
		ROUND(SUM(s.minutes_played), 1) AS total_minutes,
		ROUND(SUM(s.hours_played), 2) AS total_hours,
		MIN(s.date) AS first_listened,
		MAX(s.date) AS last_listened
	FROM streams s
	JOIN tracks t ON s.track_id = t.id
	JOIN artists a ON t.artist_id = a.id
	JOIN albums al ON t.album_id = al.id
	GROUP BY al.id, al.title, a.name
	ORDER BY total_minutes DESC;

	-- Top Tracks View
	CREATE VIEW v_top_tracks AS
	SELECT 
		t.id AS track_id,
		a.name AS artist_name,
		t.title AS track_name,
		al.title AS album_name,
		t.track_number,
		COUNT(*) AS play_count,
		SUM(CASE WHEN s.is_skipped = 0 THEN 1 ELSE 0 END) AS full_plays,
		SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) AS skips,
		ROUND(100.0 * SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) / COUNT(*), 1) AS skip_rate_pct,
		ROUND(SUM(s.minutes_played), 1) AS total_minutes,
		ROUND(SUM(s.hours_played), 2) AS total_hours,
		MIN(s.date) AS first_played,
		MAX(s.date) AS last_played
	FROM streams s
	JOIN tracks t ON s.track_id = t.id
	JOIN artists a ON t.artist_id = a.id
	LEFT JOIN albums al ON t.album_id = al.id
	GROUP BY t.id, a.name, t.title
	ORDER BY total_minutes DESC;

	-- Monthly Stats View
	CREATE VIEW v_monthly_stats AS
	SELECT 
		s.year,
		s.month,
		PRINTF('%04d-%02d', s.year, s.month) AS year_month,
		COUNT(*) AS total_streams,
		COUNT(DISTINCT t.artist_id) AS unique_artists,
		COUNT(DISTINCT t.album_id) AS unique_albums,
		COUNT(DISTINCT t.id) AS unique_tracks,
		ROUND(SUM(s.hours_played), 2) AS total_hours,
		ROUND(AVG(s.minutes_played), 2) AS avg_track_duration_mins
	FROM streams s
	JOIN tracks t ON s.track_id = t.id
	GROUP BY s.year, s.month
	ORDER BY s.year, s.month;

	-- Hourly Patterns View
	CREATE VIEW v_hourly_patterns AS
	SELECT 
		hour,
		COUNT(*) AS stream_count,
		ROUND(SUM(hours_played), 2) AS total_hours,
		ROUND(100.0 * COUNT(*) / (SELECT MAX(1, COUNT(*)) FROM streams), 2) AS pct_of_all_streams
	FROM streams
	GROUP BY hour
	ORDER BY hour;

	-- Daily Stats View
	CREATE VIEW v_daily_stats AS
	SELECT 
		s.date,
		s.day_name,
		COUNT(*) AS total_streams,
		COUNT(DISTINCT t.artist_id) AS unique_artists,
		ROUND(SUM(s.minutes_played), 1) AS total_minutes,
		ROUND(SUM(s.hours_played), 2) AS total_hours
	FROM streams s
	JOIN tracks t ON s.track_id = t.id
	GROUP BY s.date
	ORDER BY s.date DESC;

	-- Release Decades View
	CREATE VIEW v_decades AS
	SELECT 
		CASE 
			WHEN al.release_year >= 2020 THEN '2020s'
			WHEN al.release_year >= 2010 THEN '2010s'
			WHEN al.release_year >= 2000 THEN '2000s'
			WHEN al.release_year >= 1990 THEN '1990s'
			WHEN al.release_year >= 1980 THEN '1980s'
			WHEN al.release_year >= 1970 THEN '1970s'
			WHEN al.release_year IS NOT NULL THEN 'Pre-1970s'
			ELSE 'Unknown Era'
		END AS decade,
		COUNT(*) AS total_streams,
		ROUND(SUM(s.hours_played), 2) AS total_hours,
		COUNT(DISTINCT t.id) AS unique_tracks,
		ROUND(100.0 * COUNT(*) / (SELECT MAX(1, COUNT(*)) FROM streams), 2) AS percentage
	FROM streams s
	JOIN tracks t ON s.track_id = t.id
	LEFT JOIN albums al ON t.album_id = al.id
	GROUP BY 1
	ORDER BY total_streams DESC;
	`

	_, err := d.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}
	return nil
}

func isExtendedFormat(data []byte) bool {
	sample := data
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	s := string(sample)
	return strings.Contains(s, "master_metadata_track_name") || strings.Contains(s, "master_metadata_album_artist_name") || (strings.Contains(s, `"ts"`) && strings.Contains(s, `"ms_played"`))
}

func (d *Database) IngestAllSources(sourcePaths []string) (*IngestStats, error) {
	startTime := time.Now()

	var allFiles []string
	for _, p := range sourcePaths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
					allFiles = append(allFiles, filepath.Join(p, e.Name()))
				}
			}
		} else if strings.HasSuffix(strings.ToLower(p), ".json") {
			allFiles = append(allFiles, p)
		}
	}
	sort.Strings(allFiles)

	if len(allFiles) == 0 {
		return nil, fmt.Errorf("no JSON files found in provided paths")
	}

	if err := d.InitSchema(); err != nil {
		return nil, err
	}

	// Memory caches for high-speed resolution
	artistMap := make(map[string]int64)        // artistName -> artistID
	albumMap := make(map[string]int64)         // "artistID|albumTitle" -> albumID
	trackMap := make(map[string]int64)         // "artistID|trackTitle" -> trackID
	trackAlbumMap := make(map[int64]int64)     // trackID -> albumID

	tx, err := d.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	stmtArtist, err := tx.Prepare("INSERT INTO artists (name, spotify_uri) VALUES (?, ?) RETURNING id;")
	if err != nil {
		return nil, err
	}
	defer stmtArtist.Close()

	stmtAlbum, err := tx.Prepare("INSERT INTO albums (artist_id, title, album_type, release_date, release_year, cover_url, spotify_uri) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id;")
	if err != nil {
		return nil, err
	}
	defer stmtAlbum.Close()

	stmtTrack, err := tx.Prepare("INSERT INTO tracks (artist_id, album_id, title, track_number, duration_ms, spotify_uri) VALUES (?, ?, ?, ?, ?, ?) RETURNING id;")
	if err != nil {
		return nil, err
	}
	defer stmtTrack.Close()

	stmtStream, err := tx.Prepare(`
		INSERT OR IGNORE INTO streams (
			track_id, played_at, date, year, month, day, day_of_week, day_name, hour,
			ms_played, seconds_played, minutes_played, hours_played, is_skipped
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`)
	if err != nil {
		return nil, err
	}
	defer stmtStream.Close()

	getOrCreateArtist := func(name string, uri string) (int64, error) {
		name = strings.TrimSpace(name)
		if name == "" {
			name = "Unknown Artist"
		}
		if id, exists := artistMap[name]; exists {
			return id, nil
		}
		var id int64
		err := stmtArtist.QueryRow(name, uri).Scan(&id)
		if err != nil {
			// In case of conflict
			err = tx.QueryRow("SELECT id FROM artists WHERE name = ?", name).Scan(&id)
			if err != nil {
				return 0, err
			}
		}
		artistMap[name] = id
		return id, nil
	}

	getOrCreateAlbum := func(artistID int64, title string, albumType, relDate string, relYear int, cover, uri string) (int64, error) {
		title = strings.TrimSpace(title)
		if title == "" {
			return 0, nil
		}
		key := fmt.Sprintf("%d|%s", artistID, title)
		if id, exists := albumMap[key]; exists {
			return id, nil
		}
		var id int64
		err := stmtAlbum.QueryRow(artistID, title, albumType, relDate, relYear, cover, uri).Scan(&id)
		if err != nil {
			err = tx.QueryRow("SELECT id FROM albums WHERE artist_id = ? AND title = ?", artistID, title).Scan(&id)
			if err != nil {
				return 0, err
			}
		}
		albumMap[key] = id
		return id, nil
	}

	getOrCreateTrack := func(artistID int64, albumID *int64, title string, trackNum, durMs int, uri string) (int64, error) {
		title = strings.TrimSpace(title)
		if title == "" {
			title = "Unknown Track"
		}
		key := fmt.Sprintf("%d|%s", artistID, title)
		if id, exists := trackMap[key]; exists {
			if albumID != nil && *albumID > 0 {
				if _, hasAlb := trackAlbumMap[id]; !hasAlb {
					// Update track with album if it was missing
					_, _ = tx.Exec("UPDATE tracks SET album_id = ? WHERE id = ? AND album_id IS NULL", *albumID, id)
					trackAlbumMap[id] = *albumID
				}
			}
			return id, nil
		}
		var id int64
		var albVal interface{}
		if albumID != nil && *albumID > 0 {
			albVal = *albumID
		}
		err := stmtTrack.QueryRow(artistID, albVal, title, trackNum, durMs, uri).Scan(&id)
		if err != nil {
			err = tx.QueryRow("SELECT id FROM tracks WHERE artist_id = ? AND title = ?", artistID, title).Scan(&id)
			if err != nil {
				return 0, err
			}
		}
		trackMap[key] = id
		if albumID != nil && *albumID > 0 {
			trackAlbumMap[id] = *albumID
		}
		return id, nil
	}

	totalStreamsParsed := 0
	skippedStreams := 0

	// Classify files into Extended Streaming History (endsong_*.json, Streaming_History_Audio_*.json)
	// and Standard Streaming History (StreamingHistory_music_*.json)
	var extFiles []string
	var stdFiles []string
	for _, filePath := range allFiles {
		header := make([]byte, 4096)
		f, err := os.Open(filePath)
		if err != nil {
			continue
		}
		n, _ := f.Read(header)
		f.Close()
		if isExtendedFormat(header[:n]) {
			extFiles = append(extFiles, filePath)
		} else {
			stdFiles = append(stdFiles, filePath)
		}
	}

	// Process Extended history first to build the master album cache
	for _, filePath := range extFiles {
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var extStreams []RawExtendedStream
		if err := json.Unmarshal(fileBytes, &extStreams); err != nil {
			continue
		}

		for _, s := range extStreams {
			if s.Ts == "" || s.MasterMetadataTrackName == nil || s.MasterMetadataAlbumArtistName == nil {
				continue
			}
			parsedTime, err := time.Parse(time.RFC3339, s.Ts)
			if err != nil {
				continue
			}

			artistName := *s.MasterMetadataAlbumArtistName
			trackName := *s.MasterMetadataTrackName
			albumName := ""
			if s.MasterMetadataAlbumAlbumName != nil {
				albumName = *s.MasterMetadataAlbumAlbumName
			}
			trackURI := ""
			if s.SpotifyTrackUri != nil {
				trackURI = *s.SpotifyTrackUri
			}

			artistID, err := getOrCreateArtist(artistName, "")
			if err != nil {
				continue
			}

			var albumIDPtr *int64
			if albumName != "" {
				albID, err := getOrCreateAlbum(artistID, albumName, "album", "", 0, "", "")
				if err == nil && albID > 0 {
					albumIDPtr = &albID
				}
			}

			trackID, err := getOrCreateTrack(artistID, albumIDPtr, trackName, 0, int(s.MsPlayed), trackURI)
			if err != nil {
				continue
			}

			playedAtIso := parsedTime.Format("2006-01-02 15:04:05")
			dateStr := parsedTime.Format("2006-01-02")
			dayName := parsedTime.Format("Monday")
			dayOfWeek := (int(parsedTime.Weekday()) + 6) % 7
			hour := parsedTime.Hour()
			month := int(parsedTime.Month())
			day := parsedTime.Day()
			year := parsedTime.Year()

			sec := float64(s.MsPlayed) / 1000.0
			min := float64(s.MsPlayed) / 60000.0
			hrs := float64(s.MsPlayed) / 3600000.0

			isSkipped := 0
			if s.Skipped != nil && *s.Skipped {
				isSkipped = 1
				skippedStreams++
			} else if s.MsPlayed < 30000 {
				isSkipped = 1
				skippedStreams++
			}

			_, err = stmtStream.Exec(
				trackID, playedAtIso, dateStr, year, month, day, dayOfWeek, dayName, hour,
				s.MsPlayed, sec, min, hrs, isSkipped,
			)
			if err == nil {
				totalStreamsParsed++
			}
		}
	}

	// Process standard StreamingHistory files
	for _, filePath := range stdFiles {
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var stdStreams []RawSpotifyStream
		if err := json.Unmarshal(fileBytes, &stdStreams); err != nil {
			continue
		}

		for _, s := range stdStreams {
			if s.EndTime == "" || s.ArtistName == "" || s.TrackName == "" {
				continue
			}
			parsedTime, err := time.Parse("2006-01-02 15:04", s.EndTime)
			if err != nil {
				continue
			}

			artistID, err := getOrCreateArtist(s.ArtistName, "")
			if err != nil {
				continue
			}

			trackID, err := getOrCreateTrack(artistID, nil, s.TrackName, 0, int(s.MsPlayed), "")
			if err != nil {
				continue
			}

			playedAtIso := parsedTime.Format("2006-01-02 15:04:05")
			dateStr := parsedTime.Format("2006-01-02")
			dayName := parsedTime.Format("Monday")
			dayOfWeek := (int(parsedTime.Weekday()) + 6) % 7
			hour := parsedTime.Hour()
			month := int(parsedTime.Month())
			day := parsedTime.Day()
			year := parsedTime.Year()

			sec := float64(s.MsPlayed) / 1000.0
			min := float64(s.MsPlayed) / 60000.0
			hrs := float64(s.MsPlayed) / 3600000.0

			isSkipped := 0
			if s.MsPlayed < 30000 {
				isSkipped = 1
				skippedStreams++
			}

			_, err = stmtStream.Exec(
				trackID, playedAtIso, dateStr, year, month, day, dayOfWeek, dayName, hour,
				s.MsPlayed, sec, min, hrs, isSkipped,
			)
			if err == nil {
				totalStreamsParsed++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Build Full-Text Search index
	_, err = d.conn.Exec(`
		INSERT INTO streams_fts (rowid, artist_name, track_name, album_name)
		SELECT stream_id, artist_name, track_name, COALESCE(album_name, '') FROM v_streams;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to build FTS index: %w", err)
	}

	// Calculate stats
	var totalActualStreams, totalArtists, totalAlbums, totalTracks, tracksWithAlbums int
	var minDate, maxDate string
	var totalHours float64

	_ = d.conn.QueryRow("SELECT COUNT(*), COALESCE(MIN(date), ''), COALESCE(MAX(date), ''), COALESCE(ROUND(SUM(hours_played), 2), 0) FROM streams").
		Scan(&totalActualStreams, &minDate, &maxDate, &totalHours)
	_ = d.conn.QueryRow("SELECT COUNT(*) FROM artists").Scan(&totalArtists)
	_ = d.conn.QueryRow("SELECT COUNT(*) FROM albums").Scan(&totalAlbums)
	_ = d.conn.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&totalTracks)
	_ = d.conn.QueryRow("SELECT COUNT(*) FROM tracks WHERE album_id IS NOT NULL").Scan(&tracksWithAlbums)

	duplicates := totalStreamsParsed - totalActualStreams
	if duplicates < 0 {
		duplicates = 0
	}

	fileInfo, _ := os.Stat(d.path)
	dbSizeMB := 0.0
	if fileInfo != nil {
		dbSizeMB = float64(fileInfo.Size()) / (1024 * 1024)
	}

	skipRate := 0.0
	if totalActualStreams > 0 {
		skipRate = float64(skippedStreams) * 100.0 / float64(totalActualStreams)
	}

	elapsed := time.Since(startTime)

	return &IngestStats{
		TotalFiles:       len(allFiles),
		TotalStreams:     totalActualStreams,
		DuplicateStreams: duplicates,
		TotalArtists:     totalArtists,
		TotalAlbums:      totalAlbums,
		TotalTracks:      totalTracks,
		TracksWithAlbums: tracksWithAlbums,
		SkippedStreams:   skippedStreams,
		SkipRatePct:      skipRate,
		TotalHours:       totalHours,
		TotalDays:        totalHours / 24.0,
		StartDate:        minDate,
		EndDate:          maxDate,
		ElapsedDuration:  elapsed.Round(time.Millisecond).String(),
		DatabaseSizeMB:   dbSizeMB,
	}, nil
}
