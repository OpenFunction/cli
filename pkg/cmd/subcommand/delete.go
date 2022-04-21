package subcommand

import (
	"context"
	"fmt"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	cc "github.com/OpenFunction/cli/pkg/cmd/util/client"
	client "github.com/openfunction/pkg/client/clientset/versioned"
	"github.com/openfunction/pkg/client/clientset/versioned/scheme"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"

	"github.com/openfunction/apis/core/v1beta1"
	openfunction "github.com/openfunction/apis/core/v1beta1"
)

// Delete is the commandline for 'delete' sub command
type Delete struct {
	genericclioptions.IOStreams

	FilenameOptions resource.FilenameOptions
	deleteFlag

	atlas             []atlas
	options           metav1.DeleteOptions
	NamespaceIfScoped bool

	namespace        string
	enforceNamespace bool
}

const (
	deleteExample = `
# Delete a function using the name specified in demo.yaml
ofn delete -f demo.yaml

# Delete a function based on name in the YAML passed into stdin
cat demo.yaml | ofn delete -f -

# Delete all functions
ofn delete --all
`
)

// NewDelete returns an initialized Delete instance
func NewDelete(ioStreams genericclioptions.IOStreams) *Delete {
	return &Delete{
		IOStreams: ioStreams,
	}
}

func NewCmdDelete(cf *genericclioptions.ConfigFlags, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.Interface

	d := NewDelete(ioStreams)
	cmd := &cobra.Command{
		Use:                   "delete -f FILENAME",
		DisableFlagsInUseLine: true,
		Short:                 "Delete a function",
		Long: `
Delete funtion by file names, stdin and names, or by resources
`,
		Example: deleteExample,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			config, err := cf.ToRESTConfig()
			if err != nil {
				panic(err)
			}
			cc.SetConfigDefaults(config)
			fc = client.NewForConfigOrDie(config)

			d.namespace, d.enforceNamespace, err = cf.ToRawKubeConfigLoader().Namespace()
			return err
		},

		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(d.Validate(cmd, args))
			util.CheckErr(d.Complete(fc, cmd, args))
			util.CheckErr(d.Run(fc, cmd))
		},
	}

	usage := "to use to delete the funtion"
	AddFilenameOptionFlags(cmd, &d.FilenameOptions, usage)
	cmd.Flags().BoolVar(&d.IgnoreNotFound, "ignore-not-found", d.IgnoreNotFound, "Treat \"resource not found\" as a successful delete. Defaults to \"true\" when --all is specified.")
	d.deleteFlag.addFlag(cmd)
	return cmd
}

func (d *Delete) Complete(fc client.Interface, cmd *cobra.Command, args []string) (err error) {
	d.atlas = make([]atlas, 0)
	switch {
	case len(args) != 0:
		for _, name := range args {
			d.atlas = append(d.atlas, atlas{
				Name:      name,
				Namespace: d.namespace,
			})
		}
	case len(d.FilenameOptions.Filenames) != 0:
		as, err := getAtlasFromFileOpntion(cmd, d.FilenameOptions, d.namespace)
		if err != nil {
			return err
		}
		d.atlas = as
	default:
		d.NamespaceIfScoped = !d.AllNamespaces
		err := d.deleteFlag.complete(cmd)
		if err != nil {
			return err
		}
		d.atlas, err = d.listAtlas(fc)
		if err != nil {
			return err
		}
	}

	d.options = d.deleteFlag.ToOptions()
	return nil
}
func (d *Delete) Validate(cmd *cobra.Command, args []string) error {
	if d.deleteFlag.All && len(args) > 0 {
		return util.UsageErrorf(cmd, "cannot set --all and name at the same time")
	}
	return d.deleteFlag.Validate(cmd)
}

func (d *Delete) Run(fc client.Interface, cmd *cobra.Command) (err error) {
	var out string
	if d.DryRun {
		out = "deleted(dry run)"
	}

	for _, as := range d.atlas {
		err := fc.CoreV1beta1().Functions(as.Namespace).Delete(context.Background(), as.Name, d.options)
		if err != nil {
			return err
		}

		fmt.Fprintf(d.Out, "%s %s\n", as.Name, out)
	}

	return nil
}

type atlas struct {
	Name      string
	Namespace string
}

func getAtlasFromFileOpntion(cmd *cobra.Command, fo resource.FilenameOptions, namespace string) ([]atlas, error) {
	fns, err := getFromFilenameOptions(cmd, fo)
	if err != nil {
		return nil, err
	}
	return functionListToAtlas(fns, namespace), nil
}

func functionListToAtlas(fns []*openfunction.Function, namespace string) []atlas {
	as := make([]atlas, 0, len(fns))
	for _, fn := range fns {
		if fn.Namespace == "" {
			fn.Namespace = namespace
		}
		as = append(as, toAtlas(fn.ObjectMeta))
	}
	return as
}

func toAtlas(obj metav1.ObjectMeta) atlas {
	return atlas{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	}
}

func (d *Delete) listAtlas(fc client.Interface) ([]atlas, error) {
	options := metav1.ListOptions{
		LabelSelector: d.deleteFlag.LabelSelector,
		FieldSelector: d.deleteFlag.FieldSelector,
	}

	fnList := &v1beta1.FunctionList{}
	err := fc.CoreV1beta1().RESTClient().
		Get().
		NamespaceIfScoped(d.namespace, d.NamespaceIfScoped).
		Resource("functions").
		VersionedParams(&options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(fnList)
	if err != nil {
		return nil, err
	}

	as := make([]atlas, 0, len(fnList.Items))
	for _, fn := range fnList.Items {
		as = append(as, toAtlas(fn.ObjectMeta))
	}
	return as, nil
}
