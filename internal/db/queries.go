package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type OverviewStats struct {
	YearFilter         string  `json:"yearFilter"`
	TotalStreams       int     `json:"totalStreams"`
	TotalHours         float64 `json:"totalHours"`
	TotalMinutes       float64 `json:"totalMinutes"`
	TotalDays          float64 `json:"totalDays"`
	UniqueArtists      int     `json:"uniqueArtists"`
	UniqueAlbums       int     `json:"uniqueAlbums"`
	UniqueTracks       int     `json:"uniqueTracks"`
	TracksWithAlbums   int     `json:"tracksWithAlbums"`
	SkippedStreams     int     `json:"skippedStreams"`
	SkipRatePct        float64 `json:"skipRatePct"`
	ActiveDays         int     `json:"activeDays"`
	AvgStreamsPerDay   float64 `json:"avgStreamsPerDay"`
	AvgHoursPerDay     float64 `json:"avgHoursPerDay"`
	PeakDayDate        string  `json:"peakDayDate"`
	PeakDayStreams     int     `json:"peakDayStreams"`
	PeakDayHours       float64 `json:"peakDayHours"`
	TopArtist          string  `json:"topArtist"`
	TopArtistHours     float64 `json:"topArtistHours"`
	TopAlbum           string  `json:"topAlbum"`
	TopAlbumArtist     string  `json:"topAlbumArtist"`
	TopAlbumHours      float64 `json:"topAlbumHours"`
	TopTrack           string  `json:"topTrack"`
	TopTrackArtist     string  `json:"topTrackArtist"`
	TopTrackPlays      int     `json:"topTrackPlays"`
	FirstStreamDate    string  `json:"firstStreamDate"`
	LastStreamDate     string  `json:"lastStreamDate"`
	Top3ArtistsPct     float64 `json:"top3ArtistsPct"`
}

func buildYearWhere(yearFilter string, prefix string) string {
	if yearFilter == "" || strings.ToLower(yearFilter) == "all" {
		return ""
	}
	if prefix != "" && !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}
	return fmt.Sprintf("WHERE %syear = %s", prefix, yearFilter)
}

func (d *Database) GetOverview(yearFilter string) (*OverviewStats, error) {
	stats := &OverviewStats{YearFilter: yearFilter}
	whereClause := buildYearWhere(yearFilter, "s")

	queryCore := fmt.Sprintf(`
		SELECT 
			COUNT(*),
			COALESCE(ROUND(SUM(s.hours_played), 2), 0),
			COALESCE(ROUND(SUM(s.minutes_played), 1), 0),
			COUNT(DISTINCT t.artist_id),
			COUNT(DISTINCT t.album_id),
			COUNT(DISTINCT t.id),
			SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END),
			COUNT(DISTINCT s.date),
			COALESCE(MIN(s.date), ''),
			COALESCE(MAX(s.date), '')
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		%s
	`, whereClause)

	err := d.conn.QueryRow(queryCore).Scan(
		&stats.TotalStreams,
		&stats.TotalHours,
		&stats.TotalMinutes,
		&stats.UniqueArtists,
		&stats.UniqueAlbums,
		&stats.UniqueTracks,
		&stats.SkippedStreams,
		&stats.ActiveDays,
		&stats.FirstStreamDate,
		&stats.LastStreamDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query core overview: %w", err)
	}

	if stats.TotalStreams > 0 {
		stats.SkipRatePct = float64(stats.SkippedStreams) * 100.0 / float64(stats.TotalStreams)
		stats.TotalDays = stats.TotalHours / 24.0
	}
	if stats.ActiveDays > 0 {
		stats.AvgStreamsPerDay = float64(stats.TotalStreams) / float64(stats.ActiveDays)
		stats.AvgHoursPerDay = stats.TotalHours / float64(stats.ActiveDays)
	}

	// Peak Day
	_ = d.conn.QueryRow(fmt.Sprintf(`
		SELECT s.date, COUNT(*), ROUND(SUM(s.hours_played), 2)
		FROM streams s
		%s
		GROUP BY s.date
		ORDER BY SUM(s.hours_played) DESC
		LIMIT 1
	`, whereClause)).Scan(&stats.PeakDayDate, &stats.PeakDayStreams, &stats.PeakDayHours)

	// Top Artist
	_ = d.conn.QueryRow(fmt.Sprintf(`
		SELECT a.name, ROUND(SUM(s.hours_played), 2)
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		%s
		GROUP BY a.id, a.name
		ORDER BY SUM(s.hours_played) DESC
		LIMIT 1
	`, whereClause)).Scan(&stats.TopArtist, &stats.TopArtistHours)

	// Top Album
	_ = d.conn.QueryRow(fmt.Sprintf(`
		SELECT al.title, a.name, ROUND(SUM(s.hours_played), 2)
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		JOIN albums al ON t.album_id = al.id
		%s
		GROUP BY al.id, al.title, a.name
		ORDER BY SUM(s.hours_played) DESC
		LIMIT 1
	`, whereClause)).Scan(&stats.TopAlbum, &stats.TopAlbumArtist, &stats.TopAlbumHours)

	// Top Track (excluding Unknown)
	andUnknown := ""
	if whereClause == "" {
		andUnknown = "WHERE a.name != 'Unknown Artist' AND t.title != 'Unknown Track'"
	} else {
		andUnknown = "AND a.name != 'Unknown Artist' AND t.title != 'Unknown Track'"
	}
	_ = d.conn.QueryRow(fmt.Sprintf(`
		SELECT a.name, t.title, COUNT(*)
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		%s %s
		GROUP BY t.id, a.name, t.title
		ORDER BY SUM(s.hours_played) DESC
		LIMIT 1
	`, whereClause, andUnknown)).Scan(&stats.TopTrackArtist, &stats.TopTrack, &stats.TopTrackPlays)

	// Top 3 concentration
	var top3Hours float64
	_ = d.conn.QueryRow(fmt.Sprintf(`
		SELECT COALESCE(SUM(hrs), 0) FROM (
			SELECT SUM(s.hours_played) as hrs
			FROM streams s
			JOIN tracks t ON s.track_id = t.id
			%s
			GROUP BY t.artist_id
			ORDER BY hrs DESC
			LIMIT 3
		)
	`, whereClause)).Scan(&top3Hours)
	if stats.TotalHours > 0 {
		stats.Top3ArtistsPct = (top3Hours / stats.TotalHours) * 100.0
	}

	return stats, nil
}

func (d *Database) GetTopArtists(limit int, yearFilter string) ([]ArtistSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	whereClause := buildYearWhere(yearFilter, "s")

	query := fmt.Sprintf(`
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
		%s
		GROUP BY a.id, a.name
		ORDER BY total_minutes DESC
		LIMIT ?
	`, whereClause)

	rows, err := d.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top artists: %w", err)
	}
	defer rows.Close()

	var list []ArtistSummary
	rank := 1
	for rows.Next() {
		var a ArtistSummary
		a.Rank = rank
		err := rows.Scan(
			&a.ArtistID, &a.ArtistName, &a.TotalPlays, &a.FullPlays, &a.Skips, &a.SkipRatePct,
			&a.UniqueTracks, &a.UniqueAlbums, &a.TotalMinutes, &a.TotalHours, &a.FirstListened, &a.LastListened,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
		rank++
	}
	return list, nil
}

func (d *Database) GetTopAlbums(limit int, yearFilter string) ([]AlbumSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	whereClause := buildYearWhere(yearFilter, "s")

	query := fmt.Sprintf(`
		SELECT 
			al.id AS album_id,
			al.title AS album_title,
			a.name AS artist_name,
			COALESCE(al.album_type, 'album'),
			COALESCE(al.release_date, ''),
			COALESCE(al.release_year, 0),
			COALESCE(al.cover_url, ''),
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
		%s
		GROUP BY al.id, al.title, a.name
		ORDER BY total_minutes DESC
		LIMIT ?
	`, whereClause)

	rows, err := d.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top albums: %w", err)
	}
	defer rows.Close()

	var list []AlbumSummary
	rank := 1
	for rows.Next() {
		var al AlbumSummary
		al.Rank = rank
		err := rows.Scan(
			&al.AlbumID, &al.AlbumTitle, &al.ArtistName, &al.AlbumType, &al.ReleaseDate, &al.ReleaseYear, &al.CoverURL,
			&al.TotalPlays, &al.FullPlays, &al.Skips, &al.SkipRatePct, &al.TracksPlayed, &al.TotalMinutes, &al.TotalHours,
			&al.FirstListened, &al.LastListened,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, al)
		rank++
	}
	return list, nil
}

func (d *Database) GetTopTracks(limit int, yearFilter string) ([]TrackSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	whereClause := buildYearWhere(yearFilter, "s")

	query := fmt.Sprintf(`
		SELECT 
			t.id AS track_id,
			a.name AS artist_name,
			t.title AS track_name,
			COALESCE(al.title, '') AS album_name,
			COALESCE(t.track_number, 0) AS track_number,
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
		%s
		GROUP BY t.id, a.name, t.title
		ORDER BY total_minutes DESC
		LIMIT ?
	`, whereClause)

	rows, err := d.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top tracks: %w", err)
	}
	defer rows.Close()

	var list []TrackSummary
	rank := 1
	for rows.Next() {
		var t TrackSummary
		t.Rank = rank
		err := rows.Scan(
			&t.TrackID, &t.ArtistName, &t.TrackName, &t.AlbumName, &t.TrackNumber,
			&t.PlayCount, &t.FullPlays, &t.Skips, &t.SkipRatePct,
			&t.TotalMinutes, &t.TotalHours, &t.FirstPlayed, &t.LastPlayed,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
		rank++
	}
	return list, nil
}

func (d *Database) GetAlbumDetails(albumName string) (*AlbumDetailProfile, error) {
	profile := &AlbumDetailProfile{}

	// Query Album header
	row := d.conn.QueryRow(`
		SELECT 
			al.id, al.title, a.name, COALESCE(al.album_type, 'album'), COALESCE(al.release_date, ''),
			COALESCE(al.cover_url, ''), COUNT(s.id), COALESCE(ROUND(SUM(s.hours_played), 2), 0),
			COUNT(DISTINCT t.id), COALESCE(MIN(s.date), ''), COALESCE(MAX(s.date), '')
		FROM albums al
		JOIN artists a ON al.artist_id = a.id
		LEFT JOIN tracks t ON t.album_id = al.id
		LEFT JOIN streams s ON s.track_id = t.id
		WHERE al.title LIKE ?
		GROUP BY al.id
		LIMIT 1
	`, "%"+albumName+"%")

	var albumID int64
	err := row.Scan(
		&albumID, &profile.AlbumTitle, &profile.ArtistName, &profile.AlbumType, &profile.ReleaseDate,
		&profile.CoverURL, &profile.TotalPlays, &profile.TotalHours, &profile.TracksPlayed,
		&profile.FirstListened, &profile.LastListened,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no album found matching '%s'", albumName)
	} else if err != nil {
		return nil, err
	}

	// Query tracklist in order with play counts
	trackRows, err := d.conn.Query(`
		SELECT 
			COALESCE(t.track_number, 0) AS track_num,
			t.title AS track_name,
			COUNT(s.id) AS play_count,
			SUM(CASE WHEN s.is_skipped = 0 THEN 1 ELSE 0 END) AS full_plays,
			SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) AS skips,
			COALESCE(ROUND(SUM(s.minutes_played), 1), 0) AS total_minutes,
			COALESCE(ROUND(SUM(s.hours_played), 2), 0) AS total_hours
		FROM tracks t
		LEFT JOIN streams s ON s.track_id = t.id
		WHERE t.album_id = ?
		GROUP BY t.id, t.title
		ORDER BY CASE WHEN t.track_number > 0 THEN t.track_number ELSE 999 END, play_count DESC
	`, albumID)
	if err == nil {
		defer trackRows.Close()
		for trackRows.Next() {
			var ti AlbumTrackItem
			_ = trackRows.Scan(
				&ti.TrackNumber, &ti.TrackName, &ti.PlayCount, &ti.FullPlays, &ti.Skips,
				&ti.TotalMinutes, &ti.TotalHours,
			)
			if ti.PlayCount > 0 {
				ti.CompletionRate = float64(ti.FullPlays) * 100.0 / float64(ti.PlayCount)
			}
			profile.Tracks = append(profile.Tracks, ti)
		}
	}

	return profile, nil
}

func (d *Database) GetDecadeStats(yearFilter string) ([]DecadeStats, error) {
	whereClause := buildYearWhere(yearFilter, "s")
	query := fmt.Sprintf(`
		SELECT 
			CASE 
				WHEN al.release_year >= 2020 THEN '2020s'
				WHEN al.release_year >= 2010 THEN '2010s'
				WHEN al.release_year >= 2000 THEN '2000s'
				WHEN al.release_year >= 1990 THEN '1990s'
				WHEN al.release_year >= 1980 THEN '1980s'
				WHEN al.release_year >= 1970 THEN '1970s'
				WHEN al.release_year IS NOT NULL AND al.release_year > 0 THEN 'Pre-1970s'
				ELSE 'Unknown Era'
			END AS decade,
			COUNT(*) AS total_streams,
			ROUND(SUM(s.hours_played), 2) AS total_hours,
			COUNT(DISTINCT t.id) AS unique_tracks,
			ROUND(100.0 * COUNT(*) / (SELECT MAX(1, COUNT(*)) FROM streams s %s), 2) AS percentage
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		LEFT JOIN albums al ON t.album_id = al.id
		%s
		GROUP BY 1
		ORDER BY total_streams DESC
	`, whereClause, whereClause)

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query decade stats: %w", err)
	}
	defer rows.Close()

	var list []DecadeStats
	for rows.Next() {
		var ds DecadeStats
		err := rows.Scan(&ds.Decade, &ds.TotalStreams, &ds.TotalHours, &ds.UniqueTracks, &ds.Percentage)
		if err != nil {
			return nil, err
		}
		list = append(list, ds)
	}
	return list, nil
}

func (d *Database) GetMonthlyStats(yearFilter string) ([]MonthlyStats, error) {
	whereClause := buildYearWhere(yearFilter, "s")
	query := fmt.Sprintf(`
		SELECT 
			s.year,
			s.month,
			PRINTF('%%04d-%%02d', s.year, s.month) AS year_month,
			COUNT(*) AS total_streams,
			COUNT(DISTINCT t.artist_id) AS unique_artists,
			COUNT(DISTINCT t.album_id) AS unique_albums,
			COUNT(DISTINCT t.id) AS unique_tracks,
			ROUND(SUM(s.hours_played), 2) AS total_hours,
			ROUND(AVG(s.minutes_played), 2) AS avg_track_duration_mins
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		%s
		GROUP BY s.year, s.month
		ORDER BY s.year, s.month
	`, whereClause)

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly stats: %w", err)
	}
	defer rows.Close()

	var list []MonthlyStats
	for rows.Next() {
		var m MonthlyStats
		err := rows.Scan(
			&m.Year, &m.Month, &m.YearMonth, &m.TotalStreams, &m.UniqueArtists,
			&m.UniqueAlbums, &m.UniqueTracks, &m.TotalHours, &m.AvgDurationMinutes,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func (d *Database) GetHourlyHabits(yearFilter string) ([]HourlyHabit, error) {
	whereClause := buildYearWhere(yearFilter, "s")
	query := fmt.Sprintf(`
		SELECT 
			hour,
			COUNT(*) AS stream_count,
			ROUND(SUM(hours_played), 2) AS total_hours,
			ROUND(100.0 * COUNT(*) / (SELECT MAX(1, COUNT(*)) FROM streams s %s), 2) AS pct_of_all_streams
		FROM streams s
		%s
		GROUP BY hour
		ORDER BY hour
	`, whereClause, whereClause)

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly patterns: %w", err)
	}
	defer rows.Close()

	var list []HourlyHabit
	for rows.Next() {
		var h HourlyHabit
		err := rows.Scan(&h.Hour, &h.StreamCount, &h.TotalHours, &h.Percentage)
		if err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, nil
}

func (d *Database) GetDayHabits(yearFilter string) ([]DayHabit, error) {
	whereClause := buildYearWhere(yearFilter, "s")
	query := fmt.Sprintf(`
		SELECT 
			day_of_week,
			day_name,
			COUNT(*) as stream_count,
			ROUND(SUM(hours_played), 2) as total_hours,
			ROUND(100.0 * COUNT(*) / (SELECT MAX(1, COUNT(*)) FROM streams s %s), 2) as pct
		FROM streams s
		%s
		GROUP BY day_of_week, day_name
		ORDER BY day_of_week
	`, whereClause, whereClause)

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query day habits: %w", err)
	}
	defer rows.Close()

	var list []DayHabit
	for rows.Next() {
		var h DayHabit
		err := rows.Scan(&h.DayOfWeek, &h.DayName, &h.StreamCount, &h.TotalHours, &h.Percentage)
		if err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, nil
}

func (d *Database) GetObsessionTracks(minPlays int, limit int, yearFilter string) ([]ObsessionTrack, error) {
	if minPlays <= 0 {
		minPlays = 5
	}
	if limit <= 0 {
		limit = 15
	}
	whereClause := buildYearWhere(yearFilter, "s")
	whereSkipped := "WHERE s.is_skipped = 0"
	if whereClause != "" {
		whereSkipped = whereClause + " AND s.is_skipped = 0"
	}

	query := fmt.Sprintf(`
		SELECT 
			s.date, t.title AS track_name, a.name AS artist_name, COALESCE(al.title, '') AS album_name,
			COUNT(*) AS plays_in_day, ROUND(SUM(s.minutes_played), 1) AS total_mins
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		LEFT JOIN albums al ON t.album_id = al.id
		%s
		GROUP BY s.date, t.id, a.name, t.title
		HAVING COUNT(*) >= ?
		ORDER BY plays_in_day DESC, total_mins DESC
		LIMIT ?
	`, whereSkipped)

	rows, err := d.conn.Query(query, minPlays, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query obsession tracks: %w", err)
	}
	defer rows.Close()

	var list []ObsessionTrack
	for rows.Next() {
		var o ObsessionTrack
		err := rows.Scan(&o.Date, &o.TrackName, &o.ArtistName, &o.AlbumName, &o.PlaysInDay, &o.TotalMinutes)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, nil
}

func (d *Database) GetDeepArtistProfile(artistName string, yearFilter string) (*DeepArtistProfile, error) {
	profile := &DeepArtistProfile{}
	whereClause := buildYearWhere(yearFilter, "s")
	whereArtist := "WHERE a.name LIKE ?"
	if whereClause != "" {
		whereArtist = whereClause + " AND a.name LIKE ?"
	}

	// Core stats
	row := d.conn.QueryRow(fmt.Sprintf(`
		SELECT 
			a.name,
			COUNT(*) AS total_plays,
			SUM(CASE WHEN s.is_skipped = 0 THEN 1 ELSE 0 END) AS full_plays,
			SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) AS skips,
			ROUND(100.0 * SUM(CASE WHEN s.is_skipped = 1 THEN 1 ELSE 0 END) / COUNT(*), 1) AS skip_rate_pct,
			COUNT(DISTINCT t.album_id) AS unique_albums,
			COUNT(DISTINCT t.id) AS unique_tracks,
			ROUND(SUM(s.minutes_played), 1) AS total_minutes,
			ROUND(SUM(s.hours_played), 2) AS total_hours,
			MIN(s.date) AS first_listened,
			MAX(s.date) AS last_listened
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		%s
		GROUP BY a.id, a.name
		LIMIT 1
	`, whereArtist), "%"+artistName+"%")

	err := row.Scan(
		&profile.ArtistName, &profile.TotalPlays, &profile.FullPlays, &profile.Skips,
		&profile.SkipRatePct, &profile.UniqueAlbums, &profile.UniqueTracks, &profile.TotalMinutes,
		&profile.TotalHours, &profile.FirstListened, &profile.LastListened,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no streams found for artist matching '%s'", artistName)
	} else if err != nil {
		return nil, fmt.Errorf("failed to query artist: %w", err)
	}

	// Peak Day
	_ = d.conn.QueryRow(`
		SELECT s.date FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		WHERE a.name = ?
		GROUP BY s.date
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`, profile.ArtistName).Scan(&profile.PeakListeningDay)

	// Peak Hour
	_ = d.conn.QueryRow(`
		SELECT s.hour FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		WHERE a.name = ?
		GROUP BY s.hour
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`, profile.ArtistName).Scan(&profile.PeakHour)

	// Top Albums by this artist
	albumRows, err := d.conn.Query(`
		SELECT 
			al.id, al.title, a.name, COALESCE(al.album_type, 'album'), COALESCE(al.release_date, ''),
			0, '', COUNT(*) AS total_plays,
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
		WHERE a.name = ?
		GROUP BY al.id, al.title
		ORDER BY total_minutes DESC
		LIMIT 8
	`, profile.ArtistName)
	if err == nil {
		rank := 1
		for albumRows.Next() {
			var al AlbumSummary
			al.Rank = rank
			_ = albumRows.Scan(
				&al.AlbumID, &al.AlbumTitle, &al.ArtistName, &al.AlbumType, &al.ReleaseDate,
				&al.ReleaseYear, &al.CoverURL, &al.TotalPlays, &al.FullPlays, &al.Skips,
				&al.SkipRatePct, &al.TracksPlayed, &al.TotalMinutes, &al.TotalHours,
				&al.FirstListened, &al.LastListened,
			)
			profile.TopAlbums = append(profile.TopAlbums, al)
			rank++
		}
		albumRows.Close()
	}

	// Top Tracks by this artist
	trackRows, err := d.conn.Query(`
		SELECT 
			t.id, a.name, t.title, COALESCE(al.title, ''), COALESCE(t.track_number, 0),
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
		WHERE a.name = ?
		GROUP BY t.id, t.title
		ORDER BY total_minutes DESC
		LIMIT 15
	`, profile.ArtistName)
	if err == nil {
		rank := 1
		for trackRows.Next() {
			var t TrackSummary
			t.Rank = rank
			_ = trackRows.Scan(
				&t.TrackID, &t.ArtistName, &t.TrackName, &t.AlbumName, &t.TrackNumber,
				&t.PlayCount, &t.FullPlays, &t.Skips, &t.SkipRatePct,
				&t.TotalMinutes, &t.TotalHours, &t.FirstPlayed, &t.LastPlayed,
			)
			profile.TopTracks = append(profile.TopTracks, t)
			rank++
		}
		trackRows.Close()
	}

	// Monthly timeline for this artist
	monthRows, err := d.conn.Query(`
		SELECT 
			s.year, s.month, PRINTF('%04d-%02d', s.year, s.month) AS year_month,
			COUNT(*) AS total_streams, 1, COUNT(DISTINCT t.album_id),
			COUNT(DISTINCT t.id) AS unique_tracks,
			ROUND(SUM(s.hours_played), 2) AS total_hours,
			ROUND(AVG(s.minutes_played), 2) AS avg_track_duration_mins
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		WHERE a.name = ?
		GROUP BY s.year, s.month
		ORDER BY s.year, s.month
	`, profile.ArtistName)
	if err == nil {
		for monthRows.Next() {
			var m MonthlyStats
			_ = monthRows.Scan(
				&m.Year, &m.Month, &m.YearMonth, &m.TotalStreams, &m.UniqueArtists,
				&m.UniqueAlbums, &m.UniqueTracks, &m.TotalHours, &m.AvgDurationMinutes,
			)
			profile.MonthlyBreakdown = append(profile.MonthlyBreakdown, m)
		}
		monthRows.Close()
	}

	return profile, nil
}

func (d *Database) SearchFTS(query string, limit int) ([]StreamRecord, error) {
	if limit <= 0 {
		limit = 25
	}
	tokens := strings.Fields(query)
	var cleanTokens []string
	for _, tok := range tokens {
		cleanTokens = append(cleanTokens, fmt.Sprintf(`"%s"*`, strings.ReplaceAll(tok, `"`, "")))
	}
	ftsQuery := strings.Join(cleanTokens, " ")

	rows, err := d.conn.Query(`
		SELECT 
			v.stream_id, v.played_at, v.date, v.year, v.month, v.day_of_week, v.hour,
			v.artist_name, v.track_name, COALESCE(v.album_name, ''), COALESCE(v.track_number, 0),
			v.ms_played, v.seconds_played, v.minutes_played, v.hours_played, v.is_skipped
		FROM v_streams v
		JOIN streams_fts f ON v.stream_id = f.rowid
		WHERE streams_fts MATCH ?
		ORDER BY v.played_at DESC
		LIMIT ?
	`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("FTS query failed: %w", err)
	}
	defer rows.Close()

	var results []StreamRecord
	for rows.Next() {
		var r StreamRecord
		var playedAt time.Time
		err := rows.Scan(
			&r.ID, &playedAt, &r.Date, &r.Year, &r.Month, &r.DayOfWeek, &r.Hour,
			&r.ArtistName, &r.TrackName, &r.AlbumName, &r.TrackNumber,
			&r.MsPlayed, &r.SecondsPlayed, &r.MinutesPlayed,
			&r.HoursPlayed, &r.IsSkipped,
		)
		if err != nil {
			return nil, err
		}
		r.PlayedAt = playedAt
		r.PlayedAtStr = playedAt.Format("2006-01-02 15:04:05")
		results = append(results, r)
	}
	return results, nil
}

func (d *Database) ExecuteRawSQL(query string) ([]string, [][]string, error) {
	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("SQL execution error: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var data [][]string
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, err
		}

		rowStr := make([]string, len(columns))
		for i, v := range values {
			if v == nil {
				rowStr[i] = "NULL"
			} else {
				rowStr[i] = fmt.Sprintf("%v", v)
			}
		}
		data = append(data, rowStr)
	}

	return columns, data, nil
}
