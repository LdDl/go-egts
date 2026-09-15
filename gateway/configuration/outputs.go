package configuration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	LOG_FILENAME     = "application.log"
	PACKETS_FILENAME = "packets.ndjson"
	DUMP_FILENAME    = "queue.dump"
)

type OutputConflictError struct {
	First        string
	Second       string
	StderrUnsafe bool
}

func (err *OutputConflictError) Error() string {
	return fmt.Sprintf("Output destinations overlap: %s and %s", err.First, err.Second)
}

type outputDirectory struct {
	name string
	path string
	info os.FileInfo
}

type outputFile struct {
	name  string
	owner string
	info  os.FileInfo
}

func (cfg *Configuration) ValidateOutputs(stdout, stderr *os.File) error {
	directories := []outputDirectory{
		{name: "delivery_cfg.dump_directory", path: cfg.DeliveryCfg.DumpDirectory},
	}
	if cfg.LogsCfg.Output == "file" {
		directories = append(directories, outputDirectory{name: "logs_cfg.directory", path: cfg.LogsCfg.Directory})
	}
	if cfg.DestinationsCfg.File.Enabled {
		directories = append(directories, outputDirectory{name: "destinations_cfg.file.directory", path: cfg.DestinationsCfg.File.Directory})
	}
	var files []outputFile
	for i := range directories {
		directory := &directories[i]
		resolved, err := resolveDirectory(directory.path)
		if err != nil {
			return fmt.Errorf("%s: %w", directory.name, err)
		}
		directory.path = resolved
		info, err := os.Stat(resolved)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("%s: %w", directory.name, err)
		}
		directory.info = info
		entries, err := os.ReadDir(resolved)
		if err != nil {
			return fmt.Errorf("%s: %w", directory.name, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := filepath.Join(resolved, entry.Name())
			info, err := os.Stat(name)
			if err != nil {
				return fmt.Errorf("Can't inspect output file %s: %w", name, err)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("Output directory contains a non-regular file: %s", name)
			}
			files = append(files, outputFile{name: name, owner: directory.name, info: info})
		}
	}

	// stderr is also used for errors before the application logger is prepared.
	if stderr != nil {
		info, err := stderr.Stat()
		if err != nil {
			return fmt.Errorf("Can't inspect stderr: %w", err)
		}
		if info.Mode().IsRegular() {
			files = append(files, outputFile{name: "stderr", owner: "logs_cfg.directory", info: info})
		}
	}
	if stdout != nil && (cfg.LogsCfg.Output == "stdout" || cfg.DestinationsCfg.Stdout) {
		info, err := stdout.Stat()
		if err != nil {
			return fmt.Errorf("Can't inspect stdout: %w", err)
		}
		if info.Mode().IsRegular() {
			owner := "logs_cfg.directory"
			if cfg.DestinationsCfg.Stdout {
				owner = "destinations_cfg.stdout"
			}
			files = append(files, outputFile{name: "stdout", owner: owner, info: info})
		}
	}

	var conflict *OutputConflictError
	for i := range files {
		for j := 0; j < i; j++ {
			if files[i].owner == files[j].owner {
				continue
			}
			if os.SameFile(files[i].info, files[j].info) {
				conflict = &OutputConflictError{
					First:        files[i].name,
					Second:       files[j].name,
					StderrUnsafe: files[i].name == "stderr" || files[j].name == "stderr",
				}
				if conflict.StderrUnsafe {
					return conflict
				}
			}
		}
	}
	if conflict != nil {
		return conflict
	}
	if cfg.LogsCfg.Output == "stdout" && cfg.DestinationsCfg.Stdout {
		return &OutputConflictError{First: "logs_cfg.output", Second: "destinations_cfg.stdout"}
	}

	for i := range directories {
		for j := 0; j < i; j++ {
			first := directories[i]
			second := directories[j]
			if first.info != nil && second.info != nil && os.SameFile(first.info, second.info) {
				return &OutputConflictError{First: first.name, Second: second.name}
			}
			for _, pair := range [][2]string{{first.path, second.path}, {second.path, first.path}} {
				relative, err := filepath.Rel(pair[0], pair[1])
				if err != nil {
					return fmt.Errorf("Can't compare output directories: %w", err)
				}
				if relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) && !filepath.IsAbs(relative) {
					return &OutputConflictError{First: first.name, Second: second.name}
				}
			}
		}
	}
	return nil
}

func resolveDirectory(directory string) (string, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	var missing []string
	for {
		_, err = os.Lstat(absolute)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(absolute)
			if err != nil {
				return "", err
			}
			info, err := os.Stat(resolved)
			if err != nil {
				return "", err
			}
			if !info.IsDir() {
				return "", fmt.Errorf("Not a directory: %s", absolute)
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return "", err
		}
		missing = append(missing, filepath.Base(absolute))
		absolute = parent
	}
}
