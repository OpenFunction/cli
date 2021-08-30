package client

import (
	"context"
	"time"

	openfunction "github.com/openfunction/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

func (f *fnClient) GetBuilder(ctx context.Context, name string, opts metav1.GetOptions) (result *openfunction.Builder, err error) {
	result = &openfunction.Builder{}
	err = f.client.Get().
		Namespace(f.namespace).
		Resource(builderName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Name(name).
		Do(ctx).
		Into(result)
	return
}

func (f *fnClient) ListBuilder(ctx context.Context, namespaceIfScoped bool, opts metav1.ListOptions) (result *openfunction.BuilderList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}

	result = &openfunction.BuilderList{}
	err = f.client.Get().
		NamespaceIfScoped(f.namespace, namespaceIfScoped).
		Resource(builderName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}
