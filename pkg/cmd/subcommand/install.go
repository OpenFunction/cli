package subcommand

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

const (
	OpenFunctionDaprVersion            = "OPENFUNCTION_DAPR_VERSION"
	DefaultDaprVersion                 = "1.4.3"
	OpenFunctionKedaVersion            = "OPENFUNCTION_KEDA_VERSION"
	DefaultKedaVersion                 = "2.4.0"
	OpenFunctionKnativeServingVersion  = "OPENFUNCTION_KNATIVE_SERVING_VERSION"
	DefaultKnativeServingVersion       = "0.26.0"
	OpenFunctionShipwrightVersion      = "OPENFUNCTION_SHIPWRIGHT_VERSION"
	DefaultShipwrightVersion           = "0.6.0"
	OpenFunctionTektonPipelinesVersion = "OPENFUNCTION_TEKTON_PIPELINES_VERSION"
	DefaultTektonPipelinesVersion      = "v0.28.1"
	OpenFunctionCertManagerVersion     = "OPENFUNCTION_CERT_MANAGER_VERSION"
	DefaultCertManagerVersion          = "v1.5.4"
	OpenFunctionIngressNginxVersion    = "OPENFUNCTION_INGRESS_NGINX_VERSION"
	DefaultIngressNginxVersion         = "1.1.0"
)

var (
	w                 io.Writer
	availableVersions []string
)

// Install is the commandline for 'init' sub command
type Install struct {
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
	WithUpgrade         bool
}

func init() {
	w = os.Stdout
	availableVersions = []string{"v0.3.1", "v0.4.0", "latest"}
}

// NewInstall returns an initialized Init instance
func NewInstall(ioStreams genericclioptions.IOStreams) *Install {
	return &Install{
		IOStreams: ioStreams,
	}
}

func NewCmdInstall(restClient util.Getter, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var cl *k8s.Clientset

	i := NewInstall(ioStreams)

	cmd := &cobra.Command{
		Use:                   "install",
		DisableFlagsInUseLine: true,
		Short:                 "Install OpenFunction and its dependencies.",
		Long: `
This command will help you to install OpenFunction and its dependencies.

You can use fn install --all to install all components.

The dependencies to be installed for OpenFunction v0.3.1 are: Dapr, KEDA, Knative Serving, Shipwright, Tekton Pipelines.

The permitted parameters are: --sync, --async, --knative, --shipwright, --version, --verbose, --dry-run

The dependencies to be installed for OpenFunction v0.4.0 are: Dapr, KEDA, Knative Serving, Shipwright, Tekton Pipelines, Cert Manager, Ingress Nginx.

The permitted parameters are: --sync, --async, --knative, --shipwright, --cert-manager, --ingress, --version, --verbose, --dry-run
`,
		Example: "fn install --all",
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			_, cl, err = client.NewKubeConfigClient()
			if err != nil {
				return err
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(i.ValidateArgs(cmd, args))
			util.CheckErr(i.RunInstall(cl, cmd))
		},
	}

	cmd.Flags().BoolVar(&i.Verbose, "verbose", false, "Show verbose information.")
	cmd.Flags().BoolVar(&i.WithDapr, "dapr", false, "For installing Dapr.")
	cmd.Flags().BoolVar(&i.WithKeda, "keda", false, "For installing Keda.")
	cmd.Flags().BoolVar(&i.WithKnative, "knative", false, "For installing Knative Serving (with Kourier as default gateway).")
	cmd.Flags().BoolVar(&i.WithShipWright, "shipwright", false, "For installing ShipWright.")
	cmd.Flags().BoolVar(&i.WithCertManager, "cert-manager", false, "For installing Cert Manager.")
	cmd.Flags().BoolVar(&i.WithIngress, "ingress", false, "For installing Ingress Nginx.")
	cmd.Flags().BoolVar(&i.WithAsyncRuntime, "async", false, "For installing OpenFunction Async Runtime(Dapr & Keda).")
	cmd.Flags().BoolVar(&i.WithSyncRuntime, "sync", false, "For installing OpenFunction Sync Runtime(Knative).")
	cmd.Flags().BoolVar(&i.WithAll, "all", false, "For installing all dependencies.")
	cmd.Flags().BoolVar(&i.RegionCN, "region-cn", false, "For users in China to speed up the download process of dependent components.")
	cmd.Flags().BoolVar(&i.DryRun, "dry-run", false, "Used to prompt for the components and their versions to be installed by the current command.")
	cmd.Flags().BoolVar(&i.WithUpgrade, "upgrade", false, "Upgrade components to target version while installing.")
	cmd.Flags().StringVar(&i.OpenFunctionVersion, "version", "v0.4.0", "Used to specify the version of OpenFunction to be installed. The permitted versions are: v0.3.1, v0.4.0, latest.")
	return cmd
}

func (i *Install) ValidateArgs(cmd *cobra.Command, args []string) error {
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

func (i *Install) RunInstall(cl *k8s.Clientset, cmd *cobra.Command) error {
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
	inventoryExist := getExistComponentsInventory(ctx, cl)

	fmt.Fprintln(w, ti.BeforeTask("Start installing OpenFunction and its dependencies.\n"+
		"Here are the components and corresponding versions to be installed for this installation:"))
	printInventory(inventoryPending)
	fmt.Fprintln(w, ti.BeforeTask("The following components already exist in the current cluster:"))
	printInventory(inventoryExist)

	if i.DryRun {
		return nil
	}

	if i.WithUpgrade {
		if !continueFunc() {
			return nil
		}
	}

	grp1, g1ctx := errgroup.WithContext(ctx)
	defer g1ctx.Done()

	start := time.Now()

	if i.WithDapr {
		// If the Dapr component already exists in the cluster
		// and the `--upgrade` parameter is not used, skip this step.
		if exist := inventoryExist["Dapr"]; exist == "" || i.WithUpgrade {
			grp1.Go(func() error {
				return i.installDapr(g1ctx, operator)
			})
		}
		fmt.Fprintln(w, ti.SkipTask("Dapr"))
	}

	if i.WithKeda {
		// If the Keda component already exists in the cluster
		// and the `--upgrade` parameter is not used, skip this step.
		if exist := inventoryExist["Keda"]; exist == "" || i.WithUpgrade {
			grp1.Go(func() error {
				return i.installKeda(g1ctx, cl, operator)
			})
		}
		fmt.Fprintln(w, ti.SkipTask("Keda"))
	}

	if i.WithKnative {
		// If the Knative Serving component already exists in the cluster
		// and the `--upgrade` parameter is not used, skip this step.
		if exist := inventoryExist["Knative Serving"]; exist == "" || i.WithUpgrade {
			grp1.Go(func() error {
				return i.installKnativeServing(g1ctx, cl, operator)
			})
		}
		fmt.Fprintln(w, ti.SkipTask("Knative Serving & Kourier"))
	}

	if i.WithShipWright {
		// We must install Shipwright(and Tekton Pipelines)
		// with or without the `--upgrade` parameter.
		grp1.Go(func() error {
			return i.installShipwright(g1ctx, cl, operator)
		})
	}

	if i.WithCertManager && i.OpenFunctionVersion != "v0.3.1" {
		// If the Cert Manager component already exists in the cluster
		// and the `--upgrade` parameter is not used, skip this step.
		if exist := inventoryExist["Cert Manager"]; exist == "" || i.WithUpgrade {
			grp1.Go(func() error {
				return i.installCertManager(g1ctx, cl, operator)
			})
		}
		fmt.Fprintln(w, ti.SkipTask("Cert Manager"))
	}

	if i.WithIngress && i.OpenFunctionVersion != "v0.3.1" {
		// If the Ingress Nginx component already exists in the cluster
		// and the `--upgrade` parameter is not used, skip this step.
		if exist := inventoryExist["Ingress Nginx"]; exist == "" || i.WithUpgrade {
			grp1.Go(func() error {
				return i.installIngress(g1ctx, cl, operator)
			})
		}
		fmt.Fprintln(w, ti.SkipTask("Ingress Nginx"))
	}

	if err := grp1.Wait(); err != nil {
		return errors.New(ti.TaskFail(err.Error()))
	}

	grp2, g2ctx := errgroup.WithContext(ctx)
	defer g2ctx.Done()

	grp2.Go(func() error {
		return i.installOpenFunction(g2ctx, cl, operator)
	})

	if err := grp2.Wait(); err != nil {
		return errors.New(ti.TaskFail(err.Error()))
	}

	end := time.Since(start)
	fmt.Fprintln(w, ti.AllDone(end))
	return nil
}

func (i *Install) checkConditionsAndGetInventory() map[string]string {
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

func getExistComponentsInventory(ctx context.Context, cl *k8s.Clientset) map[string]string {
	// The components of Shipwright and OpenFunction don't have the "app.kubernetes.io/version" label yet.
	inventory := map[string]string{
		"Dapr":             common.GetExistComponentVersion(ctx, cl, common.DaprNamespace, "dapr-operator"),
		"Keda":             common.GetExistComponentVersion(ctx, cl, common.KedaNamespace, "keda-operator"),
		"Knative Serving":  common.GetExistComponentVersion(ctx, cl, common.KnativeServingNamespace, "controller"),
		"Ingress Nginx":    common.GetExistComponentVersion(ctx, cl, common.IngressNginxNamespace, "ingress-nginx-controller"),
		"Cert Manager":     common.GetExistComponentVersion(ctx, cl, common.CertManagerNamespace, "cert-manager"),
		"Tekton Pipelines": common.GetExistComponentVersion(ctx, cl, common.TektonPipelineNamespace, "tekton-pipelines-controller"),
	}

	return inventory
}

func printInventory(inventory map[string]string) {
	ti := util.NewTaskInformer("DRYRUN")
	ti.PrintTable(inventory)
}

func (i *Install) installDapr(ctx context.Context, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 5*time.Minute)
	defer done()

	ti := util.NewTaskInformer("DAPR")

	fmt.Fprintln(w, ti.TaskInfo("Installing Dapr..."))
	fmt.Fprintln(w, ti.TaskInfo("Downloading Dapr Cli binary..."))
	if err := operator.DownloadDaprClient(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to download Dapr client"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Initializing Dapr with Kubernetes mode..."))
	if err := operator.InitDapr(ctx); err != nil {
		if !strings.Contains(err.Error(), "still in use") {
			return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Dapr"))
		}
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installKeda(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("KEDA")

	fmt.Fprintln(w, ti.TaskInfo("Installing Keda..."))
	if err := operator.InstallKeda(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Keda"))
	}

	fmt.Fprintln(w, ti.TaskInfo("Checking if Keda is ready..."))
	if err := operator.CheckKedaIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Keda readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installKnativeServing(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("KNATIVE")

	fmt.Fprintln(w, ti.TaskInfo("Installing Knative Serving..."))
	if err := operator.InstallKnativeServing(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Knative Serving"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Installing Kourier as Knative's gateway..."))
	if err := operator.InstallKourier(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Kourier"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Configuring Knative Serving's DNS..."))
	if err := operator.ConfigKnativeServingDefaultDomain(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to config Knative Serving's DNS"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Checking if Knative Serving is ready..."))
	if err := operator.CheckKnativeServingIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Knative Serving readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installShipwright(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("SHIPWRIGHT")

	fmt.Fprintln(w, ti.TaskInfo("Installing Shipwright..."))
	if err := operator.InstallShipwright(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Shipwright"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Checking if Shipwright is ready..."))
	if err := operator.CheckShipwrightIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Shipwright readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installCertManager(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("CERTMANAGER")

	fmt.Fprintln(w, ti.TaskInfo("Installing Cert Manager..."))
	if err := operator.InstallCertManager(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Cert Manager"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Checking if Cert Manager is ready..."))
	if err := operator.CheckCertManagerIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Cert Manager readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installIngress(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("INGRESS")

	fmt.Fprintln(w, ti.TaskInfo("Installing Ingress..."))
	if err := operator.InstallIngressNginx(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Ingress"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Checking if Ingress is ready..."))
	if err := operator.CheckIngressNginxIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Ingress Nginx readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installOpenFunction(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithTimeout(ctx, 10*time.Minute)
	defer done()

	ti := util.NewTaskInformer("OPENFUNCTION")

	fmt.Fprintln(w, ti.TaskInfo("Installing OpenFunction..."))
	if err := operator.InstallOpenFunction(ctx); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install OpenFunction"))
		}
	}
	fmt.Fprintln(w, ti.TaskInfo("Checking if OpenFunction is ready..."))
	if err := operator.CheckOpenFunctionIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check OpenFunction readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}
