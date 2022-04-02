package subcommand

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/OpenFunction/cli/pkg/cmd/util/spinners"
	"github.com/OpenFunction/cli/pkg/components/common"
	"github.com/OpenFunction/cli/pkg/components/inventory"
	"github.com/oliveagle/jsonpath"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8s "k8s.io/client-go/kubernetes"
)

const (
	openFunctionLatestReleaseUrl = "https://api.github.com/repos/OpenFunction/OpenFunction/releases/latest"
)

// Install is the commandline for 'install' sub command
type Install struct {
	genericclioptions.IOStreams

	Verbose             bool
	Runtimes            []string
	Ingress             string
	WithoutCI           bool
	WithDapr            bool
	WithKeda            bool
	WithKnative         bool
	WithShipWright      bool
	WithCertManager     bool
	WithIngressNginx    bool
	WithAll             bool
	RegionCN            bool
	OpenFunctionVersion string
	DryRun              bool
	Upgrade             bool
	Yes                 bool
	Timeout             time.Duration
	openFunctionVersion *version.Version
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
ofn install --runtime async 

# For users who have limited access to gcr.io or github.com to install OpenFunction
ofn install --region-cn --all

# Install a specific version of OpenFunction
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
			util.CheckErr(i.ValidateArgs())
			util.CheckErr(i.RunInstall(cl, cmd))
		},
	}

	cmd.PersistentFlags().StringSliceVarP(&i.Runtimes, "runtime", "r", []string{"knative"}, "List of runtimes to be installed, optionally \"knative\", \"async\".")
	cmd.PersistentFlags().StringVar(&i.Ingress, "ingress", "nginx", "The type of ingress controller to be installed, optionally \"nginx\".")
	cmd.Flags().BoolVar(&i.WithoutCI, "without-ci", false, "Skip the installation of CI components.")
	cmd.Flags().BoolVar(&i.Verbose, "verbose", false, "Show verbose information.")
	cmd.Flags().BoolVar(&i.WithDapr, "with-dapr", false, "For installing Dapr.")
	cmd.Flags().BoolVar(&i.WithKeda, "with-keda", false, "For installing Keda.")
	cmd.Flags().BoolVar(&i.WithKnative, "with-knative", false, "For installing Knative Serving (with Kourier as default gateway).")
	cmd.Flags().BoolVar(&i.WithIngressNginx, "with-ingress-nginx", false, "For installing Ingress Nginx.")
	cmd.Flags().BoolVar(&i.WithAll, "all", false, "For installing all dependencies.")
	cmd.Flags().BoolVar(&i.RegionCN, "region-cn", false, "For users who have limited access to gcr.io or github.com.")
	cmd.Flags().BoolVar(&i.DryRun, "dry-run", false, "Used to prompt for the components and their versions to be installed by the current command.")
	cmd.Flags().BoolVar(&i.Upgrade, "upgrade", false, "Upgrade components to target version while installing.")
	cmd.Flags().BoolVarP(&i.Yes, "yes", "y", false, "Automatic yes to prompts.")
	cmd.Flags().StringVar(&i.OpenFunctionVersion, "version", "", "Used to specify the version of OpenFunction to be installed.")
	cmd.Flags().DurationVar(&i.Timeout, "timeout", 10*time.Minute, "Set timeout time. Default is 10 minutes.")
	// In order to avoid too many options causing misunderstandings among users,
	// we have hidden the following parameters,
	// but you can still find their usage instructions in the documentation.
	cmd.Flags().MarkHidden("with-ingress-nginx")
	cmd.Flags().MarkHidden("with-keda")
	cmd.Flags().MarkHidden("with-dapr")
	cmd.Flags().MarkHidden("with-knative")
	return cmd
}

func (i *Install) ValidateArgs() error {
	if i.OpenFunctionVersion == common.LatestVersion {
		return nil
	}

	if i.OpenFunctionVersion == "" {
		v, e := getLatestStableVersion()
		if e != nil {
			return e
		}
		i.OpenFunctionVersion = v
	}

	v, err := version.ParseGeneric(i.OpenFunctionVersion)
	if err != nil {
		return errors.New(util.TaskFail(fmt.Sprintf(
			"the specified version %s is not a valid version",
			i.OpenFunctionVersion,
		)))
	}

	if valid, err := common.IsVersionValid(v); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	} else {
		if !valid {
			return errors.New(util.TaskFail(fmt.Sprintf(
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
	operator := common.NewOperator(runtime.GOOS, runtime.GOARCH, i.OpenFunctionVersion, i.Timeout, i.RegionCN, i.Verbose)
	continueFunc := func() bool {
		reader := bufio.NewReader(os.Stdin)
		util.BeforeTask("You have specified the `--upgrade` flag, which means that the installation process " +
			"will upgrade components currently installed.\n" +
			"Please make sure that you're aware of the consequences of this command " +
			"and follow the prompts below to confirm the upgrade.\n" +
			"Enter 'y' to continue and 'n' to abort:")

		for {
			fmt.Print(util.YellowItalic("-> "))
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
	if err := i.calculateConditions(); err != nil {
		return errors.Wrap(err, "failed to calculate conditions")
	}

	inventoryPending, err := inventory.GetInventory(
		cl,
		i.RegionCN,
		i.WithKnative,
		i.WithKeda,
		i.WithDapr,
		i.WithShipWright,
		i.WithCertManager,
		i.WithIngressNginx,
		i.OpenFunctionVersion,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get pending inventory")
	}
	operator.Inventory = inventoryPending
	inventoryExist := getExistComponentsInventory(ctx, cl)

	util.BeforeTask("Start installing OpenFunction and its dependencies.\n" +
		"The following components will be installed:")
	printInventory(inventory.GetVersionMap(inventoryPending))
	if !reflect.DeepEqual(inventoryExist, map[string]bool{}) && !i.Yes {
		if !i.Upgrade {
			util.BeforeTask("The following existing components will be skipped:")
		} else {
			util.BeforeTask("The following existing components will be upgraded:")
		}
		for idx, exist := range inventoryExist {
			if exist {
				util.BeforeTask(fmt.Sprintf("\t- %s", idx))
			}
		}
	}

	if i.DryRun {
		return nil
	}

	if i.Upgrade && !i.Yes {
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

	start := time.Now()

	grp1 := spinners.NewSpinnerGroup()
	count := 0

	if i.WithDapr {
		// If Dapr already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.DaprName] || i.Upgrade {
			count += 1
			grp1.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := grp1.At(idx).WithName("Dapr")
				installDapr(ctx, spinner, operator)
			}(ctx, count-1)
		}
	}

	if i.WithKeda {
		// If Keda already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.KedaName] || i.Upgrade {
			count += 1
			grp1.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := grp1.At(idx).WithName("Keda")
				installKeda(ctx, spinner, cl, operator)
			}(ctx, count-1)
		}
	}

	if i.WithKnative {
		// If Knative Serving already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.KnativeServingName] || i.Upgrade {
			count += 1
			grp1.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := grp1.At(idx).WithName("Knative Serving")
				installKnativeServing(ctx, spinner, cl, operator)
			}(ctx, count-1)
		}
	}

	if i.WithShipWright {
		// If Shipwright already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.ShipwrightName] || i.Upgrade {
			count += 1
			grp1.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := grp1.At(idx).WithName("Shipwright")
				installShipwright(ctx, spinner, cl, operator)
			}(ctx, count-1)
		}
	}

	if i.WithCertManager {
		// If Cert Manager already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.CertManagerName] || i.Upgrade {
			count += 1
			grp1.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := grp1.At(idx).WithName("Cert Manager")
				installCertManager(ctx, spinner, cl, operator)
			}(ctx, count-1)
		}
	}

	if i.WithIngressNginx {
		// If Ingress Nginx already exists and --upgrade is not specified, skip this step.
		if !inventoryExist[inventory.IngressName] || i.Upgrade {
			count += 1
			grp1.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := grp1.At(idx).WithName("Ingress")
				installIngress(ctx, spinner, cl, operator)
			}(ctx, count-1)
		}
	}

	grp1.Start(ctx)
	if err := grp1.Wait(); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	}

	grp2 := spinners.NewSpinnerGroup()
	grp2.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp2.At(idx).WithName("OpenFunction")
		installOpenFunction(ctx, spinner, cl, operator)
	}(ctx, 0)

	grp2.Start(ctx)
	if err := grp2.Wait(); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	}

	end := time.Since(start)
	util.AllDone(end)

	util.PrintOpenFunction()
	return nil
}

func (i *Install) calculateConditions() error {

	// Enable shipwright by default
	i.WithShipWright = true

	// Calculate runtime condition
	for _, rt := range i.Runtimes {
		switch rt {
		case "knative":
			i.WithKnative = true
			i.WithIngressNginx = true
		case "async":
			i.WithDapr = true
			i.WithKeda = true
		default:
			return errors.Errorf("invalid runtime: %s", rt)
		}
	}

	// Calculate ingress condition
	switch i.Ingress {
	case "nginx":
		i.WithIngressNginx = true
	default:
		return errors.Errorf("invalid ingress controller: %s", i.Ingress)
	}

	// Update the corresponding conditions when --all is set
	if i.WithAll {
		i.WithDapr = true
		i.WithKeda = true
		i.WithIngressNginx = true
		i.WithKnative = true
	}

	// Update the corresponding conditions when --without-ci is true
	if i.WithoutCI {
		i.WithShipWright = false
	}

	// Update the corresponding conditions
	if i.OpenFunctionVersion == common.LatestVersion {
		i.WithCertManager = false
	} else {
		if i.openFunctionVersion.Major() == 0 && i.openFunctionVersion.Minor() == 3 {
			i.WithCertManager = false
		}

		if i.openFunctionVersion.Major() == 0 && i.openFunctionVersion.Minor() >= 4 && i.openFunctionVersion.Minor() <= 6 {
			i.WithCertManager = true
		}
	}

	return nil
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
	util.PrintInventory(inventory)
}

func installDapr(ctx context.Context, spinner *spinners.Spinner, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	v := operator.Inventory[inventory.DaprName].GetVersion()

	spinner.Update("Downloading Dapr CLI...")
	if err := operator.DownloadDaprClient(ctx, v); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to download Dapr CLI"))
		return
	}

	spinner.Update("Initializing Dapr with Kubernetes mode...")
	if err := operator.InitDapr(ctx, v); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to init Dapr"))
		return
	}

	// Record the version of Dapr
	operator.Records.Dapr = v

	spinner.Done()
}

func installKeda(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Installing...")

	v := operator.Inventory[inventory.KedaName].GetVersion()
	yamls, err := operator.Inventory[inventory.KedaName].GetYamlFile(v)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.InstallKeda(ctx, yamls["MAIN"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to install Keda"))
		return
	}

	// Record the version of Keda
	operator.Records.Keda = v

	spinner.Update("Checking if Keda is ready...")
	if err := operator.CheckKedaIsReady(ctx, cl); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to check Keda readiness"))
		return
	}

	spinner.Done()
}

func installKnativeServing(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Installing Knative Serving...")

	knv := operator.Inventory[inventory.KnativeServingName].GetVersion()
	knYamls, err := operator.Inventory[inventory.KnativeServingName].GetYamlFile(knv)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.InstallKnativeServing(ctx, knYamls["CRD"], knYamls["CORE"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to install Knative Serving"))
		return
	}

	// Record the version of KnativeServing
	operator.Records.KnativeServing = knv

	spinner.Update("Checking if Knative Serving is ready...")
	if err := operator.CheckKnativeServingIsReady(ctx, cl); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to check Knative Serving readiness"))
		return
	}

	spinner.Update("Configuring Knative Serving's DNS...")
	ddv := operator.Inventory[inventory.ServingDefaultDomainName].GetVersion()
	ddYamls, err := operator.Inventory[inventory.ServingDefaultDomainName].GetYamlFile(ddv)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}
	if err := operator.ConfigKnativeServingDefaultDomain(ctx, ddYamls["MAIN"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to config Knative Serving's DNS"))
		return
	}

	// Record the version of DefaultDomain
	operator.Records.DefaultDomain = ddv

	spinner.Update("Installing Kourier as Knative's gateway...")
	krv := operator.Inventory[inventory.KourierName].GetVersion()
	krYamls, err := operator.Inventory[inventory.KourierName].GetYamlFile(krv)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}
	if err := operator.InstallKourier(ctx, cl, krYamls["MAIN"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to install Kourier"))
		return
	}

	// Record the version of Kourier
	operator.Records.Kourier = krv

	spinner.Update("Checking if Kourier is ready...")
	if err := operator.CheckKourierIsReady(ctx, cl); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to check Kourier readiness"))
		return
	}

	spinner.Done()
}

func installShipwright(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Installing Tekton Pipelines...")
	tkv := operator.Inventory[inventory.TektonPipelinesName].GetVersion()
	tkYamls, err := operator.Inventory[inventory.TektonPipelinesName].GetYamlFile(tkv)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}
	if err := operator.InstallTektonPipelines(ctx, tkYamls["MAIN"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to install Tekton Pipelines"))
		return
	}

	// Record the version of TektonPipelines
	operator.Records.TektonPipelines = tkv

	spinner.Update("Checking if Tekton Pipelines is ready...")
	if err := operator.CheckTektonPipelinesIsReady(ctx, cl); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to check Tekton Pipelines readiness"))
		return
	}

	spinner.Update("Installing Shipwright...")
	swv := operator.Inventory[inventory.ShipwrightName].GetVersion()
	swYamls, err := operator.Inventory[inventory.ShipwrightName].GetYamlFile(swv)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.InstallShipwright(ctx, swYamls["MAIN"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to install Shipwright"))
		return
	}

	// Record the version of Shipwright
	operator.Records.Shipwright = swv

	spinner.Update("Checking if Shipwright is ready...")
	if err := operator.CheckShipwrightIsReady(ctx, cl); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to check Shipwright readiness"))
		return
	}

	spinner.Done()
}

func installCertManager(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Installing...")
	v := operator.Inventory[inventory.CertManagerName].GetVersion()
	yamls, err := operator.Inventory[inventory.CertManagerName].GetYamlFile(v)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.InstallCertManager(ctx, yamls["MAIN"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to install Cert Manager"))
		return
	}

	// Record the version of CertManager
	operator.Records.CertManager = v

	spinner.Update("Checking if Cert Manager is ready...")
	if err := operator.CheckCertManagerIsReady(ctx, cl); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to check Cert Manager readiness"))
		return
	}

	spinner.Done()
}

func installIngress(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Installing...")
	v := operator.Inventory[inventory.IngressName].GetVersion()
	yamls, err := operator.Inventory[inventory.IngressName].GetYamlFile(v)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.InstallIngressNginx(ctx, yamls["MAIN"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to install Ingress"))
		return
	}

	// Record the version of Ingress
	operator.Records.Ingress = v

	spinner.Update("Checking if Ingress is ready...")
	if err := operator.CheckIngressNginxIsReady(ctx, cl); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to check Ingress Nginx readiness"))
		return
	}

	spinner.Done()
}

func installOpenFunction(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Installing...")
	v := operator.Inventory[inventory.OpenFunctionName].GetVersion()
	yamls, err := operator.Inventory[inventory.OpenFunctionName].GetYamlFile(v)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.InstallOpenFunction(ctx, yamls["MAIN"]); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to install OpenFunction"))
		return
	}

	// Record the version of OpenFunction
	operator.Records.OpenFunction = v

	spinner.Update("Checking if OpenFunction is ready..")
	if err := operator.CheckOpenFunctionIsReady(ctx, cl); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to check OpenFunction readiness"))
		return
	}

	spinner.Done()
}

func getLatestStableVersion() (string, error) {
	var jsonData interface{}

	resp, err := http.Get(openFunctionLatestReleaseUrl)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch OpenFunction latest release")
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	json.Unmarshal(body, &jsonData)
	res, err := jsonpath.JsonPathLookup(jsonData, "$.tag_name")
	if err != nil {
		return "", errors.Wrap(err, "failed to find the tag_name value in release information")
	}
	return res.(string), nil
}
