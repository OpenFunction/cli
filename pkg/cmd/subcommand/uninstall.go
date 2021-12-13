package subcommand

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/OpenFunction/cli/pkg/components/common"
	"github.com/OpenFunction/cli/pkg/components/inventory"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8s "k8s.io/client-go/kubernetes"
)

// Uninstall is the commandline for 'init' sub command
type Uninstall struct {
	genericclioptions.IOStreams

	Verbose             bool
	WithDapr            bool
	WithKeda            bool
	WithKnative         bool
	WithShipWright      bool
	WithCertManager     bool
	WithIngress         bool
	WithAsyncRuntime    bool
	WithSyncRuntime     bool
	WithAll             bool
	RegionCN            bool
	OpenFunctionVersion string
	DryRun              bool
	WaitForCleared      bool
	Timeout             time.Duration
}

// NewUninstall returns an initialized Init instance
func NewUninstall(ioStreams genericclioptions.IOStreams) *Uninstall {
	return &Uninstall{
		IOStreams: ioStreams,
	}
}

func NewCmdUninstall(restClient util.Getter, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var cl *k8s.Clientset

	i := NewUninstall(ioStreams)

	cmd := &cobra.Command{
		Use:                   "uninstall <args>...",
		DisableFlagsInUseLine: true,
		Short:                 "Uninstall OpenFunction and its dependencies.",
		Long: `
This command will help you to uninstall OpenFunction and its dependencies.

You can use ofn uninstall --all to uninstall all components.

The dependencies to be uninstalled for OpenFunction v0.3.1 are: Dapr, KEDA, Knative Serving, Shipwright, Tekton Pipelines.

The permitted parameters are: --async, --knative, --shipwright, --version, --verbose, --dry-run

The dependencies to be uninstalled for OpenFunction v0.4.0 are: Dapr, KEDA, Knative Serving, Shipwright, Tekton Pipelines, Cert Manager, Ingress Nginx.

The permitted parameters are: --async, --knative, --shipwright, --cert-manager, --ingress, --version, --verbose, --dry-run
`,
		Example: "ofn uninstall --all",
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			_, cl, err = client.NewKubeConfigClient()
			if err != nil {
				return err
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(i.ValidateArgs(cmd, args))
			util.CheckErr(i.RunUninstall(cl, cmd))
		},
	}

	cmd.Flags().BoolVar(&i.Verbose, "verbose", false, "Show verbose information.")
	cmd.Flags().BoolVar(&i.WithDapr, "dapr", false, "For uninstalling Dapr.")
	cmd.Flags().BoolVar(&i.WithKeda, "keda", false, "For uninstalling KEDA.")
	cmd.Flags().BoolVar(&i.WithKnative, "knative", false, "For uninstalling Knative Serving (with Kourier as default gateway).")
	cmd.Flags().BoolVar(&i.WithShipWright, "shipwright", false, "For uninstalling ShipWright.")
	cmd.Flags().BoolVar(&i.WithCertManager, "cert-manager", false, "For uninstalling Cert Manager.")
	cmd.Flags().BoolVar(&i.WithIngress, "ingress", false, "For uninstalling Ingress Nginx.")
	cmd.Flags().BoolVar(&i.WithAsyncRuntime, "async", false, "For uninstalling OpenFunction Async Runtime (Dapr & Keda).")
	cmd.Flags().BoolVar(&i.WithSyncRuntime, "sync", false, "For uninstalling OpenFunction Sync Runtime (To be supported).")
	cmd.Flags().BoolVar(&i.WithAll, "all", false, "For uninstalling all dependencies.")
	cmd.Flags().BoolVar(&i.RegionCN, "region-cn", false, "For users in China to uninstall dependent components.")
	cmd.Flags().BoolVar(&i.DryRun, "dry-run", false, "Used to prompt for the components and their versions to be uninstalled by the current command.")
	cmd.Flags().BoolVar(&i.WaitForCleared, "wait", false, "Awaiting the results of the uninstallation.")
	cmd.Flags().StringVar(&i.OpenFunctionVersion, "version", "v0.4.0", "Used to specify the version of OpenFunction to be uninstalled.")
	cmd.Flags().DurationVar(&i.Timeout, "timeout", 5*time.Minute, "Set timeout time. Default is 5 minutes.")
	// In order to avoid too many options causing misunderstandings among users,
	// we have hidden the following parameters,
	// but you can still find their usage instructions in the documentation.
	cmd.Flags().MarkHidden("ingress")
	cmd.Flags().MarkHidden("cert-manager")
	cmd.Flags().MarkHidden("shipwright")
	cmd.Flags().MarkHidden("keda")
	cmd.Flags().MarkHidden("dapr")
	return cmd
}

func (i *Uninstall) ValidateArgs(cmd *cobra.Command, args []string) error {
	ti := util.NewTaskInformer("")

	if !util.IsInSlice(i.OpenFunctionVersion, availableVersions) {
		errMsg := fmt.Sprintf(
			"version %s is not in the list of available versions: %v",
			i.OpenFunctionVersion,
			strings.Join(availableVersions[:], ", "),
		)
		return errors.New(ti.TaskFail(errMsg))
	}
	return nil
}

func (i *Uninstall) RunUninstall(cl *k8s.Clientset, cmd *cobra.Command) error {
	operator := common.NewOperator(runtime.GOOS, i.OpenFunctionVersion, i.Timeout, i.RegionCN, i.Verbose)
	ti := util.NewTaskInformer("")
	continueFunc := func() bool {
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprintln(w, ti.BeforeTask("Please ensure that you understand the meaning of this command "+
			"and follow the prompts below to confirm the action.\n"+
			"Enter 'y' to continue and 'n' to abort:"))

		for {
			fmt.Fprint(w, ti.BeforeTask("-> "))
			text, _ := reader.ReadString('\n')
			// convert CRLF to LF
			text = strings.Replace(text, "\n", "", -1)

			if strings.Compare("y", text) == 0 {
				return true
			}
			if strings.Compare("n", text) == 0 {
				return false
			}
		}
	}

	ctx, done := context.WithTimeout(
		context.Background(),
		i.Timeout,
	)
	defer done()

	// Determine which components need to be enabled
	// and update all options to be consistent.
	i.mergeConditions()
	inventoryPending, err := inventory.GetInventory(
		cl,
		i.RegionCN,
		i.WithKnative,
		i.WithKeda,
		i.WithDapr,
		i.WithShipWright,
		i.WithCertManager,
		i.WithIngress,
		i.OpenFunctionVersion,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get pending inventory")
	}
	inventoryExist, err := operator.GetInventoryRecord(ctx, true)
	if err != nil {
		return errors.Wrap(err, "failed to get inventory record")
	}
	operator.Inventory = inventoryPending

	fmt.Fprintln(w, ti.BeforeTask("Start uninstalling OpenFunction and its dependencies."))
	fmt.Fprintln(w, ti.BeforeTask("The following components already exist:"))
	printInventory(inventoryExist)

	if i.DryRun {
		return nil
	}
	if !continueFunc() {
		return nil
	}

	// Record the list of components
	// that currently exist in the cluster.
	if _, err := operator.GetInventoryRecord(ctx, false); err != nil {
		return errors.Wrap(err, "failed to get inventory record")
	}
	defer operator.RecordInventory(ctx)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		done()
	}()

	grp, gctx := errgroup.WithContext(ctx)

	start := time.Now()

	if i.WithDapr {
		if operator.Records.Dapr != "" {
			grp.Go(func() error {
				return i.uninstallDapr(gctx, cl, operator)
			})
		}
	}

	if i.WithKeda {
		if operator.Records.Keda != "" {
			grp.Go(func() error {
				return i.uninstallKeda(gctx, cl, operator)
			})
		}
	}

	if i.WithKnative {
		if operator.Records.DefaultDomain != "" {
			grp.Go(func() error {
				return i.uninstallServingDefaultDomain(gctx, cl, operator)
			})
		}

		if operator.Records.Kourier != "" {
			grp.Go(func() error {
				return i.uninstallKourier(gctx, cl, operator)
			})
		}

		if operator.Records.KnativeServing != "" {
			grp.Go(func() error {
				return i.uninstallKnativeServing(gctx, cl, operator)
			})
		}
	}

	if i.WithShipWright {
		if operator.Records.Shipwright != "" {
			grp.Go(func() error {
				return i.uninstallShipwright(gctx, cl, operator)
			})
		}
		if operator.Records.TektonPipelines != "" {
			grp.Go(func() error {
				return i.uninstallTektonPipelines(gctx, cl, operator)
			})
		}
	}

	if i.WithCertManager {
		if operator.Records.CertManager != "" {
			grp.Go(func() error {
				return i.uninstallCertManager(gctx, cl, operator)
			})
		}
	}

	if i.WithIngress {
		if operator.Records.Ingress != "" {
			grp.Go(func() error {
				return i.uninstallIngress(gctx, cl, operator)
			})
		}
	}

	if operator.Records.OpenFunction != "" {
		grp.Go(func() error {
			return i.uninstallOpenFunction(gctx, cl, operator)
		})
	}

	if err := grp.Wait(); err != nil {
		return errors.New(ti.TaskFail(err.Error()))
	}

	end := time.Since(start)
	fmt.Fprintln(w, ti.AllDone(end))
	return nil
}

func (i *Uninstall) mergeConditions() {
	// Update the corresponding conditions when WithAll is true
	if i.WithAll {
		i.WithDapr = true
		i.WithKeda = true
		i.WithIngress = true
		i.WithKnative = true
		i.WithShipWright = true
		i.WithCertManager = true
	}

	// Update the corresponding conditions when WithAsyncRuntime is true
	if i.WithAsyncRuntime {
		i.WithDapr = true
		i.WithKeda = true
	}

	// Update the corresponding conditions when WithSyncRuntime is true
	//if i.WithSyncRuntime {
	//}
}

func (i *Uninstall) uninstallDapr(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("DAPR")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Dapr with Kubernetes mode..."))
	if err := operator.UninstallDapr(ctx, cl, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Dapr"))
	}

	// Reset version to null
	operator.Records.Dapr = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallKeda(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("KEDA")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Keda..."))

	yamls, err := operator.Inventory[inventory.KedaName].GetYamlFile(operator.Records.Keda)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.KedaNamespace, false, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Keda"))
	}

	// Reset version to null
	operator.Records.Keda = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallKnativeServing(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("KNATIVE")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Knative Serving..."))

	yamls, err := operator.Inventory[inventory.KnativeServingName].GetYamlFile(operator.Records.KnativeServing)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.UninstallKnativeServing(ctx, cl, yamls["CRD"], yamls["CORE"], i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Knative Serving"))
	}

	// Reset version to null
	operator.Records.KnativeServing = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallServingDefaultDomain(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("KNATIVE")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Serving Default Domain..."))

	yamls, err := operator.Inventory[inventory.ServingDefaultDomainName].GetYamlFile(operator.Records.DefaultDomain)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.KnativeServingNamespace, false, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Serving Default Domain"))
	}

	// Reset version to null
	operator.Records.DefaultDomain = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallKourier(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("KOURIER")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Kourier..."))

	yamls, err := operator.Inventory[inventory.KourierName].GetYamlFile(operator.Records.Kourier)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.KedaNamespace, false, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Kourier"))
	}

	// Reset version to null
	operator.Records.Kourier = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallShipwright(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("SHIPWRIGHT")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Shipwright..."))

	yamls, err := operator.Inventory[inventory.ShipwrightName].GetYamlFile(operator.Records.Shipwright)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.ShipwrightNamespace, false, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Shipwright"))
	}

	// Reset version to null
	operator.Records.Shipwright = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallTektonPipelines(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("TEKTON")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Tekton Pipeline..."))

	yamls, err := operator.Inventory[inventory.TektonPipelinesName].GetYamlFile(operator.Records.TektonPipelines)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.TektonPipelineNamespace, false, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Tekton Pipeline"))
	}

	// Reset version to null
	operator.Records.TektonPipelines = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallCertManager(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("CERTMANAGER")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Cert Manager..."))

	yamls, err := operator.Inventory[inventory.CertManagerName].GetYamlFile(operator.Records.CertManager)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.CertManagerNamespace, false, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Cert Manager"))
	}

	// Reset version to null
	operator.Records.CertManager = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallIngress(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("INGRESS")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Ingress..."))

	yamls, err := operator.Inventory[inventory.IngressName].GetYamlFile(operator.Records.Ingress)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.IngressNginxNamespace, false, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Ingress"))
	}

	// Reset version to null
	operator.Records.Ingress = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallOpenFunction(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("OPENFUNCTION")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling OpenFunction..."))

	yamls, err := operator.Inventory[inventory.OpenFunctionName].GetYamlFile(operator.Records.OpenFunction)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.OpenFunctionNamespace, false, i.WaitForCleared); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall OpenFunction"))
	}

	// Reset version to null
	operator.Records.OpenFunction = ""

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}
