package subcommand

import (
	"bufio"
	"context"
	"fmt"
	"github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/components/inventory"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/OpenFunction/cli/pkg/components/common"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8s "k8s.io/client-go/kubernetes"
)

// Demo is the commandline for 'init' sub command
type Demo struct {
	genericclioptions.IOStreams

	Verbose             bool
	RegionCN            bool
	OpenFunctionVersion string
	DryRun              bool
	AutoPrune           bool
	Timeout             time.Duration
}

const DemoYamlFile = "https://raw.githubusercontent.com/OpenFunction/OpenFunction/main/config/samples/function-sample-serving-only.yaml"
const DemoYamlFileCN = "https://cdn.jsdelivr.net/gh/OpenFunction/OpenFunction@main/config/samples/function-sample-serving-only.yaml"

func init() {
	w = os.Stdout
}

// NewDemo returns an initialized Init instance
func NewDemo(ioStreams genericclioptions.IOStreams) *Demo {
	return &Demo{
		IOStreams: ioStreams,
	}
}

func NewCmdDemo(restClient util.Getter, ioStreams genericclioptions.IOStreams) *cobra.Command {
	var cl *k8s.Clientset
	i := NewDemo(ioStreams)

	cmd := &cobra.Command{
		Use:                   "demo <args>...",
		DisableFlagsInUseLine: true,
		Short:                 "Create OpenFunction demo.",
		Long: `
         You can use ofn demo to run the OpenFunction demo.
         Available options are: --auto-prune, --region-cn, --verbose
`,
		Example: "ofn demo",
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(i.RunKind(cl, cmd))
		},
	}

	cmd.Flags().BoolVar(&i.Verbose, "verbose", false, "Show verbose information.")
	cmd.Flags().BoolVar(&i.RegionCN, "region-cn", false, "For users who have limited access to gcr.io or github.com.")
	cmd.Flags().BoolVar(&i.AutoPrune, "auto-prune", true, "Automatically clean up the demo environment.")
	cmd.Flags().DurationVar(&i.Timeout, "timeout", 10*time.Minute, "Set timeout time. Default is 10 minutes.")
	return cmd
}

func (i *Demo) RunKind(cl *k8s.Clientset, cmd *cobra.Command) error {
	if i.AutoPrune {
		defer i.DeleteKind()
	}

	operator := common.NewOperator(runtime.GOOS, i.OpenFunctionVersion, i.Timeout, i.RegionCN, i.Verbose)
	ti := util.NewTaskInformer("DEMO")
	Continue := func() bool {
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprintln(w, ti.BeforeTask("A Kind cluster will be created and the OpenFunction Demo will be launched in it...\n"+
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
	inventoryPending := i.checkConditionsAndGetInventory()

	fmt.Fprintln(w, ti.BeforeTask("Launching OpenFunction demo...\n"+
		"The following components will be installed for this demo:"))
	printInventory(inventoryPending)

	if !Continue() {
		return nil
	}

	grp1, g1ctx := errgroup.WithContext(ctx)

	start := time.Now()

	grp1.Go(func() error {
		return i.InstallKind(g1ctx, operator)
	})

	if err := grp1.Wait(); err != nil {
		return errors.New(ti.TaskFail(err.Error()))
	}

	_, cl, err := client.NewKubeConfigClient()
	if err != nil {
		return err
	}
	installinventoryPending, err := inventory.GetInventory(
		cl,
		i.RegionCN,
		true,
		true,
		true,
		true,
		true,
		false,
		i.OpenFunctionVersion,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get pending inventory")
	}
	operator.Inventory = installinventoryPending

	fmt.Fprintln(w, ti.BeforeTask("Start installing OpenFunction and its dependencies.\n"+
		"Here are the components and corresponding versions to be installed:"))
	printInventory(inventory.GetVersionMap(installinventoryPending))

	grp2, g2ctx := errgroup.WithContext(ctx)
	grp2.Go(func() error {
		return i.InstallDapr(g2ctx, operator)
	})
	grp2.Go(func() error {
		return i.installKeda(g2ctx, cl, operator)
	})
	grp2.Go(func() error {
		return i.installKnativeServing(g2ctx, cl, operator)
	})
	grp2.Go(func() error {
		return i.installShipwright(g2ctx, cl, operator)
	})
	grp2.Go(func() error {
		return i.installCertManager(g2ctx, cl, operator)
	})

	if err := grp2.Wait(); err != nil {
		util.CheckErr(i.DeleteKind())
		return errors.New(ti.TaskFail(err.Error()))
	}

	grp3, g3ctx := errgroup.WithContext(ctx)

	grp3.Go(func() error {
		return i.installOpenFunction(g3ctx, cl, operator)
	})
	if err := grp3.Wait(); err != nil {
		i.DeleteKind()
		return errors.New(ti.TaskFail(err.Error()))
	}

	grp4, g4ctx := errgroup.WithContext(ctx)

	grp4.Go(func() error {
		if err := i.RunOpenFunctionDemo(g4ctx, cl, operator); err != nil {
			return err
		}
		return nil
	})

	if err := grp4.Wait(); err != nil {
		util.CheckErr(i.DeleteKind())
		return errors.New(ti.TaskFail(err.Error()))
	}

	end := time.Since(start)
	fmt.Fprintln(w, ti.AllDone(end))
	return nil
}

func (i *Demo) checkConditionsAndGetInventory() map[string]string {
	i.OpenFunctionVersion = "v0.4.0"
	inventory := map[string]string{
		"OpenFunction": i.OpenFunctionVersion,
	}

	return inventory
}

func (i *Demo) InstallKind(ctx context.Context, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()
	ti := util.NewTaskInformer("KIND")

	fmt.Fprintln(w, ti.TaskInfo("Installing Kind..."))
	fmt.Fprintln(w, ti.TaskInfo("Downloading Kind binary..."))
	if err := operator.DownloadKind(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to download Kind"))
	}
	fmt.Fprintln(w, ti.TaskInfo("Creating cluster..."))
	if err := operator.CreateKindCluster(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to create kind cluster "))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Demo) DeleteKind() error {
	operator := common.NewOperator(runtime.GOOS, i.OpenFunctionVersion, i.Timeout, i.RegionCN, i.Verbose)
	ctx, done := context.WithTimeout(
		context.Background(),
		i.Timeout,
	)
	defer done()
	ti := util.NewTaskInformer("DeleteKIND")
	if err := operator.DeleteKind(ctx); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to delete Kind"))
	}

	return nil
}

func (i *Demo) RunOpenFunctionDemo(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	ctx, done := context.WithCancel(ctx)
	defer done()
	var demo string
	if i.RegionCN {
		demo = DemoYamlFileCN
	} else {
		demo = DemoYamlFile
	}

	ti := util.NewTaskInformer("DEMO")

	fmt.Fprintln(w, ti.TaskInfo("Run OpenFunctionDemo..."))

	if err := operator.RunOpenFunction(ctx, demo); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to run OpenFunction demo"))
	}

	NodeIP, err := operator.GetNodeIP(ctx)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to Get the node IP"))
	}

	if err := operator.PatchExternalIP(ctx, cl, NodeIP); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to Patch External IP"))
	}
	if err := operator.PatchMagicDNS(ctx, cl, NodeIP); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to Patch Magic DNS"))
	}
	EndPoint, err := operator.PrintEndpoint(ctx)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to Print the Endpoint"))
	}
	ti.TipsOnOpenfunctionDemo(EndPoint)
	if err := operator.CurlOpenFunction(ctx, EndPoint); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to Curl OpenFunction"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())

	return nil
}

func (i *Demo) InstallDapr(ctx context.Context, operator *common.Operator) error {
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

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil

}

func (i *Demo) installKeda(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
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

	fmt.Fprintln(w, ti.TaskInfo("Checking if Keda is ready..."))
	if err := operator.CheckKedaIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Keda readiness"))
	}

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil

}

func (i *Demo) installKnativeServing(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
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

	fmt.Fprintln(w, ti.TaskInfo("Installing Kourier as Knative's gateway..."))

	krv := operator.Inventory[inventory.KourierName].GetVersion()
	krYamls, err := operator.Inventory[inventory.KourierName].GetYamlFile(krv)
	if err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to get yaml file"))
	}
	if err := operator.InstallKourier(ctx, cl, krYamls["MAIN"]); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to install Kourier"))
	}

	fmt.Fprintln(w, ti.TaskInfo("Checking if Kourier is ready..."))
	if err := operator.CheckKourierIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Kourier readiness"))
	}

	fmt.Fprintln(w, ti.TaskSuccess())
	return nil

}

func (i *Demo) installShipwright(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
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

	fmt.Fprintln(w, ti.TaskInfo("Checking if Shipwright is ready..."))
	if err := operator.CheckShipwrightIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Shipwright readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Demo) installCertManager(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
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

	fmt.Fprintln(w, ti.TaskInfo("Checking if Cert Manager is ready..."))
	if err := operator.CheckCertManagerIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check Cert Manager readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}

func (i *Demo) installOpenFunction(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
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

	fmt.Fprintln(w, ti.TaskInfo("Checking if OpenFunction is ready..."))
	if err := operator.CheckOpenFunctionIsReady(ctx, cl); err != nil {
		return errors.Wrap(err, ti.TaskFailWithTitle("Failed to check OpenFunction readiness"))
	}
	fmt.Fprintln(w, ti.TaskSuccess())
	return nil
}
