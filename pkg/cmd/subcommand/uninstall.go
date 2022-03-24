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
	"github.com/OpenFunction/cli/pkg/cmd/util/spinners"
	"github.com/OpenFunction/cli/pkg/components/common"
	"github.com/OpenFunction/cli/pkg/components/inventory"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8s "k8s.io/client-go/kubernetes"
)

// Uninstall is the commandline for 'uninstall' sub command
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
	Yes                 bool
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
		Use:                   "uninstall [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Uninstall OpenFunction and its dependencies.",
		Long:                  "This command will help you to uninstall OpenFunction and its dependencies.",
		Example: `
# Uninstall OpenFunction with all dependencies
ofn uninstall --all

# Uninstall a specified runtime of OpenFunction
ofn uninstall --async

# For users who have limited access to gcr.io or github.com to uninstall OpenFunction
ofn uninstall --region-cn --all

# Uninstall OpenFunction and wait for the uninstallation to complete (default timeout is 300s/5m)
ofn uninstall --all --wait

# Uninstall a specific version of OpenFunction
ofn uninstall --all --version v0.4.0

# See more at: https://github.com/OpenFunction/cli/blob/main/docs/uninstall.md
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
	cmd.Flags().BoolVar(&i.RegionCN, "region-cn", false, "For users who have limited access to gcr.io or github.com.")
	cmd.Flags().BoolVar(&i.DryRun, "dry-run", false, "Used to prompt for the components and their versions to be uninstalled by the current command.")
	cmd.Flags().BoolVar(&i.WaitForCleared, "wait", false, "Awaiting the results of the uninstallation.")
	cmd.Flags().BoolVarP(&i.Yes, "yes", "y", false, "Automatic yes to prompts.")
	cmd.Flags().StringVar(&i.OpenFunctionVersion, "version", "", "Used to specify the version of OpenFunction to be uninstalled.")
	cmd.Flags().DurationVar(&i.Timeout, "timeout", 10*time.Minute, "Set timeout time. Default is 10 minutes.")
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
	if i.OpenFunctionVersion == common.LatestVersion {
		return nil
	}

	if i.OpenFunctionVersion == "" {
		return nil
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

	return nil
}

func (i *Uninstall) RunUninstall(cl *k8s.Clientset, cmd *cobra.Command) error {
	operator := common.NewOperator(runtime.GOOS, i.OpenFunctionVersion, i.Timeout, i.RegionCN, i.Verbose)
	continueFunc := func() bool {
		reader := bufio.NewReader(os.Stdin)
		util.BeforeTask("Please ensure that you understand the meaning of this command " +
			"and follow the prompts below to confirm the action.\n" +
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

	util.BeforeTask("Start uninstalling OpenFunction and its dependencies.")
	util.BeforeTask("The following component(s) will be uninstalled:")
	for component := range inventoryPending {
		util.BeforeTask(fmt.Sprintf("\t- %s", component))
	}

	if i.DryRun {
		return nil
	}

	if !i.Yes && !continueFunc() {
		return nil
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

	group := spinners.NewSpinnerGroup()
	count := 0

	if i.WithDapr {
		if operator.Records.Dapr != "" {
			count += 1
			group.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := group.At(idx).WithName("Dapr")
				uninstallDapr(ctx, spinner, cl, operator, i.WaitForCleared)
			}(ctx, count-1)
		}
	}

	if i.WithKeda {
		if operator.Records.Keda != "" {
			count += 1
			group.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := group.At(idx).WithName("Keda")
				uninstallKeda(ctx, spinner, cl, operator, i.WaitForCleared)
			}(ctx, count-1)
		}
	}

	if i.WithKnative {
		if operator.Records.KnativeServing != "" {
			count += 1
			group.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := group.At(idx).WithName("Knative Serving")
				uninstallKnativeServing(ctx, spinner, cl, operator, i.WaitForCleared)
			}(ctx, count-1)
		}
	}

	if i.WithShipWright {
		if operator.Records.Shipwright != "" {
			count += 1
			group.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := group.At(idx).WithName("Shipwright")
				uninstallShipwright(ctx, spinner, cl, operator, i.WaitForCleared)
			}(ctx, count-1)
		}
		if operator.Records.TektonPipelines != "" {
			count += 1
			group.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := group.At(idx).WithName("Tekton Pipelines")
				uninstallTektonPipelines(ctx, spinner, cl, operator, i.WaitForCleared)
			}(ctx, count-1)
		}
	}

	if i.WithCertManager {
		if operator.Records.CertManager != "" {
			count += 1
			group.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := group.At(idx).WithName("Cert Manager")
				uninstallCertManager(ctx, spinner, cl, operator, i.WaitForCleared)
			}(ctx, count-1)
		}
	}

	if i.WithIngress {
		if operator.Records.Ingress != "" {
			count += 1
			group.AddSpinner()
			go func(ctx context.Context, idx int) {
				spinner := group.At(idx).WithName("Ingress")
				uninstallIngress(ctx, spinner, cl, operator, i.WaitForCleared)
			}(ctx, count-1)
		}
	}

	if operator.Records.OpenFunction != "" {
		count += 1
		group.AddSpinner()
		go func(ctx context.Context, idx int) {
			spinner := group.At(idx).WithName("OpenFunction")
			uninstallOpenFunction(ctx, spinner, cl, operator, i.WaitForCleared)
		}(ctx, count-1)
	}

	group.Start(ctx)
	if err := group.Wait(); err != nil {
		return errors.New(util.TaskFail(err.Error()))
	}

	end := time.Since(start)
	util.AllDone(end)
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

func uninstallDapr(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator, waitForCleared bool) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Uninstalling...")

	if err := operator.UninstallDapr(ctx, cl, waitForCleared); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to uninstall Dapr"))
		return
	}

	// Reset version to null
	operator.Records.Dapr = ""

	spinner.Done()
}

func uninstallKeda(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator, waitForCleared bool) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Uninstalling...")
	yamls, err := operator.Inventory[inventory.KedaName].GetYamlFile(operator.Records.Keda)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.KedaNamespace, false, waitForCleared); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to uninstall Keda"))
		return
	}

	// Reset version to null
	operator.Records.Keda = ""

	spinner.Done()
}

func uninstallKnativeServing(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator, waitForCleared bool) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	if operator.Records.DefaultDomain != "" {
		spinner.Update("Uninstalling Serving Default Domain...")
		yamls, err := operator.Inventory[inventory.ServingDefaultDomainName].GetYamlFile(operator.Records.DefaultDomain)
		if err != nil {
			spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
			return
		}

		if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.KnativeServingNamespace, false, waitForCleared); err != nil {
			spinner.Error(errors.Wrap(err, "Failed to uninstall Serving Default Domain"))
			return
		}

		// Reset version to null
		operator.Records.DefaultDomain = ""
	}

	if operator.Records.Kourier != "" {
		spinner.Update("Uninstalling Kourier...")
		yamls, err := operator.Inventory[inventory.KourierName].GetYamlFile(operator.Records.Kourier)
		if err != nil {
			spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
			return
		}

		if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.KedaNamespace, true, waitForCleared); err != nil {
			spinner.Error(errors.Wrap(err, "Failed to uninstall Kourier"))
			return
		}

		// Reset version to null
		operator.Records.Kourier = ""
	}

	spinner.Update("Uninstalling Knative Serving...")
	yamls, err := operator.Inventory[inventory.KnativeServingName].GetYamlFile(operator.Records.KnativeServing)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.UninstallKnativeServing(ctx, cl, yamls["CRD"], yamls["CORE"], waitForCleared); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to uninstall Knative Serving"))
		return
	}

	// Reset version to null
	operator.Records.KnativeServing = ""

	spinner.Done()
}

func uninstallShipwright(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator, waitForCleared bool) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Uninstalling...")
	yamls, err := operator.Inventory[inventory.ShipwrightName].GetYamlFile(operator.Records.Shipwright)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.ShipwrightNamespace, false, waitForCleared); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to uninstall Shipwright"))
		return
	}

	// Reset version to null
	operator.Records.Shipwright = ""

	spinner.Done()
}

func uninstallTektonPipelines(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator, waitForCleared bool) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Uninstalling...")
	yamls, err := operator.Inventory[inventory.TektonPipelinesName].GetYamlFile(operator.Records.TektonPipelines)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.TektonPipelineNamespace, false, waitForCleared); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to uninstall Tekton Pipeline"))
		return
	}

	// Reset version to null
	operator.Records.TektonPipelines = ""

	spinner.Done()
}

func uninstallCertManager(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator, waitForCleared bool) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Uninstalling...")
	yamls, err := operator.Inventory[inventory.CertManagerName].GetYamlFile(operator.Records.CertManager)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.CertManagerNamespace, false, waitForCleared); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to uninstall Cert Manager"))
		return
	}

	// Reset version to null
	operator.Records.CertManager = ""

	spinner.Done()
}

func uninstallIngress(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator, waitForCleared bool) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Uninstalling...")
	yamls, err := operator.Inventory[inventory.IngressName].GetYamlFile(operator.Records.Ingress)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.IngressNginxNamespace, false, waitForCleared); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to uninstall Ingress"))
		return
	}

	// Reset version to null
	operator.Records.Ingress = ""

	spinner.Done()
}

func uninstallOpenFunction(ctx context.Context, spinner *spinners.Spinner, cl *k8s.Clientset, operator *common.Operator, waitForCleared bool) {
	ctx, done := context.WithCancel(ctx)
	defer done()

	spinner.Update("Uninstalling...")
	yamls, err := operator.Inventory[inventory.OpenFunctionName].GetYamlFile(operator.Records.OpenFunction)
	if err != nil {
		spinner.Error(errors.Wrap(err, "Failed to get yaml file"))
		return
	}

	if err := operator.Uninstall(ctx, cl, yamls["MAIN"], common.OpenFunctionNamespace, false, waitForCleared); err != nil {
		spinner.Error(errors.Wrap(err, "Failed to uninstall OpenFunction"))
		return
	}

	// Reset version to null
	operator.Records.OpenFunction = ""

	spinner.Done()
}
