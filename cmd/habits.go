package cmd

import (
	"fmt"

	"github.com/chrisburlacu1/spotify-history-cli/internal/analyzer"
	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var habitsCmd = &cobra.Command{
	Use:     "habits",
	Aliases: []string{"patterns", "time"},
	Short:   "Analyze hourly habits, day-of-week patterns, and listening sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		hourly, err := database.GetHourlyHabits(globalYear)
		if err != nil {
			return fmt.Errorf("error fetching hourly habits: %w", err)
		}

		days, err := database.GetDayHabits(globalYear)
		if err != nil {
			return fmt.Errorf("error fetching day habits: %w", err)
		}

		sessions, err := analyzer.AnalyzeSessions(database, 20.0, globalYear)
		if err != nil {
			return fmt.Errorf("error analyzing sessions: %w", err)
		}

		if jsonOut {
			type habitsResponse struct {
				HourlyHabits []interface{}            `json:"hourlyHabits"`
				DayHabits    []interface{}            `json:"dayHabits"`
				Sessions     *analyzer.SessionMetrics `json:"sessionMetrics"`
			}
			var hList []interface{}
			for _, h := range hourly {
				hList = append(hList, h)
			}
			var dList []interface{}
			for _, d := range days {
				dList = append(dList, d)
			}
			return printJSON(habitsResponse{
				HourlyHabits: hList,
				DayHabits:    dList,
				Sessions:     sessions,
			})
		}

		yearLabel := globalYear
		if yearLabel == "" || yearLabel == "all" {
			yearLabel = "All-Time"
		}

		fmt.Print(ui.Header(fmt.Sprintf("24-Hour Listening Distribution (%s)", yearLabel)))
		var hourLabels []string
		var hourValues []float64
		for _, h := range hourly {
			ampm := "AM"
			hr12 := h.Hour
			if h.Hour == 0 {
				hr12 = 12
			} else if h.Hour == 12 {
				ampm = "PM"
			} else if h.Hour > 12 {
				hr12 = h.Hour - 12
				ampm = "PM"
			}
			hourLabels = append(hourLabels, fmt.Sprintf("%02d %s", hr12, ampm))
			hourValues = append(hourValues, h.TotalHours)
		}
		fmt.Println(ui.RenderBarChart(hourLabels, hourValues, 0, 30, func(v float64) string {
			return fmt.Sprintf("%.1f hrs", v)
		}))

		fmt.Print(ui.Header(fmt.Sprintf("Day of the Week Distribution (%s)", yearLabel)))
		var dayLabels []string
		var dayValues []float64
		for _, d := range days {
			dayLabels = append(dayLabels, d.DayName)
			dayValues = append(dayValues, d.TotalHours)
		}
		fmt.Println(ui.RenderBarChart(dayLabels, dayValues, 0, 30, func(v float64) string {
			return fmt.Sprintf("%.1f hrs", v)
		}))

		fmt.Print(ui.Header(fmt.Sprintf("Continuous Listening Sessions (%s)", yearLabel)))
		fmt.Printf("  %sTotal Sessions%s       : %d\n", ui.Bold, ui.Reset, sessions.TotalSessions)
		fmt.Printf("  %sAvg Session Duration%s : %s%.1f minutes%s\n", ui.Bold, ui.Reset, ui.Yellow, sessions.AvgSessionMinutes, ui.Reset)
		fmt.Printf("  %sLongest Session%s      : %s%.1f hours%s (%d tracks on %s)\n",
			ui.Bold, ui.Reset, ui.Bold+ui.Green, sessions.MaxSessionMinutes/60.0, ui.Reset, sessions.LongestSessionPlays, sessions.LongestSessionDate)

		if len(sessions.Sessions) > 0 {
			fmt.Printf("\n%sTop Listening Marathons:%s\n", ui.Bold+ui.Cyan, ui.Reset)
			tbl := ui.NewTable("Start Time", "End Time", "Duration (hrs)", "Tracks", "Top Artist", "Top Album")
			tbl.SetAlignment(2, ui.AlignRight)
			tbl.SetAlignment(3, ui.AlignRight)
			for _, s := range sessions.Sessions {
				tbl.AddRow(
					s.StartTime,
					s.EndTime,
					fmt.Sprintf("%.1f hrs", s.DurationMinutes/60.0),
					s.TrackCount,
					s.TopArtist,
					s.TopAlbum,
				)
			}
			fmt.Println(tbl.Render())
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(habitsCmd)
}
