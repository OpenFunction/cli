package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

// fatalErrHandler prints the message (if provided) and then exits.
var fatalErrHandler = func(msg string, code int) {
	if klog.V(6).Enabled() {
		klog.FatalDepth(2, msg)
	}
	if len(msg) > 0 {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}

// CheckErr prints a user friendly error to STDERR
func CheckErr(err error) {
	if err != nil {
		fatalErrHandler(err.Error(), 1)
	}
}

func UsageErrorf(cmd *cobra.Command, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s\nSee '%s -h' for help and examples", msg, cmd.CommandPath())
}

func IgnoreNotFoundErr(err error) error {
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound") {
			return nil
		}
	}
	return err
}

func IsInSlice(s string, target []string) bool {
	for _, t := range target {
		if t == s {
			return true
		}
	}
	return false
}
