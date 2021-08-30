package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	openfunction "github.com/openfunction/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

func (f *fnClient) Create(ctx context.Context, fn *openfunction.Function, opts metav1.CreateOptions) (result *openfunction.Function, err error) {
	result = &openfunction.Function{}
	err = f.client.Post().
		Namespace(f.namespace).
		Resource(fnName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(fn).
		Do(ctx).
		Into(result)
	return
}

func (f *fnClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (result *openfunction.Function, err error) {
	result = &openfunction.Function{}
	err = f.client.Get().
		Namespace(f.namespace).
		Resource(fnName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Name(name).
		Do(ctx).
		Into(result)
	return
}

func (f *fnClient) List(ctx context.Context, namespaceIfScoped bool, opts metav1.ListOptions) (result *openfunction.FunctionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}

	result = &openfunction.FunctionList{}
	err = f.client.Get().
		NamespaceIfScoped(f.namespace, namespaceIfScoped).
		Resource(fnName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (f *fnClient) Apply(ctx context.Context, fn *openfunction.Function, opts metav1.ApplyOptions) (result *openfunction.Function, err error) {
	if fn == nil {
		return nil, fmt.Errorf("openfunction provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(fn)
	if err != nil {
		return nil, err
	}
	name := fn.Name
	if name == "" {
		return nil, fmt.Errorf("openfunction.Name must be provided to Apply")
	}
	result = &openfunction.Function{}
	err = f.client.Patch(types.ApplyPatchType).
		Namespace(f.namespace).
		Resource(fnName).
		Name(name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

func (f *fnClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return f.client.Delete().
		Namespace(f.namespace).
		Resource(fnName).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}
