package subcommand

import (
	"strings"
	"testing"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestGet(t *testing.T) {
	tests := []struct {
		Name   []string
		Flag   map[string]string
		expect []string
	}{
		{
			Name: []string{"sample"},
		},
		{
			expect: []string{"NAME", "sample", "sample1"},
		},
		{
			Flag: map[string]string{
				"all-namepsace": "ture",
				"output":        "name",
			},
			expect: []string{"function.core.openfunction.io/sample", "function.core.openfunction.io/sample1"},
		},
		{
			Name: []string{"sample"},
			Flag: map[string]string{
				"output": "yaml",
			},
			expect: []string{"sample", "apiVersion: core.openfunction.io/v1alpha1"},
		},
	}

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	for _, test := range tests {
		ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()
		cmd := NewCmdGet(fake, ioStreams)
		for key, flag := range test.Flag {
			cmd.Flags().Set(key, flag)
		}

		err := cmd.PreRunE(cmd, test.Name)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Run(cmd, test.Name)

		result := buf.String()
		for _, expect := range test.expect {
			if !strings.Contains(result, expect) {
				t.Errorf("unexpected output: %s", result)
			}
		}
	}
}

func TestBuilder(t *testing.T) {
	tests := []struct {
		Name   []string
		Flag   map[string]string
		expect []string
	}{
		{
			expect: []string{"NAME", "sample-builder", "sample1-builder"},
		},
		{
			Flag: map[string]string{
				"all-namepsace": "ture",
				"output":        "name",
			},
			expect: []string{"builder.core.openfunction.io/sample-builder", "builder.core.openfunction.io/sample1-builder"},
		},
		{
			Name: []string{"sample-builder"},
			Flag: map[string]string{
				"output": "yaml",
			},
			expect: []string{"sample-builder", "apiVersion: core.openfunction.io/v1alpha1"},
		},
	}

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	for _, test := range tests {
		ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()
		cmd := newCmdGetBuilder(fake, ioStreams)
		for key, flag := range test.Flag {
			cmd.Flags().Set(key, flag)
		}
		err := cmd.PreRunE(cmd, test.Name)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Run(cmd, test.Name)

		result := buf.String()
		for _, expect := range test.expect {
			if !strings.Contains(result, expect) {
				t.Errorf("unexpected output: %s", result)
			}
		}
	}
}

func TestServing(t *testing.T) {
	tests := []struct {
		Name   []string
		Flag   map[string]string
		expect []string
	}{
		{
			expect: []string{"NAME", "sample-serving", "sample1-serving"},
		},
		{
			Flag: map[string]string{
				"all-namepsace": "ture",
				"output":        "name",
			},
			expect: []string{"serving.core.openfunction.io/sample-serving", "serving.core.openfunction.io/sample1-serving"},
		},
		{
			Name: []string{"sample-serving"},
			Flag: map[string]string{
				"output": "yaml",
			},
			expect: []string{"sample-serving", "apiVersion: core.openfunction.io/v1alpha1"},
		},
	}

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	for _, test := range tests {
		ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()
		cmd := newCmdGetServing(fake, ioStreams)
		for key, flag := range test.Flag {
			cmd.Flags().Set(key, flag)
		}
		err := cmd.PreRunE(cmd, test.Name)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Run(cmd, test.Name)

		result := buf.String()
		for _, expect := range test.expect {
			if !strings.Contains(result, expect) {
				t.Errorf("unexpected output: %s", result)
			}
		}
	}
}
