package util

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
)

var (
	Yellow       = color.New(color.FgHiYellow, color.Bold).SprintFunc()
	YellowItalic = color.New(color.FgHiYellow, color.Bold, color.Italic).SprintFunc()
	Green        = color.New(color.FgHiGreen, color.Bold).SprintFunc()
	Blue         = color.New(color.FgHiBlue, color.Bold).SprintFunc()
	Cyan         = color.New(color.FgCyan, color.Bold, color.Underline).SprintFunc()
	Red          = color.New(color.FgHiRed, color.Bold).SprintFunc()
	White        = color.New(color.FgWhite).SprintFunc()
	WhiteBold    = color.New(color.FgWhite, color.Bold).SprintFunc()
	forceDetail  = "yaml"
)

type MessageLevel string

type Printer struct {
	PrintFlags  *genericclioptions.PrintFlags
	PrinterFunc PrinterFunc

	ForceDetail bool
}

func NewPrinter(operation string, scheme *runtime.Scheme) *Printer {
	return &Printer{
		PrintFlags: genericclioptions.NewPrintFlags(operation).
			WithTypeSetter(scheme),
	}
}

func (p *Printer) SetPrinterFunc(fc PrinterFunc) {
	p.PrinterFunc = fc
}

func (p *Printer) SetForceDefail() {
	if nil == p.PrintFlags.OutputFormat ||
		*p.PrintFlags.OutputFormat == "" {
		p.PrintFlags.OutputFormat = &forceDetail
	}
}

func (p *Printer) AddFlags(cmd *cobra.Command) {
	p.PrintFlags.AddFlags(cmd)
}

func (p *Printer) ShouldPrintObject() bool {
	shouldPrint := false
	output := *p.PrintFlags.OutputFormat
	if len(output) > 0 {
		shouldPrint = true
	}
	return shouldPrint
}

func (p *Printer) ToPrinterWitchColumn(columnLabels []string) (printers.ResourcePrinter, error) {
	if IsToTable(p) {
		p.SetPrinterFunc(WithTablePrinter(printers.PrintOptions{
			ColumnLabels: columnLabels,
		}))
	} else {
		p.SetPrinterFunc(WithDefaultPrinter(""))
	}

	return p.PrinterFunc(p)
}

func (p *Printer) ToPrinter() (printers.ResourcePrinter, error) {
	return p.PrinterFunc(p)
}

type PrinterFunc func(*Printer) (printers.ResourcePrinter, error)

func WithDefaultPrinter(operation string) PrinterFunc {
	return func(p *Printer) (printers.ResourcePrinter, error) {
		p.PrintFlags.NamePrintFlags.Operation = operation
		return p.PrintFlags.ToPrinter()
	}
}

func WithTablePrinter(options printers.PrintOptions) PrinterFunc {
	return func(p *Printer) (printers.ResourcePrinter, error) {
		return printers.NewTablePrinter(options), nil
	}
}

// IsToTable if printer output format want prilnt object detail,return false
func IsToTable(printer *Printer) bool {
	if printer.PrintFlags.OutputFormat != nil &&
		(*printer.PrintFlags.OutputFormat == "") &&
		!printer.ForceDetail {
		return true
	}

	return false
}

// ToTable runtime.Object convert metav1.Table to facilitate printing
func ToTable(tableRow TableRow, objs ...runtime.Object) (*metav1.Table, error) {
	tbRows, err := tableRows(tableRow, objs...)
	if err != nil {
		return nil, err
	}
	tb := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{},
		Rows:              tbRows,
	}
	return tb, nil
}

func tableRows(tableRow TableRow, objs ...runtime.Object) ([]metav1.TableRow, error) {

	rows := make([]metav1.TableRow, 0, len(objs))
	for _, obj := range objs {
		row, err := tableRow(obj)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

type TableRow func(obj interface{}) (metav1.TableRow, error)

func TranslateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

// TaskInformer is a printer of task information.
type TaskInformer struct {
	title string
}

func NewTaskInformer(title string) *TaskInformer {
	return &TaskInformer{
		title: title,
	}
}

func (ti *TaskInformer) BeforeTask(msg string) string {
	return fmt.Sprintf("%s", YellowItalic(msg))
}

func (ti *TaskInformer) SkipTask(msg string) string {
	str := fmt.Sprintf(" -> Skip the %s installation", msg)
	return fmt.Sprintf("🚗 %s", Green(str))
}

func (ti *TaskInformer) TaskInfo(msg string) string {
	str := fmt.Sprintf(" -> %s <- %s", ti.title, msg)
	return fmt.Sprintf("🔄 %s", White(str))
}

func (ti *TaskInformer) TaskFail(msg string) string {
	return fmt.Sprintf("❌ %s", Red(msg))
}

func (ti *TaskInformer) TaskFailWithTitle(msg string) string {
	return fmt.Sprintf(" -> %s <- %s", ti.title, msg)
}

func (ti *TaskInformer) TaskSuccess() string {
	str := fmt.Sprintf(" -> %s <- Done!", ti.title)
	return fmt.Sprintf("✅ %s", White(str))
}

func (ti *TaskInformer) AllDone(t time.Duration) string {
	return fmt.Sprintf("🚀 %s", WhiteBold(fmt.Sprintf("Completed in %s.", t)))
}

func (ti *TaskInformer) PrintTable(inventory map[string]string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Component", "Version"})
	for comp, version := range inventory {
		t.AppendRows([]table.Row{
			{comp, version},
		})
	}
	t.AppendSeparator()
	t.Render()
}

func (ti *TaskInformer) TipsOnUsingKnative() {
	fmt.Println(YellowItalic("Notice that you are using Knative runtime, " +
		"you can refer to the following to configure Knative's network layer (Assuming you are using Kourier) and DNS. \n" +
		"Where 1.2.3.4 can be replaced by your node address or loadbalancer address:"))
	fmt.Println(YellowItalic("\n -> Configure the externalIPs for the Kourier service"))
	fmt.Println("kubectl patch svc -n kourier-system kourier \\\n" +
		"  -p '{\"spec\": {\"type\": \"LoadBalancer\", \"externalIPs\": [\"1.2.3.4\"]}}'")
	fmt.Println(YellowItalic("\n -> Configure the domain by using MagicDNS"))
	fmt.Println("kubectl patch configmap/config-domain -n knative-serving \\\n" +
		"  --type merge --patch '{\"data\":{\"1.2.3.4.sslip.io\":\"\"}}'")
	fmt.Println()
}
