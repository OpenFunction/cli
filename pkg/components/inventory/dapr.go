package inventory

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	DaprName       = "Dapr"
	DaprRecordName = "dapr"
	DaprVersionEnv = "DAPR_VERSION"
)

type dapr struct {
	serverMajorVersion uint
	serverMinorVersion uint
	serverPatchVersion uint
	yamlTmpl           string
	regionCN           bool
}

func NewDapr(serverVersion string, regionCN bool) (*dapr, error) {
	sv, err := version.ParseGeneric(serverVersion)
	if err != nil {
		return nil, err
	}
	return &dapr{
		serverMajorVersion: sv.Major(),
		serverMinorVersion: sv.Minor(),
		serverPatchVersion: sv.Patch(),
		regionCN:           regionCN,
	}, nil
}

func (i *dapr) GetVersion() string {
	if v, ok := os.LookupEnv(DaprVersionEnv); ok {
		if v, ok := isValidVersion(v); ok {
			return fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		}
	}
	return i.getDefaultVersion()
}

func (i *dapr) GetYamlFile(ver string) (map[string]string, error) {
	return nil, nil
}

func (i *dapr) getDefaultVersion() string {
	return DefaultDaprVersion
}
