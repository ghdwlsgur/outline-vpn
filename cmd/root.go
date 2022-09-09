/*
Copyright © 2020 gjbae1212
Released under the MIT license.
(https://github.com/gjbae1212/gossm)
*/

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
)

var (
	rootCmd = &cobra.Command{
		Use:   "govpn",
		Short: `govpn is interactive CLI tool`,
		Long:  `govpn is interactive CLI tool`,
	}

	_credential              *Credential
	_credentialWithMFA       = fmt.Sprintf("%s_mfa", config.DefaultSharedConfigFilename())
	_credentialWithTemporary = fmt.Sprintf("%s_temporary", config.DefaultSharedCredentialsFilename())
)

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

	askAmi, err := internal.AskAmi(context.Background(), *_credential.awsConfig)
	if err != nil {
		panicRed(internal.WrapError(err))
	}
	color.Green("ami (%s)", askAmi.Name)

}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("profile", "p", "", `[optional] if you having multiple aws profiles, it is one of profiles (default is AWS_PROFILE environment variable or default)`)
	rootCmd.PersistentFlags().StringP("region", "r", "", `[optional] it is region in AWS would like to do something`)

	rootCmd.InitDefaultVersionFlag()

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}
