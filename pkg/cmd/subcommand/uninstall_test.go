package subcommand

import (
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type uninstallConditions struct {
	runtimes         []string
	withCI           bool
	withKeda         bool
	withDapr         bool
	withShipwright   bool
	withIngressNginx bool
	withKnative      bool
	withAll          bool
	version          string
	wantFunc         func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool
}

func TestUninstallCalculateConditions(t *testing.T) {
	conditionSets := []*uninstallConditions{
		&uninstallConditions{
			runtimes: []string{"knative", "async"},
			withCI:   false,
			withKeda: false,
			withDapr: false,
			withAll:  false,
			version:  "latest",
			wantFunc: func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool {
				return !withShipwright && withKeda && withDapr && withKnative && withIngressNginx && withCertManager && err == nil
			},
		},
		&uninstallConditions{
			runtimes: []string{"knative", "invalidRuntime"},
			withCI:   false,
			withKeda: false,
			withDapr: false,
			withAll:  false,
			version:  "v0.4.0",
			wantFunc: func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool {
				return strings.Contains(err.Error(), "invalid runtime")
			},
		},
		&uninstallConditions{
			runtimes: []string{"knative"},
			withKeda: true,
			withCI:   true,
			withDapr: false,
			withAll:  false,
			version:  "v0.6.0",
			wantFunc: func(withDapr bool, withKeda bool, withKnative bool, withShipwright bool, withIngressNginx bool, withCertManager bool, err error) bool {
				return !withDapr && withKeda && withShipwright && withKnative && withIngressNginx && withCertManager && err == nil
			},
		},
		&uninstallConditions{
			runtimes:    []string{"async"},
			withCI:      true,
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

		uninstall := NewUninstall(ioStreams)
		uninstall.Runtimes = condition.runtimes
		uninstall.WithCI = condition.withCI
		uninstall.WithDapr = condition.withDapr
		uninstall.WithKnative = condition.withKnative
		uninstall.WithKeda = condition.withKeda
		uninstall.WithIngressNginx = condition.withIngressNginx
		uninstall.WithShipWright = condition.withShipwright
		uninstall.WithAll = condition.withAll
		uninstall.OpenFunctionVersion = condition.version

		uninstall.ValidateArgs()

		err := uninstall.calculateConditions()
		if !condition.wantFunc(uninstall.WithDapr, uninstall.WithKeda, uninstall.WithKnative, uninstall.WithShipWright, uninstall.WithIngressNginx, uninstall.WithCertManager, err) {
			t.Error("failed to calculate conditions.")
		}
	}
}
