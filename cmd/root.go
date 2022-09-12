package cmd

import (
	"context"
	"fmt"
	"govpn/internal"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	_defaultProfile = "default"
	// _defaultGitUrl        = "https://github.com/ghdwlsgur/terraform"
)

var (
	_defaultTerraformPath string
	_defaultTerraformVars string

	rootCmd = &cobra.Command{
		Use:   "govpn",
		Short: `govpn is interactive CLI tool`,
		Long:  `govpn is interactive CLI tool`,
	}

	_credential              *Credential
	_credentialWithMFA       = fmt.Sprintf("%s_mfa", config.DefaultSharedConfigFilename())
	_credentialWithTemporary = fmt.Sprintf("%s_temporary", config.DefaultSharedCredentialsFilename())
)

type TerraformVarsJson struct {
	Aws_Region        string
	Ec2_Ami           string
	Instance_Type     string
	Availability_Zone string
}

type Credential struct {
	awsProfile string
	awsConfig  *aws.Config
}

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		panicRed(err)
	}
}

func panicRed(err error) {
	fmt.Println(color.RedString("[err] %s", err.Error()))
	os.Exit(1)
}

func initConfig() {
	path, _ := os.Getwd()
	_defaultTerraformPath = path + "/terraform-vpn-server"
	_defaultTerraformVars = path + "/terraform-vpn-server/terraform.tfvars.json"

	// git clone https://github.com/ghdwlsgur/terraform-vpn-server
	// if _, err := os.Stat(_defaultTerraformPath); errors.Is(err, os.ErrNotExist) {
	// 	// repo-folder (terraform-vpn-server) does not exist
	// 	_, err := git.PlainClone(_defaultTerraformPath, false, &git.CloneOptions{
	// 		URL:      _defaultGitUrl,
	// 		Progress: os.Stdout,
	// 	})
	// 	if err != nil {
	// 		panicRed(err)
	// 	}
	// } else {
	// 	// repo-folder (terraform-vpn-server) exists
	// 	repository, err := git.PlainOpen(_defaultTerraformPath)
	// 	if err != nil {
	// 		panicRed(err)
	// 	}
	// 	worktree, err := repository.Worktree()
	// 	if err != nil {
	// 		panicRed(err)
	// 	}
	// 	err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
	// 	if err != nil {
	// 		fmt.Println(color.GreenString("terraform-vpn-server (%s)", err.Error()))
	// 	}
	// }

	// if _, err := os.Stat(_defaultTerraformVars); errors.Is(err, os.ErrNotExist) {
	// 	// terraform-vpn-server/terraform.tfvars.json does not exist

	// 	_terraformVarsJson = &TerraformVarsJson{"us-east-1", "ami-0cff7528ff583bf9a", "t2.micro", "us-east-1a"}

	// 	jsonData := make(map[string]interface{})
	// 	jsonData["aws_region"] = _terraformVarsJson.Aws_Region
	// 	jsonData["ec2_ami"] = _terraformVarsJson.Ec2_Ami
	// 	jsonData["instance_type"] = _terraformVarsJson.Instance_Type
	// 	jsonData["availability_zone"] = _terraformVarsJson.Availability_Zone

	// 	varsJson, _ := json.Marshal(jsonData)
	// 	err := os.WriteFile(_defaultTerraformVars, varsJson, os.FileMode(0644))
	// 	if err != nil {
	// 		panicRed(err)
	// 	}
	// } else {
	// 	// terraform-vpn-server/terraform.tfvars.json exists
	// 	buffer, err := os.ReadFile(_defaultTerraformVars)
	// 	if err != nil {
	// 		panicRed(err)
	// 	}
	// 	_terraformVarsJson = &TerraformVarsJson{}
	// 	json.NewDecoder(bytes.NewBuffer(buffer)).Decode(&_terraformVarsJson)
	// }

	/*=======================================================

		Copyright © 2020 gjbae1212
		Released under the MIT license.
		(https://github.com/gjbae1212/gossm)

	=======================================================*/
	_credential = &Credential{}

	awsProfile := viper.GetString("profile")
	if awsProfile == "" {
		if os.Getenv("AWS_PROFILE") != "" {
			awsProfile = os.Getenv("AWS_PROFILE")
		} else {
			awsProfile = _defaultProfile
		}
	}
	_credential.awsProfile = awsProfile

	awsRegion := viper.GetString("region")

	sharedCredFile := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if sharedCredFile == "" {
		if _, err := os.Stat(_credentialWithMFA); !os.IsNotExist(err) {
			color.Yellow("[Use] gossm default mfa credential file %s", _credentialWithMFA)
			os.Setenv("AWS_SHARED_CREDENTIALS_FILE", _credentialWithMFA)
			sharedCredFile = _credentialWithMFA
		}
	}

	if sharedCredFile != "" {
		awsConfig, err := internal.NewSharedConfig(context.Background(),
			_credential.awsProfile,
			[]string{config.DefaultSharedConfigFilename()},
			[]string{sharedCredFile},
		)
		if err != nil {
			panicRed(internal.WrapError(err))
		}

		cred, err := awsConfig.Credentials.Retrieve(context.Background())

		if err != nil || cred.Expired() || cred.AccessKeyID == "" || cred.SecretAccessKey == "" {
			color.Yellow("[Expire] govpn default mfa credential file %s", sharedCredFile)
			os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
		} else {
			_credential.awsConfig = &awsConfig
		}
	}

	args := os.Args[1:]
	subcmd, _, err := rootCmd.Find(args)
	if err != nil {
		panicRed(internal.WrapError(err))
	}

	switch subcmd.Use {
	case "mfa":
		if _credential.awsConfig != nil {
			cred, err := _credential.awsConfig.Credentials.Retrieve(context.Background())
			if err != nil {
				panicRed(internal.WrapError(err))
			}

			if cred.SessionToken != "" {
				os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
				_credential.awsConfig = nil
			}
		}
	}

	if _credential.awsConfig == nil {
		var temporaryCredentials aws.Credentials
		var temporaryConfig aws.Config

		if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
			temporaryConfig, err = internal.NewConfig(context.Background(),
				os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"),
				os.Getenv("AWS_SESSION_TOKEN"), awsRegion, os.Getenv("AWS_ROLE_ARN"))
			if err != nil {
				panicRed(internal.WrapError(err))
			}

			temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
			if err != nil || temporaryCredentials.Expired() ||
				temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" ||
				(subcmd.Use == "mfa" && temporaryCredentials.SessionToken != "") {
				panicRed(internal.WrapError(fmt.Errorf("[err] invalid global environments %s", err.Error())))
			}
		} else {
			temporaryConfig, err = internal.NewSharedConfig(context.Background(), _credential.awsProfile,
				[]string{config.DefaultSharedConfigFilename()}, []string{})
			if err == nil {
				temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
			}

			if err != nil || temporaryCredentials.Expired() ||
				temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" ||
				(subcmd.Use == "mfa" && temporaryCredentials.SessionToken != "") {

				temporaryConfig, err = internal.NewSharedConfig(context.Background(), _credential.awsProfile,
					[]string{config.DefaultSharedConfigFilename()}, []string{config.DefaultSharedCredentialsFilename()})

				if err != nil {
					panicRed(internal.WrapError(err))
				}

				temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
				if err != nil {
					panicRed(internal.WrapError(err))
				}
				if temporaryCredentials.Expired() || temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" {
					panicRed(internal.WrapError(fmt.Errorf("[err] not found credentials")))
				}

				if awsRegion == "" {
					awsRegion = temporaryConfig.Region
				}
			}
		}

		temporaryCredentialsString := fmt.Sprintf(mfaCredentialFormat, _credential.awsProfile, temporaryCredentials.AccessKeyID,
			temporaryCredentials.SecretAccessKey, temporaryCredentials.SessionToken)
		if err := os.WriteFile(_credentialWithTemporary, []byte(temporaryCredentialsString), 0600); err != nil {
			panicRed(internal.WrapError(err))
		}

		os.Setenv("AWS_SHARED_CREDENTIALS_FILE", _credentialWithTemporary)
		awsConfig, err := internal.NewSharedConfig(context.Background(),
			_credential.awsProfile, []string{}, []string{_credentialWithTemporary},
		)
		if err != nil {
			panicRed(internal.WrapError(err))
		}
		_credential.awsConfig = &awsConfig
	}

	if awsRegion != "" {
		_credential.awsConfig.Region = awsRegion
	}
	if _credential.awsConfig.Region == "" {
		askRegion, err := internal.AskRegion(context.Background(), *_credential.awsConfig)
		if err != nil {
			panicRed(internal.WrapError(err))
		}
		_credential.awsConfig.Region = askRegion.Name
	}
	color.Green("region (%s)", _credential.awsConfig.Region)

}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("profile", "p", "", `[optional] if you having multiple aws profiles, it is one of profiles (default is AWS_PROFILE environment variable or default)`)
	rootCmd.PersistentFlags().StringP("region", "r", "", `[optional] it is region in AWS would like to do something`)

	rootCmd.InitDefaultVersionFlag()

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}
