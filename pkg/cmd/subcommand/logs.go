/*
Copyright 2022 The OpenFunction Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package subcommand

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	cc "github.com/OpenFunction/cli/pkg/cmd/util/client"
	client "github.com/openfunction/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8s "k8s.io/client-go/kubernetes"
)

// Logs is the commandline for `logs` sub command
type Logs struct {
	*genericclioptions.IOStreams

	functionName  string
	containerName string
	namespace     string

	Follow bool

	functionClient client.Interface
	clientSet      *k8s.Clientset
}

// NewCmdLogs builds the `logs` sub command
func NewCmdLogs(cf *genericclioptions.ConfigFlags, ioStreams genericclioptions.IOStreams) *cobra.Command {

	l := &Logs{
		IOStreams:     &ioStreams,
		containerName: "function",
	}

	cmd := &cobra.Command{
		Use:                   `logs [OPTIONS] FUNCTION_NAME [CONTAINER_NAME]`,
		DisableFlagsInUseLine: true,
		Short:                 "Get the logs from the serving pods created by the function",
		Long:                  `Get the logs from the serving pods created by the function`,
		Example: `
  # Get tht logs from the 'function' container in the serving pods created by the function whose name is 'demo-function'
  ofn logs demo-function

  # Get tht logs from the 'extra' container (a container whose name is not 'function') in the serving pods created by the function whose name is 'demo-function'
  ofn logs demo-function extra

  # Begin streaming the logs from the 'function' container in the serving pods created by the function whose name is 'demo-function'
  ofn logs -f demo-function
`,
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			return l.preRun(cf, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(l.run())
		},
	}
	cmd.Flags().BoolVarP(&l.Follow, "follow", "f", l.Follow, "Specify if the logs should be streamed")
	return cmd
}

func (l *Logs) preRun(cf *genericclioptions.ConfigFlags, args []string) error {
	config, clientSet, err := cc.NewKubeConfigClient(cf)
	if err != nil {
		panic(err)
	}
	l.clientSet = clientSet
	err = cc.SetConfigDefaults(config)
	if err != nil {
		return err
	}
	l.functionClient = client.NewForConfigOrDie(config)

	if cf.Namespace != nil && *(cf.Namespace) != "" {
		l.namespace = *(cf.Namespace)
	} else {
		if l.namespace, _, err = cf.ToRawKubeConfigLoader().Namespace(); err != nil {
			return err
		}
	}

	if len(args) < 1 {
		return errors.New("missing argument: FUNCTION_NAME")
	}
	l.functionName = args[0]
	if len(args) > 1 {
		l.containerName = args[1]
	}
	return nil
}

func (l *Logs) run() error {
	ctx := context.Background()
	f, err := l.functionClient.CoreV1beta1().Functions(l.namespace).Get(ctx, l.functionName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if f.Status.Serving == nil {
		return nil
	}
	serving := f.Status.Serving.ResourceRef
	listOpt := metav1.ListOptions{LabelSelector: fmt.Sprintf("openfunction.io/serving=%s", serving)}
	podInterface := l.clientSet.CoreV1().Pods(l.namespace)
	podList, err := podInterface.List(ctx, listOpt)
	if err != nil {
		return err
	}
	readerList := make([]io.Reader, 0, len(podList.Items))
	for _, pod := range podList.Items {
		logReader, err := podInterface.GetLogs(pod.Name, &corev1.PodLogOptions{Follow: l.Follow, Container: l.containerName}).Stream(ctx)
		defer logReader.Close()
		if err != nil {
			return err
		}
		readerList = append(readerList, logReader)
	}
	reader := io.MultiReader(readerList...)
	_, err = io.Copy(l.IOStreams.Out, reader)
	return err
}
