package cmd

import (
	"fmt"

	"github.com/chrisburlacu1/spotify-history-cli/internal/spotify"
	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var (
	enrichTopArtists int
	spotifyClientID  string
	spotifySecret    string
)

var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Enrich missing albums, track numbers, and release dates via Spotify API",
	Long:  `Uses a discography-first algorithm to query artist albums and track numbers from Spotify, drastically reducing API calls and populating full album sequencing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := spotify.NewClient(spotifyClientID, spotifySecret)
		if err != nil {
			return fmt.Errorf("%w\nTip: Set SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET environment variables", err)
		}

		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		enricher := spotify.NewEnricher(client, database)

		if !jsonOut {
			fmt.Print(ui.Header("Spotify API Discography Enrichment"))
		}

		stats, err := enricher.EnrichLibrary(enrichTopArtists, func(msg string) {
			if !jsonOut {
				fmt.Printf("  • %s\n", msg)
			}
		})
		if err != nil {
			return fmt.Errorf("enrichment error: %w", err)
		}

		if jsonOut {
			return printJSON(stats)
		}

		fmt.Printf("\n%s\n", ui.Colorize(ui.Bold+ui.Green, "Enrichment Finished in "+stats.Duration))
		fmt.Printf("  • Artists Processed : %d\n", stats.ArtistsProcessed)
		fmt.Printf("  • Albums Enriched   : %d\n", stats.AlbumsEnriched)
		fmt.Printf("  • Tracks Updated    : %d\n", stats.TracksEnriched)
		fmt.Printf("  • Total API Calls   : %d\n\n", stats.APICallsMade)

		return nil
	},
}

func init() {
	enrichCmd.Flags().IntVar(&enrichTopArtists, "top", 50, "Number of top artists to scan discographies for")
	enrichCmd.Flags().StringVar(&spotifyClientID, "client-id", "", "Spotify Client ID (or set SPOTIFY_CLIENT_ID env)")
	enrichCmd.Flags().StringVar(&spotifySecret, "client-secret", "", "Spotify Client Secret (or set SPOTIFY_CLIENT_SECRET env)")
	rootCmd.AddCommand(enrichCmd)
}
