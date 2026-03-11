package spinner

import (
	"fmt"
	"sync"
	"time"

	"github.com/Kazuto/Weave/pkg/ui"
)

type Spinner struct {
	message string
	frames  []string
	stop    chan struct{}
	done    sync.WaitGroup
}

func New(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:    make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	s.done.Add(1)
	go func() {
		defer s.done.Done()
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				// Color both the spinner frame and message in cyan (no symbol while spinning)
				coloredFrame := "\033[96m" + s.frames[i] + "\033[0m"
				fmt.Printf("\r%s %s", coloredFrame, ui.FormatCyan(s.message))
				i = (i + 1) % len(s.frames)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop(success bool) {
	close(s.stop)
	s.done.Wait()

	if success {
		fmt.Printf("\r%s\n", ui.FormatSuccess(s.message))
	} else {
		fmt.Printf("\r%s\n", ui.FormatError(s.message))
	}
}
