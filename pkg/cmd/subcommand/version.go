package subcommand

import (
	"fmt"

	"github.com/OpenFunction/cli/version"
	"github.com/spf13/cobra"
)

type VersionOptions struct {
	ShortVersion                bool
	ShowSupportedK8sVersionList bool
}

func NewVersionOptions() *VersionOptions {
	return &VersionOptions{}
}

// NewCmdVersion creates a new version command
func NewCmdVersion() *cobra.Command {
	o := NewVersionOptions()

	cmd := &cobra.Command{
		Use:   "version",
		Short: "print the client version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return printVersion(o.ShortVersion)
		},
	}
	o.AddFlags(cmd)
	return cmd
}

func (o *VersionOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&o.ShortVersion, "short", "", false, "print the version number")
}

func printVersion(short bool) error {
	v := version.Get()
	if short {
		if len(v.GitCommit) >= 7 {
			fmt.Printf("%s+g%s\n", v.Version, v.GitCommit[:7])
			return nil
		}
		fmt.Println(version.GetVersion())
	}
	fmt.Printf("%#v\n", v)
	return nil
}
