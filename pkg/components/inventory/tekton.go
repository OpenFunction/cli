package inventory

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	TektonPipelinesName                          = "Tekton Pipelines"
	TektonPipelinesRecordName                    = "tektonPipelines"
	TektonPipelinesVersionEnv                    = "TEKTON_PIPELINES_VERSION"
	TektonPipelinesYamlEnv                       = "TEKTON_PIPELINES_YAML"
	TektonPipelinesDefaultYamlFileTmpl           = "https://storage.googleapis.com/tekton-releases/pipeline/previous/%s%s/release.yaml"
	TektonPipelinesDefaultYamlFileTmplInRegionCN = "https://storage.googleapis.com/tekton-releases/pipeline/previous/%s%s/release.yaml"
)

type tektonPipelines struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewTektonPipelines(serverVersion string, regionCN bool) (*tektonPipelines, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &tektonPipelines{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *tektonPipelines) GetVersion() string {
	if v, ok := os.LookupEnv(TektonPipelinesVersionEnv); ok && i.isValidVersion(v) {
		return v
	}
	return i.getDefaultVersion()
}

func (i *tektonPipelines) GetYamlFile(ver string) (map[string]string, error) {
	yamls := map[string]string{}
	if f, ok := os.LookupEnv(TektonPipelinesYamlEnv); ok {
		yamls["MAIN"] = f
		return yamls, nil
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, err
	} else {
		tpVersion := fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		if i.regionCN {
			yamls["MAIN"] = fmt.Sprintf(TektonPipelinesDefaultYamlFileTmplInRegionCN, "v", tpVersion)
			return yamls, nil
		}
		yamls["MAIN"] = fmt.Sprintf(TektonPipelinesDefaultYamlFileTmpl, "v", tpVersion)
		return yamls, nil
	}
}

func (i *tektonPipelines) isValidVersion(ver string) bool {
	if v, err := version.ParseGeneric(ver); err != nil {
		return false
	} else {
		if i.serverMajorVersion == 1 {
			switch i.serverMinorVersion {
			case 17:
				if v.Major() == 0 && v.Minor() == 23 {
					return true
				}
			case 18:
				if v.Major() == 0 && v.Minor() >= 24 && v.Minor() <= 26 {
					return true
				}
			case 19:
				if v.Major() == 0 && v.Minor() >= 27 && v.Minor() <= 29 {
					return true
				}
			case 20:
				if v.Major() == 0 && v.Minor() == 30 {
					return true
				}
			default:
				return false
			}
		}
		return false
	}
}

func (i *tektonPipelines) getDefaultVersion() string {
	if i.serverMajorVersion == 1 {
		switch i.serverMinorVersion {
		case 17:
			return DefaultTektonPipelinesVersionOnK8Sv117
		case 18:
			return DefaultTektonPipelinesVersionOnK8Sv118
		case 19:
			return DefaultTektonPipelinesVersionOnK8Sv119
		case 20:
			return DefaultTektonPipelinesVersionOnK8Sv120
		}
	}
	return DefaultTektonPipelinesVersionOnK8Sv120
}
