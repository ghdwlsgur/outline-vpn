package cmd

import (
	"context"
	"fmt"

	"github.com/ghdwlsgur/outline-vpn/internal"
	"github.com/spf13/cobra"
)

func deleteAccessURL() error {
	var (
		tableOption = make(map[string]*internal.AccessKey)
	)

	ctx := context.Background()
	list, err := internal.ValidateOutlineJson(ctx, terraformVersion, _defaultTerraformPath)
	if err != nil {
		return err
	}

	region, err := internal.AskPromptOptionList("Choose a Workspace (Region):", list, 10)
	if err != nil {
		return err
	}

	accessKeys, err := internal.GetAccessKeys(region)
	if err != nil {
		return err
	}

	if len(accessKeys.Keys) > 0 {
		for _, v := range accessKeys.Keys {
			tableOption[fmt.Sprintf("ID: %s, (%s)", v.ID, v.AccessURL)] = &internal.AccessKey{
				ID:        v.ID,
				AccessURL: v.AccessURL,
			}
		}

		options := make([]string, 0, len(tableOption))
		for v := range tableOption {
			options = append(options, v)
		}

		answer, err := internal.AskPromptOptionList("Please select the access key you want to delete:", options, 10)
		if err != nil {
			return err
		}

		err = internal.DeleteAccessKey(region, tableOption[answer].ID)
		if err == nil {
			congratulation("Delete Success!\n")
		}

	} else {
		fmt.Println("The access key does not exist")
	}

	return nil
}

var (
	deleteCommand = &cobra.Command{
		Use:       "delete",
		Short:     "Deleting the outline resources",
		Long:      "Deleting the outline resources",
		ValidArgs: []string{"accesskey"},
		Args:      cobra.MatchAll(internal.WrapArgsError(cobra.MinimumNArgs(1)), cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(_ *cobra.Command, args []string) {
			var (
				err error
			)

			switch args[0] {
			case "accesskey":
				if deleteAccessURL(); err != nil {
					panicRed(err)
				}
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(deleteCommand)
}
