package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/fatih/color"
	"github.com/ghdwlsgur/outline-vpn/internal"
	"github.com/spf13/cobra"
)

var (
	connectCommand = &cobra.Command{
		Use:   "connect",
		Short: "Connect to an instance using the AWS SSM plugin.",
		Long:  "Connect to an instance using the AWS SSM plugin.",
		Run: func(_ *cobra.Command, _ []string) {
			var (
				err   error
				table = make(map[string]*internal.EC2)
			)

			ctx := context.Background()

			fileList, err := returnWorkspaceFileList()
			if err != nil {
				panicRed(err)
			}

			if len(fileList) > 0 {
				for _, regionName := range fileList {
					instance, err = internal.FindSpecificTagInstance(ctx, *_credential.awsConfig, regionName)
					if err != nil {
						panicRed(err)
					}

					tableKey := fmt.Sprintf("[id: %s - region: %s]",
						instance.GetID(),
						instance.GetRegion())
					table[tableKey] = instance
				}

				var option []string
				for key := range table {
					option = append(option, key)
				}

				answer, err := internal.AskPromptOptionList("Please select the instance to remove", option, len(option))
				if err != nil {
					panicRed(err)
				}

				instance = table[answer]

				input := &ssm.StartSessionInput{Target: aws.String(instance.GetID())}
				_credential.awsConfig.Region = instance.GetRegion()
				session, err := internal.CreateStartSession(ctx, *_credential.awsConfig, input)
				if err != nil {
					panicRed(err)
				}

				sessJson, err := json.Marshal(session)
				if err != nil {
					panicRed(err)
				}

				paramsJson, err := json.Marshal(input)
				if err != nil {
					panicRed(err)
				}

				if err := internal.CallProcess(_credential.ssmPluginPath, string(sessJson),
					instance.GetRegion(), "StartSession", _credential.awsProfile, string(paramsJson)); err != nil {
					color.Red("%v", err)
				}
			} else {
				notice("There are no instances with the tag 'govpn-ec2' available in all regions.\n")
				os.Exit(1)
			}

		},
	}
)

func init() {
	rootCmd.AddCommand(connectCommand)
}
