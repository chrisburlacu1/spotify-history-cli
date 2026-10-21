package cmd

import (
	"fmt"
	"strings"

	"github.com/chrisburlacu1/spotify-history-cli/internal/ui"

	"github.com/spf13/cobra"
)

var showSchema bool

var sqlCmd = &cobra.Command{
	Use:     "sql [query]",
	Aliases: []string{"query", "exec"},
	Short:   "Execute arbitrary SQL query against the SQLite database",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := getDB()
		if err != nil {
			return err
		}
		defer database.Close()

		if showSchema {
			cols, data, err := database.ExecuteRawSQL("SELECT type, name, sql FROM sqlite_master WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'streams_fts_%'")
			if err != nil {
				return fmt.Errorf("schema query error: %w", err)
			}
			if jsonOut {
				return printJSON(data)
			}
			fmt.Print(ui.Header("Database Tables & Views"))
			tbl := ui.NewTable(cols...)
			for _, row := range data {
				tbl.AddRow(row[0], row[1], row[2])
			}
			fmt.Println(tbl.Render())
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("please provide a SQL query string (e.g. `spotify-history-cli sql \"SELECT * FROM v_top_artists LIMIT 5\"`)")
		}

		query := strings.Join(args, " ")
		columns, data, err := database.ExecuteRawSQL(query)
		if err != nil {
			return fmt.Errorf("SQL execution failed: %w", err)
		}

		if jsonOut {
			var jsonRows []map[string]interface{}
			for _, row := range data {
				rowMap := make(map[string]interface{})
				for i, col := range columns {
					if i < len(row) {
						rowMap[col] = row[i]
					}
				}
				jsonRows = append(jsonRows, rowMap)
			}
			return printJSON(jsonRows)
		}

		if len(columns) == 0 {
			fmt.Println("Query executed successfully (no result set).")
			return nil
		}

		tbl := ui.NewTable(columns...)
		for _, row := range data {
			rowIface := make([]interface{}, len(row))
			for i, v := range row {
				rowIface[i] = v
			}
			tbl.AddRow(rowIface...)
		}
		fmt.Println(tbl.Render())
		fmt.Printf("\n(%d row(s) returned)\n", len(data))
		return nil
	},
}

func init() {
	sqlCmd.Flags().BoolVar(&showSchema, "schema", false, "Display database schema and views")
	rootCmd.AddCommand(sqlCmd)
}
