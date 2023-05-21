package cmd

// import (
// 	"context"
// 	"fmt"
// 	"os"

// 	"github.com/ghdwlsgur/outline-vpn/internal"
// 	"github.com/spf13/cobra"
// )

// var (
// 	keyCommand = &cobra.Command{
// 		Use:   "key",
// 		Short: "",
// 		Long:  "",
// 		Run: func(_ *cobra.Command, _ []string) {
// 			var (
// 				err     error
// 				table   = make(map[string]*internal.EC2)
// 				outline = &internal.OutlineInfo{}
// 			)
// 			ctx := context.Background()

// 			fileList, err := returnWorkspaceFileList()
// 			if err != nil {
// 				panicRed(err)
// 			}

// 			if len(fileList) > 0 {
// 				for _, regionName := range fileList {
// 					instance, err = internal.FindSpecificTagInstance(ctx, *_credential.awsConfig, regionName)
// 					if err != nil {
// 						panicRed(err)
// 					}

// 					tableKey := fmt.Sprintf("[id: %s]", instance.GetID())
// 					table[tableKey] = instance
// 				}

// 				var option []string
// 				for key := range table {
// 					option = append(option, key)
// 				}

// 				answer, err := internal.AskPromptOptionList("Please select the instance to remove", option, len(option))
// 				if err != nil {
// 					panicRed(err)
// 				}

// 				instance = table[answer]

// 				outline.CertSha256, err = internal.GetCertSha256(instance.GetRegion())
// 				if err != nil {
// 					panicRed(err)
// 				}
// 				fmt.Println(outline.CertSha256)

// 				outline.ApiURL, err = internal.GetApiURL(instance.GetRegion())
// 				if err != nil {
// 					panicRed(err)
// 				}
// 				fmt.Println(outline.ApiURL)

// 				accessKeys, err := internal.GetAccessKeys(instance.GetRegion())
// 				if err != nil {
// 					panicRed(err)
// 				}
// 				fmt.Println(accessKeys)

// 			} else {
// 				notice("There are no instances with the tag 'govpn-ec2' available in all regions.\n")
// 				os.Exit(1)
// 			}

// 		},
// 	}
// )

// func init() {
// 	rootCmd.AddCommand(keyCommand)
// }
