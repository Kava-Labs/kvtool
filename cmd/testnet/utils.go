package testnet

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// generatedPath is a utility that calls filepath.Join with generatedConfigDir as the base directory
func generatedPath(elem ...string) string {
	pieces := make([]string, 1, len(elem)+1)
	pieces[0] = generatedConfigDir
	pieces = append(pieces, elem...)
	return filepath.Join(pieces...)
}

func replaceCurrentProcess(command ...string) error {
	if len(command) < 1 {
		panic("must provide name of executable to run")
	}
	executable, err := exec.LookPath(command[0])
	if err != nil {
		return err
	}

	// pass the current environment variables to the command
	env := os.Environ()

	err = syscall.Exec(executable, command, env)
	if err != nil {
		return err
	}
	return nil
}

type stringSlice []string

func (strings stringSlice) contains(match string) bool {
	for _, s := range strings {
		if match == s {
			return true
		}
	}
	return false
}
