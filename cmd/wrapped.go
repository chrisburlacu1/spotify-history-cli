package cmd

import (
	"fmt"

	"github.com/chrisburlacu1/spotify-history-cli/internal/analyzer"
	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var wrappedCmd = &cobra.Command{
	Use:   "wrapped",
	Short: "Generate an interactive terminal Spotify Wrapped recap",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		report, err := analyzer.GenerateWrapped(database, globalYear)
		if err != nil {
			return fmt.Errorf("error generating wrapped: %w", err)
		}

		if jsonOut {
			return printJSON(report)
		}

		yearTitle := globalYear
		if yearTitle == "" || yearTitle == "all" {
			yearTitle = "ALL-TIME"
		}

		fmt.Println("\n" + ui.Bold + ui.Green + "╔════════════════════════════════════════════════════════════════╗" + ui.Reset)
		fmt.Printf("%s║                    SPOTIFY WRAPPED %s                    ║%s\n", ui.Bold+ui.Green, ui.Pad(yearTitle, 8, ui.AlignCenter), ui.Reset)
		fmt.Println(ui.Bold + ui.Green + "╚════════════════════════════════════════════════════════════════╝" + ui.Reset)

		fmt.Printf("\n%sYour Listening Archetype:%s %s%s%s\n", ui.Bold, ui.Reset, ui.Bold+ui.Yellow, report.ArchetypeTitle, ui.Reset)
		fmt.Printf("   %s%s%s\n\n", ui.Italic, report.ArchetypeDesc, ui.Reset)

		fmt.Printf("  %sTotal Minutes Listened%s : %s%.0f mins%s (%s%.1f hours%s / %s%.1f full days%s)\n",
			ui.Bold, ui.Reset, ui.Bold+ui.Green, report.TotalMinutes, ui.Reset,
			ui.Yellow, report.TotalHours, ui.Reset, ui.Cyan, report.TotalHours/24.0, ui.Reset)

		fmt.Printf("  %sArtists Discovered%s      : %s%d unique artists%s\n",
			ui.Bold, ui.Reset, ui.Bold, report.UniqueArtists, ui.Reset)
		fmt.Printf("  %sAlbums Explored%s        : %s%d unique albums%s\n",
			ui.Bold, ui.Reset, ui.Bold, report.UniqueAlbums, ui.Reset)
		fmt.Printf("  %sTracks Streamed%s        : %s%d unique tracks%s across %s%d total plays%s\n",
			ui.Bold, ui.Reset, ui.Bold, report.UniqueTracks, ui.Reset, ui.Bold, report.TotalStreams, ui.Reset)
		if report.TopMonth != "" {
			fmt.Printf("  %sPeak Month%s              : %s%s%s (%.1f listening hours)\n",
				ui.Bold, ui.Reset, ui.Magenta, report.TopMonth, ui.Reset, report.TopMonthHours)
		}
		fmt.Printf("  %sPeak Listening Day%s      : %s%s%s (%.1f hours played)\n",
			ui.Bold, ui.Reset, ui.Magenta, report.PeakDay, ui.Reset, report.PeakDayHours)
		fmt.Printf("  %sFavorite Time to Listen%s : %s%s%s\n",
			ui.Bold, ui.Reset, ui.Yellow, report.PeakListeningTime, ui.Reset)

		// Top 5 Artists
		fmt.Printf("\n%sTop 5 Artists (%s)%s\n", ui.Bold+ui.Yellow, yearTitle, ui.Reset)
		tblA := ui.NewTable("Rank", "Artist", "Total Plays", "Hours Listened")
		tblA.SetAlignment(0, ui.AlignRight)
		tblA.SetAlignment(2, ui.AlignRight)
		tblA.SetAlignment(3, ui.AlignRight)
		for _, a := range report.TopArtists {
			tblA.AddRow(fmt.Sprintf("#%d", a.Rank), a.ArtistName, a.TotalPlays, fmt.Sprintf("%.1f hrs", a.TotalHours))
		}
		fmt.Println(tblA.Render())

		// Top 5 Albums
		if len(report.TopAlbums) > 0 {
			fmt.Printf("%sTop 5 Albums (%s)%s\n", ui.Bold+ui.Yellow, yearTitle, ui.Reset)
			tblAlb := ui.NewTable("Rank", "Album", "Artist", "Plays", "Hours")
			tblAlb.SetAlignment(0, ui.AlignRight)
			tblAlb.SetAlignment(3, ui.AlignRight)
			tblAlb.SetAlignment(4, ui.AlignRight)
			for _, al := range report.TopAlbums {
				tblAlb.AddRow(fmt.Sprintf("#%d", al.Rank), al.AlbumTitle, al.ArtistName, al.TotalPlays, fmt.Sprintf("%.1f hrs", al.TotalHours))
			}
			fmt.Println(tblAlb.Render())
		}

		// Top 5 Tracks
		fmt.Printf("%sTop 5 Tracks (%s)%s\n", ui.Bold+ui.Yellow, yearTitle, ui.Reset)
		tblT := ui.NewTable("Rank", "Track", "Artist", "Plays", "Hours")
		tblT.SetAlignment(0, ui.AlignRight)
		tblT.SetAlignment(3, ui.AlignRight)
		tblT.SetAlignment(4, ui.AlignRight)
		for _, t := range report.TopTracks {
			tblT.AddRow(fmt.Sprintf("#%d", t.Rank), t.TrackName, t.ArtistName, t.PlayCount, fmt.Sprintf("%.1f hrs", t.TotalHours))
		}
		fmt.Println(tblT.Render())

		// Top Obsession
		if report.TopObsession != nil {
			fmt.Printf("%sYour #1 On-Repeat Obsession%s\n", ui.Bold+ui.Magenta, ui.Reset)
			fmt.Printf("   You played %s\"%s\"%s by %s%s%s %s%d times in a single day%s on %s (%.1f mins)!\n\n",
				ui.Bold, report.TopObsession.TrackName, ui.Reset,
				ui.Cyan, report.TopObsession.ArtistName, ui.Reset,
				ui.Bold+ui.Yellow, report.TopObsession.PlaysInDay, ui.Reset,
				report.TopObsession.Date, report.TopObsession.TotalMinutes,
			)
		}

		// Top Playthrough
		if report.TopPlaythrough != nil {
			fmt.Printf("%sTop Sequential Album Playthrough%s\n", ui.Bold+ui.Cyan, ui.Reset)
			fmt.Printf("   You listened through %s%d consecutive tracks%s of %s\"%s\"%s by %s on %s (%.1f mins)!\n\n",
				ui.Bold+ui.Yellow, report.TopPlaythrough.TracksPlayed, ui.Reset,
				ui.Bold, report.TopPlaythrough.AlbumTitle, ui.Reset,
				report.TopPlaythrough.ArtistName,
				report.TopPlaythrough.StartTime, report.TopPlaythrough.DurationMinutes,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(wrappedCmd)
}
