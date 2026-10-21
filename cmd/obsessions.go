package cmd

import (
	"fmt"

	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var (
	minPlaysInDay  int
	obsessionLimit int
)

var obsessionsCmd = &cobra.Command{
	Use:     "obsessions",
	Aliases: []string{"loops", "binges", "on-repeat"},
	Short:   "Find songs you had on repeat (single-day binge listening)",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		obsessions, err := database.GetObsessionTracks(minPlaysInDay, obsessionLimit, globalYear)
		if err != nil {
			return fmt.Errorf("error fetching obsession tracks: %w", err)
		}

		if jsonOut {
			return printJSON(obsessions)
		}

		yearLabel := globalYear
		if yearLabel == "" || yearLabel == "all" {
			yearLabel = "All-Time"
		}

		fmt.Print(ui.Header(fmt.Sprintf("Single-Day Track Obsessions (%s, Played ≥ %d times in 24h)", yearLabel, minPlaysInDay)))
		if len(obsessions) == 0 {
			fmt.Printf("No tracks found played ≥ %d times in a single day.\n", minPlaysInDay)
			return nil
		}

		tbl := ui.NewTable("Date", "Track", "Artist", "Album", "Plays in Day", "Minutes")
		tbl.SetAlignment(4, ui.AlignRight)
		tbl.SetAlignment(5, ui.AlignRight)

		for _, o := range obsessions {
			tbl.AddRow(
				o.Date,
				o.TrackName,
				o.ArtistName,
				o.AlbumName,
				fmt.Sprintf("%d plays", o.PlaysInDay),
				fmt.Sprintf("%.1f min", o.TotalMinutes),
			)
		}

		fmt.Println(tbl.Render())
		return nil
	},
}

func init() {
	obsessionsCmd.Flags().IntVarP(&minPlaysInDay, "min", "m", 5, "Minimum plays in a single day")
	obsessionsCmd.Flags().IntVarP(&obsessionLimit, "limit", "n", 15, "Number of obsessions to show")
	rootCmd.AddCommand(obsessionsCmd)
}
