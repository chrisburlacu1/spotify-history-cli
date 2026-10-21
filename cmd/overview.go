package cmd

import (
	"fmt"

	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var overviewCmd = &cobra.Command{
	Use:     "overview",
	Aliases: []string{"stats", "summary"},
	Short:   "Display comprehensive listening dashboard and metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		stats, err := database.GetOverview(globalYear)
		if err != nil {
			return fmt.Errorf("failed to get overview: %w", err)
		}

		if jsonOut {
			return printJSON(stats)
		}

		title := fmt.Sprintf("%s Listening Overview", globalYear)
		if globalYear == "" || globalYear == "all" {
			title = "All-Time Listening Overview"
		}
		fmt.Print(ui.Header(title))

		fmt.Printf("  %sTotal Listening Time%s : %s%.1f hours%s (%s%.1f days%s / %s%.0f mins%s)\n",
			ui.Bold, ui.Reset, ui.Bold+ui.Green, stats.TotalHours, ui.Reset,
			ui.Yellow, stats.TotalDays, ui.Reset, ui.Dim, stats.TotalMinutes, ui.Reset)

		fmt.Printf("  %sTotal Streams%s        : %s%d%s\n",
			ui.Bold, ui.Reset, ui.Bold, stats.TotalStreams, ui.Reset)

		fmt.Printf("  %sUnique Artists%s      : %s%d%s\n",
			ui.Bold, ui.Reset, ui.Cyan, stats.UniqueArtists, ui.Reset)

		fmt.Printf("  %sUnique Albums%s       : %s%d%s\n",
			ui.Bold, ui.Reset, ui.Cyan, stats.UniqueAlbums, ui.Reset)

		fmt.Printf("  %sUnique Tracks%s       : %s%d%s\n",
			ui.Bold, ui.Reset, ui.Cyan, stats.UniqueTracks, ui.Reset)

		fmt.Printf("  %sSkip Rate%s           : %s%.1f%%%s (%d skipped streams <30s)\n",
			ui.Bold, ui.Reset, ui.Red, stats.SkipRatePct, ui.Reset, stats.SkippedStreams)

		fmt.Printf("  %sActive Listening Days%s: %s%d days%s (Avg: %.1f streams / %.1f hrs per day)\n",
			ui.Bold, ui.Reset, ui.Bold, stats.ActiveDays, ui.Reset, stats.AvgStreamsPerDay, stats.AvgHoursPerDay)

		fmt.Printf("  %sPeak Listening Day%s   : %s%s%s (%.1f hrs, %d streams)\n",
			ui.Bold, ui.Reset, ui.Magenta, stats.PeakDayDate, ui.Reset, stats.PeakDayHours, stats.PeakDayStreams)

		fmt.Printf("  %s#1 Top Artist%s       : %s%s%s (%.1f hrs played)\n",
			ui.Bold, ui.Reset, ui.Bold+ui.Yellow, stats.TopArtist, ui.Reset, stats.TopArtistHours)

		if stats.TopAlbum != "" {
			fmt.Printf("  %s#1 Top Album%s        : %s%s%s by %s (%.1f hrs)\n",
				ui.Bold, ui.Reset, ui.Bold+ui.Yellow, stats.TopAlbum, ui.Reset, stats.TopAlbumArtist, stats.TopAlbumHours)
		}

		fmt.Printf("  %s#1 Top Track%s        : %s%s%s by %s (%d plays)\n",
			ui.Bold, ui.Reset, ui.Bold+ui.Yellow, stats.TopTrack, ui.Reset, stats.TopTrackArtist, stats.TopTrackPlays)

		fmt.Printf("  %sTop 3 Concentration%s  : %s%.1f%%%s of all listening time\n",
			ui.Bold, ui.Reset, ui.Cyan, stats.Top3ArtistsPct, ui.Reset)

		fmt.Printf("  %sDate Range%s          : %s to %s\n\n",
			ui.Bold, ui.Reset, stats.FirstStreamDate, stats.LastStreamDate)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(overviewCmd)
}
