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

type getBuilder struct {
	genericclioptions.IOStreams
	listFlag
	Printer *util.Printer

	Name              string
	NamespaceIfScoped bool
}

const (
	getbulderExample = `
# List all builder in output format
fn get builder

# Get builder in JSON output format
fn get builder sample-builder-m5sbv -o json

# Get builder in YAML output format
fn get builder sample-builder-m5sbv -o yaml
`

	getBuildLong = `
Prints a table fn the most important information.
`
)

var (
	builderColumnLabels = []string{
		"NAME",
		"NAMESPACE",
		"BUILD",
		"BUILDRUN",
		"PHASE",
		"STATE",
		"AGE",
	}
)

// newGetBuiler returns an initialized getBuilder instance
func newGetBuiler(ioStreams genericclioptions.IOStreams) *getBuilder {
	return &getBuilder{
		IOStreams: ioStreams,

		Printer: util.NewPrinter("get-builder", client.Scheme),
	}
}

func newCmdGetBuilder(restClient util.Getter, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.FnClient

	g := newGetBuiler(ioStreams)
	cmd := &cobra.Command{
		Use:                   "builder",
		DisableFlagsInUseLine: true,
		Short:                 "Display one or many builder",
		Long:                  getBuildLong,
		Example:               getbulderExample,
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

func (g *getBuilder) Complete(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		g.Name = args[0]
		g.Printer.SetForceDefail()
	}
	g.NamespaceIfScoped = !g.listFlag.AllNamespaces

	return nil
}

func (g *getBuilder) Run(fc client.FnClient, cmd *cobra.Command, args []string) error {
	var (
		objs []runtime.Object
		obj  runtime.Object
		err  error
	)

	ctx := context.Background()
	if g.Name != "" {
		obj, err = fc.GetBuilder(ctx, g.Name, metav1.GetOptions{})
		objs = []runtime.Object{obj}
	} else {
		options := g.listFlag.ToOptions()
		builderList, err := fc.ListBuilder(ctx, g.NamespaceIfScoped, options)
		if err != nil {
			return err
		}
		objs = make([]runtime.Object, 0, len(builderList.Items))
		for i := range builderList.Items {
			objs = append(objs, &builderList.Items[i])
		}

		if util.IsToTable(g.Printer) {
			obj, err = util.ToTable(builderRow, objs...)
			if err != nil {
				return err
			}
			objs = []runtime.Object{obj}
		}
	}
	if err != nil {
		return err
	}

	printer, err := g.Printer.ToPrinterWitchColumn(builderColumnLabels)
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

func builderRow(obj interface{}) (metav1.TableRow, error) {
	builder, ok := obj.(*openfunction.Builder)
	if !ok {
		return metav1.TableRow{}, fmt.Errorf("interface conversion: interface {} is not *v1alpha1.Builder")
	}
	name := builder.Name
	namespace := builder.Namespace
	phase := builder.Status.Phase
	state := builder.Status.State

	build, buildRun := getBuilderResourceRef(builder.Status.ResourceRef)

	age := util.TranslateTimestampSince(builder.CreationTimestamp)
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: builder},
	}

	row.Cells = append(row.Cells,
		name,
		namespace,
		build,
		buildRun,
		phase,
		state,
		age,
	)
	return row, nil
}

func getBuilderResourceRef(ref map[string]string) (string, string) {
	if ref == nil {
		return "", ""
	}
	return ref["shipwright.io/build"], ref["shipwright.io/buildRun"]
}
