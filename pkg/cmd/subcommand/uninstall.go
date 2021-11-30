package subcommand

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/OpenFunction/cli/pkg/dependency/common"
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
}

func init() {
	w = os.Stdout
	availableVersions = []string{"v0.4.0", "latest"}
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
		Use:                   "uninstall",
		DisableFlagsInUseLine: true,
		Short:                 "Uninstall OpenFunction and its dependencies.",
		Long: `
This command will help you to uninstall OpenFunction and its dependencies.

You can use fn uninstall --all to uninstall all components.

The dependencies to be uninstalled for OpenFunction v0.3.1 are: Dapr, KEDA, Knative Serving, Shipwright, Tekton Pipelines.

The permitted parameters are: --sync, --async, --knative, --shipwright, --version, --verbose, --dry-run

The dependencies to be uninstalled for OpenFunction v0.4.0 are: Dapr, KEDA, Knative Serving, Shipwright, Tekton Pipelines, Cert Manager, Ingress Nginx.

The permitted parameters are: --sync, --async, --knative, --shipwright, --cert-manager, --ingress, --version, --verbose, --dry-run
`,
		Example: "fn uninstall --all",
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
	cmd.Flags().BoolVar(&i.WithAsyncRuntime, "async", false, "For uninstalling OpenFunction Async Runtime(Dapr & Keda).")
	cmd.Flags().BoolVar(&i.WithSyncRuntime, "sync", false, "For uninstalling OpenFunction Sync Runtime(Knative).")
	cmd.Flags().BoolVar(&i.WithAll, "all", false, "For uninstalling all dependencies.")
	cmd.Flags().BoolVar(&i.RegionCN, "region-cn", false, "For users in China to uninstall dependent components.")
	cmd.Flags().BoolVar(&i.DryRun, "dry-run", false, "Used to prompt for the components and their versions to be uninstalled by the current command.")
	cmd.Flags().BoolVar(&i.WaitForCleared, "wait", false, "Awaiting the results of the uninstallation.")
	cmd.Flags().StringVar(&i.OpenFunctionVersion, "version", "v0.4.0", "Used to specify the version of OpenFunction to be uninstalled.")
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
	operator := common.NewOperator(runtime.GOOS, i.OpenFunctionVersion, i.RegionCN, i.Verbose)
	ti := util.NewTaskInformer("")
	continueFunc := func() bool {
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprintln(w, ti.BeforeTask("You can see the list of components to be installed and the list of components already exist in the cluster.\n"+
			"You have used the `--upgrade` parameter, which means that the installation process will overwrite the components that already exist in the cluster.\n"+
			"Make sure you know what happens when you do this.\n"+
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

	ctx, done := context.WithCancel(
		context.Background(),
	)
	defer done()

	inventoryPending := i.checkConditionsAndGetInventory()

	fmt.Fprintln(w, ti.BeforeTask("Start uninstalling OpenFunction and its dependencies.\n"+
		"Here are the components and corresponding versions to be uninstalled for this installation:"))
	printInventory(inventoryPending)

	if i.DryRun {
		return nil
	}
	if !continueFunc() {
		return nil
	}

	grp, gctx := errgroup.WithContext(ctx)
	defer gctx.Done()

	start := time.Now()

	if i.WithDapr {
		grp.Go(func() error {
			return i.uninstallDapr(gctx, cl, operator)
		})
	}

	if i.WithKeda {
		grp.Go(func() error {
			return i.uninstallKeda(gctx, cl, operator)
		})
	}

	if i.WithKnative {
		grp.Go(func() error {
			return i.uninstallKnativeServing(gctx, cl, operator)
		})
	}

	if i.WithShipWright {
		grp.Go(func() error {
			return i.uninstallShipwright(gctx, cl, operator)
		})
	}

	if i.WithCertManager && i.OpenFunctionVersion != "v0.3.1" {
		grp.Go(func() error {
			return i.uninstallCertManager(gctx, cl, operator)
		})
	}

	if i.WithIngress && i.OpenFunctionVersion != "v0.3.1" {
		grp.Go(func() error {
			return i.uninstallIngress(gctx, cl, operator)
		})
	}

	grp.Go(func() error {
		return i.uninstallOpenFunction(gctx, cl, operator)
	})

	if err := grp.Wait(); err != nil {
		return errors.New(ti.TaskFail(err.Error()))
	}

	end := time.Since(start)
	fmt.Fprintln(w, ti.AllDone(end))
	return nil
}

func (i *Uninstall) checkConditionsAndGetInventory() map[string]string {
	getVersionFromEnv := func(name string, defaultVersion string) string {
		val, ok := os.LookupEnv(name)
		if !ok {
			return defaultVersion
		} else {
			return val
		}
	}

	inventory := map[string]string{
		"OpenFunction": i.OpenFunctionVersion,
	}

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
	if i.WithSyncRuntime {
		i.WithKnative = true
	}

	if i.WithDapr {
		inventory["Dapr"] = getVersionFromEnv(OpenFunctionDaprVersion, DefaultDaprVersion)
	}

	if i.WithKeda {
		inventory["Keda"] = getVersionFromEnv(OpenFunctionKedaVersion, DefaultKedaVersion)
	}

	if i.WithKnative {
		inventory["Knative Serving"] = getVersionFromEnv(OpenFunctionKnativeServingVersion, DefaultKnativeServingVersion)
	}

	if i.WithCertManager {
		inventory["Cert Manager"] = getVersionFromEnv(OpenFunctionCertManagerVersion, DefaultCertManagerVersion)
	}

	if i.WithIngress {
		inventory["Ingress Nginx"] = getVersionFromEnv(OpenFunctionIngressNginxVersion, DefaultIngressNginxVersion)
	}

	if i.WithShipWright {
		inventory["Tekton Pipelines"] = getVersionFromEnv(OpenFunctionTektonPipelinesVersion, DefaultTektonPipelinesVersion)
		inventory["Shipwright"] = getVersionFromEnv(OpenFunctionShipwrightVersion, DefaultShipwrightVersion)
	}

	return inventory
}

func (i *Uninstall) uninstallDapr(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("DAPR")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Dapr with Kubernetes mode..."))
	if err := operator.UninstallDapr(ctx, cl, i.WaitForCleared); util.IgnoreNotFoundErr(err) != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Dapr"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallKeda(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("KEDA")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Keda..."))
	if err := operator.UninstallKeda(ctx, cl, i.WaitForCleared); util.IgnoreNotFoundErr(err) != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Keda"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallKnativeServing(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("KNATIVE")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Knative Serving..."))
	if err := operator.UninstallKnativeServing(ctx, cl, i.WaitForCleared); util.IgnoreNotFoundErr(err) != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Knative Serving"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Kourier..."))
	if err := operator.UninstallKourier(ctx, cl, i.WaitForCleared); util.IgnoreNotFoundErr(err) != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Kourier"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallShipwright(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("SHIPWRIGHT")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Tekton Pipeline & Shipwright..."))
	if err := operator.UninstallShipwright(ctx, cl, i.WaitForCleared); util.IgnoreNotFoundErr(err) != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Tekton Pipeline"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallCertManager(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("CERTMANAGER")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Cert Manager..."))
	if err := operator.UninstallCertManager(ctx, cl, i.WaitForCleared); util.IgnoreNotFoundErr(err) != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Cert Manager"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallIngress(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("INGRESS")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling Ingress..."))
	if err := operator.UninstallIngressNginx(ctx, cl, i.WaitForCleared); util.IgnoreNotFoundErr(err) != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall Ingress"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Uninstall) uninstallOpenFunction(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("OPENFUNCTION")

	fmt.Fprintln(w, ti.TaskInfo("Uninstalling OpenFunction..."))
	if err := operator.UninstallOpenFunction(ctx, cl, i.WaitForCleared); util.IgnoreNotFoundErr(err) != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to uninstall OpenFunction"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}
