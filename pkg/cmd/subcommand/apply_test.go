package subcommand

import (
	"strings"
	"testing"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestApply(t *testing.T) {
	tests := map[string]string{
		"sample-build":   "../../../testdata/fn-build.yaml",
		"sample-serving": "../../../testdata/fn-serving.yaml",
		"sample":         "../../../testdata/fn.yaml",
	}

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	for _, test := range tests {
		ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()

		cmd := NewCmdApply(fake, ioStreams)
		cmd.Flags().Set("dry-run", "client")
		cmd.Flags().Set("filename", test)

		err := cmd.PreRunE(cmd, []string{})
		if err != nil {
			t.Fatal(err)
		}
		cmd.Run(cmd, []string{})

		result := buf.String()
		if !strings.HasPrefix(result, "function.core.openfunction.io/") {
			t.Errorf("unexpected output: %s", result)
		}
	}
}
