package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and build information",
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOut {
			printJSON(map[string]string{
				"version":   Version,
				"commit":    Commit,
				"date":      Date,
				"goVersion": runtime.Version(),
				"os":        runtime.GOOS,
				"arch":      runtime.GOARCH,
			})
			return
		}
		fmt.Printf("spotify-history-cli %s (commit: %s, built: %s, %s/%s)\n",
			Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Version = Version
}
