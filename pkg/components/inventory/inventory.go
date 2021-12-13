package inventory

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/util/version"
	k8s "k8s.io/client-go/kubernetes"
)

const (
	DefaultDaprVersion                          = "1.5.1"
	DefaultKedaVersion                          = "2.4.0"
	DefaultKnativeServingVersionOnK8Sv117       = "0.21.1"
	DefaultKnativeServingVersionOnK8Sv118       = "0.23.3"
	DefaultKnativeServingVersionOnK8Sv119       = "0.25.2"
	DefaultKnativeServingVersionOnK8Sv120       = "1.0.1"
	DefaultKourierVersionOnK8Sv117              = "0.21.1"
	DefaultKourierVersionOnK8Sv118              = "0.23.3"
	DefaultKourierVersionOnK8Sv119              = "0.25.2"
	DefaultKourierVersionOnK8Sv120              = "1.0.1"
	DefaultServingDefaultDomainVersionOnK8Sv117 = "0.21.1"
	DefaultServingDefaultDomainVersionOnK8Sv118 = "0.23.3"
	DefaultServingDefaultDomainVersionOnK8Sv119 = "0.25.2"
	DefaultServingDefaultDomainVersionOnK8Sv120 = "1.0.1"
	DefaultTektonPipelinesVersionOnK8Sv117      = "0.23.0"
	DefaultTektonPipelinesVersionOnK8Sv118      = "0.26.0"
	DefaultTektonPipelinesVersionOnK8Sv119      = "0.29.0"
	DefaultTektonPipelinesVersionOnK8Sv120      = "0.30.0"
	DefaultShipwrightVersion                    = "0.6.1"
	DefaultCertManagerVersion                   = "1.1.0"
	DefaultIngressNginxVersion                  = "1.5.4"
	DefaultOpenFunctionVersion                  = "0.4.0"
)

type Record struct {
	OpenFunction    string `yaml:"openFunction"`
	KnativeServing  string `yaml:"knativeServing,omitempty"`
	Kourier         string `yaml:"kourier,omitempty"`
	DefaultDomain   string `yaml:"defaultDomain,omitempty"`
	Keda            string `yaml:"keda,omitempty"`
	Dapr            string `yaml:"dapr,omitempty"`
	TektonPipelines string `yaml:"tektonPipelines,omitempty"`
	Shipwright      string `yaml:"shipwright,omitempty"`
	CertManager     string `yaml:"certManager,omitempty"`
	Ingress         string `yaml:"ingress,omitempty"`
}

type Interface interface {
	GetVersion() string
	GetYamlFile(version string) (map[string]string, error)
}

func getKubernetesServerVersion(cl *k8s.Clientset) (string, error) {
	if sv, err := cl.ServerVersion(); err != nil {
		return "", err
	} else {
		return sv.String(), nil
	}
}

func GetInventory(
	cl *k8s.Clientset,
	regionCN bool,
	withKnative bool,
	withKeda bool,
	withDapr bool,
	withShipwright bool,
	withCertManager bool,
	withIngress bool,
	openFunctionVersion string,
) (map[string]Interface, error) {
	serverVersion, err := getKubernetesServerVersion(cl)
	if err != nil {
		return nil, err
	}

	inventory := map[string]Interface{}

	if withKnative {
		if iv, err := NewKnativeServing(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[KnativeServingName] = iv
		}

		if iv, err := NewKourier(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[KourierName] = iv
		}

		if iv, err := NewDefaultDomain(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[ServingDefaultDomainName] = iv
		}
	}

	if withKeda {
		if iv, err := NewKeda(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[KedaName] = iv
		}
	}

	if withDapr {
		if iv, err := NewDapr(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[DaprName] = iv
		}
	}

	if withShipwright {
		if iv, err := NewTektonPipelines(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[TektonPipelinesName] = iv
		}

		if iv, err := NewShipwright(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[ShipwrightName] = iv
		}
	}

	if withCertManager && openFunctionVersion != "v0.3.1" {
		if iv, err := NewCertManager(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[CertManagerName] = iv
		}
	}

	if withIngress && openFunctionVersion == "latest" {
		if iv, err := NewIngressNginx(serverVersion, regionCN); err != nil {
			return nil, err
		} else {
			inventory[IngressName] = iv
		}
	}

	if iv, err := NewOpenFunction(serverVersion, openFunctionVersion, regionCN); err != nil {
		return nil, err
	} else {
		inventory[OpenFunctionName] = iv
	}

	return inventory, nil
}

func GetVersionMap(inventory map[string]Interface) map[string]string {
	versionMap := map[string]string{}
	for name, inv := range inventory {
		versionMap[name] = inv.GetVersion()
	}
	return versionMap
}

func isValidVersion(ver string) (*version.Version, bool) {
	if v, err := version.ParseGeneric(ver); err != nil {
		return nil, false
	} else {
		return v, true
	}
}

func NewRecord(inventoryMap map[string]string) (*Record, error) {
	r := &Record{}
	if inventoryMapBytes, err := json.Marshal(inventoryMap); err != nil {
		return nil, err
	} else {
		if err := json.Unmarshal(inventoryMapBytes, r); err != nil {
			return nil, err
		}
		return r, nil
	}
}

func (r *Record) Update(newRecord *Record) {
	r.OpenFunction = newRecord.OpenFunction

	if &newRecord.KnativeServing != nil {
		r.KnativeServing = newRecord.KnativeServing
	}

	if &newRecord.Kourier != nil {
		r.Kourier = newRecord.Kourier
	}

	if &newRecord.DefaultDomain != nil {
		r.DefaultDomain = newRecord.DefaultDomain
	}

	if &newRecord.Keda != nil {
		r.Keda = newRecord.Keda
	}

	if &newRecord.Dapr != nil {
		r.Dapr = newRecord.Dapr
	}

	if &newRecord.Shipwright != nil {
		r.Shipwright = newRecord.Shipwright
	}

	if &newRecord.TektonPipelines != nil {
		r.TektonPipelines = newRecord.TektonPipelines
	}

	if &newRecord.CertManager != nil {
		r.CertManager = newRecord.CertManager
	}

	if &newRecord.Ingress != nil {
		r.Ingress = newRecord.Ingress
	}
}

func (r *Record) ToMap(humanize bool) map[string]string {
	m := map[string]string{}
	if &r.OpenFunction != nil && r.OpenFunction != "" {
		if humanize {
			m[OpenFunctionName] = r.OpenFunction
		} else {
			m[OpenFunctionRecordName] = r.OpenFunction
		}
	}

	if &r.KnativeServing != nil && r.KnativeServing != "" {
		if humanize {
			m[KnativeServingName] = r.KnativeServing
		} else {
			m[KnativeServingRecordName] = r.KnativeServing
		}
	}

	if &r.Kourier != nil && r.Kourier != "" {
		if humanize {
			m[KourierName] = r.Kourier
		} else {
			m[KourierRecordName] = r.Kourier
		}
	}

	if &r.DefaultDomain != nil && r.DefaultDomain != "" {
		if humanize {
			m[ServingDefaultDomainName] = r.DefaultDomain
		} else {
			m[ServingDefaultDomainRecordName] = r.DefaultDomain
		}

	}

	if &r.Keda != nil && r.Keda != "" {
		if humanize {
			m[KedaName] = r.Keda
		} else {
			m[KedaRecordName] = r.Keda
		}
	}

	if &r.Dapr != nil && r.Dapr != "" {
		if humanize {
			m[DaprName] = r.Dapr
		} else {
			m[DaprRecordName] = r.Dapr
		}
	}

	if &r.Shipwright != nil && r.Shipwright != "" {
		if humanize {
			m[ShipwrightName] = r.Shipwright
		} else {
			m[ShipwrightRecordName] = r.Shipwright
		}
	}

	if &r.TektonPipelines != nil && r.TektonPipelines != "" {
		if humanize {
			m[TektonPipelinesName] = r.TektonPipelines
		} else {
			m[TektonPipelinesRecordName] = r.TektonPipelines
		}
	}

	if &r.CertManager != nil && r.CertManager != "" {
		if humanize {
			m[CertManagerName] = r.CertManager
		} else {
			m[CertManagerRecordName] = r.CertManager
		}
	}

	if &r.Ingress != nil && r.Ingress != "" {
		if humanize {
			m[IngressName] = r.Ingress
		} else {
			m[IngressRecordName] = r.Ingress
		}
	}

	return m
}
