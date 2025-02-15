package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/ghdwlsgur/outline-vpn/internal"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/spf13/cobra"
)

type IPRange struct {
	IPNet   *net.IPNet
	Country string
	State   string
	City    string
}

const (
	terraformVersion = "1.7.3"
	icloudCSV        = "https://mask-api.icloud.com/egress-ip-ranges.csv"
)

func verifyPrivateRelay() (bool, error) {
	currentIPv4, err := internal.GetPublicIP()
	if err != nil {
		return false, err
	}

	ipRanges, err := fetchIPRanges(icloudCSV)
	if err != nil {
		return false, err
	}

	ip := net.ParseIP(currentIPv4)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address")
	}

	found := false
	for _, ipRange := range ipRanges {
		if ipRange.IPNet.Contains(ip) {
			found = true
			return found, fmt.Errorf("you need to disable Private Relay")
		}
	}

	return found, nil
}

func fetchIPRanges(url string) ([]IPRange, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ipRanges := make([]IPRange, 0)

	reader := csv.NewReader(resp.Body)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		ipNet, err := parseIPNet(record[0])
		if err != nil {
			log.Println("Failed to parse IP range:", err)
			continue
		}

		ipRange := IPRange{
			IPNet:   ipNet,
			Country: record[1],
			State:   record[2],
			City:    record[3],
		}
		ipRanges = append(ipRanges, ipRange)
	}

	return ipRanges, nil
}

func parseIPNet(cidr string) (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	return ipNet, nil
}

func terraformReady(ctx context.Context, version string) (*root, error) {
	r := &root{}

	r.execPath, err = internal.TerraformReady(ctx, version)
	if err != nil {
		return nil, err
	}
	r.workspace, err = internal.SetRoot(r.execPath, _defaultTerraformPath)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func terraformInit(r *root, ctx context.Context) error {
	if _, err := os.Stat(_defaultTerraformPath + "/.terraform"); err != nil {
		if err = r.workspace.Init(ctx, tfexec.Upgrade(true)); err != nil {
			return fmt.Errorf("failed to terraform init")
		}
		internal.PrintProvisioning("[root]", "terraform init:", "success")
	} else {
		internal.PrintProvisioning("[root]", "terraform init:", "already-done")
	}
	return nil
}

func findInstance(ctx context.Context, r *root) error {
	instance, err = internal.FindSpecificTagInstance(ctx, *_credential.awsConfig, _credential.awsConfig.Region)
	if err != nil {
		return err
	}

	switch instance.Existence {
	case true:
		return fmt.Errorf("⚠️  You already have EC2 %s", _credential.awsConfig.Region)
	case false:
		workSpace, err = internal.SelectWorkspace(ctx,
			r.execPath,
			_defaultTerraformPath,
			_credential.awsConfig.Region,
			workSpace,
		)

		if err != nil {
			return err
		}
		fmt.Printf("%s %s\n", color.HiBlackString("terraform workspace select"), color.HiMagentaString(workSpace.Now))
	}

	return nil
}

type root struct {
	execPath  string
	workspace *tfexec.Terraform
}

var (
	ami          *internal.Ami
	az           *internal.AvailabilityZone
	instanceType *internal.InstanceType
	defaultVpc   *internal.DefaultVpc

	instance *internal.EC2
	err      error

	_terraformVarsJSON = &TerraformVarsJSON{}
	workSpace          = &internal.Workspace{}
)

func decodeTerraformVarsFile() (string, error) {

	buffer, err := os.ReadFile(_defaultTerraformVars)
	if err != nil {
		return "", err
	}
	json.NewDecoder(bytes.NewBuffer(buffer)).Decode(&_terraformVarsJSON)

	answer, err := internal.AskNewTfVars(
		_terraformVarsJSON.AWSRegion,
		_terraformVarsJSON.AvailabilityZone,
		_terraformVarsJSON.InstanceType,
		_terraformVarsJSON.EC2Ami,
	)
	if err != nil {
		return "", err
	}
	_credential.awsConfig.Region = _terraformVarsJSON.AWSRegion

	return strings.Split(answer, ",")[0], nil
}

func isExistDefaultSubnet(ctx context.Context) error {
	defaultSubnet, err := internal.ExistsDefaultSubnet(ctx, *_credential.awsConfig, _terraformVarsJSON.AvailabilityZone)
	if err != nil {
		return err
	}

	if !defaultSubnet.Existence {
		answer, err := internal.AskCreateDefaultSubnet()
		if err != nil {
			return err
		}

		if answer == "Yes" {
			_, err = internal.CreateDefaultSubnet(ctx, *_credential.awsConfig, _terraformVarsJSON.AvailabilityZone)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid default subnet")
		}
	}
	return nil
}

func isExistDefaultVpc(ctx context.Context) error {
	defaultVpc, err = internal.ExistsDefaultVpc(ctx, *_credential.awsConfig)
	if err != nil {
		panicRed(err)
	}

	if !defaultVpc.Existence {
		answer, err := internal.AskCreateDefaultVpc()
		if err != nil {
			return err
		}

		if answer == "Yes" {
			vpc, err := internal.CreateDefaultVpc(ctx, *_credential.awsConfig)
			if err != nil {
				return err
			}
			internal.PrintReady("[create-vpc]", _credential.awsConfig.Region, "vpc-id", vpc.Id)
		} else {
			os.Exit(1)
		}
	}
	return nil
}

func inputRegion(ctx context.Context) error {
	if _credential.awsConfig.Region == "" {
		region, err := internal.AskRegion(ctx, *_credential.awsConfig)
		if err != nil {
			return err
		}
		_credential.awsConfig.Region = region.Name
	}
	_terraformVarsJSON.AWSRegion = _credential.awsConfig.Region
	return nil
}

func inputAvailabilityZone(ctx context.Context) error {
	if az == nil {
		az, err := internal.AskAvailabilityZone(ctx, *_credential.awsConfig)
		if err != nil {
			return err
		}
		_terraformVarsJSON.AvailabilityZone = az.Name
	}
	return nil
}

func inputAmi(ctx context.Context) error {
	if ami == nil {
		ami, err = internal.AskAmi(ctx, *_credential.awsConfig)
		if err != nil {
			return err
		}
		_terraformVarsJSON.EC2Ami = ami.Name
	}
	return nil
}

func inputInstanceType(ctx context.Context) error {
	if instanceType == nil {
		instanceType, err := internal.AskInstanceType(ctx, *_credential.awsConfig, _terraformVarsJSON.AvailabilityZone)
		if err != nil {
			return err
		}
		_terraformVarsJSON.InstanceType = instanceType.Name
	}
	return nil
}

func inputTerraformVariable(ctx context.Context) error {

	err := inputRegion(ctx)
	if err != nil {
		return fmt.Errorf("inputRegion function : %s", err)
	}

	err = isExistDefaultVpc(ctx)
	if err != nil {
		return fmt.Errorf("isExistDefaultVpc function : %s", err)
	}

	err = inputAvailabilityZone(ctx)
	if err != nil {
		return fmt.Errorf("inputAvailabilityZone function: %s", err)
	}

	err = isExistDefaultSubnet(ctx)
	if err != nil {
		return fmt.Errorf("isExistDefaultSubnet function: %s", err)
	}

	err = inputAmi(ctx)
	if err != nil {
		return fmt.Errorf("inputAmi function: %s", err)
	}

	err = inputInstanceType(ctx)
	if err != nil {
		return fmt.Errorf("inputInstanceType function: %s", err)
	}

	// save tfvars ===============================
	jsonData := make(map[string]interface{})
	jsonData["aws_region"] = _terraformVarsJSON.AWSRegion
	jsonData["ec2_ami"] = _terraformVarsJSON.EC2Ami
	jsonData["instance_type"] = _terraformVarsJSON.InstanceType
	jsonData["availability_zone"] = _terraformVarsJSON.AvailabilityZone

	_, err = internal.SaveTerraformVariable(jsonData, _defaultTerraformVars)
	if err != nil {
		return err
	}

	internal.PrintReady("[variable]", _credential.awsConfig.Region, "availability-zone", _terraformVarsJSON.AvailabilityZone)
	internal.PrintReady("[variable]", _credential.awsConfig.Region, "image-id", _terraformVarsJSON.EC2Ami)
	internal.PrintReady("[variable]", _credential.awsConfig.Region, "instance-type", _terraformVarsJSON.InstanceType)

	return nil
}

var (
	applyCommand = &cobra.Command{
		Use:   "apply",
		Short: "Create an instance that can be used as an outline VPN server and all its resources.",
		Long:  "Create an instance that can be used as an outline VPN server and all its resources.",
		Run: func(_ *cobra.Command, _ []string) {
			ctx := context.Background()

			s := spinner.New(spinner.CharSets[17], 100*time.Millisecond)
			s.UpdateCharSet(spinner.CharSets[17])
			s.Color("fgHiCyan")
			s.Restart()
			s.Prefix = color.HiCyanString("Checking the status of Private Relay usage ")

			usePrivateRelay, err := verifyPrivateRelay()
			if usePrivateRelay {
				fmt.Println()
				panicRed(err)
			}
			s.Stop()

			if _, err := os.Stat(_defaultTerraformVars); err == nil {
				answer, err := decodeTerraformVarsFile()
				if err != nil {
					panicRed(err)
				}
				if answer == "No" {
					fmt.Println(color.HiGreenString("The following region list represents the Active regions in my current AWS account."))
					askRegion, err := internal.AskRegion(ctx, *_credential.awsConfig)
					if err != nil {
						panicRed(err)
					}
					_credential.awsConfig.Region = askRegion.Name

					err = isExistDefaultVpc(ctx)
					if err != nil {
						panicRed(err)
					}

					err = inputTerraformVariable(ctx)
					if err != nil {
						panicRed(err)
					}
				}
			} else {
				fmt.Println(color.HiGreenString("The following region list represents the Active regions in my current AWS account."))
				askRegion, err := internal.AskRegion(ctx, *_credential.awsConfig)
				if err != nil {
					panicRed(err)
				}
				_credential.awsConfig.Region = askRegion.Name

				err = inputTerraformVariable(ctx)
				if err != nil {
					panicRed(err)
				}
			}

			if _credential.awsConfig.Region != _terraformVarsJSON.AWSRegion {
				panicRed(err)
			}

			err = isExistDefaultVpc(ctx)
			if err != nil {
				panicRed(err)
			}

			// terraform ready [root] =============================================
			r, err := terraformReady(ctx, terraformVersion)
			if err != nil {
				panicRed(err)
			}
			internal.PrintProvisioning("[root]", "terraform-state:", "ready")

			// terraform init [root] =============================================
			err = terraformInit(r, ctx)
			if err != nil {
				panicRed(err)
			}

			workSpace, err = internal.ExistsWorkspace(ctx, r.execPath, _defaultTerraformPath, _credential.awsConfig.Region)
			if err != nil {
				panicRed(err)
			}
			if workSpace.Existence {
				err = findInstance(ctx, r)
				if err != nil {
					panicRed(err)
				}
			} else {

				if err = internal.CreateWorkspace(ctx,
					r.execPath, _defaultTerraformPath, _credential.awsConfig.Region); err != nil {
					panicRed(err)
				}
				workSpace.Now = _credential.awsConfig.Region
				fmt.Printf("%s %s\n", color.HiBlackString("terraform workspace new"), color.HiMagentaString(workSpace.Now))

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
			internal.PrintProvisioning("[workspace]", "terraform-state:", "ready")

			// terraform init [workspace] =============================================
			if err = workSpaceTf.Init(ctx, tfexec.Upgrade(true)); err != nil {
				panicRed(fmt.Errorf("failed to terraform init %s", err))
			}
			internal.PrintProvisioning("[workspace]", "terraform-init:", "success")

			// terraform plan [workspace] =============================================
			if _, err = workSpaceTf.Plan(ctx, tfexec.VarFile(_defaultTerraformVars)); err != nil {
				panicRed(fmt.Errorf("failed to terraform plan"))
			}
			internal.PrintProvisioning("[workspace]", "terraform-plan:", "success")

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
				congratulation("🎉 Provisioning Complete! 🎉\n")
				result := fmt.Sprintf("accessKey: %v\n", state.Values.Outputs["access_key"].Value)
				congratulation(result)

				apiURL, err := internal.GetApiURL(_credential.awsConfig.Region)
				if err != nil {
					panicRed(err)
				}
				congratulation("apiURL: " + apiURL + "\n")

				certSha256, err := internal.GetCertSha256(_credential.awsConfig.Region)
				if err != nil {
					panicRed(err)
				}
				congratulation("certSha256: " + certSha256 + "\n")

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

func init() {
	rootCmd.AddCommand(applyCommand)
}
