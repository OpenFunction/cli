package inventory

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	ShipwrightName                          = "Shipwright"
	ShipwrightRecordName                    = "shipwright"
	ShipwrightVersionEnv                    = "SHIPWRIGHT_VERSION"
	ShipwrightYamlEnv                       = "SHIPWRIGHT_YAML"
	ShipwrightDefaultYamlFileTmpl           = "https://openfunction.sh1a.qingstor.com/shipwright/%s%s/release.yaml"
	ShipwrightDefaultYamlFileTmplInRegionCN = "https://openfunction.sh1a.qingstor.com/shipwright/%s%s/release.yaml"
)

type shipwright struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewShipwright(serverVersion string, regionCN bool) (*shipwright, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &shipwright{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *shipwright) GetVersion() string {
	if v, ok := os.LookupEnv(ShipwrightVersionEnv); ok && i.isValidVersion(v) {
		return v
	}
	return i.getDefaultVersion()
}

func (i *shipwright) GetYamlFile(ver string) (map[string]string, error) {
	yamls := map[string]string{}
	if f, ok := os.LookupEnv(ShipwrightYamlEnv); ok {
		yamls["MAIN"] = f
		return yamls, nil
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, err
	} else {
		swVersion := fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		if i.regionCN {
			yamls["MAIN"] = fmt.Sprintf(ShipwrightDefaultYamlFileTmplInRegionCN, "v", swVersion)
			return yamls, nil
		}
		yamls["MAIN"] = fmt.Sprintf(ShipwrightDefaultYamlFileTmpl, "v", swVersion)
		return yamls, nil
	}
}

func (i *shipwright) isValidVersion(ver string) bool {
	if ver != "0.6.1" {
		return false
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return false
	} else {
		if v.Major() == 0 && v.Minor() == 6 && v.Patch() == 1 {
			return true
		}
		return false
	}
}

func (i *shipwright) getDefaultVersion() string {
	return DefaultShipwrightVersion
}
