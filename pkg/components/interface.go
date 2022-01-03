package components

import (
	"context"

	"github.com/OpenFunction/cli/pkg/components/inventory"
)

const (
	OpenFunctionDir    = ".ofn"
	RecordFileNameTmpl = "%s-inventory.yaml"
)

// OperatorExecutor is an executor abstraction
// that guides the implementation of executors
// under different operating systems.
type OperatorExecutor interface {
	Exec(cmd string) (string, string, error)
	DownloadDaprClient(version string, inRegionCN bool) error
	KubectlExec(ctx context.Context, cmd string, wait bool) error
	RecordInventory(ctx context.Context, inventoryMap map[string]string) error
	GetInventoryRecord(ctx context.Context) (*inventory.Record, error)
	DownloadKind(ctx context.Context) error
	GetNodeIP(ctx context.Context) (string, error)
	CurlOpenFunction(ctx context.Context, endPoint string) (string, error)
}
