package spinners

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ahmetalpbalkan/go-cursor"
	"github.com/leaanthony/synx"
)

var (
	spinnerFrames = []string{
		"⠈⠁",
		"⠈⠑",
		"⠈⠱",
		"⠈⡱",
		"⢀⡱",
		"⢄⡱",
		"⢄⡱",
		"⢆⡱",
		"⢎⡱",
		"⢎⡰",
		"⢎⡠",
		"⢎⡀",
		"⢎⠁",
		"⠎⠁",
		"⠊⠁",
	}
)

// SpinnerGroup is a group of Spinners
type SpinnerGroup struct {
	sync.Mutex
	sync.WaitGroup
	spinners        []*Spinner
	frames          []string
	currentFrameIdx int
	successSymbol   string
	errorSymbol     string
	running         bool
	drawn           bool
	errC            chan error
	err             error
}

// At returns the Spinner at given 0-based index
func (g *SpinnerGroup) At(idx int) *Spinner {
	return g.spinners[idx]
}

// Start the spinners
func (g *SpinnerGroup) Start(ctx context.Context) {
	g.Lock()
	defer g.Unlock()

	if g.running {
		return
	}
	g.running = true

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for g.running {
			select {
			case <-ticker.C:
				g.redraw()
				g.checkIfNeedToTerminate()
			case <-ctx.Done():
				g.stop(ctx.Err().Error())
			case err := <-g.errC:
				g.err = err
				g.stop(stopped)
			}
		}
	}()
}

func (g *SpinnerGroup) stop(errMsg string) {
	for _, s := range g.spinners {
		if s.IsActive() {
			s.ErrorWithMessage(errMsg, nil)
		}
	}
}

// Stop the spinners
func (g *SpinnerGroup) Stop() {
	g.Lock()
	defer g.Unlock()
	g.running = false
}

// Wait for all spinners to finish
func (g *SpinnerGroup) Wait() error {
	g.WaitGroup.Wait()
	g.Stop()
	return g.err
}

func (g *SpinnerGroup) redraw() {
	g.Lock()
	defer g.Unlock()
	if !g.running {
		return
	}
	if g.drawn {
		fmt.Print(cursor.MoveUp(len(g.spinners)))
	}
	for _, spinner := range g.spinners {
		fmt.Print(cursor.ClearEntireLine())
		fmt.Println(spinner.refresh())
	}
	g.currentFrameIdx = (g.currentFrameIdx + 1) % len(g.frames)
	g.drawn = true
}

func (g *SpinnerGroup) checkIfNeedToTerminate() {
	i := 0
	for _, s := range g.spinners {
		if !s.IsActive() {
			i += 1
			if !s.IsDead {
				s.stop(s.status.GetValue())
				g.Done()
				s.IsDead = true
			}
		}
	}
	if len(g.spinners) == i {
		g.Stop()
	}
}

func (g *SpinnerGroup) currentFrame() string {
	return g.frames[g.currentFrameIdx]
}

func (g *SpinnerGroup) AddSpinner() *SpinnerGroup {
	idx := len(g.spinners)
	g.spinners = append(g.spinners, &Spinner{
		message: synx.NewString(fmt.Sprintf("Spinner #%d", idx+1)),
		status:  synx.NewInt(runningStatus),
		group:   g,
		IsDead:  false,
	})
	g.Add(1)
	return g
}

// NewSpinnerGroupWithSize creates a SpinnerGroup with size
func NewSpinnerGroupWithSize(size int) *SpinnerGroup {
	group := &SpinnerGroup{
		spinners:        make([]*Spinner, size),
		frames:          spinnerFrames,
		currentFrameIdx: 0,
		successSymbol:   " ✓",
		errorSymbol:     " ✗",
		running:         false,
		drawn:           false,
	}
	for i := 0; i < size; i++ {
		group.spinners[i] = &Spinner{
			message: synx.NewString(fmt.Sprintf("Spinner #%d", i+1)),
			status:  synx.NewInt(runningStatus),
			group:   group,
			IsDead:  false,
		}
	}
	group.Add(size)
	return group
}

// NewSpinnerGroup creates a SpinnerGroup
func NewSpinnerGroup() *SpinnerGroup {
	group := &SpinnerGroup{
		spinners:        []*Spinner{},
		frames:          spinnerFrames,
		currentFrameIdx: 0,
		successSymbol:   " ✓",
		errorSymbol:     " ✗",
		running:         false,
		drawn:           false,
		errC:            make(chan error, 1),
	}
	return group
}
