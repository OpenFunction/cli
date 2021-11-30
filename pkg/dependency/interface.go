package dependency

import (
	"context"
)

// OperatorExecutor is an executor abstraction
// that guides the implementation of executors
// under different operating systems.
type OperatorExecutor interface {
	Exec(cmd string) (string, string, error)
	DownloadDaprClient(inRegionCN bool) error
	GetExistDaprVerion() string
	KubectlApplyAndCreateAndDelete(ctx context.Context, operator string, yamlFile string) error
}
