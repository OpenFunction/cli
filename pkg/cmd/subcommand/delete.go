package subcommand

import (
	"context"
	"fmt"

	client "github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	openfunction "github.com/openfunction/apis/core/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

// Delete is the commandline for 'delete' sub command
type Delete struct {
	genericclioptions.IOStreams

	FilenameOptions resource.FilenameOptions
	deleteFlag

	atlas             []atlas
	options           metav1.DeleteOptions
	NamespaceIfScoped bool
}

const (
	deleteExample = `
# Delete a function using the name specified in demo.yaml
fn delete -f demo.yaml

# Delete a function based on name in the YAML passed into stdin
cat demo.yaml | fn delete -f -

# Delete all functions
fn delete --all
`
)

// NewDelete returns an initialized Delete instance
func NewDelete(ioStreams genericclioptions.IOStreams) *Delete {
	return &Delete{
		IOStreams: ioStreams,
	}
}

func NewCmdDelete(restClient util.Getter, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.FnClient

	d := NewDelete(ioStreams)
	cmd := &cobra.Command{
		Use:                   "delete -f FILENAME",
		DisableFlagsInUseLine: true,
		Short:                 "Delete a function",
		Long: `
Delete funtion by file names, stdin and names, or by resources
`,
		Example: deleteExample,
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			fc, err = client.NewFnClient(restClient)
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

func (d *Delete) Complete(fc client.FnClient, cmd *cobra.Command, args []string) (err error) {
	d.atlas = make([]atlas, 0)
	switch {
	case len(args) != 0:
		for _, name := range args {
			d.atlas = append(d.atlas, atlas{
				Name: name,
			})
		}
	case len(d.FilenameOptions.Filenames) != 0:
		as, err := getAtlasFromFileOpntion(cmd, d.FilenameOptions)
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

func (d *Delete) Run(fc client.FnClient, cmd *cobra.Command) (err error) {
	var out string
	if d.DryRun {
		out = "deleted(dry run)"
	}

	for _, as := range d.atlas {
		err := fc.Namespace(as.Namespace).Delete(context.Background(), as.Name, d.options)
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

func getAtlasFromFileOpntion(cmd *cobra.Command, fo resource.FilenameOptions) ([]atlas, error) {
	fns, err := getFromFilenameOptions(cmd, fo)
	if err != nil {
		return nil, err
	}
	return functionListToAtlas(fns), nil
}

func functionListToAtlas(fns []*openfunction.Function) []atlas {
	as := make([]atlas, 0, len(fns))
	for _, fn := range fns {
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

func (d *Delete) listAtlas(fc client.FnClient) ([]atlas, error) {
	options := metav1.ListOptions{
		LabelSelector: d.deleteFlag.LabelSelector,
		FieldSelector: d.deleteFlag.FieldSelector,
	}

	fnList, err := fc.List(context.Background(), d.NamespaceIfScoped, options)
	if err != nil {
		return nil, err
	}

	as := make([]atlas, 0, len(fnList.Items))
	for _, fn := range fnList.Items {
		as = append(as, toAtlas(fn.ObjectMeta))
	}
	return as, nil
}
