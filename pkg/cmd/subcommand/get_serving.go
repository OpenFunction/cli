package subcommand

import (
	"context"
	"fmt"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	cc "github.com/OpenFunction/cli/pkg/cmd/util/client"
	"github.com/openfunction/apis/core/v1beta1"
	openfunction "github.com/openfunction/apis/core/v1beta1"
	client "github.com/openfunction/pkg/client/clientset/versioned"
	scheme "github.com/openfunction/pkg/client/clientset/versioned/scheme"
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

	namespace        string
	enforceNamespace bool
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

		Printer: util.NewPrinter("get-serving", scheme.Scheme),
	}
}

func newCmdGetServing(cf *genericclioptions.ConfigFlags, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.Interface

	g := newGetServing(ioStreams)
	cmd := &cobra.Command{
		Use:                   "serving",
		DisableFlagsInUseLine: true,
		Short:                 "Display one or many serving",
		Long:                  getServingLong,
		Example:               getServingExample,
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

func (g *getServing) Complete(cmd *cobra.Command, args []string) error {
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

func (g *getServing) Run(fc client.Interface, cmd *cobra.Command, args []string) error {
	var (
		objs []runtime.Object
		obj  runtime.Object
		err  error
	)

	ctx := context.Background()
	if g.Name != "" {
		obj, err = fc.CoreV1beta1().Servings(g.namespace).Get(ctx, g.Name, metav1.GetOptions{})
		objs = []runtime.Object{obj}
	} else {
		opt := g.listFlag.ToOptions()
		result := &v1beta1.ServingList{}
		err := fc.CoreV1beta1().RESTClient().
			Get().
			NamespaceIfScoped(g.namespace, g.NamespaceIfScoped).
			Resource("servings").
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
		serving.Spec.Runtime,
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
