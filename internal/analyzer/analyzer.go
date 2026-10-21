package analyzer

import (
	"fmt"
	"sort"
	"github.com/chrisburlacu1/spotify-history-cli/internal/db"
	"time"
)

type SessionMetrics struct {
	TotalSessions       int                   `json:"totalSessions"`
	AvgSessionMinutes   float64               `json:"avgSessionMinutes"`
	MaxSessionMinutes   float64               `json:"maxSessionMinutes"`
	LongestSessionDate  string                `json:"longestSessionDate"`
	LongestSessionPlays int                   `json:"longestSessionPlays"`
	Sessions            []db.ListeningSession `json:"topSessions,omitempty"`
}

type PlaythroughSummary struct {
	TotalPlaythroughs int                   `json:"totalPlaythroughs"`
	TopPlaythroughs   []db.AlbumPlaythrough `json:"topPlaythroughs"`
}

type playItem struct {
	playedAt   time.Time
	artist     string
	albumID    int64
	albumTitle string
	track      string
	trackNum   int
	isSkipped  int
	minutes    float64
}

type sessPlay struct {
	playedAt time.Time
	artist   string
	album    string
	track    string
	minutes  float64
}

// AnalyzePlaythroughs detects sequential album playthroughs (e.g. track 1 -> 2 -> 3 -> ...)
func AnalyzePlaythroughs(database *db.Database, minTracks int, yearFilter string) (*PlaythroughSummary, error) {
	if minTracks <= 0 {
		minTracks = 4
	}

	whereClause := ""
	if yearFilter != "" && yearFilter != "all" {
		whereClause = fmt.Sprintf("WHERE s.year = %s", yearFilter)
	}

	query := fmt.Sprintf(`
		SELECT 
			s.played_at, a.name, COALESCE(al.id, 0), COALESCE(al.title, ''),
			t.title, COALESCE(t.track_number, 0), s.is_skipped, s.minutes_played
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		LEFT JOIN albums al ON t.album_id = al.id
		%s
		ORDER BY s.played_at ASC
	`, whereClause)

	rows, err := database.Conn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query streams for playthrough analysis: %w", err)
	}
	defer rows.Close()

	var allPlays []playItem
	for rows.Next() {
		var p playItem
		var playedAt time.Time
		if err := rows.Scan(&playedAt, &p.artist, &p.albumID, &p.albumTitle, &p.track, &p.trackNum, &p.isSkipped, &p.minutes); err != nil {
			continue
		}
		p.playedAt = playedAt
		allPlays = append(allPlays, p)
	}

	if len(allPlays) == 0 {
		return &PlaythroughSummary{}, nil
	}

	var playthroughs []db.AlbumPlaythrough
	var currentSeq []playItem

	for _, play := range allPlays {
		if play.albumID == 0 || play.trackNum <= 0 || play.isSkipped == 1 {
			if len(currentSeq) >= minTracks {
				playthroughs = append(playthroughs, buildPlaythrough(currentSeq))
			}
			currentSeq = nil
			continue
		}

		if len(currentSeq) == 0 {
			currentSeq = append(currentSeq, play)
			continue
		}

		prev := currentSeq[len(currentSeq)-1]
		gapMins := play.playedAt.Sub(prev.playedAt).Minutes()

		// Continuous stream on same album with sequential track number
		if play.albumID == prev.albumID && play.trackNum == prev.trackNum+1 && gapMins <= 15.0 && gapMins >= 0 {
			currentSeq = append(currentSeq, play)
		} else {
			if len(currentSeq) >= minTracks {
				playthroughs = append(playthroughs, buildPlaythrough(currentSeq))
			}
			currentSeq = []playItem{play}
		}
	}

	if len(currentSeq) >= minTracks {
		playthroughs = append(playthroughs, buildPlaythrough(currentSeq))
	}

	// Sort by tracks played descending, then duration
	sort.Slice(playthroughs, func(i, j int) bool {
		if playthroughs[i].TracksPlayed == playthroughs[j].TracksPlayed {
			return playthroughs[i].DurationMinutes > playthroughs[j].DurationMinutes
		}
		return playthroughs[i].TracksPlayed > playthroughs[j].TracksPlayed
	})

	topLimit := 15
	if len(playthroughs) < topLimit {
		topLimit = len(playthroughs)
	}

	return &PlaythroughSummary{
		TotalPlaythroughs: len(playthroughs),
		TopPlaythroughs:   playthroughs[:topLimit],
	}, nil
}

func buildPlaythrough(seq []playItem) db.AlbumPlaythrough {
	first := seq[0]
	last := seq[len(seq)-1]
	totalMins := 0.0
	for _, p := range seq {
		totalMins += p.minutes
	}

	isFull := first.trackNum == 1 && len(seq) >= 8

	return db.AlbumPlaythrough{
		AlbumTitle:      first.albumTitle,
		ArtistName:      first.artist,
		StartTime:       first.playedAt.Format("2006-01-02 15:04"),
		EndTime:         last.playedAt.Format("2006-01-02 15:04"),
		TracksPlayed:    len(seq),
		StartTrackNum:   first.trackNum,
		EndTrackNum:     last.trackNum,
		IsFullAlbum:     isFull,
		DurationMinutes: totalMins,
	}
}

// AnalyzeSessions detects listening sessions where streams have gap < maxGapMinutes
func AnalyzeSessions(database *db.Database, maxGapMinutes float64, yearFilter string) (*SessionMetrics, error) {
	if maxGapMinutes <= 0 {
		maxGapMinutes = 20.0
	}

	whereClause := ""
	if yearFilter != "" && yearFilter != "all" {
		whereClause = fmt.Sprintf("WHERE s.year = %s", yearFilter)
	}

	query := fmt.Sprintf(`
		SELECT s.played_at, a.name, COALESCE(al.title, ''), t.title, s.minutes_played
		FROM streams s
		JOIN tracks t ON s.track_id = t.id
		JOIN artists a ON t.artist_id = a.id
		LEFT JOIN albums al ON t.album_id = al.id
		%s
		ORDER BY s.played_at ASC
	`, whereClause)

	rows, err := database.Conn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch streams for session analysis: %w", err)
	}
	defer rows.Close()

	var allPlays []sessPlay
	for rows.Next() {
		var sp sessPlay
		var playedAt time.Time
		if err := rows.Scan(&playedAt, &sp.artist, &sp.album, &sp.track, &sp.minutes); err != nil {
			continue
		}
		sp.playedAt = playedAt
		allPlays = append(allPlays, sp)
	}

	if len(allPlays) == 0 {
		return &SessionMetrics{}, nil
	}

	var sessions []db.ListeningSession
	var currentSessionPlays []sessPlay

	for i, play := range allPlays {
		if i == 0 {
			currentSessionPlays = append(currentSessionPlays, play)
			continue
		}

		prev := allPlays[i-1]
		diff := play.playedAt.Sub(prev.playedAt).Minutes()

		if diff <= maxGapMinutes && diff >= 0 {
			currentSessionPlays = append(currentSessionPlays, play)
		} else {
			if len(currentSessionPlays) > 0 {
				sessions = append(sessions, buildListeningSession(currentSessionPlays))
			}
			currentSessionPlays = []sessPlay{play}
		}
	}

	if len(currentSessionPlays) > 0 {
		sessions = append(sessions, buildListeningSession(currentSessionPlays))
	}

	totalDuration := 0.0
	maxDuration := 0.0
	longestDate := ""
	longestPlays := 0

	for _, s := range sessions {
		totalDuration += s.DurationMinutes
		if s.DurationMinutes > maxDuration {
			maxDuration = s.DurationMinutes
			longestDate = s.StartTime
			longestPlays = s.TrackCount
		}
	}

	avgDuration := 0.0
	if len(sessions) > 0 {
		avgDuration = totalDuration / float64(len(sessions))
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].DurationMinutes > sessions[j].DurationMinutes
	})

	topLimit := 5
	if len(sessions) < topLimit {
		topLimit = len(sessions)
	}

	return &SessionMetrics{
		TotalSessions:       len(sessions),
		AvgSessionMinutes:   avgDuration,
		MaxSessionMinutes:   maxDuration,
		LongestSessionDate:  longestDate,
		LongestSessionPlays: longestPlays,
		Sessions:            sessions[:topLimit],
	}, nil
}

func buildListeningSession(plays []sessPlay) db.ListeningSession {
	start := plays[0].playedAt
	end := plays[len(plays)-1].playedAt

	totalMins := 0.0
	artistCounts := make(map[string]int)
	albumCounts := make(map[string]int)

	for _, p := range plays {
		totalMins += p.minutes
		artistCounts[p.artist]++
		if p.album != "" {
			albumCounts[p.album]++
		}
	}

	topArtist := ""
	topArtistCount := 0
	for a, c := range artistCounts {
		if c > topArtistCount {
			topArtistCount = c
			topArtist = a
		}
	}

	topAlbum := ""
	topAlbumCount := 0
	for al, c := range albumCounts {
		if c > topAlbumCount {
			topAlbumCount = c
			topAlbum = al
		}
	}

	return db.ListeningSession{
		StartTime:       start.Format("2006-01-02 15:04"),
		EndTime:         end.Format("2006-01-02 15:04"),
		DurationMinutes: totalMins,
		TrackCount:      len(plays),
		TopArtist:       topArtist,
		TopAlbum:        topAlbum,
	}
}

// GenerateWrapped compiles a full wrapped experience with listening archetype
func GenerateWrapped(database *db.Database, yearFilter string) (*db.WrappedReport, error) {
	overview, err := database.GetOverview(yearFilter)
	if err != nil {
		return nil, err
	}

	topArtists, err := database.GetTopArtists(5, yearFilter)
	if err != nil {
		return nil, err
	}

	topAlbums, err := database.GetTopAlbums(5, yearFilter)
	if err != nil {
		return nil, err
	}

	topTracks, err := database.GetTopTracks(5, yearFilter)
	if err != nil {
		return nil, err
	}

	monthlyStats, err := database.GetMonthlyStats(yearFilter)
	if err != nil {
		return nil, err
	}

	hourlyHabits, err := database.GetHourlyHabits(yearFilter)
	if err != nil {
		return nil, err
	}

	obsessions, _ := database.GetObsessionTracks(5, 1, yearFilter)
	playthroughSummary, _ := AnalyzePlaythroughs(database, 4, yearFilter)

	// Top Month
	topMonth := ""
	topMonthHours := 0.0
	for _, m := range monthlyStats {
		if m.TotalHours > topMonthHours {
			topMonthHours = m.TotalHours
			topMonth = m.YearMonth
		}
	}

	// Peak Listening Time Window
	lateNightStreams := 0
	morningStreams := 0
	afternoonStreams := 0
	eveningStreams := 0

	for _, h := range hourlyHabits {
		if h.Hour >= 23 || h.Hour < 6 {
			lateNightStreams += h.StreamCount
		} else if h.Hour >= 6 && h.Hour < 12 {
			morningStreams += h.StreamCount
		} else if h.Hour >= 12 && h.Hour < 18 {
			afternoonStreams += h.StreamCount
		} else {
			eveningStreams += h.StreamCount
		}
	}

	peakTime := "Afternoons (12 PM - 6 PM)"
	if lateNightStreams >= morningStreams && lateNightStreams >= afternoonStreams && lateNightStreams >= eveningStreams {
		peakTime = "Late Night (11 PM - 6 AM)"
	} else if morningStreams >= afternoonStreams && morningStreams >= eveningStreams {
		peakTime = "Mornings (6 AM - 12 PM)"
	} else if eveningStreams >= afternoonStreams {
		peakTime = "Evenings (6 PM - 11 PM)"
	}

	// Archetype determination
	title := "The Eclectic Voyager"
	desc := "You love exploring a massive range of artists and albums across every mood."

	if overview.Top3ArtistsPct > 45.0 {
		title = "The Ultimate Loyalist"
		desc = fmt.Sprintf("When you find an artist you love (%s), you stick with them relentlessly. Over %.1f%% of your entire listening went to your top 3 artists.", overview.TopArtist, overview.Top3ArtistsPct)
	} else if lateNightStreams*100/max(1, overview.TotalStreams) > 30 {
		title = "The Midnight Dreamer"
		desc = "The moon is your DJ. Over 30% of your listening happens during the late night hours."
	} else if overview.UniqueAlbums > 200 {
		title = "The Album Purist"
		desc = fmt.Sprintf("You deeply explore full artist catalogs and complete records (%d unique albums explored).", overview.UniqueAlbums)
	} else if overview.UniqueArtists > 500 {
		title = "The Musical Explorer"
		desc = fmt.Sprintf("You sampled music from %d different artists—your sonic horizons are limitless.", overview.UniqueArtists)
	}

	var topObsession *db.ObsessionTrack
	if len(obsessions) > 0 {
		topObsession = &obsessions[0]
	}

	var topPlaythrough *db.AlbumPlaythrough
	if playthroughSummary != nil && len(playthroughSummary.TopPlaythroughs) > 0 {
		topPlaythrough = &playthroughSummary.TopPlaythroughs[0]
	}

	return &db.WrappedReport{
		Year:              yearFilter,
		TotalHours:        overview.TotalHours,
		TotalMinutes:      overview.TotalMinutes,
		TotalStreams:      overview.TotalStreams,
		UniqueArtists:     overview.UniqueArtists,
		UniqueAlbums:      overview.UniqueAlbums,
		UniqueTracks:      overview.UniqueTracks,
		SkipRatePct:       overview.SkipRatePct,
		ArchetypeTitle:    title,
		ArchetypeDesc:     desc,
		TopArtists:        topArtists,
		TopAlbums:         topAlbums,
		TopTracks:         topTracks,
		TopMonth:          topMonth,
		TopMonthHours:     topMonthHours,
		PeakDay:           overview.PeakDayDate,
		PeakDayHours:      overview.PeakDayHours,
		PeakDayStreams:    overview.PeakDayStreams,
		PeakListeningTime: peakTime,
		TopObsession:      topObsession,
		TopPlaythrough:    topPlaythrough,
	}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
