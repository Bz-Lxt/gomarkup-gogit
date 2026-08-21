package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestServerProcess(t *testing.T) {
	if os.Getenv("GOGIT_SERVER_HELPER") != "1" {
		return
	}
	main()
}

func TestServerShutdownHonorsDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestServerProcess$")
	cmd.Env = append(os.Environ(),
		"GOGIT_SERVER_HELPER=1",
		fmt.Sprintf("PORT=%d", port),
		"GOGIT_REPO="+filepath.Join(t.TempDir(), "repo"),
		"GOGIT_WEB="+filepath.Join(t.TempDir(), "missing-web"),
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	var conn net.Conn
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server did not start: %v", err)
	}
	defer conn.Close()

	request := "PUT /api/v1/files HTTP/1.1\r\n" +
		"Host: " + addr + "\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 1048576\r\n\r\n" +
		`{"path":"pending.txt","content":"`
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	_, _ = bufio.NewReader(conn).Peek(1)

	started := time.Now()
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server exited unsuccessfully: %v", err)
		}
		if elapsed := time.Since(started); elapsed > 10*time.Second {
			t.Fatalf("server exceeded shutdown deadline: %v", elapsed)
		}
	case <-time.After(11 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatal("server did not exit after its shutdown deadline")
	}
}
