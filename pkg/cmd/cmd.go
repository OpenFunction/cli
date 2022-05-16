package cmd

import (
	"flag"
	"io"
	"os"

	"github.com/OpenFunction/cli/pkg/cmd/subcommand"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
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
    https://github.com/OpenFunction/OpenFunction/blob/main/README.md
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

	cmd.AddCommand(subcommand.NewCmdCreate(kubeConfigFlags, ioStreams))
	cmd.AddCommand(subcommand.NewCmdDelete(kubeConfigFlags, ioStreams))
	cmd.AddCommand(subcommand.NewCmdGet(kubeConfigFlags, ioStreams))
	cmd.AddCommand(subcommand.NewCmdInstall(kubeConfigFlags, ioStreams))
	cmd.AddCommand(subcommand.NewCmdUninstall(kubeConfigFlags, ioStreams))
	cmd.AddCommand(subcommand.NewCmdDemo(kubeConfigFlags, ioStreams))
	cmd.AddCommand(subcommand.NewCmdVersion())
	return cmd
}

func addCmdHeaderHooks(cmds *cobra.Command, kubeConfigFlags *genericclioptions.ConfigFlags) {
	existingPreRunE := cmds.PersistentPreRunE
	cmds.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if existingPreRunE != nil {
			return existingPreRunE(cmd, args)
		}
		return nil
	}
}
