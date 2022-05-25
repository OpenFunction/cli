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
	openfunction "github.com/openfunction/apis/core/v1beta1"
	client "github.com/openfunction/pkg/client/clientset/versioned"
	swclient "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	swClient       swclient.Interface
}

// NewCmdLogs builds the `logs` sub command
func NewCmdLogs(cf *genericclioptions.ConfigFlags, ioStreams genericclioptions.IOStreams) *cobra.Command {

	l := &Logs{
		IOStreams: &ioStreams,
	}

	cmd := &cobra.Command{
		Use:                   `logs [OPTIONS] FUNCTION_NAME [CONTAINER_NAME]`,
		DisableFlagsInUseLine: true,
		Short:                 "Get the logs from the build and serving pods created by the function",
		Long:                  `Get the logs from the build and serving pods created by the function`,
		Example: `
  # Get tht logs from all container in the build and serving pods created by the function whose name is 'demo-function'
  ofn logs demo-function

  # Get tht logs from the 'extra' container (a container whose name is not 'function') in the build or serving pods created by the function whose name is 'demo-function'
  ofn logs demo-function extra

  # Begin streaming the logs from the 'function' container in the build and serving pods created by the function whose name is 'demo-function'
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
	l.swClient = swclient.NewForConfigOrDie(config)

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

	// Build stage
	if f.Status.Build != nil && f.Status.Build.State != openfunction.Skipped {
		builderRef := f.Status.Build.ResourceRef
		if builderRef != "" {
			// get openfunction builder ref
			builder, err := l.functionClient.CoreV1beta1().Builders(f.Namespace).Get(ctx, builderRef, metav1.GetOptions{})
			if err != nil {
				statusError, ok := err.(*k8serrors.StatusError)
				// builder has been cleaned up
				if !ok || statusError.Status().Code != 404 {
					return err
				}
			}
			// get shipwright builder-buildrun
			swbuildrunRef := builder.Status.ResourceRef["shipwright.io/buildRun"]
			if swbuildrunRef != "" {
				swBuildRun, err := l.swClient.ShipwrightV1alpha1().BuildRuns(f.Namespace).Get(ctx, swbuildrunRef, metav1.GetOptions{})
				if err != nil {
					statusError, ok := err.(*k8serrors.StatusError)
					// buildrun has been cleaned up
					if !ok || statusError.Status().Code != 404 {
						return err
					}
				}
				err = l.logsForPods(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("buildrun.shipwright.io/name=%s", swBuildRun.Name)})
				if err != nil {
					return err
				}
			}
		}
	}

	// Serving stage
	if f.Status.Serving != nil && f.Status.Serving.State != openfunction.Skipped {
		servingRef := f.Status.Serving.ResourceRef
		if servingRef != "" {
			if l.containerName == "" {
				l.containerName = "function"
			}
			err := l.logsForPods(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("openfunction.io/serving=%s", servingRef)})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *Logs) logsForPods(ctx context.Context, listOpt metav1.ListOptions) error {
	// retrieves pod list according to the options
	podInterface := l.clientSet.CoreV1().Pods(l.namespace)
	readerList := make([]io.Reader, 0, 5)
	podList, err := podInterface.List(ctx, listOpt)
	if err != nil {
		return err
	}

	// append stream of each container
	for _, pod := range podList.Items {
		var logReader io.ReadCloser
		if l.containerName != "" {
			logReader, err = podInterface.GetLogs(pod.Name, &corev1.PodLogOptions{Follow: l.Follow, Container: l.containerName}).Stream(ctx)
			// the pod may be deleted due to scale, so we should exclude NotFound errors
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}
			defer logReader.Close()
			readerList = append(readerList, logReader)
		} else {
			for _, container := range pod.Spec.Containers {
				logReader, err = podInterface.GetLogs(pod.Name, &corev1.PodLogOptions{Follow: l.Follow, Container: container.Name}).Stream(ctx)
				// the pod may be deleted due to scale, so we should exclude NotFound errors
				if err != nil && !k8serrors.IsNotFound(err) {
					return err
				}

				defer logReader.Close()
				readerList = append(readerList, logReader)
			}
		}
		if err != nil {
			return err
		}
	}

	reader := io.MultiReader(readerList...)
	io.Copy(l.IOStreams.Out, reader)

	return err
}
