package util

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Getter interface {
	ToRESTConfig() (*restclient.Config, error)
	ToRESTMapper() (meta.RESTMapper, error)
	ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error)
	ToRawKubeConfigLoader() clientcmd.ClientConfig
}

type RESTClientGetter struct {
	clientGetter genericclioptions.RESTClientGetter
}

func NewRESTClientGetter(clientGetter genericclioptions.RESTClientGetter) *RESTClientGetter {
	if clientGetter == nil {
		panic("attempt to instantiate client_access_factory with nil clientGetter")
	}
	r := &RESTClientGetter{
		clientGetter: clientGetter,
	}

	return r
}

func (r *RESTClientGetter) ToRESTConfig() (*restclient.Config, error) {
	return r.clientGetter.ToRESTConfig()
}

func (r *RESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return r.clientGetter.ToRESTMapper()
}

func (r *RESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return r.clientGetter.ToDiscoveryClient()
}

func (r *RESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return r.clientGetter.ToRawKubeConfigLoader()
}
