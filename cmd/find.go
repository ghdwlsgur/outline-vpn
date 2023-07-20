package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/ghdwlsgur/outline-vpn/internal"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	findCommand = &cobra.Command{
		Use:   "find",
		Short: "Find instances with the tag [govpn-ec2] in all available regions.",
		Long:  "Find instances with the tag [govpn-ec2] in all available regions.",
		Run: func(_ *cobra.Command, _ []string) {
			var (
				ec2Table = make(map[string]*internal.EC2)
			)

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
				fmt.Print(color.HiRedString("No instances created with outline-vpn cli were found.\n"))
			} else {
				for _, region := range regionList {
					f := fmt.Sprintf(" %s", region)
					result += f
				}

				for _, region := range regionList {
					instance, err = internal.FindSpecificTagInstance(ctx, *_credential.awsConfig, region)
					if err != nil {
						panicRed(err)
					}
					ec2Table[region] = instance
				}

				t := table.NewWriter()
				t.SetOutputMirror(os.Stdout)

				t.AppendHeader(table.Row{"ID", "Public IP", "Launch Time", "Instance Type", "Region"})
				for region := range ec2Table {
					t.AppendRow(table.Row{
						ec2Table[region].GetID(),
						ec2Table[region].GetPublicIP(),
						ec2Table[region].GetLaunchTime(),
						ec2Table[region].GetInstanceType(),
						ec2Table[region].GetRegion(),
					})
				}

				t.Render()
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
