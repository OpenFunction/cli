package inventory

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	KourierName                          = "Kourier"
	KourierRecordName                    = "kourier"
	KourierVersionEnv                    = "KOURIER_VERSION"
	KourierYamlEnv                       = "KOURIER_YAML"
	KourierDefaultYamlFileTmpl           = "https://github.com/knative-sandbox/net-kourier/releases/download/%s%d.%d.%d/kourier.yaml"
	KourierDefaultYamlFileTmplInRegionCN = "https://github.com/knative-sandbox/net-kourier/releases/download/%s%d.%d.%d/release.yaml"
)

type kourier struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewKourier(serverVersion string, regionCN bool) (*kourier, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &kourier{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *kourier) GetVersion() string {
	if v, ok := os.LookupEnv(KourierVersionEnv); ok && i.isValidVersion(v) {
		return v
	}
	return i.getDefaultVersion()
}

func (i *kourier) GetYamlFile(ver string) (map[string]string, error) {
	yamls := map[string]string{}
	if f, ok := os.LookupEnv(KourierYamlEnv); ok {
		yamls["MAIN"] = f
		return yamls, nil
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, err
	} else {
		switch v.Major() {
		case 0:
			if i.regionCN {
				yamls["MAIN"] = fmt.Sprintf(KourierDefaultYamlFileTmplInRegionCN, "v", v.Major(), v.Minor(), v.Patch())
				return yamls, nil
			}
			yamls["MAIN"] = fmt.Sprintf(KourierDefaultYamlFileTmpl, "v", v.Major(), v.Minor(), v.Patch())
			return yamls, nil
		case 1:
			if i.regionCN {
				yamls["MAIN"] = fmt.Sprintf(KourierDefaultYamlFileTmpl, "knative-v", v.Major(), v.Minor(), v.Patch())
				return yamls, nil
			}
			yamls["MAIN"] = fmt.Sprintf(KourierDefaultYamlFileTmpl, "knative-v", v.Major(), v.Minor(), v.Patch())
			return yamls, nil
		default:
			return nil, errors.New("wrong format")
		}
	}
}

func (i *kourier) isValidVersion(knVersion string) bool {
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

func (i *kourier) getDefaultVersion() string {
	if i.serverMajorVersion == 1 {
		switch i.serverMinorVersion {
		case 17:
			return DefaultKourierVersionOnK8Sv117
		case 18:
			return DefaultKourierVersionOnK8Sv118
		case 19:
			return DefaultKourierVersionOnK8Sv119
		case 20:
			return DefaultKourierVersionOnK8Sv120
		}
	}
	return DefaultKourierVersionOnK8Sv120
}
