package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/fatih/color"
	"github.com/ghdwlsgur/outline-vpn/internal"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	git "gopkg.in/src-d/go-git.v4"

	which "github.com/hairyhenderson/go-which"
)

const (
	_defaultProfile = "default"
	_defaultGitURL  = "https://github.com/ghdwlsgur/govpn-terraform"
)

var (
	// /opt/homebrew/lib/outline-vpn
	path = func() string {
		path := which.Which("outline-vpn")
		path = strings.Replace(path, "bin", "lib", -1)
		return path
	}()

	_defaultTerraformPath = func(path, terraformPath string) string {
		return path + terraformPath
	}(path, "/govpn-terraform")

	_defaultTerraformVars = func(path, tfvarsJsonPath string) string {
		return path + tfvarsJsonPath
	}(path, "/govpn-terraform/terraform.tfvars.json")

	rootCmd = &cobra.Command{
		Use:   "outline-vpn",
		Short: `outline-vpn is interactive CLI tool to quickly provision a cloud server to use Outline VPN`,
		Long:  `After the user selects an machine image, instance type, region, and availability zone, an EC2 is created in the default subnet within the selected availability zone in the default vpc. If you don't have a default vpc or default subnet, we'll help you create defulat vpc or default subnet. You can create one EC2 instance for each region. You can use the vpn service by pasting access key on the Outline Client App.`,
	}

	_credential              *Credential
	_credentialWithMFA       = fmt.Sprintf("%s_mfa", config.DefaultSharedConfigFilename())
	_credentialWithTemporary = fmt.Sprintf("%s_temporary", config.DefaultSharedCredentialsFilename())

	congratulation = color.New(color.Bold, color.FgHiGreen).PrintFunc()
	notice         = color.New(color.Bold, color.FgHiRed).PrintfFunc()
)

type TerraformVarsJSON struct {
	AWSRegion        string `json:"aws_region"`
	EC2Ami           string `json:"ec2_ami"`
	InstanceType     string `json:"instance_type"`
	AvailabilityZone string `json:"availability_zone"`
}

type Credential struct {
	awsProfile    string
	awsConfig     *aws.Config
	homePath      string
	ssmPluginPath string
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

func gitInit() {
	// git clone https://github.com/ghdwlsgur/govpn-terraform
	if _, err := os.Stat(_defaultTerraformPath); errors.Is(err, os.ErrNotExist) {
		// govpn-terraform folder does not exist
		_, err := git.PlainClone(_defaultTerraformPath, false, &git.CloneOptions{
			URL:      _defaultGitURL,
			Progress: os.Stdout,
		})
		if err != nil {
			panicRed(err)
		}

		fmt.Println(color.GreenString("🎉 Terrafom File Download Complete! 🎉"))
	} else {
		// govpn-terraform folder exists
		repository, err := git.PlainOpen(_defaultTerraformPath)
		if err != nil {
			panicRed(err)
		}

		worktree, err := repository.Worktree()
		if err != nil {
			panicRed(err)
		}
		err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
		// if err != nil {
		// 	fmt.Println(color.GreenString("govpn-terraform \t(%s)", err.Error()))
		// } else {
		// 	fmt.Println(color.GreenString("govpn-terraform \t(%s)", "pull complete"))
		// }
	}
}

func findProfile() {
	awsProfile := viper.GetString("profile")

	if awsProfile == "" {
		if os.Getenv("AWS_PROFILE") != "" {
			awsProfile = os.Getenv("AWS_PROFILE")
		} else {
			awsProfile = _defaultProfile
		}
	}
	_credential.awsProfile = awsProfile
}

func findSharedCredFile() {
	sharedCredFile := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if sharedCredFile == "" {
		if _, err := os.Stat(_credentialWithMFA); !os.IsNotExist(err) {
			color.Yellow("[Use] outline-vpn default mfa credential file %s", _credentialWithMFA)
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

		if err != nil {
			if cred.Expired() || cred.AccessKeyID == "" || cred.SecretAccessKey == "" {
				color.Yellow("[Expire] outline-vpn default mfa credential file %s", sharedCredFile)
				os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
			} else {
				_credential.awsConfig = &awsConfig
			}
		}
	}
}

func findRegion(awsRegion string) {
	if awsRegion != "" {
		_credential.awsConfig.Region = awsRegion
	}

	if _credential.awsConfig.Region == "" {
		region, err := internal.AskRegion(context.Background(), *_credential.awsConfig)
		if err != nil {
			panicRed(internal.WrapError(err))
		}
		_credential.awsConfig.Region = region.Name
	}
}

func setTempConfig(awsRegion string) (string, aws.Config) {
	var temporaryCredentials aws.Credentials
	var temporaryConfig aws.Config

	var temporaryCredentialsInvalid = func(temporaryCredentials aws.Credentials) bool {
		return temporaryCredentials.Expired() ||
			temporaryCredentials.AccessKeyID == "" ||
			temporaryCredentials.SecretAccessKey == "" ||
			temporaryCredentials.SessionToken == ""
	}

	var temporaryCredentialsError = func(temporaryCredentials aws.Credentials, err error) bool {
		return temporaryCredentialsInvalid(temporaryCredentials)
	}

	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		temporaryConfig, err = internal.NewConfig(context.Background(),
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			os.Getenv("AWS_SESSION_TOKEN"),
			awsRegion,
			os.Getenv("AWS_ROLE_ARN"))
		if err != nil {
			panicRed(internal.WrapError(err))
		}
		temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
		if temporaryCredentialsError(temporaryCredentials, err) {
			panicRed(internal.WrapError(fmt.Errorf("invalid global environments %s", err.Error())))
		}
	} else {
		temporaryConfig, err = internal.NewSharedConfig(context.Background(),
			_credential.awsProfile,
			[]string{config.DefaultSharedConfigFilename()},
			[]string{})
		if err == nil {
			temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
		}

		if temporaryCredentialsError(temporaryCredentials, err) {
			temporaryConfig, err = internal.NewSharedConfig(context.Background(),
				_credential.awsProfile,
				[]string{config.DefaultSharedConfigFilename()},
				[]string{config.DefaultSharedCredentialsFilename()})
			if err != nil {
				panicRed(internal.WrapError(err))
			}

			temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
			if err != nil {
				panicRed(internal.WrapError(err))
			}
			if temporaryCredentialsInvalid(temporaryCredentials) {
				panicRed(internal.WrapError(fmt.Errorf("not found credentials")))
			}
		}
	}

	return fmt.Sprintf(mfaCredentialFormat,
		_credential.awsProfile,
		temporaryCredentials.AccessKeyID,
		temporaryCredentials.SecretAccessKey,
		temporaryCredentials.SessionToken,
	), temporaryConfig
}

func createTemporaryCredentialsFile(temporaryCredentialsString, awsRegion string, temporaryConfig aws.Config) string {
	if err := os.WriteFile(_credentialWithTemporary, []byte(temporaryCredentialsString), 0600); err != nil {
		panicRed(internal.WrapError(err))
	}

	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", _credentialWithTemporary)
	awsConfig, err := internal.NewSharedConfig(context.Background(),
		_credential.awsProfile,
		[]string{},
		[]string{_credentialWithTemporary})
	if err != nil {
		panicRed(err)
		panicRed(internal.WrapError(err))
	}
	_credential.awsConfig = &awsConfig

	if awsRegion == "" {
		return temporaryConfig.Region
	}
	return awsRegion
}

func libraryCheck(lib string) error {
	_, err := exec.LookPath(lib)
	if err != nil {
		return err
	}
	return nil
}

func libPrerequisite(libList []string) {
	for _, lib := range libList {
		if err = libraryCheck(lib); err != nil {
			panicRed(fmt.Errorf("⚠️  %s is not installed\n[required] jq, rsync and terraform must be installed as prerequisites.", lib))
		}
		// else {
		// 	PrintFunc(lib, "ready")
		// }
	}
	fmt.Println()
}

func PrintFunc(field, value string) {
	if len(field) < 8 {
		fmt.Printf("%s\t\t\t%s\n", color.HiBlackString(field), color.GreenString(fmt.Sprintf("("+value+")")))
	} else {
		fmt.Printf("%s\t\t%s\n", color.HiBlackString(field), color.GreenString(fmt.Sprintf("("+value+")")))
	}
}

func setUpPlugin() {
	home, err := homedir.Dir()
	if err != nil {
		panicRed(internal.WrapError(err))
	}

	_credential.homePath = filepath.Join(home, ".outline-vpn")
	if _, err := os.Stat(_credential.homePath); os.IsNotExist(err) {
		if err := os.MkdirAll(_credential.homePath, os.ModePerm); err != nil {
			panicRed(internal.WrapError(err))
		}
	}

	plugin, err := internal.GetSSMPlugin()
	if err != nil {
		panicRed(internal.WrapError(err))
	}

	_credential.ssmPluginPath = filepath.Join(_credential.homePath,
		internal.GetSSMPluginName())
	if info, err := os.Stat(_credential.ssmPluginPath); os.IsNotExist(err) {
		color.Green("[create] aws ssm plugin")
		if err := ioutil.WriteFile(_credential.ssmPluginPath, plugin, 0755); err != nil {
			panicRed(internal.WrapError(err))
		}
	} else if err != nil {
		panicRed(internal.WrapError(err))
	} else {
		if int(info.Size()) != len(plugin) {
			color.Green("[update] aws ssm plugin")
			if err := ioutil.WriteFile(_credential.ssmPluginPath, plugin, 0755); err != nil {
				panicRed(internal.WrapError(err))
			}
		}
	}
}

func initConfig() {

	_credential = &Credential{}
	gitInit()

	findProfile()
	findSharedCredFile()
	setUpPlugin()

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

	awsRegion := viper.GetString("region")
	if _credential.awsConfig == nil {
		temporaryCredentialsString, temporaryConfig := setTempConfig(awsRegion)
		awsRegion = createTemporaryCredentialsFile(temporaryCredentialsString, awsRegion, temporaryConfig)
	}
	findRegion(awsRegion)

	libList := []string{"jq", "rsync", "terraform"}
	libPrerequisite(libList)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("profile", "p", "", `[optional] if you having multiple aws profiles, it is one of profiles (default is AWS_PROFILE environment variable or default)`)
	rootCmd.PersistentFlags().StringP("region", "r", "", `[optional] it is region in AWS would like to do something`)

	rootCmd.InitDefaultVersionFlag()

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}
