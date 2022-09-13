package cmd

import (
	"context"
	"govpn/internal"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	deleteCommand = &cobra.Command{
		Use:   "delete",
		Short: "Exec test",
		Long:  "Exec test",
		Run: func(_ *cobra.Command, _ []string) {
			var (
				instance *internal.EC2
			)

			ctx := context.Background()

			notice := color.New(color.Bold, color.FgHiRed).PrintfFunc()
			congratulation := color.New(color.Bold, color.FgHiGreen).PrintFunc()

			tagSubnet, err := internal.ExistsTagSubnet(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}
			if tagSubnet.Existence {
				answer, err := internal.AskDeleteTagSubnet()
				if err != nil {
					panicRed(err)
				}

				if answer == "Yes" {
					internal.DeleteTagSubnet(ctx, *_credential.awsConfig, tagSubnet.Id)
				}
			}

			instance, err = internal.FindTagInstance(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}

			if instance.Existence {
				vpnConnect, err := internal.CheckOutlineConnect(instance)
				if err != nil {
					panicRed(err)
				}
				if vpnConnect {
					notice("[err] Please Disconnect Outline VPN and Try Again\n")
					os.Exit(1)
				}

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
				notice("You haven't EC2 [govpn-ec2-%s]\n", _credential.awsConfig.Region)
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
	rootCmd.AddCommand(deleteCommand)
}
