package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	openfunction "github.com/openfunction/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func newFakeClient() (FnClient, error) {
	header := http.Header{}
	header.Set("Content-Type", runtime.ContentTypeJSON)

	codec := Codecs.LegacyCodec(openfunction.GroupVersion)

	return NewFakeFnClient("test", func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case "DELETE":
			fn := new(openfunction.Function)

			fn.Name = "sample"
			return &http.Response{StatusCode: http.StatusCreated, Header: header,
				Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, fn))))}, nil
		case "GET":
			switch r.URL.Path {
			case "/apis/namespaces/test/functions":
				fnList := new(openfunction.FunctionList)
				fnList.Items = append(fnList.Items, openfunction.Function{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sample",
					},
				})
				fnList.Items = append(fnList.Items, openfunction.Function{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sample1",
					},
				})
				return &http.Response{StatusCode: http.StatusCreated, Header: header,
					Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, fnList))))}, nil
			case "/apis/namespaces/test/functions/sample":
				fn := new(openfunction.Function)
				fn.Name = "sample"
				return &http.Response{StatusCode: http.StatusCreated, Header: header,
					Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, fn))))}, nil
			case "/apis/namespaces/test/builders":
				builderList := new(openfunction.BuilderList)
				builderList.Items = append(builderList.Items, openfunction.Builder{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sample-builder",
					},
				})
				builderList.Items = append(builderList.Items, openfunction.Builder{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sample1-builder",
					},
				})
				return &http.Response{StatusCode: http.StatusCreated, Header: header,
					Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, builderList))))}, nil
			case "/apis/namespaces/test/builders/sample-builder":
				builder := new(openfunction.Builder)
				builder.Name = "sample-builder"
				return &http.Response{StatusCode: http.StatusCreated, Header: header,
					Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, builder))))}, nil
			case "/apis/namespaces/test/servings":
				ServingList := new(openfunction.ServingList)
				ServingList.Items = append(ServingList.Items, openfunction.Serving{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sample-serving",
					},
				})
				ServingList.Items = append(ServingList.Items, openfunction.Serving{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sample1-serving",
					},
				})
				return &http.Response{StatusCode: http.StatusCreated, Header: header,
					Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, ServingList))))}, nil
			case "/apis/namespaces/test/servings/sample-serving":
				serving := new(openfunction.Serving)
				serving.Name = "sample-serving"
				return &http.Response{StatusCode: http.StatusCreated, Header: header,
					Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, serving))))}, nil
			}

		default:
			fn := new(openfunction.Function)
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return &http.Response{StatusCode: http.StatusInternalServerError, Header: header,
					Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, fn))))}, err
			}

			err = json.Unmarshal(body, fn)
			if err != nil {
				return &http.Response{StatusCode: http.StatusInternalServerError, Header: header,
					Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, fn))))}, err
			}
			return &http.Response{StatusCode: http.StatusCreated, Header: header,
				Body: ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, fn))))}, nil
		}

		return &http.Response{StatusCode: http.StatusInternalServerError, Header: header,
			Body: ioutil.NopCloser(bytes.NewReader([]byte("")))}, nil
	})
}
