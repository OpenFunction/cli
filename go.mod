module github.com/OpenFunction/cli

go 1.16

require (
	github.com/openfunction v0.3.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/cli-runtime v0.22.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/component-base v0.22.0
	k8s.io/klog/v2 v2.10.0
)

replace (
	github.com/openfunction => github.com/OpenFunction/OpenFunction v0.3.0
	github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2
	k8s.io/api => k8s.io/api v0.0.0-20210716001550-68328c152cca
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20210712060818-a644435e2c13
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20210803003910-24147945b9ef
	k8s.io/client-go => k8s.io/client-go v0.0.0-20210803001025-5629b666e53e
)
