package client

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	openfunction "github.com/openfunction/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	fnName              = "functions"
	builderName         = "builders"
	servingName         = "servings"
	kubeConfigDelimiter = ":"
)

func init() {
	_ = AddToScheme(scheme.Scheme)
}

type FnClient interface {
	Namespace(ns string) *fnClient
	EnforceNamespace() *fnClient

	Create(ctx context.Context, fn *openfunction.Function, opts metav1.CreateOptions) (result *openfunction.Function, err error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (result *openfunction.Function, err error)
	List(ctx context.Context, namespaceIfScoped bool, options metav1.ListOptions) (*openfunction.FunctionList, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Apply(ctx context.Context, fn *openfunction.Function, opts metav1.ApplyOptions) (result *openfunction.Function, err error)

	GetBuilder(ctx context.Context, name string, opts metav1.GetOptions) (result *openfunction.Builder, err error)
	ListBuilder(ctx context.Context, namespaceIfScoped bool, opts metav1.ListOptions) (result *openfunction.BuilderList, err error)

	GetServing(ctx context.Context, name string, opts metav1.GetOptions) (result *openfunction.Serving, err error)
	ListServing(ctx context.Context, namespaceIfScoped bool, opts metav1.ListOptions) (result *openfunction.ServingList, err error)
}

// NewFnClient new a openFunction rest client from RESTClientGetter
func NewFnClient(r util.Getter) (FnClient, error) {
	restConfig, err := r.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	SetConfigDefaults(restConfig)
	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		return nil, err
	}

	cmdNamespace, enforceNamespace, err := r.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}
	if _, ok := r.(*util.FakeRESTClientGetter); ok {
		return newFakeClient()
	}
	fc := newFnClient(restClient, cmdNamespace)
	if enforceNamespace {
		return fc.EnforceNamespace(), nil
	}

	return fc, nil
}

// NewKubeConfigClient returns the kubeconfig and the client created from the kubeconfig.
func NewKubeConfigClient() (*rest.Config, *k8s.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, nil, err
	}
	client, err := k8s.NewForConfig(config)
	if err != nil {
		return config, nil, err
	}
	return config, client, nil
}

func NewFakeFnClient(namespace string, roundTripper func(r *http.Request) (*http.Response, error)) (FnClient, error) {
	fakeClient := &fake.RESTClient{
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         openfunction.GroupVersion,
		VersionedAPIPath:     "/apis",
		Client:               fake.CreateHTTPClient(roundTripper),
	}
	fc := newFnClient(fakeClient, namespace)

	return fc, nil
}

func newFnClient(client rest.Interface, namespace string) FnClient {
	return &fnClient{
		client:    client,
		namespace: namespace,
	}
}

func SetConfigDefaults(config *rest.Config) error {
	gv := openfunction.GroupVersion
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

type fnClient struct {
	enforceNamespace bool

	namespace string
	client    rest.Interface
}

func (f *fnClient) Namespace(ns string) *fnClient {
	if ns != "" && !f.enforceNamespace {
		f.namespace = ns
	}

	return f
}

func (f *fnClient) EnforceNamespace() *fnClient {
	f.enforceNamespace = true
	return f
}
