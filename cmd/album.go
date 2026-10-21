package cmd

import (
	"fmt"
	"strings"

	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var albumCmd = &cobra.Command{
	Use:   "album [album title]",
	Short: "Inspect an album's sequential tracklist, play counts, and completion rates",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		albumTitle := strings.Join(args, " ")
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		details, err := database.GetAlbumDetails(albumTitle)
		if err != nil {
			return fmt.Errorf("error fetching album details: %w", err)
		}

		if jsonOut {
			return printJSON(details)
		}

		fmt.Print(ui.Header(fmt.Sprintf("Album: %s by %s", details.AlbumTitle, details.ArtistName)))
		fmt.Printf("  %sTotal Listening%s  : %s%.1f hours%s (%d plays across %d tracks)\n",
			ui.Bold, ui.Reset, ui.Bold+ui.Green, details.TotalHours, ui.Reset, details.TotalPlays, details.TracksPlayed)
		if details.ReleaseDate != "" {
			fmt.Printf("  %sRelease Date%s    : %s (%s)\n", ui.Bold, ui.Reset, details.ReleaseDate, details.AlbumType)
		}
		fmt.Printf("  %sTimeline%s        : First listened on %s, last on %s\n\n",
			ui.Bold, ui.Reset, details.FirstListened, details.LastListened)

		if len(details.Tracks) > 0 {
			fmt.Printf("%sSequential Tracklist & Drop-off Heatmap:%s\n", ui.Bold+ui.Cyan, ui.Reset)
			tbl := ui.NewTable("#", "Track Title", "Plays", "Full", "Skips", "Completion", "Hours")
			tbl.SetAlignment(0, ui.AlignRight)
			tbl.SetAlignment(2, ui.AlignRight)
			tbl.SetAlignment(3, ui.AlignRight)
			tbl.SetAlignment(4, ui.AlignRight)
			tbl.SetAlignment(5, ui.AlignRight)
			tbl.SetAlignment(6, ui.AlignRight)

			for _, t := range details.Tracks {
				numStr := "-"
				if t.TrackNumber > 0 {
					numStr = fmt.Sprintf("%02d", t.TrackNumber)
				}
				compStr := "-"
				if t.PlayCount > 0 {
					compStr = fmt.Sprintf("%.1f%%", t.CompletionRate)
				}
				tbl.AddRow(
					numStr,
					t.TrackName,
					t.PlayCount,
					t.FullPlays,
					t.Skips,
					compStr,
					fmt.Sprintf("%.1f", t.TotalHours),
				)
			}
			fmt.Println(tbl.Render())
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(albumCmd)
}
