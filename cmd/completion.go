package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script for the specified shell",
	Long: `Generate shell completion script for spotify-history-cli.

To load completions:

Bash:
  $ source <(spotify-history-cli completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ spotify-history-cli completion bash > /etc/bash_completion.d/spotify-history-cli
  # macOS:
  $ spotify-history-cli completion bash > $(brew --prefix)/etc/bash_completion.d/spotify-history-cli

Zsh:
  # If shell completion is not already enabled in your environment:
  autoload -U compinit; compinit
  $ spotify-history-cli completion zsh > "${fpath[1]}/_spotify-history-cli"

Fish:
  $ spotify-history-cli completion fish | source
  # To load completions for each session:
  $ spotify-history-cli completion fish > ~/.config/fish/completions/spotify-history-cli.fish

PowerShell:
  PS> spotify-history-cli completion powershell | Out-String | Invoke-Expression
  # To load completions for every session, add the above line to your $PROFILE.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
