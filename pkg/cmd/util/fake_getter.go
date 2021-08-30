package util

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type FakeRESTClientGetter struct {
	clientGetter genericclioptions.RESTClientGetter
}

func NewFakeRESTClientGetter(clientGetter genericclioptions.RESTClientGetter) *FakeRESTClientGetter {
	if clientGetter == nil {
		panic("attempt to instantiate client_access_factory with nil fakeRESTClientGetter")
	}
	r := &FakeRESTClientGetter{
		clientGetter: clientGetter,
	}

	return r
}

func (r *FakeRESTClientGetter) ToRESTConfig() (*restclient.Config, error) {
	return r.clientGetter.ToRESTConfig()
}

func (r *FakeRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return r.clientGetter.ToRESTMapper()
}

func (r *FakeRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return r.clientGetter.ToDiscoveryClient()
}

func (r *FakeRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return r.clientGetter.ToRawKubeConfigLoader()
}
