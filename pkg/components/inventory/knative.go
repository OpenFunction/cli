package inventory

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	KnativeServingName                          = "Knative Serving"
	KnativeServingRecordName                    = "knativeServing"
	KnativeServingVersionEnv                    = "KNATIVE_SERVING_VERSION"
	KnativeServingCrdYamlEnv                    = "KNATIVE_SERVING_CRD_YAML"
	KnativeServingCoreYamlEnv                   = "KNATIVE_SERVING_CORE_YAML"
	KnativeServingDefaultYamlFileTmpl           = "https://github.com/knative/serving/releases/download/%s%d.%d.%d/serving-%s.yaml"
	KnativeServingDefaultYamlFileTmplInRegionCN = "https://github.com/knative/serving/releases/download/%s%d.%d.%d/serving-%s.yaml"
)

type knativeServing struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewKnativeServing(serverVersion string, regionCN bool) (*knativeServing, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &knativeServing{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *knativeServing) GetVersion() string {
	if v, ok := os.LookupEnv(KnativeServingVersionEnv); ok && i.isValidVersion(v) {
		return v
	}
	return i.getDefaultVersion()
}

func (i *knativeServing) GetYamlFile(knVersion string) (map[string]string, error) {
	yamls := map[string]string{}
	if crdYaml, ok := os.LookupEnv(KnativeServingCrdYamlEnv); ok {
		if coreYaml, ok := os.LookupEnv(KnativeServingCoreYamlEnv); ok {
			yamls["CRD"] = crdYaml
			yamls["CORE"] = coreYaml
			return yamls, nil
		}
	}

	if v, err := version.ParseGeneric(knVersion); err != nil {
		return nil, err
	} else {
		switch v.Major() {
		case 0:
			if i.regionCN {
				yamls["CRD"] = fmt.Sprintf(KnativeServingDefaultYamlFileTmplInRegionCN, "v", v.Major(), v.Minor(), v.Patch(), "crds")
				yamls["CORE"] = fmt.Sprintf(KnativeServingDefaultYamlFileTmplInRegionCN, "v", v.Major(), v.Minor(), v.Patch(), "core")
				return yamls, nil
			}
			yamls["CRD"] = fmt.Sprintf(KnativeServingDefaultYamlFileTmpl, "v", v.Major(), v.Minor(), v.Patch(), "crds")
			yamls["CORE"] = fmt.Sprintf(KnativeServingDefaultYamlFileTmpl, "v", v.Major(), v.Minor(), v.Patch(), "core")
			return yamls, nil
		case 1:
			if i.regionCN {
				yamls["CRD"] = fmt.Sprintf(KnativeServingDefaultYamlFileTmplInRegionCN, "knative-v", v.Major(), v.Minor(), v.Patch(), "crds")
				yamls["CORE"] = fmt.Sprintf(KnativeServingDefaultYamlFileTmplInRegionCN, "knative-v", v.Major(), v.Minor(), v.Patch(), "core")
				return yamls, nil
			}
			yamls["CRD"] = fmt.Sprintf(KnativeServingDefaultYamlFileTmpl, "knative-v", v.Major(), v.Minor(), v.Patch(), "crds")
			yamls["CORE"] = fmt.Sprintf(KnativeServingDefaultYamlFileTmpl, "knative-v", v.Major(), v.Minor(), v.Patch(), "core")
			return yamls, nil
		default:
			return nil, errors.New("wrong format")
		}
	}
}

func (i *knativeServing) isValidVersion(knVersion string) bool {
	if v, err := version.ParseGeneric(knVersion); err != nil {
		return false
	} else {
		if i.serverMajorVersion == 1 {
			switch i.serverMinorVersion {
			case 17:
				if v.Major() == 0 && v.Minor() == 21 {
					return true
				}
			case 18:
				if v.Major() == 0 && v.Minor() >= 22 && v.Minor() <= 23 {
					return true
				}
			case 19:
				if v.Major() == 0 && v.Minor() >= 24 && v.Minor() <= 25 {
					return true
				}
			case 20:
				if (v.Major() == 0 && v.Minor() == 26) || (v.Major() == 1 && v.Minor() == 0) {
					return true
				}
			default:
				return false
			}
		}
		return false
	}
}

func (i *knativeServing) getDefaultVersion() string {
	if i.serverMajorVersion == 1 {
		switch i.serverMinorVersion {
		case 17:
			return DefaultKnativeServingVersionOnK8Sv117
		case 18:
			return DefaultKnativeServingVersionOnK8Sv118
		case 19:
			return DefaultKnativeServingVersionOnK8Sv119
		case 20:
			return DefaultKnativeServingVersionOnK8Sv120
		}
	}
	return DefaultKnativeServingVersionOnK8Sv120
}
