package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/ghdwlsgur/govpn/internal"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/spf13/cobra"
)

const (
	terraformVersion = "1.2.9"
)

var (
	ami           *internal.Ami
	az            *internal.AvailabilityZone
	ec2Type       *internal.InstanceType
	defaultVpc    *internal.DefaultVpc
	defaultSubnet *internal.DefaultSubnet

	instance *internal.EC2
	err      error

	_terraformVarsJSON = &TerraformVarsJSON{}
	workSpace          = &internal.Workspace{}
)

var (
	applyCommand = &cobra.Command{
		Use:   "apply",
		Short: "Create an instance that can be used as an outline VPN server and all its resources.",
		Long:  "Create an instance that can be used as an outline VPN server and all its resources.",
		Run: func(_ *cobra.Command, _ []string) {
			ctx := context.Background()

			if _, err := os.Stat(_defaultTerraformVars); err == nil {

				buffer, err := os.ReadFile(_defaultTerraformVars)
				if err != nil {
					panicRed(err)
				}
				json.NewDecoder(bytes.NewBuffer(buffer)).Decode(&_terraformVarsJSON)

				answer, err := internal.AskNewTfVars(_terraformVarsJSON.AWSRegion, _terraformVarsJSON.AvailabilityZone, _terraformVarsJSON.InstanceType, _terraformVarsJSON.EC2Ami)
				if err != nil {
					panicRed(err)
				}
				_credential.awsConfig.Region = _terraformVarsJSON.AWSRegion

				if strings.Split(answer, ",")[0] == "No" {
					askRegion, err := internal.AskRegion(ctx, *_credential.awsConfig)
					if err != nil {
						panicRed(err)
					}
					_credential.awsConfig.Region = askRegion.Name

					defaultVpc, err = internal.ExistsDefaultVpc(ctx, *_credential.awsConfig)
					if err != nil {
						panicRed(err)
					}

					if !defaultVpc.Existence {
						answer, err := internal.AskCreateDefaultVpc()
						if err != nil {
							panicRed(err)
						}

						if answer == "Yes" {
							vpc, err := internal.CreateDefaultVpc(ctx, *_credential.awsConfig)
							if err != nil {
								panicRed(err)
							}
							internal.PrintReady("[create-vpc]", _credential.awsConfig.Region, "vpc-id", vpc.Id)
						} else {
							os.Exit(1)
						}
					}

					scanVariable(ctx)
				}
			} else {
				scanVariable(ctx)
			}

			if _credential.awsConfig.Region != _terraformVarsJSON.AWSRegion {
				panicRed(err)
			}

			defaultVpc, err = internal.ExistsDefaultVpc(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}

			if !defaultVpc.Existence {
				answer, err := internal.AskCreateDefaultVpc()
				if err != nil {
					panicRed(err)
				}

				if answer == "Yes" {
					vpc, err := internal.CreateDefaultVpc(ctx, *_credential.awsConfig)
					if err != nil {
						panicRed(err)
					}
					internal.PrintReady("[create-vpc]", _credential.awsConfig.Region, "vpc-id", vpc.Id)
				} else {
					os.Exit(1)
				}
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
			internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "[root] terraform-state", "ready")

			// terraform init [root] =============================================
			if _, err := os.Stat(_defaultTerraformPath + "/.terraform"); err != nil {
				if err = rootTf.Init(ctx, tfexec.Upgrade(true)); err != nil {
					panicRed(fmt.Errorf("failed to terraform init"))
				}
				internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "[root] terraform init", "success")
			} else {
				internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "[root] terraform init", "already-done")
			}

			workSpace, err = internal.ExistsWorkspace(ctx, rootExecPath, _defaultTerraformPath, _credential.awsConfig.Region)
			if err != nil {
				panicRed(err)
			}

			if workSpace.Existence {
				instance, err = internal.FindSpecificTagInstance(ctx, *_credential.awsConfig, _credential.awsConfig.Region)
				if err != nil {
					panicRed(err)
				}

				if instance.Existence {
					panicRed(fmt.Errorf("⚠️  You already have EC2 %s", _credential.awsConfig.Region))
				} else {
					workSpace, err = internal.SelectWorkspace(ctx, rootExecPath, _defaultTerraformPath, _credential.awsConfig.Region, workSpace)
					if err != nil {
						panicRed(err)
					}
					fmt.Printf("%s %s\n", color.HiCyanString("[terraform-workspace-select]"), color.HiCyanString(workSpace.Now))
				}
			} else {
				if err = internal.CreateWorkspace(ctx,
					rootExecPath, _defaultTerraformPath, _credential.awsConfig.Region); err != nil {
					panicRed(err)
				}
				workSpace.Now = _credential.awsConfig.Region
				fmt.Printf("%s %s\n", color.HiCyanString("[terraform-workspace-new]"), color.HiCyanString(workSpace.Now))
			}

			// create tf file [ main.tf / key.tf / output.tf / provider.tf ]
			workSpace.Path = _defaultTerraformPath + "/terraform.tfstate.d/" + _credential.awsConfig.Region
			err = internal.CreateTf(workSpace.Path, _terraformVarsJSON.AWSRegion, _terraformVarsJSON.EC2Ami, _terraformVarsJSON.InstanceType, _terraformVarsJSON.AvailabilityZone)
			if err != nil {
				panicRed(err)
			}

			// terraform ready [workspace] =============================================
			workSpaceExecPath, err := internal.TerraformReady(ctx, terraformVersion)
			if err != nil {
				panicRed(err)
			}
			workSpaceTf, err := internal.SetRoot(workSpaceExecPath, workSpace.Path)
			if err != nil {
				panicRed(err)
			}
			internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "[workspace] terraform-state", "ready")

			// terraform init [workspace] =============================================
			if err = workSpaceTf.Init(ctx, tfexec.Upgrade(true)); err != nil {
				panicRed(fmt.Errorf("failed to terraform init"))
			}
			internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "[workspace] terraform init", "success")

			// terraform plan [workspace] =============================================
			if _, err = workSpaceTf.Plan(ctx, tfexec.VarFile(_defaultTerraformVars)); err != nil {
				panicRed(fmt.Errorf("failed to terraform plan"))
			}

			internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "[workspace] terraform plan", "success")

			answer, err := internal.AskTerraformExecution("Do You Provision EC2 Instance:")
			if err != nil {
				panicRed(err)
			}

			if answer == "Yes" {
				s := spinner.New(spinner.CharSets[8], 100*time.Millisecond)
				s.UpdateCharSet(spinner.CharSets[59])
				s.Color("fgHiGreen")
				s.Restart()
				s.Prefix = color.HiGreenString("EC2 Creating ")

				// terraform apply [workspace] =============================================
				err = workSpaceTf.Apply(ctx)
				if err != nil {
					panicRed(fmt.Errorf("failed to terraform apply"))
				}

				ctx, cancel := context.WithTimeout(ctx, time.Minute)
				defer cancel()

				// terraform show [workspace] =============================================
				state, err := workSpaceTf.Show(ctx)
				if err != nil {
					panicRed(err)
				}

				s.Stop()
				congratulation := color.New(color.Bold, color.FgHiGreen).PrintFunc()
				congratulation("🎉 Provisioning Complete! 🎉\n")
				congratulation(state.Values.Outputs["access_key"].Value)

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
		},
	}
)

func scanVariable(ctx context.Context) error {

	if _credential.awsConfig.Region == "" {
		askRegion, err := internal.AskRegion(ctx, *_credential.awsConfig)
		if err != nil {
			panicRed(err)
		}
		_credential.awsConfig.Region = askRegion.Name
	}
	_terraformVarsJSON.AWSRegion = _credential.awsConfig.Region

	defaultVpc, err = internal.ExistsDefaultVpc(ctx, *_credential.awsConfig)
	if err != nil {
		panicRed(err)
	}

	if !defaultVpc.Existence {
		answer, err := internal.AskCreateDefaultVpc()
		if err != nil {
			panicRed(err)
		}

		if answer == "Yes" {
			vpc, err := internal.CreateDefaultVpc(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}
			internal.PrintReady("[create-vpc]", _credential.awsConfig.Region, "vpc-id", vpc.Id)
		} else {
			os.Exit(1)
		}
	}

	// user inputs Availability Zone value
	if az == nil {
		az, err = internal.AskAvailabilityZone(ctx, *_credential.awsConfig)
		if err != nil {
			panicRed(err)
		}
		_terraformVarsJSON.AvailabilityZone = az.Name
	}
	internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "availability-zone", az.Name)

	// If the user hasn't default subnet in availability zone, ask whether create or not
	defaultSubnet, err = internal.ExistsDefaultSubnet(ctx, *_credential.awsConfig, az.Name)
	if err != nil {
		panicRed(fmt.Errorf(`
		⚠️  [privacy] Direct permission modification is required.
		1. Aws Console -> IAM -> Account Settings
		2. Click Activate for the region where you want to create the default VPC.
				`))
	}

	if !defaultSubnet.Existence {
		answer, err := internal.AskCreateDefaultSubnet()
		if err != nil {
			if err != nil {
				panicRed(fmt.Errorf(`
				⚠️  [privacy] Direct permission modification is required.
				1. Aws Console -> IAM -> Account Settings
				2. Click Activate for the region where you want to create the default VPC.
						`))
			}
		}

		if answer == "Yes" {
			_, err = internal.CreateDefaultSubnet(ctx, *_credential.awsConfig, az.Name)
			if err != nil {
				panicRed(fmt.Errorf(`
				⚠️  [privacy] Direct permission modification is required.
				1. Aws Console -> IAM -> Account Settings
				2. Click Activate for the region where you want to create the default VPC.
						`))
			}
		} else {
			panicRed(fmt.Errorf("invalid default subnet"))
		}
	}

	// user inputs Instance Type value
	if ec2Type == nil {
		ec2Type, err = internal.AskInstanceType(ctx, *_credential.awsConfig, az.Name)
		if err != nil {
			panicRed(err)
		}
		_terraformVarsJSON.InstanceType = ec2Type.Name
	}
	internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "instance-type", ec2Type.Name)

	// user inputs Amazon Machine Image
	if ami == nil {
		ami, err = internal.AskAmi(ctx, *_credential.awsConfig)
		if err != nil {
			panicRed(err)
		}
		_terraformVarsJSON.EC2Ami = ami.Name
	}
	internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "ami-id", ami.Name)

	// save tfvars ===============================
	jsonData := make(map[string]interface{})
	jsonData["aws_region"] = _terraformVarsJSON.AWSRegion
	jsonData["eC2ami"] = _terraformVarsJSON.EC2Ami
	jsonData["instancetype"] = _terraformVarsJSON.InstanceType
	jsonData["availabilityzone"] = _terraformVarsJSON.AvailabilityZone

	_, err = internal.SaveTerraformVariable(jsonData, _defaultTerraformVars)
	if err != nil {
		panicRed(err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(applyCommand)
}
