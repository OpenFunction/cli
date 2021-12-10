package inventory

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	OpenFunctionName                          = "OpenFunction"
	OpenFunctionRecordName                    = "openFunction"
	OpenFunctionYamlEnv                       = "OPENFUNCTION_YAML"
	OpenFunctionDefaultYamlFileTmpl           = "https://github.com/OpenFunction/OpenFunction/releases/download/%s%s/bundle.yaml"
	OpenFunctionDefaultYamlFileTmplInRegionCN = "https://github.com/OpenFunction/OpenFunction/releases/download/%s%s/bundle.yaml"
)

type openFunction struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
	version            string
}

func NewOpenFunction(serverVersion string, openFunctionVersion string, regionCN bool) (*openFunction, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &openFunction{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
		version:            openFunctionVersion,
	}, nil
}

func (i *openFunction) GetVersion() string {
	if i.version == "latest" {
		return i.version
	}

	if v, ok := isValidVersion(i.version); ok {
		return fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
	}

	return i.getDefaultVersion()
}

func (i *openFunction) GetYamlFile(ver string) (map[string]string, error) {
	yamls := map[string]string{}
	if f, ok := os.LookupEnv(OpenFunctionYamlEnv); ok {
		yamls["MAIN"] = f
		return yamls, nil
	}

	if ver == "latest" {
		yamls["MAIN"] = "https://raw.githubusercontent.com/OpenFunction/OpenFunction/main/config/bundle.yaml"
		return yamls, nil
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, err
	} else {
		ofVersion := fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		if i.regionCN {
			yamls["MAIN"] = fmt.Sprintf(OpenFunctionDefaultYamlFileTmplInRegionCN, "v", ofVersion)
			return yamls, nil
		}
		yamls["MAIN"] = fmt.Sprintf(OpenFunctionDefaultYamlFileTmpl, "v", ofVersion)
		return yamls, nil
	}
}

func (i *openFunction) getDefaultVersion() string {
	return DefaultOpenFunctionVersion
}
