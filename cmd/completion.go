package cmd

import (
	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long:  "Generate a shell completion script for splitdns. Load it into your shell to enable tab completion.",
		Example: `  # Load completions for the current zsh session
  source <(splitdns completion zsh)

  # Install bash completions system-wide
  splitdns completion bash | sudo tee /etc/bash_completion.d/splitdns

  # Generate fish completions
  splitdns completion fish > ~/.config/fish/completions/splitdns.fish`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(env.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(env.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(env.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(env.Stdout)
			}
			return argErrorf("unsupported shell %q", args[0])
		},
	}
	return cmd
}
