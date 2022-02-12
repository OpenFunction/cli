package cmd

import (
	"flag"
	"io"
	"net/http"
	"os"

	"github.com/OpenFunction/cli/pkg/cmd/subcommand"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	cflg "k8s.io/component-base/cli/flag"
)

// NewDefaultCommand creates the default command with default arguments
func NewDefaultCommand() *cobra.Command {
	return NewDefaultCommandWithArgs(os.Args, os.Stdin, os.Stdout, os.Stderr)
}

// NewDefaultCommandWithArgs creates the default command with arguments
func NewDefaultCommandWithArgs(args []string, in io.Reader, out, errout io.Writer) *cobra.Command {
	return NewCommand(in, out, errout)
}

// NewCommand creates the command
func NewCommand(in io.Reader, out, errout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ofn",
		Short: "ofn controller the openfunction manager",
		Long: `
ofn controller the openfunction manager.

Find more information at:
    https://https://github.com/OpenFunction/OpenFunction/blob/main/README.md
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	flags := cmd.PersistentFlags()
	flags.SetNormalizeFunc(cflg.WarnWordSepNormalizeFunc)
	flags.SetNormalizeFunc(cflg.WordSepNormalizeFunc)

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.AddFlags(flags)
	addCmdHeaderHooks(cmd, kubeConfigFlags)
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	cmd.SetGlobalNormalizationFunc(cflg.WarnWordSepNormalizeFunc)

	ioStreams := genericclioptions.IOStreams{In: in, Out: out, ErrOut: errout}
	restClient := util.NewRESTClientGetter(kubeConfigFlags)

	cmd.AddCommand(subcommand.NewCmdInit(ioStreams))
	cmd.AddCommand(subcommand.NewCmdCreate(restClient, ioStreams))
	cmd.AddCommand(subcommand.NewCmdApply(restClient, ioStreams))
	cmd.AddCommand(subcommand.NewCmdDelete(restClient, ioStreams))
	cmd.AddCommand(subcommand.NewCmdGet(restClient, ioStreams))
	cmd.AddCommand(subcommand.NewCmdInstall(restClient, ioStreams))
	cmd.AddCommand(subcommand.NewCmdUninstall(restClient, ioStreams))
	cmd.AddCommand(subcommand.NewCmdDemo(restClient, ioStreams))
	cmd.AddCommand(subcommand.NewCmdVersion())
	return cmd
}

func addCmdHeaderHooks(cmds *cobra.Command, kubeConfigFlags *genericclioptions.ConfigFlags) {
	crt := &genericclioptions.CommandHeaderRoundTripper{}

	existingPreRunE := cmds.PersistentPreRunE
	cmds.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		crt.ParseCommandHeaders(cmd, args)
		if existingPreRunE != nil {
			return existingPreRunE(cmd, args)
		}
		return nil
	}
	kubeConfigFlags.WrapConfigFn = func(c *rest.Config) *rest.Config {
		c.Wrap(func(rt http.RoundTripper) http.RoundTripper {
			crt.Delegate = rt
			return crt
		})
		return c
	}
}
