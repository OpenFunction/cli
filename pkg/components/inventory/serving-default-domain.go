package inventory

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	ServingDefaultDomainName                          = "DefaultDomain"
	ServingDefaultDomainRecordName                    = "defaultDomain"
	ServingDefaultDomainVersionEnv                    = "DEFAULT_DOMAIN_VERSION"
	ServingDefaultDomainYamlEnv                       = "DEFAULT_DOMAIN_YAML"
	ServingDefaultDomainDefaultYamlFileTmpl           = "https://github.com/knative/serving/releases/download/%s%s/serving-default-domain.yaml"
	ServingDefaultDomainDefaultYamlFileTmplInRegionCN = "https://github.com/knative-sandbox/net-kourier/releases/download/%s%d.%d.%d/release.yaml"
)

type defaultDomain struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewDefaultDomain(serverVersion string, regionCN bool) (*defaultDomain, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &defaultDomain{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *defaultDomain) GetVersion() string {
	if v, ok := os.LookupEnv(ServingDefaultDomainVersionEnv); ok && i.isValidVersion(v) {
		return v
	}
	return i.getDefaultVersion()
}

func (i *defaultDomain) GetYamlFile(ver string) (map[string]string, error) {
	yamls := map[string]string{}
	if f, ok := os.LookupEnv(ServingDefaultDomainYamlEnv); ok {
		yamls["MAIN"] = f
		return yamls, nil
	}

	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, err
	} else {
		ddVersion := fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		switch v.Major() {
		case 0:
			if i.regionCN {
				yamls["MAIN"] = fmt.Sprintf(ServingDefaultDomainDefaultYamlFileTmplInRegionCN, "v", v.Major(), v.Minor(), v.Patch())
				return yamls, nil
			}
			yamls["MAIN"] = fmt.Sprintf(ServingDefaultDomainDefaultYamlFileTmpl, "v", ddVersion)
			return yamls, nil
		case 1:
			if i.regionCN {
				yamls["MAIN"] = fmt.Sprintf(ServingDefaultDomainDefaultYamlFileTmplInRegionCN, "knative-v", v.Major(), v.Minor(), v.Patch())
				return yamls, nil
			}
			yamls["MAIN"] = fmt.Sprintf(ServingDefaultDomainDefaultYamlFileTmpl, "knative-v", ddVersion)
			return yamls, nil
		default:
			return nil, errors.New("wrong format")
		}
	}
}

func (i *defaultDomain) isValidVersion(knVersion string) bool {
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

func (i *defaultDomain) getDefaultVersion() string {
	if i.serverMajorVersion == 1 {
		switch i.serverMinorVersion {
		case 17:
			return DefaultServingDefaultDomainVersionOnK8Sv117
		case 18:
			return DefaultServingDefaultDomainVersionOnK8Sv118
		case 19:
			return DefaultServingDefaultDomainVersionOnK8Sv119
		case 20:
			return DefaultServingDefaultDomainVersionOnK8Sv120
		}
	}
	return DefaultServingDefaultDomainVersionOnK8Sv120
}
