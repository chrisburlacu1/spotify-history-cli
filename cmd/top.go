package cmd

import (
	"fmt"

	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var (
	topLimit    int
	onlyTracks  bool
	onlyArtists bool
	onlyAlbums  bool
)

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Show top artists, albums, and tracks ranked by listening duration",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		showAll := !onlyTracks && !onlyArtists && !onlyAlbums
		showArtists := showAll || onlyArtists
		showAlbums := showAll || onlyAlbums
		showTracks := showAll || onlyTracks

		if jsonOut {
			type topResponse struct {
				TopArtists interface{} `json:"topArtists,omitempty"`
				TopAlbums  interface{} `json:"topAlbums,omitempty"`
				TopTracks  interface{} `json:"topTracks,omitempty"`
			}
			res := topResponse{}
			if showArtists {
				artists, err := database.GetTopArtists(topLimit, globalYear)
				if err != nil {
					return fmt.Errorf("error fetching top artists: %w", err)
				}
				res.TopArtists = artists
			}
			if showAlbums {
				albums, err := database.GetTopAlbums(topLimit, globalYear)
				if err != nil {
					return fmt.Errorf("error fetching top albums: %w", err)
				}
				res.TopAlbums = albums
			}
			if showTracks {
				tracks, err := database.GetTopTracks(topLimit, globalYear)
				if err != nil {
					return fmt.Errorf("error fetching top tracks: %w", err)
				}
				res.TopTracks = tracks
			}
			return printJSON(res)
		}

		yearLabel := globalYear
		if yearLabel == "" || yearLabel == "all" {
			yearLabel = "All-Time"
		}

		if showArtists {
			artists, err := database.GetTopArtists(topLimit, globalYear)
			if err != nil {
				return fmt.Errorf("error fetching top artists: %w", err)
			}

			fmt.Print(ui.Header(fmt.Sprintf("Top %d Artists (%s)", len(artists), yearLabel)))
			tbl := ui.NewTable("Rank", "Artist", "Plays", "Full", "Skips", "Skip %", "Albums", "Tracks", "Hours")
			tbl.SetAlignment(0, ui.AlignRight)
			tbl.SetAlignment(2, ui.AlignRight)
			tbl.SetAlignment(3, ui.AlignRight)
			tbl.SetAlignment(4, ui.AlignRight)
			tbl.SetAlignment(5, ui.AlignRight)
			tbl.SetAlignment(6, ui.AlignRight)
			tbl.SetAlignment(7, ui.AlignRight)
			tbl.SetAlignment(8, ui.AlignRight)

			for _, a := range artists {
				tbl.AddRow(
					fmt.Sprintf("#%d", a.Rank),
					a.ArtistName,
					a.TotalPlays,
					a.FullPlays,
					a.Skips,
					fmt.Sprintf("%.1f%%", a.SkipRatePct),
					a.UniqueAlbums,
					a.UniqueTracks,
					fmt.Sprintf("%.1f", a.TotalHours),
				)
			}
			fmt.Println(tbl.Render())
		}

		if showAlbums {
			albums, err := database.GetTopAlbums(topLimit, globalYear)
			if err != nil {
				return fmt.Errorf("error fetching top albums: %w", err)
			}

			fmt.Print(ui.Header(fmt.Sprintf("Top %d Albums (%s)", len(albums), yearLabel)))
			tbl := ui.NewTable("Rank", "Album", "Artist", "Plays", "Full", "Skips", "Tracks", "Hours")
			tbl.SetAlignment(0, ui.AlignRight)
			tbl.SetAlignment(3, ui.AlignRight)
			tbl.SetAlignment(4, ui.AlignRight)
			tbl.SetAlignment(5, ui.AlignRight)
			tbl.SetAlignment(6, ui.AlignRight)
			tbl.SetAlignment(7, ui.AlignRight)

			for _, al := range albums {
				tbl.AddRow(
					fmt.Sprintf("#%d", al.Rank),
					al.AlbumTitle,
					al.ArtistName,
					al.TotalPlays,
					al.FullPlays,
					al.Skips,
					al.TracksPlayed,
					fmt.Sprintf("%.1f", al.TotalHours),
				)
			}
			fmt.Println(tbl.Render())
		}

		if showTracks {
			tracks, err := database.GetTopTracks(topLimit, globalYear)
			if err != nil {
				return fmt.Errorf("error fetching top tracks: %w", err)
			}

			fmt.Print(ui.Header(fmt.Sprintf("Top %d Tracks (%s)", len(tracks), yearLabel)))
			tbl := ui.NewTable("Rank", "Track", "Artist", "Album", "Plays", "Full", "Skips", "Hours")
			tbl.SetAlignment(0, ui.AlignRight)
			tbl.SetAlignment(4, ui.AlignRight)
			tbl.SetAlignment(5, ui.AlignRight)
			tbl.SetAlignment(6, ui.AlignRight)
			tbl.SetAlignment(7, ui.AlignRight)

			for _, t := range tracks {
				tbl.AddRow(
					fmt.Sprintf("#%d", t.Rank),
					t.TrackName,
					t.ArtistName,
					t.AlbumName,
					t.PlayCount,
					t.FullPlays,
					t.Skips,
					fmt.Sprintf("%.1f", t.TotalHours),
				)
			}
			fmt.Println(tbl.Render())
		}

		return nil
	},
}

func init() {
	topCmd.Flags().IntVarP(&topLimit, "limit", "n", 10, "Number of items to display")
	topCmd.Flags().BoolVar(&onlyArtists, "artists", false, "Display only top artists")
	topCmd.Flags().BoolVar(&onlyAlbums, "albums", false, "Display only top albums")
	topCmd.Flags().BoolVar(&onlyTracks, "tracks", false, "Display only top tracks")
	rootCmd.AddCommand(topCmd)
}
