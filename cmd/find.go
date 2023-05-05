package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/ghdwlsgur/outline-vpn/internal"
	"github.com/spf13/cobra"
)

var (
	findCommand = &cobra.Command{
		Use:   "find",
		Short: "Find instances with the tag [govpn-ec2] in all available regions.",
		Long:  "Find instances with the tag [govpn-ec2] in all available regions.",
		Run: func(_ *cobra.Command, _ []string) {
			ctx := context.Background()

			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			s := spinner.New(spinner.CharSets[8], 100*time.Millisecond)
			s.UpdateCharSet(spinner.CharSets[39])
			s.Color("fgHiGreen")
			s.Restart()
			s.Prefix = color.HiGreenString("Searching for EC2 instances with the tag 'govpn-ec2' ")

			regionList, err := internal.FindTagInstance(ctx, *_credential.awsConfig)
			if err != nil {
				panicRed(err)
			}
			s.Stop()

			var result string
			if len(regionList) == 0 {
				fmt.Printf(color.HiRedString("No instances created with govpn cli were found.\n"))
			} else {
				for _, region := range regionList {
					f := fmt.Sprintf(" %s", region)
					result += f
				}
				fmt.Printf("%s%s\n", "There exists an instance on", color.HiGreenString(result))
			}

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

		},
	}
)

func init() {
	rootCmd.AddCommand(findCommand)
}
