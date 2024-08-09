package testnet

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// mustReachHeightOrLog waits for the specified block height to be reached on the
// specified chain, or prints container logs and exits the program with an error.
func mustReachHeightOrLog(
	n int64,
	timeout time.Duration,
	chainDockerServiceName string,
) error {
	err := waitForBlock(n, timeout, chainDockerServiceName)
	// **No** error, successfully reached target block
	if err == nil {
		return nil
	}

	fmt.Printf("error waiting for %s block: %s\n", chainDockerServiceName, err)

	containerID, err := getContainerID(chainDockerServiceName)
	if err != nil {
		return fmt.Errorf("failed getting container ID for logs: %w", err)
	}

	logs, err := getContainerLogs(chainDockerServiceName)
	if err != nil {
		return fmt.Errorf("failed getting container logs: %w", err)
	}

	// Dumps the container logs here, it shouldn't be too much since
	// it will fail either immediately or after 20 seconds of
	// retries.
	fmt.Printf("%s (container ID %s) logs shown below:\n", chainDockerServiceName, containerID)
	fmt.Printf("========================================\n")
	fmt.Println(logs)

	// Custom error handling, no return of error to cobra
	os.Exit(1)

	return nil
}

// checkContainerStatus returns an error if the specified container is not
// running.
func checkContainerStatus(
	chainDockerServiceName string,
) error {
	// check state of container
	out, err := exec.Command(
		"docker",
		"compose",
		"-f",
		generatedPath("docker-compose.yaml"),
		"ps",
		"-a", // all including exited
		"--format",
		"{{.State}}",
		chainDockerServiceName,
	).Output()
	if err != nil {
		stderr := ""
		if errors.Is(err, &exec.ExitError{}) {
			stderr = string(err.(*exec.ExitError).Stderr)
		}

		return fmt.Errorf("error checking container state, %s: %w", stderr, err)
	}

	containerState := strings.TrimSpace(string(out))

	if containerState != "running" {
		return fmt.Errorf(
			"%s container is not running, current state is \"%s\"",
			chainDockerServiceName,
			containerState,
		)
	}

	return nil
}

func getContainerID(
	chainDockerServiceName string,
) (string, error) {
	out, err := exec.Command(
		"docker",
		"compose",
		"-f",
		generatedPath("docker-compose.yaml"),
		"ps",
		"-a", // all including exited
		"--format",
		"{{.ID}}",
		chainDockerServiceName,
	).Output()
	if err != nil {
		stderr := ""

		exitErr, ok := err.(*exec.ExitError)
		if ok {
			stderr = string(exitErr.Stderr)
		}

		return "", fmt.Errorf("failed compose ps \"%s\": %w", stderr, err)
	}

	containerID := strings.TrimSpace(string(out))
	return containerID, nil
}

func getContainerLogs(
	chainDockerServiceName string,
) (string, error) {
	containerID, err := getContainerID(chainDockerServiceName)
	if err != nil {
		return "", fmt.Errorf("failed getting container ID: %w", err)
	}

	out, err := exec.Command(
		"docker",
		"logs",
		containerID,
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}

	return string(out), nil
}

// getContainerLogsChannel returns a channel that streams logs from a container
func getContainerLogsChannel(
	ctx context.Context,
	chainDockerServiceName string,
) (<-chan string, error) {
	containerID, err := getContainerID(chainDockerServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed getting container ID: %w", err)
	}

	// Run with CommandContext so it automatically cancels when the context is
	// done.
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"logs",
		"-f",
		containerID,
	)
	// pipe all stdout to a ReadCloser we can scan
	cmdReader, err := cmd.StdoutPipe()
	// redirect all stderr output to stdout
	cmd.Stderr = cmd.Stdout
	if err != nil {
		return nil, fmt.Errorf("failed to get command stdout pipe: %w", err)
	}

	scanner := bufio.NewScanner(cmdReader)
	out := make(chan string)

	go func(scanner *bufio.Scanner, out chan string) {
		for scanner.Scan() {
			// Check if the context is done, if so close the channel and return.
			// Don't need to manually stop the cmd since we use CommandContext()
			select {
			case <-ctx.Done():
				close(out)
				return
			default:
				out <- scanner.Text()
			}
		}
	}(scanner, out)

	// Start process, but don't wait for it to finish since it follows logs.
	// Don't use .Run() as it will block until it completes, causing a hang.
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to run docker logs -f %s: %w", containerID, err)
	}

	return out, nil
}
