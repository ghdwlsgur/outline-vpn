package cmd

import (
	"context"
	"govpn/internal"

	"github.com/spf13/cobra"
)

var (
	showCommand = &cobra.Command{
		Use:   "show",
		Short: "Exec test",
		Long:  "Exec test",
		Run: func(_ *cobra.Command, _ []string) {
			ctx := context.Background()

			_, err := internal.FindTagInstanceAllRegion(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}

		},
	}
)

func init() {
	rootCmd.AddCommand(showCommand)
}
