package logger_test

import (
	"sync"
	"testing"
	"time"

	"gogit/internal/logger"
)

type concurrentWriteProbe struct {
	mu       sync.Mutex
	active   int
	maxWrite int
}

func (w *concurrentWriteProbe) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.active++
	if w.active > w.maxWrite {
		w.maxWrite = w.active
	}
	w.mu.Unlock()

	time.Sleep(5 * time.Millisecond)

	w.mu.Lock()
	w.active--
	w.mu.Unlock()
	return len(p), nil
}

func (w *concurrentWriteProbe) maximumConcurrentWrites() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.maxWrite
}

func TestLoggerSerializesConcurrentOutput(t *testing.T) {
	output := &concurrentWriteProbe{}
	log := logger.New(logger.Info, output)

	start := make(chan struct{})
	var callers sync.WaitGroup
	for i := 0; i < 24; i++ {
		callers.Add(1)
		go func(id int) {
			defer callers.Done()
			<-start
			log.Info("request complete", "request_id", id)
		}(i)
	}
	close(start)
	callers.Wait()

	if got := output.maximumConcurrentWrites(); got != 1 {
		t.Fatalf("logger wrote to its output concurrently: maximum overlapping writes = %d, want 1", got)
	}
}
