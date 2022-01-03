package spinners

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/leaanthony/synx"
)

// Status code constants.
const (
	runningStatus int = iota
	successStatus
	errorStatus
	completed = "Completed!"
	failed    = "Failed!"
	stopped   = "Stopped!"
)

var (
	green = color.New(color.Bold, color.FgGreen).Sprintf
	red   = color.New(color.Bold, color.FgRed).Sprintf
	blue  = color.New(color.Bold, color.FgBlue).Sprintf
)

// Spinner defines a single s
type Spinner struct {
	message *synx.String
	status  *synx.Int
	group   *SpinnerGroup
	name    *string
}

func (s *Spinner) WithName(name string) *Spinner {
	s.name = &name
	return s
}

func (s *Spinner) handleMessage(message string) string {
	var msg string
	if s.name != nil {
		msg = fmt.Sprintf("%s - %s", *s.name, message)
	} else {
		msg = message
	}
	return msg
}

// Update updates the spinner message
func (s *Spinner) Update(message string) {
	s.message.SetValue(s.handleMessage(message))
}

// Done marks spinner as success
func (s *Spinner) Done() {
	s.Update(completed)
	s.stop(successStatus)
}

// Error marks spinner as error
func (s *Spinner) Error(err error) {
	s.ErrorWithMessage(failed, err)
}

// ErrorWithMessage marks spinner as error and update message
func (s *Spinner) ErrorWithMessage(message string, err error) {
	s.Update(message)
	s.stop(errorStatus)
	if err != nil {
		s.group.err = err
		s.group.errC <- err
	}
}

func (s *Spinner) stop(status int) {
	s.status.SetValue(status)
	s.group.redraw()
	s.group.Done()
}

func (s *Spinner) refresh() string {
	switch s.status.GetValue() {
	case successStatus:
		return fmt.Sprintf("%s %s", green(s.getSymbol()), s.message.GetValue())
	case errorStatus:
		return fmt.Sprintf("%s %s", red(s.getSymbol()), s.message.GetValue())
	default:
		return fmt.Sprintf("%s %s", blue(s.getSymbol()), s.message.GetValue())
	}
}

func (s *Spinner) getSymbol() string {
	switch s.status.GetValue() {
	case successStatus:
		return s.group.successSymbol
	case errorStatus:
		return s.group.errorSymbol
	default:
		return s.group.currentFrame()
	}
}

func (s *Spinner) IsActive() bool {
	if s.status.GetValue() == runningStatus {
		return true
	}
	return false
}
