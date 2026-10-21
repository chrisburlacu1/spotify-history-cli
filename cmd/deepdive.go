package cmd

import (
	"fmt"
	"strings"

	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var deepdiveCmd = &cobra.Command{
	Use:     "deepdive [artist name]",
	Aliases: []string{"artist", "profile"},
	Short:   "Deep-dive analysis into an artist's listening timeline, albums, and tracks",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		artistName := strings.Join(args, " ")
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		profile, err := database.GetDeepArtistProfile(artistName, globalYear)
		if err != nil {
			return fmt.Errorf("error fetching artist profile: %w", err)
		}

		if jsonOut {
			return printJSON(profile)
		}

		yearLabel := globalYear
		if yearLabel == "" || yearLabel == "all" {
			yearLabel = "All-Time"
		}

		fmt.Print(ui.Header(fmt.Sprintf("Deep Dive: %s (%s)", profile.ArtistName, yearLabel)))

		fmt.Printf("  %sTotal Listening%s  : %s%.1f hours%s (%.0f mins across %d plays)\n",
			ui.Bold, ui.Reset, ui.Bold+ui.Green, profile.TotalHours, ui.Reset, profile.TotalMinutes, profile.TotalPlays)
		fmt.Printf("  %sFull vs Skips%s   : %d full plays / %d skips (%s%.1f%% skip rate%s)\n",
			ui.Bold, ui.Reset, profile.FullPlays, profile.Skips, ui.Red, profile.SkipRatePct, ui.Reset)
		fmt.Printf("  %sCatalog Breadth%s : %d unique tracks across %d albums explored\n",
			ui.Bold, ui.Reset, profile.UniqueTracks, profile.UniqueAlbums)
		fmt.Printf("  %sTimeline%s        : First listened on %s, last on %s\n",
			ui.Bold, ui.Reset, profile.FirstListened, profile.LastListened)
		fmt.Printf("  %sPeak Day%s        : %s\n",
			ui.Bold, ui.Reset, profile.PeakListeningDay)
		fmt.Printf("  %sPeak Hour%s       : %02d:00 (%d:00 %s)\n",
			ui.Bold, ui.Reset, profile.PeakHour, (profile.PeakHour+11)%12+1, map[bool]string{true: "AM", false: "PM"}[profile.PeakHour < 12])

		// Top Albums
		if len(profile.TopAlbums) > 0 {
			fmt.Printf("\n%sTop Albums by %s%s\n", ui.Bold+ui.Cyan, profile.ArtistName, ui.Reset)
			tblAlb := ui.NewTable("Rank", "Album Title", "Plays", "Full", "Skips", "Tracks", "Hours")
			tblAlb.SetAlignment(0, ui.AlignRight)
			tblAlb.SetAlignment(2, ui.AlignRight)
			tblAlb.SetAlignment(3, ui.AlignRight)
			tblAlb.SetAlignment(4, ui.AlignRight)
			tblAlb.SetAlignment(5, ui.AlignRight)
			tblAlb.SetAlignment(6, ui.AlignRight)

			for _, al := range profile.TopAlbums {
				tblAlb.AddRow(
					fmt.Sprintf("#%d", al.Rank),
					al.AlbumTitle,
					al.TotalPlays,
					al.FullPlays,
					al.Skips,
					al.TracksPlayed,
					fmt.Sprintf("%.1f", al.TotalHours),
				)
			}
			fmt.Println(tblAlb.Render())
		}

		// Top Tracks
		if len(profile.TopTracks) > 0 {
			fmt.Printf("%sTop Tracks by %s%s\n", ui.Bold+ui.Cyan, profile.ArtistName, ui.Reset)
			tblTrk := ui.NewTable("Rank", "Track", "Album", "Plays", "Full", "Skips", "Hours")
			tblTrk.SetAlignment(0, ui.AlignRight)
			tblTrk.SetAlignment(3, ui.AlignRight)
			tblTrk.SetAlignment(4, ui.AlignRight)
			tblTrk.SetAlignment(5, ui.AlignRight)
			tblTrk.SetAlignment(6, ui.AlignRight)

			for _, t := range profile.TopTracks {
				tblTrk.AddRow(
					fmt.Sprintf("#%d", t.Rank),
					t.TrackName,
					t.AlbumName,
					t.PlayCount,
					t.FullPlays,
					t.Skips,
					fmt.Sprintf("%.1f", t.TotalHours),
				)
			}
			fmt.Println(tblTrk.Render())
		}

		// Monthly Listening Bar Chart
		if len(profile.MonthlyBreakdown) > 0 {
			fmt.Printf("%sMonthly Trend%s\n", ui.Bold+ui.Cyan, ui.Reset)
			var labels []string
			var values []float64
			for _, m := range profile.MonthlyBreakdown {
				labels = append(labels, m.YearMonth)
				values = append(values, m.TotalHours)
			}
			fmt.Println(ui.RenderBarChart(labels, values, 0, 25, func(v float64) string {
				return fmt.Sprintf("%.1f hrs", v)
			}))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deepdiveCmd)
}
