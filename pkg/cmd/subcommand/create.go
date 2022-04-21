package subcommand

import (
	"context"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	cc "github.com/OpenFunction/cli/pkg/cmd/util/client"
	client "github.com/openfunction/pkg/client/clientset/versioned"
	"github.com/openfunction/pkg/client/clientset/versioned/scheme"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"

	fn "github.com/openfunction/apis/core/v1beta1"
	openfunction "github.com/openfunction/apis/core/v1beta1"
)

// Create is the commandline for 'create' sub command
type Create struct {
	genericclioptions.IOStreams
	Printer *util.Printer

	FilenameOptions resource.FilenameOptions
	DryRun          bool

	Name             string
	Image            string
	Version          string
	Port             int32
	ImageCredentials string
	Build            *openfunction.BuildImpl
	Serving          *openfunction.ServingImpl
	runtime          string

	namespace string

	options metav1.CreateOptions
	printer printers.ResourcePrinter
}

const (
	createExample = `
# Create a function using the data in function.yaml
ofn create -f function.yaml

# Create a function based on the YAML passed into stdin
cat function.yaml | ofn create -f -
`
)

// NewCreate returns an initialized Create instance
func NewCreate(ioStreams genericclioptions.IOStreams) *Create {
	return &Create{
		IOStreams: ioStreams,

		Printer: util.NewPrinter("create", scheme.Scheme),
		Build:   &fn.BuildImpl{},
		Serving: &fn.ServingImpl{},
	}
}

func NewCmdCreate(cf *genericclioptions.ConfigFlags, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var fc client.Interface

	c := NewCreate(ioStreams)
	cmd := &cobra.Command{
		Use:                   "create -f FILENAME",
		DisableFlagsInUseLine: true,
		Short:                 "Create a resource from a file or from stdin",
		Long: `
Create a resource from a file or from stdin
`,
		Example: createExample,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			config, err := cf.ToRESTConfig()
			if err != nil {
				panic(err)
			}
			cc.SetConfigDefaults(config)
			fc = client.NewForConfigOrDie(config)

			c.namespace, _, err = cf.ToRawKubeConfigLoader().Namespace()

			return err
		},

		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(c.Complete(cmd, args))
			util.CheckErr(c.Validate(cmd))
			util.CheckErr(c.RunCreate(fc, cmd, args))
		},
	}

	usage := "to use to create the function"
	AddFilenameOptionFlags(cmd, &c.FilenameOptions, usage)
	cmd.Flags().StringVarP(&c.Image, "image", "i", c.Image, "Function image name")
	cmd.Flags().StringVarP(&c.Version, "version", "v", c.Version, "Function version in format like v1.0.0")
	cmd.Flags().StringVarP(&c.ImageCredentials, "image-credentials", "", c.ImageCredentials, "ImageCredentials references a Secret that contains credentials to access the image repository")
	cmd.Flags().Int32VarP(&c.Port, "port", "", c.Port, "The port on which the function will be invoked")
	cmd.Flags().BoolVarP(&c.DryRun, "dry-run", "", c.DryRun, "Only print the object that would be sent, without sending it")
	cmd.Flags().StringVarP(&c.runtime, "runtime", "", "", "The configuration of the backend runtime for running function.")
	c.Printer.AddFlags(cmd)
	AddBuild(cmd, c.Build)
	AddServing(cmd, c.Serving)
	return cmd
}

func (c *Create) Complete(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		c.Name = args[0]
	}

	if c.DryRun {
		c.options.DryRun = []string{
			metav1.DryRunAll,
		}
	}

	c.Printer.SetPrinterFunc(util.WithDefaultPrinter(""))

	var err error
	c.printer, err = c.Printer.ToPrinter()
	if err != nil {
		return err
	}

	if c.runtime == "" {
		c.Serving = nil
	} else {
		c.Serving.Runtime = openfunction.Runtime(c.runtime)
	}

	if c.Build != nil {
		if c.Build.SrcRepo.Credentials.Name == "" {
			c.Build.SrcRepo.Credentials = nil
		}
	}
	return nil
}

func (c *Create) Validate(cmd *cobra.Command) error {
	if len(c.FilenameOptions.Filenames) != 0 {
		return nil
	}

	if c.Name == "" {
		return util.UsageErrorf(cmd, "a name is required")
	}
	if c.Version == "" {
		return util.UsageErrorf(cmd, "spec.version is required")
	}
	if c.Image == "" {
		return util.UsageErrorf(cmd, "spec.image is required")
	}

	return nil
}

func (c *Create) RunCreate(fc client.Interface, cmd *cobra.Command, args []string) error {
	var (
		fns []*openfunction.Function
		err error
	)
	if len(c.FilenameOptions.Filenames) != 0 {
		fns, err = getFromFilenameOptions(cmd, c.FilenameOptions)
	} else {
		fns = []*openfunction.Function{{
			ObjectMeta: metav1.ObjectMeta{
				Name: c.Name,
			},
			Spec: openfunction.FunctionSpec{
				Version: &c.Version,
				Image:   c.Image,
				ImageCredentials: &corev1.LocalObjectReference{
					Name: c.ImageCredentials,
				},
				Port:    &c.Port,
				Build:   c.Build,
				Serving: c.Serving,
			},
		},
		}
	}
	if err != nil {
		return err
	}

	for _, fn := range fns {
		result, err := c.create(fc, cmd, fn)
		if err != nil {
			return err
		}

		if c.printer != nil {
			if err = c.printer.PrintObj(result, c.Out); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Create) create(fc client.Interface, cmd *cobra.Command, fn *openfunction.Function) (*openfunction.Function, error) {
	opt := metav1.CreateOptions{}
	if c.DryRun {
		opt.DryRun = []string{metav1.DryRunAll}
	}

	if fn.Namespace == "" {
		fn.Namespace = c.namespace
	}

	result, err := fc.CoreV1beta1().Functions(fn.Namespace).Create(context.Background(), fn, opt)
	if err != nil {
		return nil, err
	}

	return result, nil
}
