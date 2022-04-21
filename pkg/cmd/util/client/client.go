package client

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	corebeta1 "github.com/openfunction/apis/core/v1beta1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8s "k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	kubeConfigDelimiter = ":"
)

// NewKubeConfigClient returns the kubeconfig and the client created from the kubeconfig.
func NewKubeConfigClient(cf *genericclioptions.ConfigFlags) (*rest.Config, *k8s.Clientset, error) {
	config, err := cf.ToRESTConfig()
	if err != nil {
		return nil, nil, err
	}
	client, err := k8s.NewForConfig(config)
	if err != nil {
		return config, nil, err
	}
	return config, client, nil
}

func SetConfigDefaults(config *rest.Config) error {
	gv := corebeta1.GroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

func getConfig() (*rest.Config, error) {
	var (
		doOnce     sync.Once
		kubeconfig *string
	)

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	doOnce.Do(func() {
		flag.Parse()
	})
	kubeConfigEnv := os.Getenv("KUBECONFIG")
	delimiterBelongsToPath := strings.Count(*kubeconfig, kubeConfigDelimiter) == 1 && strings.EqualFold(*kubeconfig, kubeConfigEnv)

	if len(kubeConfigEnv) != 0 && !delimiterBelongsToPath {
		kubeConfigs := strings.Split(kubeConfigEnv, kubeConfigDelimiter)
		if len(kubeConfigs) > 1 {
			return nil, fmt.Errorf("multiple kubeconfigs in KUBECONFIG environment variable - %s", kubeConfigEnv)
		}
		kubeconfig = &kubeConfigs[0]
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}
