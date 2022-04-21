package common

import (
	"context"
	"encoding/json"
	"fmt"
	ospkg "os"
	"strings"
	"time"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/OpenFunction/cli/pkg/components"
	"github.com/OpenFunction/cli/pkg/components/inventory"
	"github.com/OpenFunction/cli/pkg/components/linux"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	k8s "k8s.io/client-go/kubernetes"
)

const (
	k8sVersionLabel = "app.kubernetes.io/version"
	k8sNameLabel    = "app.kubernetes.io/name"

	DaprNamespace           = "dapr-system"
	KedaNamespace           = "keda"
	KnativeServingNamespace = "knative-serving"
	KourierNamespace        = "kourier-system"
	TektonPipelineNamespace = "tekton-pipelines"
	ShipwrightNamespace     = "shipwright-build"
	CertManagerNamespace    = "cert-manager"
	IngressNginxNamespace   = "ingress-nginx"
	OpenFunctionNamespace   = "openfunction"

	BaseVersion   = "v0.3.1"
	LatestVersion = "latest"
)

type Operator struct {
	os         string
	version    string
	inRegionCN bool
	verbose    bool
	executor   components.OperatorExecutor
	timeout    time.Duration
	Inventory  map[string]inventory.Interface
	Records    *inventory.Record
}

type PatchExternalIP struct {
	Spec Spec `json:"spec"`
}
type Spec struct {
	MyType      string   `json:"type"`
	ExternalIPs []string `json:"externalIPs"`
}

func NewOperator(os, arch, version string, timeout time.Duration, inRegionCN bool, verbose bool) *Operator {
	op := &Operator{
		os:         os,
		version:    version,
		inRegionCN: inRegionCN,
		verbose:    verbose,
		timeout:    timeout,
	}

	if arch != "amd64" {
		fmt.Fprint(ospkg.Stderr, "unsupported arch: ", arch)
		ospkg.Exit(1)
	}

	switch os {
	case "linux", "darwin":
		op.executor = linux.NewExecutor(verbose)
	default:
		fmt.Fprint(ospkg.Stderr, "unsupported os: ", os)
		ospkg.Exit(1)
	}
	return op
}

func (o *Operator) RecordInventory(ctx context.Context) error {
	if o.Records == nil {
		return errors.New("the inventory record is nil")
	}
	return o.executor.RecordInventory(ctx, o.Records.ToMap(false))
}

func (o *Operator) GetInventoryRecord(ctx context.Context, humanize bool) (map[string]string, error) {
	if rec, err := o.executor.GetInventoryRecord(ctx); err != nil {
		return nil, err
	} else {
		o.Records = rec
		return rec.ToMap(humanize), nil
	}
}

func (o *Operator) DownloadDaprClient(ctx context.Context, daprVersion string) error {
	return o.executor.DownloadDaprClient(daprVersion, o.inRegionCN)
}

func (o *Operator) InitDapr(ctx context.Context, daprVersion string) error {
	cmd := fmt.Sprintf("dapr init -k --log-as-json --runtime-version %s", daprVersion)
	if _, _, err := o.executor.Exec(cmd); err != nil && !strings.Contains(err.Error(), "still in use") {
		return err
	}
	return nil
}

func (o *Operator) InstallKeda(ctx context.Context, yamlFile string) error {
	cmd := fmt.Sprintf("apply -f %s", yamlFile)
	return o.executor.KubectlExec(ctx, cmd, false)
}

func (o *Operator) CheckKedaIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, KedaNamespace)
}

func (o *Operator) InstallKnativeServing(ctx context.Context, crdYamlFile string, coreYamlFile string) error {
	// Ensure that the CRDs are ready before installing the CORE file
	// See more at: https://github.com/knative/serving/issues/6571
	applyCore := func(ctx context.Context) error {
		ctx, done := context.WithCancel(ctx)
		defer done()

		t := time.NewTicker(5 * time.Second)
		defer t.Stop()

		cmd := fmt.Sprintf("apply -f %s", coreYamlFile)

		for {
			select {
			case <-t.C:
				if err := o.executor.KubectlExec(ctx, cmd, false); err != nil {
					if strings.Contains(err.Error(), "no matches for kind") {
						t.Reset(5 * time.Second)
						continue
					}
					return err
				}
				return nil
			case <-ctx.Done():
				return errors.Wrap(
					ctx.Err(),
					"context marked done. stopping check loop",
				)
			}
		}
	}

	cmd := fmt.Sprintf("apply -f %s", crdYamlFile)
	if err := o.executor.KubectlExec(ctx, cmd, true); err != nil {
		return err
	}

	return applyCore(ctx)
}

func (o *Operator) InstallKourier(ctx context.Context, cl *k8s.Clientset, yamlFile string) error {
	patchData := map[string]map[string]string{
		"data": {
			"ingress.class": "kourier.ingress.networking.knative.dev",
		},
	}
	patchDataBytes, err := json.Marshal(patchData)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("apply -f %s", yamlFile)
	if err := o.executor.KubectlExec(ctx, cmd, false); err != nil {
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

func (o *Operator) ConfigKnativeServingDefaultDomain(ctx context.Context, yamlFile string) error {
	cmd := fmt.Sprintf("apply -f %s", yamlFile)
	return o.executor.KubectlExec(ctx, cmd, false)
}

func (o *Operator) CheckKnativeServingIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, KnativeServingNamespace)
}

func (o *Operator) CheckKourierIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, KourierNamespace)
}

func (o *Operator) InstallTektonPipelines(ctx context.Context, yamlFile string) error {
	cmd := fmt.Sprintf("apply -f %s", yamlFile)
	return o.executor.KubectlExec(ctx, cmd, true)
}

func (o *Operator) InstallShipwright(ctx context.Context, yamlFile string) error {
	cmd := fmt.Sprintf("apply -f %s", yamlFile)
	return o.executor.KubectlExec(ctx, cmd, false)
}

func (o *Operator) CheckShipwrightIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, ShipwrightNamespace)
}

func (o *Operator) CheckTektonPipelinesIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, TektonPipelineNamespace)
}

func (o *Operator) InstallCertManager(ctx context.Context, yamlFile string) error {
	cmd := fmt.Sprintf("apply -f %s", yamlFile)
	return o.executor.KubectlExec(ctx, cmd, false)
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

func (o *Operator) InstallIngressNginx(ctx context.Context, yamlFile string) error {
	cmd := fmt.Sprintf("apply -f %s", yamlFile)
	return o.executor.KubectlExec(ctx, cmd, false)
}

func (o *Operator) CheckIngressNginxIsReady(ctx context.Context, cl *k8s.Clientset) error {
	return checkDeploymentIsReady(ctx, cl, IngressNginxNamespace)
}

func (o *Operator) InstallOpenFunction(ctx context.Context, yamlFile string) error {
	var cmd string
	if o.version == "v0.3.1" {
		cmd = fmt.Sprintf("apply -f %s", yamlFile)
		if err := o.executor.KubectlExec(ctx, cmd, false); err != nil {
			return err
		}
	} else {
		cmd = fmt.Sprintf("create -f %s", yamlFile)
		if err := o.executor.KubectlExec(
			ctx,
			cmd,
			false,
		); err != nil && !strings.Contains(err.Error(), "already exists") {
			return err
		}
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

	cmd = fmt.Sprintf("delete namespace %s", DaprNamespace)
	if err := o.executor.KubectlExec(ctx, cmd, false); util.IgnoreNotFoundErr(err) != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, DaprNamespace)
	}
	return nil
}

func (o *Operator) UninstallKnativeServing(
	ctx context.Context,
	cl *k8s.Clientset,
	crdYamlFile string,
	coreYamlFile string,
	waitForCleared bool,
) error {
	var cmd string
	cmd = fmt.Sprintf("delete -f %s", coreYamlFile)
	if err := o.executor.KubectlExec(ctx, cmd, true); util.IgnoreNotFoundErr(err) != nil {
		return err
	}
	cmd = fmt.Sprintf("delete -f %s", crdYamlFile)
	if err := o.executor.KubectlExec(ctx, cmd, true); util.IgnoreNotFoundErr(err) != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, KnativeServingNamespace)
	}
	return nil
}

func (o *Operator) Uninstall(
	ctx context.Context,
	cl *k8s.Clientset,
	yamlFile string,
	namespace string,
	waitForDelete bool,
	waitForCleared bool,
) error {
	cmd := fmt.Sprintf("delete -f %s", yamlFile)
	if err := o.executor.KubectlExec(ctx, cmd, waitForDelete); util.IgnoreNotFoundErr(err) != nil {
		return err
	}

	if waitForCleared {
		return checkNamespaceIsCleared(ctx, cl, namespace)
	}
	return nil
}

func (o *Operator) DownloadKind(ctx context.Context) error {
	return o.executor.DownloadKind(ctx)
}

func (o *Operator) CreateKindCluster(ctx context.Context) error {
	cmd := "kind create cluster --name openfunction"
	if _, _, err := o.executor.Exec(cmd); err != nil {
		return err
	}
	return nil
}

func (o *Operator) DeleteKind(ctx context.Context) error {
	cmd := "kind delete cluster --name openfunction"
	if _, _, err := o.executor.Exec(cmd); err != nil {
		return err
	}
	return nil
}

func (o *Operator) RunOpenFunction(ctx context.Context, demoYamlFile string) error {
	cmd := fmt.Sprintf("apply -f %s", demoYamlFile)
	return o.executor.KubectlExec(ctx, cmd, false)
}

func (o *Operator) GetNodeIP(ctx context.Context) (string, error) {
	return o.executor.GetNodeIP(ctx)
}

func (o *Operator) PatchExternalIP(ctx context.Context, cl *k8s.Clientset, ip string) error {

	patchData := PatchExternalIP{
		Spec: Spec{
			MyType:      "LoadBalancer",
			ExternalIPs: []string{ip},
		},
	}

	patchDataBytes, err := json.Marshal(patchData)
	if err != nil {
		return err
	}

	if _, err := cl.CoreV1().Services(KourierNamespace).Patch(
		ctx,
		"kourier",
		types.MergePatchType,
		patchDataBytes,
		metav1.PatchOptions{},
	); err != nil {
		return err
	}

	return nil
}

func (o *Operator) PatchMagicDNS(ctx context.Context, cl *k8s.Clientset, ip string) error {

	patchData := map[string]map[string]string{
		"data": {fmt.Sprintf("%s.sslip.io", ip): ""},
	}
	patchDataBytes, err := json.Marshal(patchData)
	if err != nil {
		return err
	}

	if _, err := cl.CoreV1().ConfigMaps(KnativeServingNamespace).Patch(
		ctx,
		"config-domain",
		types.MergePatchType,
		patchDataBytes,
		metav1.PatchOptions{},
	); err != nil {
		return err
	}
	return nil
}

func (o *Operator) PrintEndpoint(ctx context.Context) (string, error) {
	statusCMD := "kubectl get ksvc -l openfunction.io/serving=$(kubectl get functions function-sample-serving-only -o jsonpath='{.status.serving.resourceRef}') -o jsonpath='{.items[0].status.conditions[2].status}'"
	for {
		status, _, err := o.executor.Exec(statusCMD)
		if err != nil {
			return "", err
		}
		if status == "True" {
			break
		} else {
			time.Sleep(2 * time.Second)
		}

	}
	endpointCMD := "kubectl get ksvc -l openfunction.io/serving=$(kubectl get functions function-sample-serving-only -o jsonpath='{.status.serving.resourceRef}') -o jsonpath='{.items[0].status.url}'"
	endpoint, _, err := o.executor.Exec(endpointCMD)
	if err != nil {
		return "", err
	}
	return endpoint, nil
}

func (o *Operator) CurlOpenFunction(ctx context.Context, endPoint string) (string, error) {
	return o.executor.CurlOpenFunction(ctx, endPoint)
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

func IsComponentExist(ctx context.Context, cl *k8s.Clientset, ns string, resourceName string) bool {
	// For the serving-default-domain component,
	// we determine whether the component exists
	// based on the status.active number of its Jobs resource.
	if resourceName == "default-domain" {
		if job, err := cl.BatchV1().Jobs(ns).Get(ctx, resourceName, metav1.GetOptions{}); err != nil {
			return false
		} else {
			active := job.Status.Active
			return active >= 1
		}
	} else {
		if deploy, err := cl.AppsV1().Deployments(ns).Get(ctx, resourceName, metav1.GetOptions{}); err != nil {
			return false
		} else {
			if status := getDeploymentStatusByType(
				deploy.Status.Conditions,
				appsv1.DeploymentAvailable,
			); status != nil && *status == corev1.ConditionTrue {
				return true
			}
			return false
		}
	}
}

func IsVersionValid(ofVersion *version.Version) (bool, error) {
	base, err := version.ParseGeneric(BaseVersion)
	if err != nil {
		return false, err
	}
	if ofVersion.LessThan(base) {
		return false, nil
	} else {
		return true, nil
	}
}
