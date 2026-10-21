package cmd

import (
	"fmt"
	"strings"

	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var searchLimit int

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Full-text search for tracks and artists across your streaming history",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		results, err := database.SearchFTS(query, searchLimit)
		if err != nil {
			return fmt.Errorf("search error: %w", err)
		}

		if jsonOut {
			return printJSON(results)
		}

		fmt.Print(ui.Header(fmt.Sprintf("Search Results for \"%s\" (%d matches)", query, len(results))))
		if len(results) == 0 {
			fmt.Println("No matches found.")
			return nil
		}

		tbl := ui.NewTable("Played At", "Artist", "Track", "Duration", "Skipped?")
		tbl.SetAlignment(3, ui.AlignRight)
		tbl.SetAlignment(4, ui.AlignCenter)

		for _, r := range results {
			skippedStr := "No"
			if r.IsSkipped == 1 {
				skippedStr = ui.Colorize(ui.Red, "Yes (<30s)")
			}
			tbl.AddRow(
				r.PlayedAtStr,
				r.ArtistName,
				r.TrackName,
				fmt.Sprintf("%.1f min", r.MinutesPlayed),
				skippedStr,
			)
		}
		fmt.Println(tbl.Render())
		return nil
	},
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 20, "Maximum search results to return")
	rootCmd.AddCommand(searchCmd)
}
