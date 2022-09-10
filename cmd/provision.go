package cmd

import (
	"context"
	"govpn/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startProvisionCommand = &cobra.Command{
		Use:   "start",
		Short: "Exec `start-session`",
		Long:  "Exec `start-session`",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				ami *internal.Ami
				err error
			)
			ctx := context.Background()

			if ami == nil {

				ami, err = internal.AskAmi(ctx, *_credential.awsConfig)
				if err != nil {
					panicRed(err)
				}
			}
			internal.PrintReady("start-provisioning", _credential.awsConfig.Region, ami.Name)

		},
	}
)

func init() {
	startProvisionCommand.Flags().StringP("ami", "a", "", "")
	viper.BindPFlag("start-session-target", startProvisionCommand.Flags().Lookup("ami"))

	rootCmd.AddCommand(startProvisionCommand)
}
