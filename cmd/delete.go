package cmd

import (
	"context"
	"govpn/internal"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	deleteProvisionCommand = &cobra.Command{
		Use:   "delete",
		Short: "Exec test",
		Long:  "Exec test",
		Run: func(_ *cobra.Command, _ []string) {

			ctx := context.Background()

			notice := color.New(color.Bold, color.FgHiRed).PrintfFunc()
			congratulation := color.New(color.Bold, color.FgHiGreen).PrintFunc()

			existServer, err := internal.FindTagEc2(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}

			if existServer {
				answer, err := internal.AskTerraformDestroy()
				if err != nil {
					panicRed(err)
				}
				if answer == "Yes" {
					if err = internal.TerraformDestroy(_defaultTerraformPath); err != nil {
						panicRed(err)
					}
					congratulation("🎉 Delete EC2 Instance Complete! 🎉")
				}
			} else {
				notice("You haven't EC2 [govpn-EC2-%s]\n", _credential.awsConfig.Region)
			}

			if internal.ExistsKeyPair() {
				err := internal.DeleteKeyPair()
				if err != nil {
					panicRed(err)
				}
			}

			tagVpc, err := internal.TagVpcExists(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}

			if tagVpc.Existence {
				answer, err := internal.AskDeleteTagVpc()
				if err != nil {
					panicRed(err)
				}
				if answer == "Yes" {
					_, err := internal.DeleteTagVpc(ctx, *_credential.awsConfig, tagVpc.Id)
					if err != nil {
						panicRed(err)
					}
					congratulation("🎉 Delete VPC Complete! 🎉")
				}
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(deleteProvisionCommand)
}
