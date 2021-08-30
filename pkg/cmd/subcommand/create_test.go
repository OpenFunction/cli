package subcommand

import (
	"fmt"
	"strings"
	"testing"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	openfunction "github.com/openfunction/apis/core/v1alpha1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestCreateFromFile(t *testing.T) {
	tests := map[string]string{
		"sample-build":   "../../../testdata/fn-build.yaml",
		"sample-serving": "../../../testdata/fn-serving.yaml",
		"sample":         "../../../testdata/fn.yaml",
	}

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()

	for _, test := range tests {
		cmd := NewCmdCreate(fake, ioStreams)
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

func TestCreate(t *testing.T) {
	builder := "openfunction/builder:v1"
	srcRepoURL := "https://github.com/OpenFunction/samples.git"
	srcRepoSourceSubPath := "functions/Knative/hello-world-go"
	runtime := openfunction.Runtime("Knative")

	build := openfunction.BuildImpl{
		Builder: &builder,
		SrcRepo: &openfunction.GitRepo{
			Url:           srcRepoURL,
			SourceSubPath: &srcRepoSourceSubPath,
		},
	}
	serving := openfunction.ServingImpl{
		Runtime: &runtime,
	}

	tests := []struct {
		name             string
		version          string
		image            string
		imageCredentials struct {
			name string
		}
		port    int32
		build   openfunction.BuildImpl
		serving openfunction.ServingImpl
	}{
		{
			name:    "sample",
			version: "v1.0.0",
			image:   "openfunctiondev/sample-go-func:latest",
			port:    8080,
			build:   build,
			serving: serving,
		},
		{
			name:    "sample-build",
			version: "v1.0.0",
			image:   "openfunctiondev/sample-go-func:latest",
			port:    8080,
			build:   build,
		},
		{
			name:    "sample-serving",
			version: "v1.0.0",
			image:   "openfunctiondev/sample-go-func:latest",
			port:    8080,
			serving: serving,
		},
	}

	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	fake := util.NewFakeRESTClientGetter(kubeConfigFlags)

	for _, test := range tests {
		ioStreams, _, buf, _ := genericclioptions.NewTestIOStreams()

		cmd := NewCmdCreate(fake, ioStreams)
		cmd.Flags().Set("dry-run", "client")
		cmd.Flags().Set("version", test.version)
		cmd.Flags().Set("image", test.image)
		if test.build.Builder != nil {
			cmd.Flags().Set("builder", *test.build.Builder)
		}
		if test.build.SrcRepo != nil {
			cmd.Flags().Set("git-repo-url", test.build.SrcRepo.Url)
			cmd.Flags().Set("git-repo-source-sub-path", *test.build.SrcRepo.SourceSubPath)
		}
		if test.serving.Runtime != nil {
			cmd.Flags().Set("runtime", string(*test.serving.Runtime))
		}

		err := cmd.PreRunE(cmd, []string{test.name})
		if err != nil {
			t.Fatal(err)
		}
		cmd.Run(cmd, []string{test.name})

		result := buf.String()
		fmt.Println(result)
		if !strings.HasPrefix(result, "function.core.openfunction.io/") {
			t.Errorf("unexpected output: %s", result)
		}
	}
}
