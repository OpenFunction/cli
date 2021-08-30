package subcommand

import (
	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type deleteFlag struct {
	AllNamespaces bool

	All            bool
	LabelSelector  string
	FieldSelector  string
	IgnoreNotFound bool

	GracePeriodSeconds *int64
	DryRun             bool
}

func (d *deleteFlag) addFlag(cmd *cobra.Command) {
	var gracePeriodSeconds int64
	d.GracePeriodSeconds = &gracePeriodSeconds

	cmd.Flags().BoolVarP(&d.AllNamespaces, "all-namespaces", "A", d.AllNamespaces, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace")
	cmd.Flags().BoolVarP(&d.All, "all", "", d.All, "Delete all resources, including uninitialized ones, in the namespace of the specified resource types.")
	cmd.Flags().StringVarP(&d.LabelSelector, "selector", "l", d.LabelSelector, "Selector (label query) to filter on, not including uninitialized ones")
	cmd.Flags().StringVarP(&d.FieldSelector, "field-selector", "", d.FieldSelector, "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type")
	cmd.Flags().Int64VarP(d.GracePeriodSeconds, "grace-period-seconds", "", *d.GracePeriodSeconds, "The duration in seconds before the object should be deletedl")
	cmd.Flags().BoolVarP(&d.DryRun, "dry-run", "", d.DryRun, "Only print the object that would be sent, without sending it")
}

func (d *deleteFlag) complete(cmd *cobra.Command) error {
	if d.All || len(d.LabelSelector) > 0 || len(d.FieldSelector) > 0 {
		if f := cmd.Flags().Lookup("ignore-not-found"); f != nil && !f.Changed {
			d.IgnoreNotFound = true
		}
	}

	return nil
}

func (d *deleteFlag) Validate(cmd *cobra.Command) error {
	if d.All && len(d.LabelSelector) > 0 {
		return util.UsageErrorf(cmd, "cannot set --all and --selector at the same time")
	}
	if d.All && len(d.FieldSelector) > 0 {
		return util.UsageErrorf(cmd, "cannot set --all and --field-selector at the same time")
	}

	return nil
}

func (d *deleteFlag) ToOptions() metav1.DeleteOptions {

	options := metav1.DeleteOptions{}
	options.GracePeriodSeconds = d.GracePeriodSeconds

	if d.DryRun {
		options.DryRun = []string{metav1.DryRunAll}
	}

	return options
}
