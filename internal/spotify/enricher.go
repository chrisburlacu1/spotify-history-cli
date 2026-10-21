package spotify

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"github.com/chrisburlacu1/spotify-history-cli/internal/db"
	"strconv"
	"time"
)

type Enricher struct {
	client *Client
	db     *db.Database
}

type EnrichmentStats struct {
	ArtistsProcessed int    `json:"artistsProcessed"`
	AlbumsEnriched   int    `json:"albumsEnriched"`
	TracksEnriched   int    `json:"tracksEnriched"`
	APICallsMade     int    `json:"apiCallsMade"`
	Duration         string `json:"duration"`
}

func NewEnricher(client *Client, database *db.Database) *Enricher {
	return &Enricher{
		client: client,
		db:     database,
	}
}

// Structs for parsing Spotify API Responses
type searchArtistResponse struct {
	Artists struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			URI  string `json:"uri"`
		} `json:"items"`
	} `json:"artists"`
}

type artistAlbumsResponse struct {
	Items []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		AlbumType   string `json:"album_type"`
		ReleaseDate string `json:"release_date"`
		Images      []struct {
			URL string `json:"url"`
		} `json:"images"`
		URI string `json:"uri"`
	} `json:"items"`
}

type albumDetailsResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AlbumType   string `json:"album_type"`
	ReleaseDate string `json:"release_date"`
	Images      []struct {
		URL string `json:"url"`
	} `json:"images"`
	Tracks struct {
		Items []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			TrackNumber int    `json:"track_number"`
			DurationMs  int    `json:"duration_ms"`
			URI         string `json:"uri"`
			Artists     []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"items"`
	} `json:"tracks"`
}

type searchTrackResponse struct {
	Tracks struct {
		Items []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			TrackNumber int    `json:"track_number"`
			DurationMs  int    `json:"duration_ms"`
			URI         string `json:"uri"`
			Album       struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				AlbumType   string `json:"album_type"`
				ReleaseDate string `json:"release_date"`
				Images      []struct {
					URL string `json:"url"`
				} `json:"images"`
				URI string `json:"uri"`
			} `json:"album"`
			Artists []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"items"`
	} `json:"tracks"`
}

func parseReleaseYear(relDate string) int {
	if len(relDate) >= 4 {
		if y, err := strconv.Atoi(relDate[:4]); err == nil {
			return y
		}
	}
	return 0
}

func (e *Enricher) EnrichLibrary(topArtistsLimit int, onProgress func(msg string)) (*EnrichmentStats, error) {
	start := time.Now()
	stats := &EnrichmentStats{}

	// Step 1: Query artists that have tracks missing track numbers or album metadata, sorted by listening hours
	artistRows, err := e.db.Conn().Query(`
		SELECT a.id, a.name, COUNT(DISTINCT t.id) as missing_count
		FROM artists a
		JOIN tracks t ON t.artist_id = a.id
		JOIN streams s ON s.track_id = t.id
		WHERE t.track_number IS NULL OR t.track_number = 0 OR t.album_id IS NULL
		GROUP BY a.id, a.name
		ORDER BY SUM(s.hours_played) DESC
		LIMIT ?
	`, topArtistsLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to query artists needing enrichment: %w", err)
	}
	defer artistRows.Close()

	type artistTask struct {
		id           int64
		name         string
		missingCount int
	}

	var heavyArtists []artistTask
	for artistRows.Next() {
		var a artistTask
		if err := artistRows.Scan(&a.id, &a.name, &a.missingCount); err == nil {
			heavyArtists = append(heavyArtists, a)
		}
	}
	artistRows.Close()

	if onProgress != nil {
		onProgress(fmt.Sprintf("Found %d top artists with missing album metadata", len(heavyArtists)))
	}

	// Phase 1: Smart Discography Lookup for Heavy Artists (>= 4 missing tracks)
	for _, at := range heavyArtists {
		if at.missingCount < 3 {
			continue
		}

		if onProgress != nil {
			onProgress(fmt.Sprintf("Scanning discography for: %s (%d tracks in library)", at.name, at.missingCount))
		}

		// 1. Search Artist ID
		q := url.Values{}
		q.Set("q", fmt.Sprintf(`artist:"%s"`, at.name))
		q.Set("type", "artist")
		q.Set("limit", "1")

		body, err := e.client.Get("/search", q)
		stats.APICallsMade++
		if err != nil {
			continue
		}

		var sResp searchArtistResponse
		if err := json.Unmarshal(body, &sResp); err != nil || len(sResp.Artists.Items) == 0 {
			continue
		}
		spotifyArtistID := sResp.Artists.Items[0].ID
		stats.ArtistsProcessed++

		// 2. Get Artist Albums
		albQ := url.Values{}
		albQ.Set("include_groups", "album,single")
		albQ.Set("limit", "50")

		albBody, err := e.client.Get(fmt.Sprintf("/artists/%s/albums", spotifyArtistID), albQ)
		stats.APICallsMade++
		if err != nil {
			continue
		}

		var albResp artistAlbumsResponse
		if err := json.Unmarshal(albBody, &albResp); err != nil {
			continue
		}

		// 2.5 Sort albums so full studio albums are processed before singles
		items := albResp.Items
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].AlbumType == "album" && items[j].AlbumType != "album" {
				return true
			}
			return false
		})

		// 3. For each album, fetch tracklist and update SQLite
		for _, alb := range items {
			detailBody, err := e.client.Get(fmt.Sprintf("/albums/%s", alb.ID), nil)
			stats.APICallsMade++
			if err != nil {
				continue
			}

			var detailResp albumDetailsResponse
			if err := json.Unmarshal(detailBody, &detailResp); err != nil {
				continue
			}

			coverURL := ""
			if len(detailResp.Images) > 0 {
				coverURL = detailResp.Images[0].URL
			}
			relYear := parseReleaseYear(detailResp.ReleaseDate)

			// Get or Create Album in SQLite (protecting studio album status against singles)
			var albumID int64
			err = e.db.Conn().QueryRow(`
				INSERT INTO albums (artist_id, title, album_type, release_date, release_year, cover_url, spotify_uri)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(artist_id, title) DO UPDATE SET
					album_type = CASE 
						WHEN albums.album_type = 'album' AND excluded.album_type = 'single' THEN 'album'
						ELSE excluded.album_type
					END,
					release_date = CASE 
						WHEN albums.album_type = 'album' AND excluded.album_type = 'single' THEN albums.release_date
						ELSE excluded.release_date
					END,
					release_year = CASE 
						WHEN albums.album_type = 'album' AND excluded.album_type = 'single' THEN albums.release_year
						ELSE excluded.release_year
					END,
					cover_url = COALESCE(excluded.cover_url, albums.cover_url),
					spotify_uri = COALESCE(excluded.spotify_uri, albums.spotify_uri)
				RETURNING id
			`, at.id, detailResp.Name, detailResp.AlbumType, detailResp.ReleaseDate, relYear, coverURL, "spotify:album:"+detailResp.ID).Scan(&albumID)
			if err != nil {
				_ = e.db.Conn().QueryRow("SELECT id FROM albums WHERE artist_id = ? AND title = ?", at.id, detailResp.Name).Scan(&albumID)
			}
			if albumID > 0 {
				stats.AlbumsEnriched++
			}

			// Match tracks on album (Studio albums take priority over singles)
			for _, trk := range detailResp.Tracks.Items {
				res, err := e.db.Conn().Exec(`
					UPDATE tracks 
					SET album_id = ?, track_number = ?, duration_ms = ?, spotify_uri = ?
					WHERE artist_id = ? AND title = ?
					AND (
						album_id IS NULL 
						OR (SELECT album_type FROM albums WHERE id = tracks.album_id) != 'album'
						OR ? = 'album'
					)
				`, albumID, trk.TrackNumber, trk.DurationMs, trk.URI, at.id, trk.Name, detailResp.AlbumType)
				if err == nil {
					if rowsAff, _ := res.RowsAffected(); rowsAff > 0 {
						stats.TracksEnriched += int(rowsAff)
					}
				}
			}
		}
	}

	// Phase 2: Direct Track Search for remaining unmatched tracks (ordered by listening hours)
	unmatchedRows, err := e.db.Conn().Query(`
		SELECT t.id, a.id, a.name, t.title
		FROM tracks t
		JOIN artists a ON t.artist_id = a.id
		JOIN streams s ON s.track_id = t.id
		WHERE t.track_number IS NULL OR t.track_number = 0 OR t.album_id IS NULL
		GROUP BY t.id, a.id, a.name, t.title
		ORDER BY SUM(s.hours_played) DESC
		LIMIT 150
	`)
	if err == nil {
		type trackTask struct {
			trackID  int64
			artistID int64
			artist   string
			title    string
		}
		var unmatchedTracks []trackTask
		for unmatchedRows.Next() {
			var u trackTask
			if err := unmatchedRows.Scan(&u.trackID, &u.artistID, &u.artist, &u.title); err == nil {
				unmatchedTracks = append(unmatchedTracks, u)
			}
		}
		unmatchedRows.Close()

		if len(unmatchedTracks) > 0 && onProgress != nil {
			onProgress(fmt.Sprintf("Running direct search for %d single unmatched tracks...", len(unmatchedTracks)))
		}

		for _, ut := range unmatchedTracks {
			tQ := url.Values{}
			tQ.Set("q", fmt.Sprintf(`track:"%s" artist:"%s"`, ut.title, ut.artist))
			tQ.Set("type", "track")
			tQ.Set("limit", "1")

			body, err := e.client.Get("/search", tQ)
			stats.APICallsMade++
			if err != nil {
				continue
			}

			var trkResp searchTrackResponse
			if err := json.Unmarshal(body, &trkResp); err != nil || len(trkResp.Tracks.Items) == 0 {
				continue
			}

			item := trkResp.Tracks.Items[0]
			coverURL := ""
			if len(item.Album.Images) > 0 {
				coverURL = item.Album.Images[0].URL
			}
			relYear := parseReleaseYear(item.Album.ReleaseDate)

			var albumID int64
			_ = e.db.Conn().QueryRow(`
				INSERT INTO albums (artist_id, title, album_type, release_date, release_year, cover_url, spotify_uri)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(artist_id, title) DO UPDATE SET
					album_type = excluded.album_type,
					release_date = excluded.release_date,
					release_year = excluded.release_year,
					cover_url = excluded.cover_url
				RETURNING id
			`, ut.artistID, item.Album.Name, item.Album.AlbumType, item.Album.ReleaseDate, relYear, coverURL, item.Album.URI).Scan(&albumID)

			_, _ = e.db.Conn().Exec(`
				UPDATE tracks 
				SET album_id = ?, track_number = ?, duration_ms = ?, spotify_uri = ?
				WHERE id = ?
			`, albumID, item.TrackNumber, item.DurationMs, item.URI, ut.trackID)

			stats.TracksEnriched++
		}
	}

	stats.Duration = time.Since(start).Round(time.Millisecond).String()
	return stats, nil
}
