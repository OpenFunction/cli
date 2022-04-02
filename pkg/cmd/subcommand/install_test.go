package subcommand

import (
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type installConditions struct {
	runtimes         []string
	ingress          string
	withoutCI        bool
	withKeda         bool
	withDapr         bool
	withShipwright   bool
	withIngressNginx bool
	withKnative      bool
	withAll          bool
	version          string
	wantFunc         func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool
}

func TestInstallCalculateConditions(t *testing.T) {
	conditionSets := []*installConditions{
		&installConditions{
			runtimes:  []string{"knative", "async"},
			ingress:   "nginx",
			withoutCI: true,
			withKeda:  false,
			withDapr:  false,
			withAll:   false,
			version:   "latest",
			wantFunc: func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool {
				return !withShipwright && withKeda && withDapr && withKnative && withIngressNginx && !withCertManager && err == nil
			},
		},
		&installConditions{
			runtimes:  []string{"knative", "invalidRuntime"},
			ingress:   "nginx",
			withoutCI: false,
			withKeda:  false,
			withDapr:  false,
			withAll:   false,
			version:   "v0.4.0",
			wantFunc: func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool {
				return strings.Contains(err.Error(), "invalid runtime")
			},
		},
		&installConditions{
			runtimes:  []string{"knative"},
			ingress:   "invalidIngress",
			withoutCI: false,
			withKeda:  false,
			withDapr:  false,
			withAll:   false,
			version:   "v0.6.0",
			wantFunc: func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool {
				return strings.Contains(err.Error(), "invalid ingress")
			},
		},
		&installConditions{
			runtimes:    []string{"async"},
			ingress:     "nginx",
			withoutCI:   false,
			withKeda:    false,
			withDapr:    false,
			withKnative: false,
			withAll:     true,
			version:     "v0.5.0",
			wantFunc: func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool {
				return withDapr && withKeda && withShipwright && withKnative && withIngressNginx && withCertManager && err == nil
			},
		},
	}

	for _, condition := range conditionSets {
		ioStreams, _, _, _ := genericclioptions.NewTestIOStreams()

		install := NewInstall(ioStreams)
		install.Runtimes = condition.runtimes
		install.Ingress = condition.ingress
		install.WithoutCI = condition.withoutCI
		install.WithDapr = condition.withDapr
		install.WithKnative = condition.withKnative
		install.WithKeda = condition.withKeda
		install.WithIngressNginx = condition.withIngressNginx
		install.WithShipWright = condition.withShipwright
		install.WithAll = condition.withAll
		install.OpenFunctionVersion = condition.version

		install.ValidateArgs()

		err := install.calculateConditions()
		if !condition.wantFunc(install.WithDapr, install.WithKeda, install.WithKnative, install.WithShipWright, install.WithIngressNginx, install.WithCertManager, err) {
			t.Error("failed to calculate conditions.")
		}
	}
}
