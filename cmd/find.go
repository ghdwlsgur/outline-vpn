package cmd

import (
	"context"
	"fmt"

	"github.com/ghdwlsgur/govpn/internal"
	"github.com/spf13/cobra"
)

var (
	findCommand = &cobra.Command{
		Use:   "find",
		Short: "Find instances with the tag [govpn-ec2] in all available regions.",
		Long:  "Find instances with the tag [govpn-ec2] in all available regions.",
		Run: func(_ *cobra.Command, _ []string) {
			ctx := context.Background()

			regionList, err := internal.FindTagInstance(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}

			fmt.Println(regionList)
		},
	}
)

func init() {
	rootCmd.AddCommand(findCommand)
}
