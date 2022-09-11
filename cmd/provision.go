package cmd

import (
	"context"
	"fmt"
	"govpn/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startProvisionCommand = &cobra.Command{
		Use:   "start",
		Short: "Exec `start-session`",
		Long:  "Exec `start-session`",
		Run: func(_ *cobra.Command, _ []string) {
			var (
				ami        *internal.Ami
				az         *internal.AvailabilityZone
				ec2Type    *internal.InstanceType
				defaultVpc *internal.DefaultVpc
				err        error
			)
			ctx := context.Background()

			defaultVpc, err = internal.DefaultVpcExists(ctx, *_credential.awsConfig)
			if err != nil {
				// defaultVpc.Existence false -> survey
				panicRed(err)
			}

			check, err := internal.CreateDefaultVpc(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}
			fmt.Println(check)

			fmt.Println(defaultVpc.Existence)

			if az == nil {
				az, err = internal.AskAvailabilityZone(ctx, *_credential.awsConfig)
				if err != nil {
					panicRed(err)
				}
			}
			internal.PrintReady("start-provisioning", _credential.awsConfig.Region, az.Name)

			if ec2Type == nil {
				ec2Type, err = internal.AskInstanceType(ctx, *_credential.awsConfig, az.Name)
				if err != nil {
					panicRed(err)
				}
			}

			internal.PrintReady("start-provisioning", _credential.awsConfig.Region, ec2Type.Name)

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
