package subcommand

import (
	"context"

	client "github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
)

// Apply is the commandline for 'apply' sub command
type Apply struct {
	genericclioptions.IOStreams
	Printer *util.Printer

	FilenameOptions resource.FilenameOptions
	DryRun          bool
	FieldManager    string

	options metav1.ApplyOptions
	printer printers.ResourcePrinter
}

const (
	applyExample = `
# Apply a function using the data in function.yaml
ofn apply -f function.yaml

# Create a function based on the YAML passed into stdin
cat function.yaml | fn apply -f -
`

	applyLong = `
Apply a configuration to a function by file name or stdin. This function will Apply a configuration to a resource by 
file name or stdin. This function will be created if it doesn't exist yet. To use 'apply', always create the function 
initially. created if it doesn't exist yet.
   `
)

const (
	fieldManagerClientSideApply = "client-side-apply"
)

// NewApply returns an initialized Apply instance
func NewApply(ioStreams genericclioptions.IOStreams) *Apply {
	return &Apply{
		IOStreams: ioStreams,

		Printer: util.NewPrinter("apply", client.Scheme),
	}
}

func NewCmdApply(restClient util.Getter, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.FnClient

	a := NewApply(ioStreams)
	cmd := &cobra.Command{
		Use:                   "apply (-f FILENAME | -k DIRECTORY) [options]",
		DisableFlagsInUseLine: true,
		Short:                 "Apply a resource from a file",
		Long:                  applyLong,
		Example:               applyExample,
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			fc, err = client.NewFnClient(restClient)
			return err
		},
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(a.Complete(cmd, args))
			util.CheckErr(a.Validate(cmd, args))
			util.CheckErr(a.RunApply(fc, cmd))
		},
	}

	usage := "to use to apply the function"
	AddFilenameOptionFlags(cmd, &a.FilenameOptions, usage)
	a.Printer.AddFlags(cmd)
	cmd.Flags().StringVar(&a.FieldManager, "field-manager", fieldManagerClientSideApply, "Name of the manager used to track field ownership.")
	cmd.Flags().BoolVarP(&a.DryRun, "dry-run", "", a.DryRun, "Only print the object that would be sent, without sending it")
	return cmd
}

func (a *Apply) Complete(cmd *cobra.Command, args []string) error {
	if a.DryRun {
		a.options.DryRun = []string{metav1.DryRunAll}
	}
	a.options.FieldManager = a.FieldManager

	a.Printer.SetPrinterFunc(util.WithDefaultPrinter("configured"))

	var err error
	a.printer, err = a.Printer.ToPrinter()
	if err != nil {
		return err
	}

	return nil
}

func (a *Apply) Validate(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return util.UsageErrorf(cmd, "Unexpected args: %v", args)
	}
	if len(a.FilenameOptions.Filenames) == 0 {
		return util.UsageErrorf(cmd, "use at least one file")
	}
	return nil
}

func (a *Apply) RunApply(fc client.FnClient, cmd *cobra.Command) error {
	return a.applyFromFile(fc, cmd)
}

func (a *Apply) applyFromFile(fc client.FnClient, cmd *cobra.Command) error {
	fns, err := getFromFilenameOptions(cmd, a.FilenameOptions)
	if err != nil {
		return err
	}

	for _, fn := range fns {
		result, err := fc.Namespace(fn.Namespace).Apply(context.Background(), fn, a.options)
		if err != nil {
			return err
		}

		if a.printer != nil {
			if err = a.printer.PrintObj(result, a.Out); err != nil {
				return err
			}
		}
	}
	return err
}
