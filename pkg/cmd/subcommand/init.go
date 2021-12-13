package subcommand

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/OpenFunction/cli/pkg/cmd/util"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Init is the commandline for 'init' sub command
type Init struct {
	genericclioptions.IOStreams

	Language    string
	Path        string
	ProjectName string
	Repo        string
	OutPutPath  string

	frameworkPath string
}

// NewInit returns an initialized Init instance
func NewInit(ioStreams genericclioptions.IOStreams) *Init {
	return &Init{
		IOStreams: ioStreams,
	}
}

func NewCmdInit(ioStreams genericclioptions.IOStreams) *cobra.Command {
	i := NewInit(ioStreams)

	cmd := &cobra.Command{
		Use:                   "init",
		DisableFlagsInUseLine: true,
		Short:                 "Init a project from the specified framework",
		Long: `
Init a project from the specified framework
`,
		Example: "ofn init -l go",
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(i.ValidateArgs(cmd, args))
			util.CheckErr(i.RunInit(cmd))
		},
	}

	cmd.Flags().StringVarP(&i.Path, "path", "p", i.Path, "framework path")
	cmd.Flags().StringVarP(&i.ProjectName, "project-name", "", i.ProjectName, "name of this project")
	cmd.Flags().StringVarP(&i.Repo, "repo", "", i.Repo, "name to use for go module")
	cmd.Flags().StringVarP(&i.OutPutPath, "output", "o", ".", "project output path")
	cmd.Flags().StringVarP(&i.Language, "lang", "l", "go", "language or framework to use")
	return cmd
}

const (
	frameworkPath = ".openfunction/framework"
)

func (i *Init) ValidateArgs(cmd *cobra.Command, args []string) error {
	exist, err := i.checkFramework()
	if err != nil {
		return err
	}

	if !exist {
		// TODO Download from github and store in the default directory
		return fmt.Errorf("framework of [%s] is not exists", i.Language)
	}

	i.OutPutPath = filepath.Join(i.OutPutPath, i.ProjectName)

	i.OutPutPath, err = abs(i.OutPutPath)
	if err != nil {
		return err
	}

	exist, err = checkPathExists(i.OutPutPath)
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("the path [%s] is not empty", i.OutPutPath)
	}

	return nil
}

func (i *Init) RunInit(cmd *cobra.Command) error {
	value := i.mergeVariables()
	return i.templateExecute("", value)
}

// checkFramework check if framework exists
// two sources of framework:
//     - path: custom path
//     - default: ~/.openfunction/framework
func (i *Init) checkFramework() (bool, error) {
	var err error

	i.frameworkPath = i.Path
	if i.frameworkPath == "" {
		i.frameworkPath, err = os.UserHomeDir()
		if err != nil {
			return false, err
		}

		i.frameworkPath = filepath.Join(i.Path, frameworkPath, i.Language)
	}

	i.frameworkPath, err = abs(i.frameworkPath)
	if err != nil {
		return false, err
	}

	return checkPathExists(i.Path)
}

func abs(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Abs(path)

	}

	return path, nil
}

func checkPathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func mkdir(path string) error {
	return os.MkdirAll(path, os.ModeDir)
}

func (i *Init) mergeVariables() map[string]interface{} {
	values := map[string]interface{}{}

	values["ProjectName"] = i.ProjectName
	values["Repo"] = i.Repo

	return values
}

func (i *Init) templateExecute(subPath string, value interface{}) error {
	var parseFS = func(fp, fo, name string) error {
		if !strings.HasSuffix(name, ".template") {
			return nil
		}
		fp = filepath.Join(fp, name)
		tpl, err := template.ParseFiles(fp)
		if err != nil {
			return err
		}

		wr, err := os.Create(filepath.Join(fo, strings.TrimSuffix(name, ".template")))
		if err != nil {
			return err
		}

		err = tpl.Execute(wr, value)
		if err != nil {
			return err
		}
		return nil
	}

	fp := filepath.Join(i.frameworkPath, subPath)
	fo := filepath.Join(i.OutPutPath, subPath)
	dirEntry, err := os.ReadDir(fp)
	if err != nil {
		return err
	}

	if err := mkdir(fo); err != nil {
		return err
	}
	for _, elem := range dirEntry {
		if elem.IsDir() {
			err = i.templateExecute(elem.Name(), value)
			if err != nil {
				return err
			}
			continue
		}

		err = parseFS(fp, fo, elem.Name())
		if err != nil {
			return err
		}
	}

	return nil
}
