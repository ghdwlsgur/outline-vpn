package cmd

import (
	"context"
	"os"

	"github.com/ghdwlsgur/outline-vpn/internal"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func createAccessURL() error {

	ctx := context.Background()
	list, err := internal.ValidateOutlineJson(ctx, terraformVersion, _defaultTerraformPath)
	if err != nil {
		return err
	}

	answer, err := internal.AskPromptOptionList("Choose a Workspace (Region):", list, 10)
	if err != nil {
		return err
	}

	accessKey, err := internal.CreateAccessKey(answer)
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"ID", "AccessURL", "Password", "Region"})
	t.AppendRow(table.Row{accessKey.ID, accessKey.AccessURL, accessKey.Password, answer})

	t.Render()

	return nil
}

var (
	createCommand = &cobra.Command{
		Use:       "create",
		Short:     "Creating the outline resources",
		Long:      "Creating the outline resources",
		ValidArgs: []string{"accesskey"},
		Args:      cobra.MatchAll(internal.WrapArgsError(cobra.MinimumNArgs(1)), cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(_ *cobra.Command, args []string) {
			var (
				err error
			)

			switch args[0] {
			case "accesskey":
				if createAccessURL(); err != nil {
					panicRed(err)
				}
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(createCommand)
}
