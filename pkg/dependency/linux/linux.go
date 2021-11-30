package linux

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/OpenFunction/cli/pkg/dependency"
	"github.com/pkg/errors"
)

type Executor struct {
	verbose bool
}

func NewExecutor(verbose bool) dependency.OperatorExecutor {
	return &Executor{
		verbose: verbose,
	}
}

func (e *Executor) Exec(cmd string) (string, string, error) {
	command := exec.Command("/bin/bash", "-c", cmd)
	if e.verbose {
		fmt.Printf("command:\n%s\n", command)
	}

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	if err != nil {
		if e.verbose {
			fmt.Printf("out:\n%s\nerr:\n%s\n", outStr, errStr)
			fmt.Printf("command: %s, failed with %s\n", cmd, err)
		}
		return outStr, errStr, errors.Wrap(err, fmt.Sprintf("\n%s", errStr))
	}
	return outStr, errStr, nil
}

func (e *Executor) DownloadDaprClient(inRegionCN bool) error {
	var cmd string
	if inRegionCN {
		cmd = "wget -q https://openfunction.sh1a.qingstor.com/dapr/install.sh -O - | /bin/bash -s 1.4.0"
	} else {
		cmd = "wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash -s 1.4.0"
	}
	if _, _, err := e.Exec(cmd); err != nil {
		return err
	}
	return nil
}

func (e *Executor) GetExistDaprVerion() string {
	cmd := "kubectl get namespace dapr-system"
	if _, _, err := e.Exec(cmd); err != nil {
		return "None"
	} else {
		cmd = "kubectl get po -n dapr-system -L app.kubernetes.io/version|grep operator|awk '{ print $6 }'"
		if out, _, err := e.Exec(cmd); err != nil {
			return "None"
		} else {
			return out
		}
	}
}

func (e *Executor) KubectlApplyAndCreateAndDelete(
	ctx context.Context,
	operator string,
	yamlFile string,
) error {
	cmd := fmt.Sprintf("kubectl %s --filename %s", operator, yamlFile)

	if _, _, err := e.Exec(cmd); err != nil {
		return err
	}
	return nil
}
