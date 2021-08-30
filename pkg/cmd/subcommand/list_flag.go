package subcommand

import (
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type listFlag struct {
	LabelSelector string
	FieldSelector string
	AllNamespaces bool
	Watch         bool
	Limit         int64
	Timeout       int64
}

func (l *listFlag) addListFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&l.LabelSelector, "selector", "l", l.LabelSelector, "Selector (label query) to filter on, not including uninitialized ones")
	cmd.Flags().StringVarP(&l.FieldSelector, "field-selector", "", l.FieldSelector, "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type")
	cmd.Flags().BoolVarP(&l.AllNamespaces, "all-namespaces", "A", l.AllNamespaces, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace")
	cmd.Flags().Int64VarP(&l.Timeout, "timeout", "", l.Timeout, "Timeout for the list/watch call")
	cmd.Flags().Int64VarP(&l.Limit, "limit", "", l.Limit, "limit is a maximum number of responses to return for a list call")
	cmd.Flags().BoolVarP(&l.Watch, "watch", "w", l.Watch, "After listing the requested object, watch for changes. Uninitialized objects are excluded if no object name is provided")
}

func (l *listFlag) ToOptions() metav1.ListOptions {

	options := metav1.ListOptions{}

	options.LabelSelector = l.LabelSelector
	options.FieldSelector = l.FieldSelector

	timeout := l.Timeout
	if timeout == 0 {
		timeout = 30 * 60
	}
	options.TimeoutSeconds = &timeout

	options.Limit = l.Limit
	options.Watch = l.Watch

	return options
}
