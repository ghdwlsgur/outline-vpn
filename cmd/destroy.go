package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/ghdwlsgur/govpn/internal"
	"github.com/spf13/cobra"
)

func deleteTagSubnet(ctx context.Context) error {
	tagSubnet, err := internal.ExistsTagSubnet(ctx, *_credential.awsConfig)
	if err != nil {
		return err
	}

	if tagSubnet.Existence {
		answer, err := internal.AskDeleteTagSubnet()
		if err != nil {
			return err
		}

		if answer == "Yes" {
			_, err := internal.DeleteTagSubnet(ctx, *_credential.awsConfig, tagSubnet.ID)
			if err != nil {
				return err
			}
			congratulation("🎉 Delete Subnet Complete! 🎉\n")
		}
	}
	return nil
}

func deleteTagVPC(ctx context.Context) error {
	tagVpc, err := internal.ExistsTagVpc(ctx, *_credential.awsConfig)
	if err != nil {
		return err
	}

	if tagVpc.Existence {
		answer, err := internal.AskDeleteTagVpc()
		if err != nil {
			return err
		}

		if answer == "Yes" {
			_, err := internal.DeleteTagVpc(ctx, *_credential.awsConfig, tagVpc.Id)
			if err != nil {
				return err
			}
			congratulation("🎉 Delete VPC Complete! 🎉\n")
		}
	}
	return nil
}

var (
	destroyCommand = &cobra.Command{
		Use:   "destroy",
		Short: "Delete the EC2 instance you created as the outline VPN server and all resources associated with it.",
		Long:  "Delete the EC2 instance you created as the outline VPN server and all resources associated with it.",
		Run: func(_ *cobra.Command, _ []string) {
			var (
				instance *internal.EC2
			)

			ctx := context.Background()

			instance, err = internal.FindSpecificTagInstance(ctx, *_credential.awsConfig, _credential.awsConfig.Region)
			if err != nil {
				panicRed(err)
			}

			if instance.Existence {
				vpnConnect, err := internal.CheckOutlineConnect(instance)
				if err != nil {
					panicRed(err)
				}

				if vpnConnect {
					panicRed(fmt.Errorf(`⚠️  Please Disconnect Outline VPN and Try Again`))
				}

				answer, err := internal.AskTerraformExecution("Do You Execute Terraform Destroy:")
				if err != nil {
					panicRed(err)
				}

				if answer == "Yes" {
					s := spinner.New(spinner.CharSets[8], 100*time.Millisecond)
					s.UpdateCharSet(spinner.CharSets[59])
					s.Color("fgHiRed")
					s.Restart()
					s.Prefix = color.HiRedString("EC2 Destroying ")

					workSpace.Path = _defaultTerraformPath + "/terraform.tfstate.d/" + _credential.awsConfig.Region

					// terraform ready [workspace] =============================================
					workSpaceExecPath, err := internal.TerraformReady(ctx, terraformVersion)
					if err != nil {
						panicRed(err)
					}

					workSpaceTf, err := internal.SetRoot(workSpaceExecPath, workSpace.Path)
					if err != nil {
						panicRed(err)
					}

					err = workSpaceTf.Destroy(ctx)
					if err != nil {
						panicRed(err)
					}

					// terraform ready [root] =============================================
					rootExecPath, err := internal.TerraformReady(ctx, terraformVersion)
					if err != nil {
						panicRed(err)
					}

					rootTf, err := internal.SetRoot(rootExecPath, _defaultTerraformPath)
					if err != nil {
						panicRed(err)
					}

					// terraform workspace select [root] =============================================
					err = rootTf.WorkspaceSelect(ctx, "default")
					if err != nil {
						panicRed(err)
					}

					// terraform workspace delete [root] =============================================
					err = rootTf.WorkspaceDelete(ctx, _credential.awsConfig.Region)
					if err != nil {
						panicRed(err)
					}

					ctx, cancel := context.WithTimeout(ctx, time.Minute)
					defer cancel()

					s.Stop()
					congratulation("🎉 Delete EC2 Instance Complete! 🎉\n")

					go func() {
						cancel()
					}()

				delay:
					for {
						select {
						case <-time.After(time.Second):
						case <-ctx.Done():
							break delay
						}
					}

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

			err := deleteTagSubnet(ctx)
			if err != nil {
				panicRed(err)
			}

			err = deleteTagVPC(ctx)
			if err != nil {
				panicRed(err)
			}

		},
	}
)

func init() {
	rootCmd.AddCommand(destroyCommand)
}
