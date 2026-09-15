package destination

import (
	"fmt"
	"os"
	"syscall"
)

func openStdout() (*os.File, error) {
	info, err := os.Stdout.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		return os.Stdout, nil
	}
	// Reopen independently so deadlines on a pipe do not change the process stdout descriptor.
	return os.OpenFile(fmt.Sprintf("/proc/self/fd/%d", os.Stdout.Fd()), os.O_WRONLY|os.O_APPEND|syscall.O_NONBLOCK, 0)
}
