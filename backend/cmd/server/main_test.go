package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const serverHelperEnv = "GOGIT_SERVER_HELPER"

type serverResponse struct {
	status int
	err    error
}

func TestServerFinishesInFlightRequestDuringShutdown(t *testing.T) {
	repoDir := t.TempDir()
	port := freePort(t)

	var logs bytes.Buffer
	cmd := exec.Command(os.Args[0], "-test.run", "^TestServerHelperProcess$")
	cmd.Env = append(os.Environ(),
		serverHelperEnv+"=1",
		"GOGIT_REPO="+repoDir,
		"GOGIT_WEB="+filepath.Join(repoDir, "missing-web"),
		"PORT="+strconv.Itoa(port),
	)
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server process: %v", err)
	}
	processDone := make(chan struct{})
	var processErr error
	go func() {
		processErr = cmd.Wait()
		close(processDone)
	}()
	t.Cleanup(func() {
		select {
		case <-processDone:
		default:
			_ = cmd.Process.Kill()
			<-processDone
		}
	})

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForServer(t, baseURL, processDone, &processErr, &logs)

	pipePath := filepath.Join(repoDir, "slow-input")
	if err := syscall.Mkfifo(pipePath, 0o600); err != nil {
		t.Fatalf("create blocking worktree file: %v", err)
	}

	responseDone := make(chan serverResponse, 1)
	go func() {
		resp, err := http.Post(baseURL+"/api/v1/index/add", "application/json", strings.NewReader(`{"paths":["slow-input"]}`))
		if err != nil {
			responseDone <- serverResponse{err: err}
			return
		}
		_, readErr := io.Copy(io.Discard, resp.Body)
		closeErr := resp.Body.Close()
		responseDone <- serverResponse{status: resp.StatusCode, err: errors.Join(readErr, closeErr)}
	}()

	writer := openPipeWriter(t, pipePath, responseDone, processDone, &processErr, &logs)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = writer.Close()
		t.Fatalf("signal server process: %v", err)
	}

	select {
	case <-processDone:
		_ = writer.Close()
		t.Fatalf("server exited before its in-flight request completed: %v\n%s", processErr, logs.String())
	case <-time.After(300 * time.Millisecond):
	}

	if _, err := writer.Write([]byte("content available during shutdown\n")); err != nil {
		t.Fatalf("release in-flight request: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close blocking worktree file: %v", err)
	}

	select {
	case got := <-responseDone:
		if got.err != nil {
			t.Fatalf("in-flight request was interrupted during shutdown: %v", got.err)
		}
		if got.status != http.StatusOK {
			t.Fatalf("in-flight request status = %d, want %d", got.status, http.StatusOK)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight request did not finish during shutdown grace period")
	}

	select {
	case <-processDone:
		if processErr != nil {
			t.Fatalf("server process exited with error: %v\n%s", processErr, logs.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not exit after the in-flight request completed")
	}
}

func TestServerHelperProcess(t *testing.T) {
	if os.Getenv(serverHelperEnv) != "1" {
		return
	}
	main()
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve server port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("release server port: %v", err)
	}
	return port
}

func waitForServer(t *testing.T, baseURL string, processDone <-chan struct{}, processErr *error, logs *bytes.Buffer) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-processDone:
			t.Fatalf("server exited during startup: %v\n%s", *processErr, logs.String())
		default:
		}
		resp, err := client.Get(baseURL + "/api/v1/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not become ready\n%s", logs.String())
}

func openPipeWriter(t *testing.T, path string, responseDone <-chan serverResponse, processDone <-chan struct{}, processErr *error, logs *bytes.Buffer) *os.File {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		writer, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			return writer
		}
		if !errors.Is(err, syscall.ENXIO) {
			t.Fatalf("open blocking worktree file: %v", err)
		}
		select {
		case got := <-responseDone:
			t.Fatalf("request finished before reading the blocking file: status=%d err=%v", got.status, got.err)
		case <-processDone:
			t.Fatalf("server exited while waiting for request to start: %v\n%s", *processErr, logs.String())
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("request did not start reading its worktree file")
	return nil
}
