package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"govpn/internal"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	_terraformVarsJson = &TerraformVarsJson{}
	workSpace          = &internal.Workspace{}
)

var (
	startProvisionCommand = &cobra.Command{
		Use:   "start",
		Short: "Exec `start-session`",
		Long:  "Exec `start-session`",
		Run: func(_ *cobra.Command, _ []string) {
			ctx := context.Background()

			defaultVpc, err = internal.DefaultVpcExists(ctx, *_credential.awsConfig)
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

			if _, err := os.Stat(_defaultTerraformVars); err == nil {

				buffer, err := os.ReadFile(_defaultTerraformVars)
				if err != nil {
					panicRed(err)
				}
				json.NewDecoder(bytes.NewBuffer(buffer)).Decode(&_terraformVarsJson)

				answer, err := internal.AskNewTfVars(_terraformVarsJson.Aws_Region, _terraformVarsJson.Availability_Zone, _terraformVarsJson.Instance_Type, _terraformVarsJson.Ec2_Ami)
				if err != nil {
					panicRed(err)
				}

				_credential.awsConfig.Region = _terraformVarsJson.Aws_Region

				if strings.Split(answer, ",")[0] == "No" {

					awsRegion, err := internal.AskRegion(ctx, *_credential.awsConfig)
					if err != nil {
						panicRed(err)
					}

					_credential.awsConfig.Region = awsRegion.Name
					_terraformVarsJson.Aws_Region = awsRegion.Name
					inputTfvars()
				}
			} else {
				inputTfvars()
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
					panicRed(fmt.Errorf("[err] failed to terraform init"))
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
			err = internal.CreateTf(workSpace.Path, _terraformVarsJson.Aws_Region, _terraformVarsJson.Ec2_Ami, _terraformVarsJson.Instance_Type, _terraformVarsJson.Availability_Zone)
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
				panicRed(fmt.Errorf("[err] failed to terraform init"))
			}
			internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "[workspace] terraform init", "success")

			// terraform plan [workspace] =============================================
			if _, err = workSpaceTf.Plan(ctx, tfexec.VarFile(_defaultTerraformVars)); err != nil {
				panicRed(fmt.Errorf("[err] failed to terraform plan"))
			}
			internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "[workspace] terraform plan", "success")

			answer, err := internal.AskTerraformApply()
			if err != nil {
				panicRed(err)
			}

			if answer == "Yes" {

				// terraform apply [workspace] =============================================
				if err = workSpaceTf.Apply(ctx); err != nil {
					panicRed(fmt.Errorf("failed to terraform apply"))
				}

				// terraform show [workspace] =============================================
				state, err := workSpaceTf.Show(ctx)
				if err != nil {
					panicRed(err)
				}

				congratulation := color.New(color.Bold, color.FgHiGreen).PrintFunc()

				congratulation("🎉 Provisioning Complete! 🎉\n")
				congratulation(state.Values.Outputs["access_key"].Value)
			}
		},
	}
)

func inputTfvars() {
	ctx := context.Background()

	// user inputs Availability Zone value
	if az == nil {
		az, err = internal.AskAvailabilityZone(ctx, *_credential.awsConfig)
		if err != nil {
			panicRed(err)
		}
		_terraformVarsJson.Availability_Zone = az.Name
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
		_terraformVarsJson.Instance_Type = ec2Type.Name
	}
	internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "instance-type", ec2Type.Name)

	// user inputs Amazon Machine Image
	if ami == nil {
		ami, err = internal.AskAmi(ctx, *_credential.awsConfig)
		if err != nil {
			panicRed(err)
		}
		_terraformVarsJson.Ec2_Ami = ami.Name
	}
	internal.PrintReady("[start-provisioning]", _credential.awsConfig.Region, "ami-id", ami.Name)

	// save tfvars ===============================
	jsonData := make(map[string]interface{})
	jsonData["aws_region"] = _terraformVarsJson.Aws_Region
	jsonData["ec2_ami"] = _terraformVarsJson.Ec2_Ami
	jsonData["instance_type"] = _terraformVarsJson.Instance_Type
	jsonData["availability_zone"] = _terraformVarsJson.Availability_Zone

	_, err = internal.SaveTerraformVariable(jsonData, _defaultTerraformVars)
	if err != nil {
		panicRed(err)
	}
}

func init() {
	startProvisionCommand.Flags().StringP("ami", "a", "", "")
	viper.BindPFlag("start-session-target", startProvisionCommand.Flags().Lookup("ami"))

	rootCmd.AddCommand(startProvisionCommand)
}
