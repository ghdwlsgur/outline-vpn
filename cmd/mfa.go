/*
Copyright © 2020 gjbae1212
Released under the MIT license.
(https://github.com/gjbae1212/gossm)
*/

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	virtualMFADevice    = "arn:aws:iam:%s:mfa/%s"
	mfaCredentialFormat = "[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\naws_session_token = %s\n"
)

var (
	mfaCommand = &cobra.Command{
		Use:   "mfa",
		Short: "It's to authenticate MFA on AWS, and save authenticated mfa token in .aws/credentials_mfa.",
		Long:  `It's to authenticate MFA on AWS, and save authenticated mfa token in .aws/credentials_mfa.`,
		Run: func(_ *cobra.Command, args []string) {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, time.Second*60)
			defer cancel()

			if len(args) != 1 {
				panicRed(fmt.Errorf("invalid mfa code"))
			}
			code := strings.TrimSpace(args[0])
			if code == "" {
				panicRed(fmt.Errorf("invalid mfa code"))
			}

			client := sts.NewFromConfig(*_credential.awsConfig)

			deadline := viper.GetInt32("mfa-deadline")
			device := viper.GetString("mfa-device")

			if device == "" {
				identity, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
				if err != nil {
					panicRed(err)
				}
				username := strings.Split(*identity.Arn, "/")[1]
				device = fmt.Sprintf(virtualMFADevice, aws.ToString(identity.Account), username)
			}

			output, err := client.GetSessionToken(ctx, &sts.GetSessionTokenInput{
				DurationSeconds: aws.Int32(deadline),
				SerialNumber:    aws.String(device),
				TokenCode:       aws.String(code),
			})
			if err != nil {
				panicRed(err)
			}

			newCredential := fmt.Sprintf(mfaCredentialFormat, _defaultProfile, *output.Credentials.AccessKeyId,
				*output.Credentials.SecretAccessKey, *output.Credentials.SessionToken)
			if err := os.WriteFile(_credentialWithMFA, []byte(newCredential), 0600); err != nil {
				panicRed(err)
			}

			color.Green("[SUCCESS] Temporary MFA credential creates %s (%s)", _credentialWithMFA, output.Credentials.Expiration.UTC())
			fmt.Printf("%s `%s` %s\n",
				color.YellowString("[INFO] For Use AWS CLI using temporary MFA credential, Set To"),
				color.CyanString("export AWS_SHARED_CREDENTIALS_FILE=%s", _credentialWithMFA),
				color.YellowString("in $HOME/.bash_profile, $HOME/.zshrc"))
		},
	}
)

func init() {
	mfaCommand.Flags().Int32P("deadline", "", 21600, "[optional] deadline seconds for issued credentials. (default is 6 hours)")
	mfaCommand.Flags().StringP("device", "", "", "[optional] mfa device. (default is your virtual mfa device")

	viper.BindPFlag("mfa-deadline", mfaCommand.Flags().Lookup("deadline"))
	viper.BindPFlag("mfa-device", mfaCommand.Flags().Lookup("device"))
	rootCmd.AddCommand(mfaCommand)
}
