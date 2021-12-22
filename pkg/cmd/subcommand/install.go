package subcommand

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/util/version"

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

var (
	w io.Writer
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
	Timeout             time.Duration
	openFunctionVersion *version.Version
}

func init() {
	w = os.Stdout
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
		Use:                   "install [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Install OpenFunction and its dependencies.",
		Long:                  "This command will help you to install OpenFunction and its dependencies.",
		Example: `
# Install OpenFunction with all dependencies
ofn install --all

# Install OpenFunction with a specific runtime
ofn install --async

# For users who have limited access to gcr.io or github.com to install OpenFunction
ofn install --region-cn --all

# Install a specific version of OpenFunction (default is v0.4.0)
ofn install --all --version v0.4.0

# See more at: https://github.com/OpenFunction/cli/blob/main/docs/install.md
`,
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
	cmd.Flags().BoolVar(&i.WithAsyncRuntime, "async", false, "For installing OpenFunction Async Runtime (Dapr & Keda).")
	cmd.Flags().BoolVar(&i.WithSyncRuntime, "sync", false, "For installing OpenFunction Sync Runtime (To be supported).")
	cmd.Flags().BoolVar(&i.WithAll, "all", false, "For installing all dependencies.")
	cmd.Flags().BoolVar(&i.RegionCN, "region-cn", false, "For users who have limited access to gcr.io or github.com.")
	cmd.Flags().BoolVar(&i.DryRun, "dry-run", false, "Used to prompt for the components and their versions to be installed by the current command.")
	cmd.Flags().BoolVar(&i.WithUpgrade, "upgrade", false, "Upgrade components to target version while installing.")
	cmd.Flags().StringVar(&i.OpenFunctionVersion, "version", "v0.4.0", "Used to specify the version of OpenFunction to be installed.")
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

func (i *Install) ValidateArgs(cmd *cobra.Command, args []string) error {
	ti := util.NewTaskInformer("")

	if i.OpenFunctionVersion == common.LatestVersion {
		return nil
	}

	v, err := version.ParseGeneric(i.OpenFunctionVersion)
	if err != nil {
		return errors.New(ti.TaskFail(fmt.Sprintf(
			"the specified version %s is not a valid version",
			i.OpenFunctionVersion,
		)))
	}

	if valid, err := common.IsVersionValid(v); err != nil {
		return errors.New(ti.TaskFail(err.Error()))
	} else {
		if !valid {
			return errors.New(ti.TaskFail(fmt.Sprintf(
				"the specified version %s is lower than the supported version %s",
				i.OpenFunctionVersion,
				common.BaseVersion,
			)))
		}
	}

	i.openFunctionVersion = v

	return nil
}

func (i *Install) RunInstall(cl *k8s.Clientset, cmd *cobra.Command) error {
	operator := common.NewOperator(runtime.GOOS, i.OpenFunctionVersion, i.Timeout, i.RegionCN, i.Verbose)
	ti := util.NewTaskInformer("")
	continueFunc := func() bool {
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprintln(w, ti.BeforeTask("You have used the `--upgrade` parameter, which means that the installation process "+
			"will overwrite the components that already exist.\n"+
			"Please ensure that you understand the meaning of this command "+
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
	operator.Inventory = inventoryPending
	inventoryExist := getExistComponentsInventory(ctx, cl)

	fmt.Fprintln(w, ti.BeforeTask("Start installing OpenFunction and its dependencies.\n"+
		"Here are the components and corresponding versions to be installed:"))
	printInventory(inventory.GetVersionMap(inventoryPending))
	if !reflect.DeepEqual(inventoryExist, map[string]bool{}) {
		fmt.Fprintln(w, ti.BeforeTask("The following components already exist:"))
		for i, exist := range inventoryExist {
			if exist {
				fmt.Fprintln(w, ti.BeforeTask(fmt.Sprintf("\t- %s", i)))
			}
		}
	}

	if i.DryRun {
		return nil
	}

	if i.WithUpgrade {
		if !continueFunc() {
			return nil
		}
	}

	// Record the list of components
	// that currently exist in the cluster.
	if _, err := operator.GetInventoryRecord(ctx, false); err != nil {
		return errors.Wrap(err, "failed to get inventory record")
	}
	defer operator.RecordInventory(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		done()
	}()

	grp, gctx := errgroup.WithContext(ctx)

	start := time.Now()

	if i.WithDapr {
		// If Dapr already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.DaprName] || i.WithUpgrade {
			grp.Go(func() error {
				return i.installDapr(gctx, operator)
			})
		} else {
			fmt.Fprintln(w, ti.SkipTask(inventory.DaprName))
		}
	}

	if i.WithKeda {
		// If Keda already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.KedaName] || i.WithUpgrade {
			grp.Go(func() error {
				return i.installKeda(gctx, cl, operator)
			})
		} else {
			fmt.Fprintln(w, ti.SkipTask("Keda"))
		}
	}

	if i.WithKnative {
		// If Knative Serving already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.KnativeServingName] || i.WithUpgrade {
			grp.Go(func() error {
				return i.installKnativeServing(gctx, cl, operator)
			})
		} else {
			fmt.Fprintln(w, ti.SkipTask(inventory.KnativeServingName))
		}
	}

	if i.WithShipWright {
		grp.Go(func() error {
			return i.installShipwright(gctx, cl, operator)
		})
	}

	if i.WithCertManager {
		// If Cert Manager already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.CertManagerName] || i.WithUpgrade {
			grp.Go(func() error {
				return i.installCertManager(gctx, cl, operator)
			})
		} else {
			fmt.Fprintln(w, ti.SkipTask(inventory.CertManagerName))
		}
	}

	if i.WithIngress {
		// If Ingress Nginx already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.IngressName] || i.WithUpgrade {
			grp.Go(func() error {
				return i.installIngress(gctx, cl, operator)
			})
		} else {
			fmt.Fprintln(w, ti.SkipTask(inventory.IngressName))
		}
	}

	if err := grp.Wait(); err != nil {
		return errors.New(ti.TaskFail(err.Error()))
	}

	if err := i.installOpenFunction(ctx, cl, operator); err != nil {
		return errors.New(ti.TaskFail(err.Error()))
	}

	end := time.Since(start)
	fmt.Fprintln(w, ti.AllDone(end))

	if i.WithKnative {
		ti.TipsOnUsingKnative()
	}

	ti.PrintOpenFunction()
	return nil
}

func (i *Install) mergeConditions() {
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

	if i.OpenFunctionVersion == common.LatestVersion {
		i.WithCertManager = true
	} else {
		if i.openFunctionVersion.Major() == 0 && i.openFunctionVersion.Minor() == 3 {
			i.WithIngress = false
			i.WithCertManager = false
		}

		if i.openFunctionVersion.Major() == 0 && i.openFunctionVersion.Minor() == 4 {
			i.WithIngress = false
			i.WithCertManager = true
		}
	}
}

func getExistComponentsInventory(ctx context.Context, cl *k8s.Clientset) map[string]bool {
	// We assume that a component exists when its deployment is in the available state.
	m := map[string]bool{}

	if exist := common.IsComponentExist(ctx, cl, common.DaprNamespace, "dapr-operator"); exist {
		m[inventory.DaprName] = true
	}

	if exist := common.IsComponentExist(ctx, cl, common.KedaNamespace, "keda-operator"); exist {
		m[inventory.KedaName] = true
	}

	if exist := common.IsComponentExist(ctx, cl, common.KnativeServingNamespace, "controller"); exist {
		m[inventory.KnativeServingName] = true
	}

	if exist := common.IsComponentExist(ctx, cl, common.KourierNamespace, "3scale-kourier-gateway"); exist {
		m[inventory.KourierName] = true
	}

	if exist := common.IsComponentExist(ctx, cl, common.KnativeServingNamespace, "default-domain"); exist {
		m[inventory.ServingDefaultDomainName] = true
	}

	if exist := common.IsComponentExist(ctx, cl, common.TektonPipelineNamespace, "tekton-pipelines-controller"); exist {
		m[inventory.TektonPipelinesName] = true
	}

	if exist := common.IsComponentExist(ctx, cl, common.ShipwrightNamespace, "shipwright-build-controller"); exist {
		m[inventory.ShipwrightName] = true
	}

	if exist := common.IsComponentExist(ctx, cl, common.CertManagerNamespace, "cert-manager"); exist {
		m[inventory.CertManagerName] = true
	}

	if exist := common.IsComponentExist(ctx, cl, common.IngressNginxNamespace, "ingress-nginx-controller"); exist {
		m[inventory.IngressName] = true
	}

	return m
}

func printInventory(inventory map[string]string) {
	ti := util.NewTaskInformer("DRYRUN")
	ti.PrintTable(inventory)
}

func (i *Install) installDapr(ctx context.Context, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("DAPR")

	v := operator.Inventory[inventory.DaprName].GetVersion()

	fmt.Fprintln(w, ti.TaskInfo("Installing Dapr..."))
	fmt.Fprintln(w, ti.TaskInfo("Downloading Dapr Cli binary..."))
	if err := operator.DownloadDaprClient(ctx, v); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to download Dapr client"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Initializing Dapr with Kubernetes mode..."))
	if err := operator.InitDapr(ctx, v); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Dapr"))
	}

	// Record the version of Dapr
	operator.Records.Dapr = v

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installKeda(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("KEDA")

	fmt.Fprintln(w, ti.TaskInfo("Installing Keda..."))

	v := operator.Inventory[inventory.KedaName].GetVersion()
	yamls, err := operator.Inventory[inventory.KedaName].GetYamlFile(v)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.InstallKeda(ctx, yamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Keda"))
	}

	// Record the version of Keda
	operator.Records.Keda = v

	fmt.Fprintln(w, ti.TaskInfo("Checking if Keda is ready..."))
	if err := operator.CheckKedaIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Keda readiness"))
	}

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installKnativeServing(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("KNATIVE")

	fmt.Fprintln(w, ti.TaskInfo("Installing Knative Serving..."))

	knv := operator.Inventory[inventory.KnativeServingName].GetVersion()
	knYamls, err := operator.Inventory[inventory.KnativeServingName].GetYamlFile(knv)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.InstallKnativeServing(ctx, knYamls["CRD"], knYamls["CORE"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Knative Serving"))
	}

	// Record the version of KnativeServing
	operator.Records.KnativeServing = knv

	fmt.Fprintln(w, ti.TaskInfo("Checking if Knative Serving is ready..."))
	if err := operator.CheckKnativeServingIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Knative Serving readiness"))
	}

	fmt.Fprintln(w, ti.TaskInfo("Configuring Knative Serving's DNS..."))

	ddv := operator.Inventory[inventory.ServingDefaultDomainName].GetVersion()
	ddYamls, err := operator.Inventory[inventory.ServingDefaultDomainName].GetYamlFile(ddv)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}
	if err := operator.ConfigKnativeServingDefaultDomain(ctx, ddYamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to config Knative Serving's DNS"))
	}

	// Record the version of DefaultDomain
	operator.Records.DefaultDomain = ddv

	fmt.Fprintln(w, ti.TaskInfo("Installing Kourier as Knative's gateway..."))

	krv := operator.Inventory[inventory.KourierName].GetVersion()
	krYamls, err := operator.Inventory[inventory.KourierName].GetYamlFile(krv)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}
	if err := operator.InstallKourier(ctx, cl, krYamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Kourier"))
	}

	// Record the version of Kourier
	operator.Records.Kourier = krv

	fmt.Fprintln(w, ti.TaskInfo("Checking if Kourier is ready..."))
	if err := operator.CheckKourierIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Kourier readiness"))
	}

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installShipwright(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("SHIPWRIGHT")

	fmt.Fprintln(w, ti.TaskInfo("Installing Tekton Pipelines..."))

	tkv := operator.Inventory[inventory.TektonPipelinesName].GetVersion()
	tkYamls, err := operator.Inventory[inventory.TektonPipelinesName].GetYamlFile(tkv)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}
	if err := operator.InstallTektonPipelines(ctx, tkYamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Tekton Pipelines"))
	}

	// Record the version of TektonPipelines
	operator.Records.TektonPipelines = tkv

	fmt.Fprintln(w, ti.TaskInfo("Checking if Tekton Pipelines is ready..."))
	if err := operator.CheckTektonPipelinesIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Tekton Pipelines readiness"))
	}

	fmt.Fprintln(w, ti.TaskInfo("Installing Shipwright..."))

	swv := operator.Inventory[inventory.ShipwrightName].GetVersion()
	swYamls, err := operator.Inventory[inventory.ShipwrightName].GetYamlFile(swv)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.InstallShipwright(ctx, swYamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Shipwright"))
	}

	// Record the version of Shipwright
	operator.Records.Shipwright = swv

	fmt.Fprintln(w, ti.TaskInfo("Checking if Shipwright is ready..."))
	if err := operator.CheckShipwrightIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Shipwright readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installCertManager(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("CERTMANAGER")

	fmt.Fprintln(w, ti.TaskInfo("Installing Cert Manager..."))

	v := operator.Inventory[inventory.CertManagerName].GetVersion()
	yamls, err := operator.Inventory[inventory.CertManagerName].GetYamlFile(v)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.InstallCertManager(ctx, yamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Cert Manager"))
	}

	// Record the version of CertManager
	operator.Records.CertManager = v

	fmt.Fprintln(w, ti.TaskInfo("Checking if Cert Manager is ready..."))
	if err := operator.CheckCertManagerIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Cert Manager readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installIngress(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("INGRESS")

	fmt.Fprintln(w, ti.TaskInfo("Installing Ingress..."))

	v := operator.Inventory[inventory.IngressName].GetVersion()
	yamls, err := operator.Inventory[inventory.IngressName].GetYamlFile(v)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	if err := operator.InstallIngressNginx(ctx, yamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Ingress"))
	}

	// Record the version of Ingress
	operator.Records.Ingress = v

	fmt.Fprintln(w, ti.TaskInfo("Checking if Ingress is ready..."))
	if err := operator.CheckIngressNginxIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Ingress Nginx readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Install) installOpenFunction(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	ti := util.NewTaskInformer("OPENFUNCTION")

	v := operator.Inventory[inventory.OpenFunctionName].GetVersion()
	yamls, err := operator.Inventory[inventory.OpenFunctionName].GetYamlFile(v)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}

	fmt.Fprintln(w, ti.TaskInfo("Installing OpenFunction..."))
	if err := operator.InstallOpenFunction(ctx, yamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install OpenFunction"))
	}

	// Record the version of OpenFunction
	operator.Records.OpenFunction = v

	fmt.Fprintln(w, ti.TaskInfo("Checking if OpenFunction is ready..."))
	if err := operator.CheckOpenFunctionIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check OpenFunction readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}
