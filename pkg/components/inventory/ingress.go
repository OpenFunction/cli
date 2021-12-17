package inventory

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	IngressName                          = "IngressNginx"
	IngressRecordName                    = "ingress"
	IngressVersionEnv                    = "INGRESS_NGINX_VERSION"
	IngressYamlEnv                       = "INGRESS_NGINX_YAML"
	IngressDefaultYamlFileTmpl           = "https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-%s%s/deploy/static/provider/cloud/deploy.yaml"
	IngressDefaultYamlFileTmplInRegionCN = "https://openfunction.sh1a.qingstor.com/ingress-nginx/%s%s/deploy.yml"
)

type ingress struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewIngressNginx(serverVersion string, regionCN bool) (*ingress, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &ingress{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *ingress) GetVersion() string {
	if v, ok := os.LookupEnv(IngressVersionEnv); ok && i.isValidVersion(v) {
		return v
	}
	return i.getDefaultVersion()
}

func (i *ingress) GetYamlFile(ver string) (map[string]string, error) {
	yamls := map[string]string{}
	if f, ok := os.LookupEnv(IngressYamlEnv); ok {
		yamls["MAIN"] = f
		return yamls, nil
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, err
	} else {
		ingressVersion := fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		if i.regionCN {
			yamls["MAIN"] = fmt.Sprintf(IngressDefaultYamlFileTmplInRegionCN, "v", ingressVersion)
			return yamls, nil
		}
		yamls["MAIN"] = fmt.Sprintf(IngressDefaultYamlFileTmpl, "v", ingressVersion)
		return yamls, nil
	}
}

func (i *ingress) isValidVersion(ver string) bool {
	if _, err := version.ParseGeneric(ver); err != nil {
		return false
	}
	return true
}

func (i *ingress) getDefaultVersion() string {
	return DefaultIngressNginxVersion
}
