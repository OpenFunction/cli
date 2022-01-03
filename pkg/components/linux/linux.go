package linux

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/OpenFunction/cli/pkg/components"
	"github.com/OpenFunction/cli/pkg/components/inventory"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Executor struct {
	verbose bool
}

func NewExecutor(verbose bool) components.OperatorExecutor {
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

func (e *Executor) DownloadDaprClient(daprVersion string, inRegionCN bool) error {
	var cmd string
	if inRegionCN {
		cmd = fmt.Sprintf(
			"wget -q https://openfunction.sh1a.qingstor.com/dapr/install.sh -O - | /bin/bash -s %s", daprVersion)
	} else {
		cmd = fmt.Sprintf(
			"wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash -s %s", daprVersion)
	}
	if _, _, err := e.Exec(cmd); err != nil {
		return err
	}
	return nil
}

func (e *Executor) KubectlExec(
	ctx context.Context,
	cmd string,
	wait bool,
) error {
	var kubectlCMD string
	kubectlCMD = fmt.Sprintf("kubectl %s", cmd)

	// The `kubectl create` command does not support using the --wait option.
	if !wait && !strings.Contains(kubectlCMD, "create") {
		kubectlCMD += " --wait=false"
	}

	if _, _, err := e.Exec(kubectlCMD); err != nil {
		return err
	}
	return nil
}

func (e *Executor) getClusterName(ctx context.Context) (string, error) {
	cmd := "kubectl config current-context"

	if out, _, err := e.Exec(cmd); err != nil {
		return "", err
	} else {
		return strings.TrimSuffix(out, "\n"), nil
	}
}

func (e *Executor) RecordInventory(ctx context.Context, inventoryMap map[string]string) error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err := os.Mkdir(filepath.Join(dirname, components.OpenFunctionDir), os.ModePerm); err != nil {
		if !strings.Contains(err.Error(), "file exists") {
			return err
		}
	}

	cName, err := e.getClusterName(ctx)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf(components.RecordFileNameTmpl, cName)

	filePath := filepath.Join(dirname, components.OpenFunctionDir, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer file.Close()

	oldYaml, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return err
	}

	record := &inventory.Record{}
	err = yaml.Unmarshal(oldYaml, record)
	if err != nil {
		return err
	}

	newRecord, err := inventory.NewRecord(inventoryMap)
	if err != nil {
		return err
	}

	record.Update(newRecord)

	recordData, err := yaml.Marshal(record)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePath, recordData, 0644)
	if err != nil {
		panic(err)
	}
	return nil
}

func (e *Executor) GetInventoryRecord(ctx context.Context) (*inventory.Record, error) {
	var file *os.File

	dirname, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	if err := os.Mkdir(filepath.Join(dirname, components.OpenFunctionDir), os.ModePerm); err != nil {
		if !strings.Contains(err.Error(), "file exists") {
			return nil, err
		}
	}

	cName, err := e.getClusterName(ctx)
	if err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf(components.RecordFileNameTmpl, cName)

	filePath := filepath.Join(dirname, components.OpenFunctionDir, fileName)

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		file, err = os.Create(filePath)
		if err != nil {
			return nil, err
		}
	}

	file, err = os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	recordYaml, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return nil, err
	}

	record := &inventory.Record{}
	err = yaml.Unmarshal(recordYaml, record)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (e *Executor) DownloadKind(ctx context.Context) error {
	downloadCmd := "curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.11.1/kind-linux-amd64"
	if _, _, err := e.Exec(downloadCmd); err != nil {
		return err
	}
	chmodCmd := "chmod +x ./kind"
	if _, _, err := e.Exec(chmodCmd); err != nil {
		return err
	}
	mvCmd := "mv ./kind /usr/local/bin/kind"
	if _, _, err := e.Exec(mvCmd); err != nil {
		return err
	}
	return nil
}

func (e *Executor) GetNodeIP(ctx context.Context) (string, error) {
	getNodeIP := "docker exec openfunction-control-plane sh -c \"ip addr | grep eth0$ | grep -Eo 'inet ([0-9]*\\.){3}[0-9]*' | grep -Eo '([0-9]*\\.){3}[0-9]*' | tr -d '\\n'\""
	nodeIP, _, err := e.Exec(getNodeIP)
	if err != nil {
		return "", err
	}
	return nodeIP, nil
}

func (e *Executor) CurlOpenFunction(ctx context.Context, endpoint string) (string, error) {
	curlCMD := fmt.Sprintf("curl %s", endpoint)
	res, _, err := e.Exec(curlCMD)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(res, "\n"), nil
}
