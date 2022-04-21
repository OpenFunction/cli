package subcommand

import (
	"context"
	"fmt"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	cc "github.com/OpenFunction/cli/pkg/cmd/util/client"
	client "github.com/openfunction/pkg/client/clientset/versioned"
	scheme "github.com/openfunction/pkg/client/clientset/versioned/scheme"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/openfunction/apis/core/v1beta1"
	openfunction "github.com/openfunction/apis/core/v1beta1"
)

type getBuilder struct {
	genericclioptions.IOStreams
	listFlag
	Printer *util.Printer

	Name              string
	NamespaceIfScoped bool

	namespace        string
	enforceNamespace bool
}

const (
	getbulderExample = `
# List all builder in output format
ofn get builder

# Get builder in JSON output format
ofn get builder sample-builder-m5sbv -o json

# Get builder in YAML output format
ofn get builder sample-builder-m5sbv -o yaml
`

	getBuildLong = `
Prints a table ofn the most important information.
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

		Printer: util.NewPrinter("get-builder", scheme.Scheme),
	}
}

func newCmdGetBuilder(cf *genericclioptions.ConfigFlags, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.Interface

	g := newGetBuiler(ioStreams)
	cmd := &cobra.Command{
		Use:                   "builder",
		DisableFlagsInUseLine: true,
		Short:                 "Display one or many builder",
		Long:                  getBuildLong,
		Example:               getbulderExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			config, err := cf.ToRESTConfig()
			if err != nil {
				panic(err)
			}
			cc.SetConfigDefaults(config)
			fc = client.NewForConfigOrDie(config)

			g.namespace, g.enforceNamespace, err = cf.ToRawKubeConfigLoader().Namespace()
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

	g.NamespaceIfScoped = true
	if !g.enforceNamespace {
		g.NamespaceIfScoped = !g.AllNamespaces
	}

	return nil
}

func (g *getBuilder) Run(fc client.Interface, cmd *cobra.Command, args []string) error {
	var (
		objs []runtime.Object
		obj  runtime.Object
		err  error
	)

	ctx := context.Background()
	if g.Name != "" {
		obj, err = fc.CoreV1beta1().Builders(g.namespace).Get(ctx, g.Name, metav1.GetOptions{})
		objs = []runtime.Object{obj}
	} else {
		opt := g.listFlag.ToOptions()
		result := &v1beta1.BuilderList{}
		err := fc.CoreV1beta1().RESTClient().
			Get().
			NamespaceIfScoped(g.namespace, g.NamespaceIfScoped).
			Resource("builders").
			VersionedParams(&opt, scheme.ParameterCodec).
			Do(ctx).
			Into(result)
		if err != nil {
			return err
		}
		objs = make([]runtime.Object, 0, len(result.Items))
		for i := range result.Items {
			objs = append(objs, &result.Items[i])
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
