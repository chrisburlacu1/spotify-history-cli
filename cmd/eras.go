package cmd

import (
	"fmt"

	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var erasCmd = &cobra.Command{
	Use:     "eras",
	Aliases: []string{"decades", "vintage"},
	Short:   "Analyze listening breakdown by release decade and music vintage",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		decades, err := database.GetDecadeStats(globalYear)
		if err != nil {
			return fmt.Errorf("error fetching decade stats: %w", err)
		}

		if jsonOut {
			return printJSON(decades)
		}

		yearLabel := globalYear
		if yearLabel == "" || yearLabel == "all" {
			yearLabel = "All-Time"
		}

		fmt.Print(ui.Header(fmt.Sprintf("Music Vintage & Release Decades (%s)", yearLabel)))
		var labels []string
		var values []float64
		for _, d := range decades {
			labels = append(labels, d.Decade)
			values = append(values, d.TotalHours)
		}
		fmt.Println(ui.RenderBarChart(labels, values, 0, 30, func(v float64) string {
			return fmt.Sprintf("%.1f hrs", v)
		}))

		fmt.Println()
		tbl := ui.NewTable("Decade / Era", "Total Streams", "Hours Played", "Unique Tracks", "Share")
		tbl.SetAlignment(1, ui.AlignRight)
		tbl.SetAlignment(2, ui.AlignRight)
		tbl.SetAlignment(3, ui.AlignRight)
		tbl.SetAlignment(4, ui.AlignRight)

		for _, d := range decades {
			tbl.AddRow(
				d.Decade,
				d.TotalStreams,
				fmt.Sprintf("%.1f hrs", d.TotalHours),
				d.UniqueTracks,
				fmt.Sprintf("%.1f%%", d.Percentage),
			)
		}
		fmt.Println(tbl.Render())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(erasCmd)
}
