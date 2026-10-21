package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chrisburlacu1/spotify-history-cli/internal/db"
	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var (
	sourceDirsFlag string
)

var ingestCmd = &cobra.Command{
	Use:   "ingest [paths...]",
	Short: "Ingest Spotify history files or directories into relational SQLite",
	Long: `Parses both Extended & Standard Spotify streaming history formats (endsong_*.json, 
Streaming_History_Audio_*.json, StreamingHistory_music_*.json), populates relational 
artists, albums, tracks, and streams, and deduplicates identical records.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var candidatePaths []string
		if len(args) > 0 {
			candidatePaths = args
		} else if sourceDirsFlag != "" {
			for _, d := range strings.Split(sourceDirsFlag, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					candidatePaths = append(candidatePaths, d)
				}
			}
		}

		var validPaths []string
		for _, p := range candidatePaths {
			if _, err := os.Stat(p); err == nil {
				validPaths = append(validPaths, p)
			}
		}

		if len(validPaths) == 0 {
			return fmt.Errorf("no valid Spotify data files or directories found.\n\nUsage:\n  spotify-history-cli ingest <path/to/folder-or-files...> [flags]\n\nExamples:\n  spotify-history-cli ingest ./my_spotify_data/\n  spotify-history-cli ingest endsong_0.json endsong_1.json\n  spotify-history-cli ingest --dirs ./data,./extra-data")
		}

		targetDB := resolveDBPath()
		if parent := filepath.Dir(targetDB); parent != "" && parent != "." {
			_ = os.MkdirAll(parent, 0755)
		}

		database, err := db.Open(targetDB)
		if err != nil {
			return fmt.Errorf("failed to initialize database at '%s': %w", targetDB, err)
		}
		defer database.Close()

		if !jsonOut {
			fmt.Printf("%s\n", ui.Header("Starting Multi-Source Ingestion"))
			fmt.Printf("  Sources   : %s%s%s\n", ui.Cyan, strings.Join(validPaths, ", "), ui.Reset)
			fmt.Printf("  Target DB : %s%s%s\n\n", ui.Cyan, targetDB, ui.Reset)
		}

		stats, err := database.IngestAllSources(validPaths)
		if err != nil {
			return fmt.Errorf("ingestion failed: %w", err)
		}

		if jsonOut {
			return printJSON(stats)
		}

		fmt.Printf("%s\n", ui.Colorize(ui.Bold+ui.Green, "Relational Ingestion Completed in "+stats.ElapsedDuration))
		fmt.Printf("──────────────────────────────────────────────────\n")
		fmt.Printf("  • Total Streams Ingested : %s%d%s\n", ui.Bold, stats.TotalStreams, ui.Reset)
		if stats.DuplicateStreams > 0 {
			fmt.Printf("  • Deduplicated Streams   : %s%d%s (filtered out)\n", ui.Dim, stats.DuplicateStreams, ui.Reset)
		}
		fmt.Printf("  • Time Range             : %s to %s\n", stats.StartDate, stats.EndDate)
		fmt.Printf("  • Total Listening Time   : %s%.1f hours%s (%.1f days)\n", ui.Bold+ui.Yellow, stats.TotalHours, ui.Reset, stats.TotalDays)
		fmt.Printf("  • Unique Artists         : %s%d%s\n", ui.Bold, stats.TotalArtists, ui.Reset)
		fmt.Printf("  • Unique Albums          : %s%d%s\n", ui.Bold, stats.TotalAlbums, ui.Reset)
		fmt.Printf("  • Unique Tracks          : %s%d%s (%d with albums resolved)\n", ui.Bold, stats.TotalTracks, ui.Reset, stats.TracksWithAlbums)
		fmt.Printf("  • Skipped Tracks (<30s)  : %d (%s%.1f%%%s)\n", stats.SkippedStreams, ui.Red, stats.SkipRatePct, ui.Reset)
		fmt.Printf("  • Database File Size     : %.2f MB\n", stats.DatabaseSizeMB)
		fmt.Printf("──────────────────────────────────────────────────\n")

		return nil
	},
}

func init() {
	ingestCmd.Flags().StringVar(&sourceDirsFlag, "dirs", "./data,./extra-data", "Comma-separated list of directories containing Spotify JSON files")
	rootCmd.AddCommand(ingestCmd)
}
