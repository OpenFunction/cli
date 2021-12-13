package subcommand

import (
	"context"
	"fmt"

	client "github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	openfunction "github.com/openfunction/apis/core/v1alpha1"
)

// Get is the commandline for 'get' sub command
type Get struct {
	genericclioptions.IOStreams
	listFlag
	Printer *util.Printer

	Name              string
	NamespaceIfScoped bool
}

const (
	getExample = `
# List all function in output format
ofn get

# Get function in JSON output format
ofn get sample -o json

# Get function in YAML output format
ofn get sample -o yaml

# Return only the state ofn build
ofn get sample --template={{.status.build.state}}
`
	getLong = `
Prints a table of the most important information.
`
)

var (
	fnColumnLabels = []string{
		"NAME",
		"NAMESPACE",
		"BUILDSTATE",
		"SERVINGSTATE",
		"BUILDER",
		"SERVING",
		"AGE",
	}
)

// NewGet returns an initialized Get instance
func NewGet(ioStreams genericclioptions.IOStreams) *Get {
	return &Get{
		IOStreams: ioStreams,

		Printer: util.NewPrinter("get", client.Scheme),
	}
}

func NewCmdGet(restClient util.Getter, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.FnClient

	g := NewGet(ioStreams)
	cmd := &cobra.Command{
		Use:                   "get",
		DisableFlagsInUseLine: true,
		Short:                 "Display one or many function",
		Long:                  getLong,
		Example:               getExample,
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			fc, err = client.NewFnClient(restClient)
			return err
		},
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(g.Complete(cmd, args))
			util.CheckErr(g.Run(fc, cmd, args))
		},
	}

	g.Printer.AddFlags(cmd)
	g.listFlag.addListFlag(cmd)

	cmd.AddCommand(newCmdGetBuilder(restClient, ioStreams))
	cmd.AddCommand(newCmdGetServing(restClient, ioStreams))
	return cmd
}

func (g *Get) Complete(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		g.Name = args[0]
		g.Printer.SetForceDefail()
	}

	g.NamespaceIfScoped = !g.AllNamespaces

	return nil
}

func (g *Get) Run(fc client.FnClient, cmd *cobra.Command, args []string) error {
	var (
		objs []runtime.Object
		obj  runtime.Object
		err  error
	)

	ctx := context.Background()
	if g.Name != "" {
		obj, err = fc.Get(ctx, g.Name, metav1.GetOptions{})
		objs = []runtime.Object{obj}
	} else {
		options := g.listFlag.ToOptions()
		fnList, err := fc.List(ctx, g.NamespaceIfScoped, options)
		if err != nil {
			return err
		}

		objs = make([]runtime.Object, 0, len(fnList.Items))
		for i := range fnList.Items {
			objs = append(objs, &fnList.Items[i])
		}

		if util.IsToTable(g.Printer) {
			obj, err = util.ToTable(fnRow, objs...)
			if err != nil {
				return err
			}
			objs = []runtime.Object{obj}
		}
	}

	if err != nil {
		return err
	}

	printer, err := g.Printer.ToPrinterWitchColumn(fnColumnLabels)
	if err != nil {
		return err
	}

	for _, obj := range objs {
		if err = printer.PrintObj(obj, g.Out); err != nil {
			return err
		}
	}
	return nil
}

func fnRow(obj interface{}) (metav1.TableRow, error) {
	fn, ok := obj.(*openfunction.Function)
	if !ok {
		return metav1.TableRow{}, fmt.Errorf("interface conversion: interface {} is not *v1alpha1.Function")
	}

	name := fn.Name
	namespace := fn.Namespace
	var builder, buildState, serving, servingState string
	if fn.Status.Build != nil {
		buildState = fn.Status.Build.State
		builder = fn.Status.Build.ResourceRef
	}
	if fn.Status.Serving != nil {
		servingState = fn.Status.Serving.State
		serving = fn.Status.Serving.ResourceRef
	}

	age := util.TranslateTimestampSince(fn.CreationTimestamp)
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: fn},
	}

	row.Cells = append(row.Cells,
		name,
		namespace,
		buildState,
		servingState,
		builder,
		serving,
		age,
	)
	return row, nil
}
