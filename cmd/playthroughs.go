package cmd

import (
	"fmt"

	"github.com/chrisburlacu1/spotify-history-cli/internal/analyzer"
	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var minPlaythroughTracks int

var playthroughsCmd = &cobra.Command{
	Use:     "playthroughs",
	Aliases: []string{"album-loops", "marathons"},
	Short:   "Find continuous sequential album playthroughs (Track 1 -> 2 -> 3...)",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		summary, err := analyzer.AnalyzePlaythroughs(database, minPlaythroughTracks, globalYear)
		if err != nil {
			return fmt.Errorf("error analyzing album playthroughs: %w", err)
		}

		if jsonOut {
			return printJSON(summary)
		}

		yearLabel := globalYear
		if yearLabel == "" || yearLabel == "all" {
			yearLabel = "All-Time"
		}

		fmt.Print(ui.Header(fmt.Sprintf("Album Sequential Playthroughs (%s, ≥%d tracks in order)", yearLabel, minPlaythroughTracks)))
		if len(summary.TopPlaythroughs) == 0 {
			fmt.Printf("No sequential album listens found with ≥%d tracks.\n", minPlaythroughTracks)
			fmt.Println("Tip: Run `spotify-history-cli enrich` to fetch track numbers for albums.")
			return nil
		}

		tbl := ui.NewTable("Album Title", "Artist", "Start Time", "Tracks", "Span", "Duration")
		tbl.SetAlignment(3, ui.AlignRight)
		tbl.SetAlignment(5, ui.AlignRight)

		for _, p := range summary.TopPlaythroughs {
			spanStr := fmt.Sprintf("Tracks %d-%d", p.StartTrackNum, p.EndTrackNum)
			if p.IsFullAlbum {
				spanStr = ui.Colorize(ui.Bold+ui.Yellow, fmt.Sprintf("FULL (1-%d)", p.EndTrackNum))
			}
			tbl.AddRow(
				p.AlbumTitle,
				p.ArtistName,
				p.StartTime,
				fmt.Sprintf("%d tracks", p.TracksPlayed),
				spanStr,
				fmt.Sprintf("%.1f min", p.DurationMinutes),
			)
		}

		fmt.Println(tbl.Render())
		return nil
	},
}

func init() {
	playthroughsCmd.Flags().IntVarP(&minPlaythroughTracks, "min", "m", 4, "Minimum consecutive tracks on same album")
	rootCmd.AddCommand(playthroughsCmd)
}
