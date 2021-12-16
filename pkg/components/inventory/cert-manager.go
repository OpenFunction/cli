package inventory

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	CertManagerName                          = "CertManager"
	CertManagerRecordName                    = "CertManager"
	CertManagerVersionEnv                    = "CERT_MANAGER_VERSION"
	CertManagerYamlEnv                       = "CERT_MANAGER_YAML"
	CertManagerDefaultYamlFileTmpl           = "https://github.com/jetstack/cert-manager/releases/download/%s%s/cert-manager.yaml"
	CertManagerDefaultYamlFileTmplInRegionCN = "https://openfunction.sh1a.qingstor.com/cert-manager/%s%s/cert-manager.yaml"
)

type certManager struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewCertManager(serverVersion string, regionCN bool) (*certManager, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &certManager{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *certManager) GetVersion() string {
	if v, ok := os.LookupEnv(CertManagerVersionEnv); ok && i.isValidVersion(v) {
		return v
	}
	return i.getDefaultVersion()
}

func (i *certManager) GetYamlFile(ver string) (map[string]string, error) {
	yamls := map[string]string{}
	if f, ok := os.LookupEnv(CertManagerYamlEnv); ok {
		yamls["MAIN"] = f
		return yamls, nil
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, err
	} else {
		cmVersion := fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		if i.regionCN {
			yamls["MAIN"] = fmt.Sprintf(CertManagerDefaultYamlFileTmplInRegionCN, "v", cmVersion)
			return yamls, nil
		}
		yamls["MAIN"] = fmt.Sprintf(CertManagerDefaultYamlFileTmpl, "v", cmVersion)
		return yamls, nil
	}
}

func (i *certManager) isValidVersion(ver string) bool {
	if _, err := version.ParseGeneric(ver); err != nil {
		return false
	}
	return true
}

func (i *certManager) getDefaultVersion() string {
	return DefaultCertManagerVersion
}
