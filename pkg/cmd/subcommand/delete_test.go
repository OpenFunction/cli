package subcommand

import (
	"strings"
	"testing"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestDeleteFromFile(t *testing.T) {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()
	cmd := NewCmdDelete(fake, ioStreams)
	cmd.Flags().Set("dry-run", "client")
	cmd.Flags().Set("filename", "../../../testdata/fn.yaml")

	err := cmd.PreRunE(cmd, []string{})
	if err != nil {
		t.Fatal(err)
	}
	cmd.Run(cmd, []string{})

	result := buf.String()
	if !strings.HasPrefix(result, "sample") {
		t.Errorf("unexpected output: %s", result)
	}
}

func TestDelete(t *testing.T) {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()
	cmd := NewCmdDelete(fake, ioStreams)
	cmd.Flags().Set("dry-run", "client")

	err := cmd.PreRunE(cmd, []string{})
	if err != nil {
		t.Fatal(err)
	}
	cmd.Run(cmd, []string{"sample"})

	result := buf.String()
	if !strings.HasPrefix(result, "sample") {
		t.Errorf("unexpected output: %s", result)
	}
}

func TestAll(t *testing.T) {
	tests := []map[string]string{
		{
			"all": "true",
		},
		{
			"all":           "true",
			"all-namespace": "true",
		},
		{
			"all":     "true",
			"dry-run": "true",
			"output":  "yaml",
		},
	}

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	for _, test := range tests {
		ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()

		cmd := NewCmdDelete(fake, ioStreams)
		for key, flag := range test {
			cmd.Flags().Set(key, flag)
		}

		err := cmd.PreRunE(cmd, []string{})
		if err != nil {
			t.Fatal(err)
		}
		cmd.Run(cmd, []string{})

		result := buf.String()
		if !strings.HasPrefix(result, "sample") {
			t.Errorf("unexpected output: %s", result)
		}
	}
}
