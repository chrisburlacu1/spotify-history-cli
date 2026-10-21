package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chrisburlacu1/spotify-history-cli/internal/db"
	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var (
	dbPath     string
	globalYear string
	jsonOut    bool
	noColor    bool
)

var rootCmd = &cobra.Command{
	Use:          "spotify-history-cli",
	Short:        "Relational Spotify Streaming History Analytics Engine",
	SilenceUsage: true,
	Long: `spotify-history-cli is an ultra-fast relational analytics engine and CLI tool for exploring, 
deep-diving, and querying your Spotify streaming history.

Supports multi-year history, album playthrough detection, release eras, 
and AI agent integration with --json.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor {
			ui.DisableColors()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "./spotify.db", "Path to SQLite database file")
	rootCmd.PersistentFlags().StringVarP(&globalYear, "year", "y", "2026", "Year filter ('2026', '2025', '2024', or 'all')")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output results in structured JSON (agent-friendly)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable ANSI color formatting")
}

func resolveDBPath() string {
	if dbPath != "" && dbPath != "./spotify.db" {
		return dbPath
	}
	if env := os.Getenv("SPOTIFY_DB"); env != "" {
		return env
	}
	if _, err := os.Stat("./spotify.db"); err == nil {
		return "./spotify.db"
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		appDB := filepath.Join(configDir, "spotify-history-cli", "spotify.db")
		if _, err := os.Stat(appDB); err == nil {
			return appDB
		}
	}
	return dbPath
}

func getDB() (*db.Database, error) {
	targetPath := resolveDBPath()

	// Check if database file exists on disk before attempting to connect
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("spotify database not found at '%s'.\nHint: Run `spotify-history-cli ingest <path-to-spotify-data>` first to build your database", targetPath)
	}

	database, err := db.Open(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database at '%s': %w", targetPath, err)
	}

	// Verify the database has the required schema
	var tableCount int
	_ = database.Conn().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='streams'").Scan(&tableCount)
	if tableCount == 0 {
		database.Close()
		return nil, fmt.Errorf("database at '%s' is empty or uninitialized.\nHint: Run `spotify-history-cli ingest <path-to-spotify-data>` to populate your listening history", targetPath)
	}

	return database, nil
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("JSON encoding error: %w", err)
	}
	return nil
}
