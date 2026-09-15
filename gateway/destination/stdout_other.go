//go:build !linux

package destination

import "os"

func openStdout() (*os.File, error) {
	return os.Stdout, nil
}
