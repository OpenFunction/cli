package subcommand

import (
	"context"
	"fmt"

	client "github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	openfunction "github.com/openfunction/apis/core/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type getServing struct {
	genericclioptions.IOStreams
	listFlag
	Printer *util.Printer

	Name              string
	NamespaceIfScoped bool
}

const (
	getServingExample = `
# List all serving in output format
fn get builder

# Get serving in JSON output format
fn get serving sample-serving-c2dsf -o json

# Get serving in YAML output format
fn get serving sample-serving-c2dsf -o yaml
`

	getServingLong = `
Prints a table fn the most important information.
`
)

var (
	servingColumnLabels = []string{
		"NAME",
		"NAMESPACE",
		"RUNTIME",
		"RESOURCE",
		"PHASE",
		"STATE",
		"AGE",
	}
)

// newGetServing returns an initialized getServing instance
func newGetServing(ioStreams genericclioptions.IOStreams) *getServing {
	return &getServing{
		IOStreams: ioStreams,

		Printer: util.NewPrinter("get-serving", client.Scheme),
	}
}

func newCmdGetServing(restClient util.Getter, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.FnClient

	g := newGetServing(ioStreams)
	cmd := &cobra.Command{
		Use:                   "serving",
		DisableFlagsInUseLine: true,
		Short:                 "Display one or many serving",
		Long:                  getServingLong,
		Example:               getServingExample,
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

	return cmd
}

func (g *getServing) Complete(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		g.Name = args[0]
		g.Printer.SetForceDefail()
	}
	g.NamespaceIfScoped = !g.listFlag.AllNamespaces

	return nil
}

func (g *getServing) Run(fc client.FnClient, cmd *cobra.Command, args []string) error {
	var (
		objs []runtime.Object
		obj  runtime.Object
		err  error
	)

	ctx := context.Background()
	if g.Name != "" {
		obj, err = fc.GetServing(ctx, g.Name, metav1.GetOptions{})
		objs = []runtime.Object{obj}
	} else {
		options := g.listFlag.ToOptions()
		servingList, err := fc.ListServing(ctx, g.NamespaceIfScoped, options)
		if err != nil {
			return err
		}
		objs = make([]runtime.Object, 0, len(servingList.Items))
		for i := range servingList.Items {
			objs = append(objs, &servingList.Items[i])
		}

		if util.IsToTable(g.Printer) {
			obj, err = util.ToTable(servingRow, objs...)
			if err != nil {
				return err
			}
			objs = []runtime.Object{obj}
		}
	}
	if err != nil {
		return err
	}

	printer, err := g.Printer.ToPrinterWitchColumn(servingColumnLabels)
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

func servingRow(obj interface{}) (metav1.TableRow, error) {
	serving, ok := obj.(*openfunction.Serving)
	if !ok {
		return metav1.TableRow{}, fmt.Errorf("interface conversion: interface {} is not *v1alpha1.Function")
	}

	name := serving.Name
	namespace := serving.Namespace
	var rt openfunction.Runtime
	if serving.Spec.Runtime != nil {
		rt = *serving.Spec.Runtime
	}
	phase := serving.Status.Phase
	state := serving.Status.State
	resourceRef := gerSeringResourceRef(serving.Status.ResourceRef)

	age := util.TranslateTimestampSince(serving.CreationTimestamp)
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: serving},
	}

	row.Cells = append(row.Cells,
		name,
		namespace,
		rt,
		resourceRef,
		phase,
		state,
		age,
	)
	return row, nil
}

func gerSeringResourceRef(ref map[string]string) string {
	if ref == nil {
		return ""
	}
	for _, value := range ref {
		return value
	}

	return ""
}
