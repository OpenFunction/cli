module github.com/OpenFunction/cli

go 1.16

require (
	github.com/ahmetalpbalkan/go-cursor v0.0.0-20131010032410-8136607ea412
	github.com/fatih/color v1.10.0
	github.com/jedib0t/go-pretty/v6 v6.3.1
	github.com/leaanthony/synx v0.1.0
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852
	github.com/openfunction v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/component-base v0.21.4
	k8s.io/klog/v2 v2.9.0
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.2
	github.com/openfunction => github.com/OpenFunction/OpenFunction v0.6.0
	k8s.io/api => k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.4
	k8s.io/client-go => k8s.io/client-go v0.21.4
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.9.7
)
