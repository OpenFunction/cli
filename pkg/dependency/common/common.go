package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/OpenFunction/cli/pkg/dependency"
	"github.com/OpenFunction/cli/pkg/dependency/linux"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "k8s.io/client-go/kubernetes"
)

const (
	k8sVersionLabel = "app.kubernetes.io/version"
	k8sNameLabel    = "app.kubernetes.io/name"
	KubectlCreate   = "create"
	KubectlApply    = "apply"
	KubectlDelete   = "delete"

	DaprInRegionCn               = "DaprInRegionCN"
	Dapr                         = "Dapr"
	KedaInRegionCn               = "KedaInRegionCN"
	Keda                         = "Keda"
	KnativeServingCrdInRegionCn  = "KnativeServingCRDInRegionCN"
	KnativeServingCrd            = "KnativeServingCRD"
	KnativeServingCoreInRegionCn = "KnativeServingCoreInRegionCN"
	KnativeServingCore           = "KnativeServingCore"
	KourierInRegionCn            = "KourierInRegionCN"
	Kourier                      = "Kourier"
	DefaultDomainInRegionCn      = "DefaultDomainInRegionCN"
	DefaultDomain                = "DefaultDomain"
	TektonPipelineInRegionCn     = "TektonPipelineInRegionCN"
	TektonPipeline               = "TektonPipeline"
	ShipwrightInRegionCn         = "ShipwrightInRegionCN"
	Shipwright                   = "Shipwright"
	CertManagerInRegionCn        = "CertManagerInRegionCN"
	CertManager                  = "CertManager"
	IngressNginxInRegionCn       = "IngressNginxInRegionCN"
	IngressNginx                 = "IngressNginx"
	OpenfunctionLatest           = "OpenFunctionLatest"
	OpenfunctionTmpl             = "OpenFunctionTemplate"

	DaprNamespace           = "dapr-system"
	KedaNamespace           = "keda"
	KnativeServingNamespace = "knative-serving"
	KourierNamespace        = "kourier-system"
	TektonPipelineNamespace = "tekton-pipelines"
	ShipwrightNamespace     = "shipwright-build"
	CertManagerNamespace    = "cert-manager"
	IngressNginxNamespace   = "ingress-nginx"
	OpenFunctionNamespace   = "openfunction"
)

var onlineFileMap map[string]string

func init() {
	onlineFileMap = map[string]string{
		DaprInRegionCn:               "https://openfunction.sh1a.qingstor.com/dapr/install.sh",
		Dapr:                         "https://raw.githubusercontent.com/dapr/cli/master/install/install.sh",
		KedaInRegionCn:               "https://openfunction.sh1a.qingstor.com/keda/v2.4.0/keda-2.4.0.yaml",
		Keda:                         "https://github.com/kedacore/keda/releases/download/v2.4.0/keda-2.4.0.yaml",
		KnativeServingCrdInRegionCn:  "https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-crds.yaml",
		KnativeServingCrd:            "https://github.com/knative/serving/releases/download/v0.26.0/serving-crds.yaml",
		KnativeServingCoreInRegionCn: "https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-core.yaml",
		KnativeServingCore:           "https://github.com/knative/serving/releases/download/v0.26.0/serving-core.yaml",
		KourierInRegionCn:            "https://openfunction.sh1a.qingstor.com/knative/net-kourier/v0.26.0/kourier.yaml",
		Kourier:                      "https://github.com/knative/net-kourier/releases/download/v0.26.0/kourier.yaml",
		DefaultDomainInRegionCn:      "https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-default-domain.yaml",
		DefaultDomain:                "https://github.com/knative/serving/releases/download/v0.26.0/serving-default-domain.yaml",
		TektonPipelineInRegionCn:     "https://openfunction.sh1a.qingstor.com/tekton/pipeline/v0.28.1/release.yaml",
		TektonPipeline:               "https://github.com/tektoncd/pipeline/releases/download/v0.28.1/release.yaml",
		ShipwrightInRegionCn:         "https://openfunction.sh1a.qingstor.com/shipwright/v0.6.0/release.yaml",
		Shipwright:                   "https://github.com/shipwright-io/build/releases/download/v0.6.0/release.yaml",
		CertManagerInRegionCn:        "https://openfunction.sh1a.qingstor.com/cert-manager/v1.5.4/cert-manager.yaml",
		CertManager:                  "https://github.com/jetstack/cert-manager/releases/download/v1.5.4/cert-manager.yaml",
		IngressNginxInRegionCn:       "https://openfunction.sh1a.qingstor.com/ingress-nginx/deploy/static/provider/cloud/deploy.yaml",
		IngressNginx:                 "https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml",
		OpenfunctionLatest:           "https://raw.githubusercontent.com/OpenFunction/OpenFunction/main/config/bundle.yaml",
		OpenfunctionTmpl:             "https://github.com/OpenFunction/OpenFunction/releases/download/%s/bundle.yaml",
	}
}

type Operator struct {
	os                     string
	version                string
	inRegionCN             bool
	verbose                bool
	executor               dependency.OperatorExecutor
	downloadDaprClientFunc func(inRegionCN bool, verbose bool) error
	timeout                time.Duration
}

func NewOperator(os string, version string, timeout time.Duration, inRegionCN bool, verbose bool) *Operator {
	op := &Operator{
		os:         os,
		version:    version,
		inRegionCN: inRegionCN,
		verbose:    verbose,
		timeout:    timeout,
	}
	switch os {
	case "linux":
		op.executor = linux.NewExecutor(verbose)
	}
	return op
}

func (o *Operator) DownloadDaprClient(ctx context.Context) error {
	return o.executor.DownloadDaprClient(o.inRegionCN)
}

func (o *Operator) InitDapr(ctx context.Context) error {
	cmd := "dapr init -k --runtime-version 1.4.3"
	if _, _, err := o.executor.Exec(cmd); err != nil {
		return err
	}
	return nil
}

func (o *Operator) InstallKeda(ctx context.Context) error {
	var yamlFile string
	if o.inRegionCN {
		yamlFile = onlineFileMap[KedaInRegionCn]
	} else {
		yamlFile = onlineFileMap[Keda]
	}
	return o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, yamlFile)
}

func (o *Operator) CheckKedaIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, KedaNamespace)
}

func (o *Operator) InstallKnativeServing(ctx context.Context) error {
	var crdYamlFile string
	var coreYamlFile string
	if o.inRegionCN {
		crdYamlFile = onlineFileMap[KnativeServingCrdInRegionCn]
		coreYamlFile = onlineFileMap[KnativeServingCoreInRegionCn]
	} else {
		crdYamlFile = onlineFileMap[KnativeServingCrd]
		coreYamlFile = onlineFileMap[KnativeServingCore]
	}

	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, crdYamlFile); err != nil {
		return err
	}
	return o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, coreYamlFile)
}

func (o *Operator) InstallKourier(ctx context.Context, cl *k8s.Clientset) error {
	var yamlFile string

	patchData := map[string]string{
		"ingress.class": "kourier.ingress.networking.knative.dev",
	}
	patchDataBytes, err := json.Marshal(patchData)
	if err != nil {
		return err
	}

	if o.inRegionCN {
		yamlFile = onlineFileMap[KourierInRegionCn]
	} else {
		yamlFile = onlineFileMap[Kourier]
	}

	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, yamlFile); err != nil {
		return err
	}

	if _, err := cl.CoreV1().ConfigMaps(KnativeServingNamespace).Patch(
		ctx,
		"config-network",
		types.MergePatchType,
		patchDataBytes,
		metav1.PatchOptions{},
	); err != nil {
		return err
	}
	return nil
}

func (o *Operator) ConfigKnativeServingDefaultDomain(ctx context.Context) error {
	var yamlFile string
	if o.inRegionCN {
		yamlFile = onlineFileMap[DefaultDomainInRegionCn]
	} else {
		yamlFile = onlineFileMap[DefaultDomain]
	}
	return o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, yamlFile)
}

func (o *Operator) CheckKnativeServingIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, KnativeServingNamespace)
}

func (o *Operator) InstallShipwright(ctx context.Context) error {
	var tektonPipelineYamlFile string
	var shipwrightYamlFile string

	if o.inRegionCN {
		tektonPipelineYamlFile = onlineFileMap[TektonPipelineInRegionCn]
		shipwrightYamlFile = onlineFileMap[ShipwrightInRegionCn]
	} else {
		tektonPipelineYamlFile = onlineFileMap[TektonPipeline]
		shipwrightYamlFile = onlineFileMap[Shipwright]
	}

	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, tektonPipelineYamlFile); err != nil {
		return err
	}
	return o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, shipwrightYamlFile)
}

func (o *Operator) CheckShipwrightIsReady(ctx context.Context, cl *k8s.Clientset) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	grp, ctx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		return checkDeploymentIsReady(ctx, cl, TektonPipelineNamespace)
	})

	grp.Go(func() error {
		return checkDeploymentIsReady(ctx, cl, ShipwrightNamespace)
	})
	return grp.Wait()
}

func (o *Operator) InstallCertManager(ctx context.Context) error {
	var yamlFile string
	if o.inRegionCN {
		yamlFile = onlineFileMap[CertManagerInRegionCn]
	} else {
		yamlFile = onlineFileMap[CertManager]
	}
	return o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, yamlFile)
}

func (o *Operator) CheckCertManagerIsReady(ctx context.Context, cl *k8s.Clientset) error {
	if err := checkDeploymentIsReady(ctx, cl, CertManagerNamespace); err != nil {
		return err
	} else {
		if err := checkPodIsReady(
			ctx,
			cl,
			CertManagerNamespace,
			fmt.Sprintf("%s=%s", k8sNameLabel, "webhook"),
		); err != nil {
			return err
		}
	}
	return nil
}

func (o *Operator) InstallIngressNginx(ctx context.Context) error {
	var yamlFile string
	if o.inRegionCN {
		yamlFile = onlineFileMap[IngressNginxInRegionCn]
	} else {
		yamlFile = onlineFileMap[IngressNginx]
	}
	return o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlApply, yamlFile)
}

func (o *Operator) CheckIngressNginxIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, IngressNginxNamespace)
}

func (o *Operator) InstallOpenFunction(ctx context.Context) error {
	var yamlFile string
	if o.version == "latest" {
		yamlFile = onlineFileMap[OpenfunctionLatest]
	} else {
		yamlFile = fmt.Sprintf(onlineFileMap[OpenfunctionTmpl], o.version)
	}
	if err := o.executor.KubectlApplyAndCreateAndDelete(
		ctx,
		KubectlCreate,
		yamlFile,
	); err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	return nil
}

func (o *Operator) CheckOpenFunctionIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, OpenFunctionNamespace)
}

func (o *Operator) UninstallDapr(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var cmd string

	cmd = "dapr uninstall -k --all"
	if _, _, err := o.executor.Exec(cmd); err != nil {
		return err
	}

	if err := cl.CoreV1().Namespaces().Delete(ctx, DaprNamespace, metav1.DeleteOptions{}); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, DaprNamespace)
	}
	return nil
}

func (o *Operator) UninstallKeda(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var yamlFile string

	if o.inRegionCN {
		yamlFile = onlineFileMap[KedaInRegionCn]
	} else {
		yamlFile = onlineFileMap[Keda]
	}
	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, yamlFile); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, KedaNamespace)
	}
	return nil
}

func (o *Operator) UninstallKnativeServing(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var crdYamlFile string
	var coreYamlFile string

	if o.inRegionCN {
		crdYamlFile = onlineFileMap[KnativeServingCrdInRegionCn]
		coreYamlFile = onlineFileMap[KnativeServingCoreInRegionCn]
	} else {
		crdYamlFile = onlineFileMap[KnativeServingCrd]
		coreYamlFile = onlineFileMap[KnativeServingCore]
	}

	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, crdYamlFile); err != nil {
		return err
	}
	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, coreYamlFile); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, KnativeServingNamespace)
	}
	return nil
}

func (o *Operator) UninstallKourier(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var yamlFile string

	if o.inRegionCN {
		yamlFile = onlineFileMap[KourierInRegionCn]
	} else {
		yamlFile = onlineFileMap[Kourier]
	}

	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, yamlFile); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, KourierNamespace)
	}
	return nil
}

func (o *Operator) UninstallShipwright(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var yamlFile string

	if o.inRegionCN {
		yamlFile = onlineFileMap[ShipwrightInRegionCn]
	} else {
		yamlFile = onlineFileMap[Shipwright]
	}

	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, yamlFile); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, ShipwrightNamespace)
	}
	return nil
}

func (o *Operator) UninstallTektonPipelines(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var yamlFile string

	if o.inRegionCN {
		yamlFile = onlineFileMap[TektonPipelineInRegionCn]
	} else {
		yamlFile = onlineFileMap[TektonPipeline]
	}

	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, yamlFile); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, TektonPipelineNamespace)
	}
	return nil
}

func (o *Operator) UninstallCertManager(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var yamlFile string

	if o.inRegionCN {
		yamlFile = onlineFileMap[CertManagerInRegionCn]
	} else {
		yamlFile = onlineFileMap[CertManager]
	}

	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, yamlFile); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, CertManagerNamespace)
	}
	return nil
}

func (o *Operator) UninstallIngressNginx(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var yamlFile string

	if o.inRegionCN {
		yamlFile = onlineFileMap[IngressNginxInRegionCn]
	} else {
		yamlFile = onlineFileMap[IngressNginx]
	}
	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, yamlFile); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, IngressNginxNamespace)
	}
	return nil
}

func (o *Operator) UninstallOpenFunction(ctx context.Context, cl *k8s.Clientset, waitForCleared bool) error {
	var yamlFile string

	if o.version == "latest" {
		yamlFile = onlineFileMap[OpenfunctionLatest]
	} else {
		yamlFile = fmt.Sprintf(onlineFileMap[OpenfunctionTmpl], o.version)
	}
	if err := o.executor.KubectlApplyAndCreateAndDelete(ctx, KubectlDelete, yamlFile); err != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, OpenFunctionNamespace)
	}

	return nil
}

func checkDeploymentIsReady(
	ctx context.Context,
	cl *k8s.Clientset,
	ns string,
) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if dpls, err := cl.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{}); err == nil {
				ready := 0
				for _, deploy := range dpls.Items {
					if status := getDeploymentStatusByType(
						deploy.Status.Conditions,
						appsv1.DeploymentAvailable,
					); status != nil && *status == corev1.ConditionTrue {
						ready += 1
					}
				}
				if len(dpls.Items) != ready {
					t.Reset(5 * time.Second)
				} else {
					return nil
				}
			}
		case <-ctx.Done():
			return errors.Wrap(
				ctx.Err(),
				"context marked done. stopping check loop",
			)
		}
	}
}

func getDeploymentStatusByType(
	conditions []appsv1.DeploymentCondition,
	deploymentType appsv1.DeploymentConditionType,
) *corev1.ConditionStatus {
	for _, condition := range conditions {
		if condition.Type == deploymentType {
			return &condition.Status
		}
	}
	return nil
}

func checkNamespaceIsCleared(
	ctx context.Context,
	cl *k8s.Clientset,
	ns string,
) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if _, err := cl.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{}); err != nil {
				return nil
			}
			t.Reset(5 * time.Second)
		case <-ctx.Done():
			return errors.Wrap(
				ctx.Err(),
				"context marked done. stopping check loop",
			)
		}
	}
}

func checkPodIsReady(
	ctx context.Context,
	cl *k8s.Clientset,
	ns string,
	label string,
) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if pods, err := cl.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: label}); err == nil {
				for _, pod := range pods.Items {
					if status := getPodStatusByType(
						pod.Status.Conditions,
						corev1.PodReady,
					); status != nil && *status == corev1.ConditionTrue {
						return nil
					}
				}
			}
			t.Reset(5 * time.Second)
		case <-ctx.Done():
			return errors.Wrap(
				ctx.Err(),
				"context marked done. stopping check loop",
			)
		}
	}
}

func getPodStatusByType(
	conditions []corev1.PodCondition,
	podType corev1.PodConditionType,
) *corev1.ConditionStatus {
	for _, condition := range conditions {
		if condition.Type == podType {
			return &condition.Status
		}
	}
	return nil
}

func GetExistComponentVersion(ctx context.Context, cl *k8s.Clientset, ns string, deploymentName string) string {
	if deploy, err := cl.AppsV1().Deployments(ns).Get(ctx, deploymentName, metav1.GetOptions{}); err != nil {
		return ""
	} else {
		var labels map[string]string
		if ns == IngressNginxNamespace || ns == KedaNamespace {
			labels = deploy.GetLabels()
		} else {
			labels = deploy.Spec.Template.GetLabels()
		}
		if version, ok := labels[k8sVersionLabel]; !ok {
			return ""
		} else {
			return version
		}
	}
}
