package subcommand

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/OpenFunction/cli/pkg/client"
	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/OpenFunction/cli/pkg/cmd/util/spinners"
	"github.com/OpenFunction/cli/pkg/components/common"
	"github.com/OpenFunction/cli/pkg/components/inventory"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8s "k8s.io/client-go/kubernetes"
)

// Demo is the commandline for 'demo' sub command
type Demo struct {
	genericclioptions.IOStreams

	Verbose             bool
	RegionCN            bool
	OpenFunctionVersion string
	DryRun              bool
	AutoPrune           bool
	Timeout             time.Duration
}

const (
	DemoYamlFile   = "https://raw.githubusercontent.com/OpenFunction/OpenFunction/main/config/samples/function-sample-serving-only.yaml"
	DemoYamlFileCN = "https://cdn.jsdelivr.net/gh/OpenFunction/OpenFunction@main/config/samples/function-sample-serving-only.yaml"
)

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
			util.CheckErr(i.ValidateArgs(cmd, args))
			util.CheckErr(i.RunKind(cl, cmd))
		},
	}

	cmd.Flags().BoolVar(&i.Verbose, "verbose", false, "Show verbose information.")
	cmd.Flags().BoolVar(&i.RegionCN, "region-cn", false, "For users who have limited access to gcr.io or github.com.")
	cmd.Flags().BoolVar(&i.AutoPrune, "auto-prune", true, "Automatically clean up the demo environment.")
	cmd.Flags().DurationVar(&i.Timeout, "timeout", 20*time.Minute, "Set timeout time. Default is 20 minutes.")
	return cmd
}

func (i *Demo) ValidateArgs(cmd *cobra.Command, args []string) error {
	v, e := getLatestStableVersion()
	if e != nil {
		return e
	}
	i.OpenFunctionVersion = v
	return nil
}

func (i *Demo) RunKind(cl *k8s.Clientset, cmd *cobra.Command) error {
	defer func() {
		if i.AutoPrune {
			i.deleteCluster()
		}
	}()

	operator := common.NewOperator(runtime.GOOS, runtime.GOARCH, i.OpenFunctionVersion, i.Timeout, i.RegionCN, i.Verbose)

	ctx, done := context.WithTimeout(
		context.Background(),
		i.Timeout,
	)
	defer done()

	util.BeforeTask(" -> The OpenFunction demonstration <-\n" +
		"Start launching the cluster.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		done()
	}()

	start := time.Now()

	// grp1 for installing the cluster via Kind
	grp1 := spinners.NewSpinnerGroup()
	grp1.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp1.At(idx).WithName("Kind Cluster")
		i.createCluster(ctx, spinner, operator)
	}(ctx, 0)

	grp1.Start(ctx)
	if err := grp1.Wait(); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	}

	_, cl, err := client.NewKubeConfigClient()
	if err != nil {
		return err
	}
	inventoryPending, err := inventory.GetInventory(
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
	operator.Inventory = inventoryPending

	util.BeforeTask("Start installing OpenFunction and its dependencies.\n" +
		"Here are the components and corresponding versions to be installed:")
	printInventory(inventory.GetVersionMap(inventoryPending))

	// Record the list of components
	// that currently exist in the cluster.
	if _, err := operator.GetInventoryRecord(ctx, false); err != nil {
		return errors.Wrap(err, "failed to get inventory record")
	}
	defer operator.RecordInventory(ctx)

	// grp2 for installing dependent components
	grp2 := spinners.NewSpinnerGroup()
	count := 0

	count += 1
	grp2.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp2.At(idx).WithName("Dapr")
		installDapr(ctx, spinner, operator)
	}(ctx, count-1)

	count += 1
	grp2.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp2.At(idx).WithName("Keda")
		installKeda(ctx, spinner, cl, operator)
	}(ctx, count-1)

	count += 1
	grp2.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp2.At(idx).WithName("Knative Serving")
		installKnativeServing(ctx, spinner, cl, operator)
	}(ctx, count-1)

	count += 1
	grp2.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp2.At(idx).WithName("Shipwright")
		installShipwright(ctx, spinner, cl, operator)
	}(ctx, count-1)

	count += 1
	grp2.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp2.At(idx).WithName("Cert Manager")
		installCertManager(ctx, spinner, cl, operator)
	}(ctx, count-1)

	grp2.Start(ctx)
	if err := grp2.Wait(); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	}

	// grp3 for installing the latest stable OpenFunction
	grp3 := spinners.NewSpinnerGroup()

	grp3.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp3.At(idx).WithName("OpenFunction")
		installOpenFunction(ctx, spinner, cl, operator)
	}(ctx, 0)

	grp3.Start(ctx)
	if err := grp3.Wait(); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	}

	// grp4 for provisioning the demo
	grp4 := spinners.NewSpinnerGroup()

	grp4.AddSpinner()
	go func(ctx context.Context, idx int) {
		spinner := grp4.At(idx).WithName("Demo")
		i.provisionDemoFunction(ctx, spinner, cl, operator)
	}(ctx, 0)

	grp4.Start(ctx)
	if err := grp4.Wait(); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	}

	if err := i.accessDemoFunction(ctx, cl, operator); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	}

	end := time.Since(start)
	util.AllDone(end)
	return nil
}

func (i *Demo) createCluster(ctx context.Context, spinner *spinners.Spinner, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Downloading Kind binary...")
	if err := operator.DownloadKind(ctx); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to download Kind"))
		return
	}

	spinner.Update("Creating cluster...")
	if err := operator.CreateKindCluster(ctx); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to create kind cluster"))
		return
	}

	spinner.Done()
}

func (i *Demo) deleteCluster() {
	operator := common.NewOperator(runtime.GOOS, runtime.GOARCH, i.OpenFunctionVersion, i.Timeout, i.RegionCN, i.Verbose)
	ctx, done := context.WithTimeout(
		context.Background(),
		i.Timeout,
	)
	defer done()

	if err := operator.DeleteKind(ctx); err != nil {
		util.TaskFail(err.Error())
		return
	}
}

func (i *Demo) provisionDemoFunction(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator) {
	ctx, done := context.WithCancel(ctx)
	defer done()
	var demo string
	if i.RegionCN {
		demo = DemoYamlFileCN
	} else {
		demo = DemoYamlFile
	}

	spinner.Update("Provisioning OpenFunction demo...")

	if err := operator.RunOpenFunction(ctx, demo); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to provision OpenFunction demo"))
		return
	}

	NodeIP, err := operator.GetNodeIP(ctx)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get the Node IP address"))
		return
	}

	if err := operator.PatchExternalIP(ctx, cl, NodeIP); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to patch the External IP address"))
		return
	}
	if err := operator.PatchMagicDNS(ctx, cl, NodeIP); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to patch the Magic DNS"))
		return
	}

	spinner.Done()
}

func (i *Demo) accessDemoFunction(ctx context.Context, cl *k8s.Clientset, operator *common.Operator) error {
	fmt.Print(util.YellowItalic(" -> Fetching the URL of demo function...\r"))
	endpoint, err := operator.PrintEndpoint(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch the Endpoint")
	}
	fmt.Println(util.YellowItalic(" -> You can use the following URL to access the demo function:"))
	fmt.Println(endpoint)
	fmt.Print(util.YellowItalic("\n -> We are now accessing the URL above...\r"))
	if rt, err := operator.CurlOpenFunction(ctx, endpoint); err != nil {
		return errors.Wrap(err, "Failed to access the URL of OpenFunction demo")
	} else {
		fmt.Println(util.YellowItalic(" -> We have accessed the URL above and get the following information:"))
		fmt.Println(rt)
	}

	return nil
}
