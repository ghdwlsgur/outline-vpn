package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ghdwlsgur/outline-vpn/internal"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func getAccessURL() error {

	ctx := context.Background()
	list, err := internal.ValidateOutlineJson(ctx, terraformVersion, _defaultTerraformPath)
	if err != nil {
		return err
	}

	answer, err := internal.AskPromptOptionList("Choose a Workspace (Region):", list, 10)
	if err != nil {
		return err
	}

	accessKeys, err := internal.GetAccessKeys(answer)
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	if len(accessKeys.Keys) > 0 {
		t.AppendHeader(table.Row{"ID", "AccessURL", "Password", "Region"})
		for _, v := range accessKeys.Keys {
			t.AppendRow(table.Row{v.ID, v.AccessURL, v.Password, answer})
		}
	} else {
		fmt.Println("The access key does not exist")
	}

	t.Render()

	return nil
}

var (
	getCommand = &cobra.Command{
		Use:       "get",
		Short:     "Retrieving the outline resources",
		Long:      "Retrieving the outline resources",
		ValidArgs: []string{"accesskey"},
		Args:      cobra.MatchAll(internal.WrapArgsError(cobra.MinimumNArgs(1)), cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(_ *cobra.Command, args []string) {
			var (
				err error
			)

			switch args[0] {
			case "accesskey":
				if getAccessURL(); err != nil {
					panicRed(err)
				}
			}

		},
	}
)

func init() {
	rootCmd.AddCommand(getCommand)
}
