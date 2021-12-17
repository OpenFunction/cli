package inventory

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	KedaName                          = "Keda"
	KedaRecordName                    = "keda"
	KedaVersionEnv                    = "KEDA_VERSION"
	KedaYamlEnv                       = "KEDA_YAML"
	KedaDefaultYamlFileTmpl           = "https://github.com/kedacore/keda/releases/download/%s%s/keda-%s.yaml"
	KedaDefaultYamlFileTmplInRegionCN = "https://openfunction.sh1a.qingstor.com/keda/%s%s/keda-%s.yaml"
)

type keda struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewKeda(serverVersion string, regionCN bool) (*keda, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &keda{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *keda) GetVersion() string {
	if v, ok := os.LookupEnv(KedaVersionEnv); ok {
		if v, ok := isValidVersion(v); ok {
			return fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		}
	}
	return i.getDefaultVersion()
}

func (i *keda) GetYamlFile(ver string) (map[string]string, error) {
	yamls := map[string]string{}
	if f, ok := os.LookupEnv(KedaYamlEnv); ok {
		yamls["MAIN"] = f
		return yamls, nil
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, err
	} else {
		kedaVersion := fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		if i.regionCN {
			yamls["MAIN"] = fmt.Sprintf(KedaDefaultYamlFileTmplInRegionCN, "v", kedaVersion, kedaVersion)
			return yamls, nil
		}
		yamls["MAIN"] = fmt.Sprintf(KedaDefaultYamlFileTmpl, "v", kedaVersion, kedaVersion)
		return yamls, nil
	}
}

func (i *keda) getDefaultVersion() string {
	return DefaultKedaVersion
}
