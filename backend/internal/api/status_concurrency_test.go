package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"gogit/internal/api"
	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestConcurrentStatusRequestsRemainSuccessful(t *testing.T) {
	repo, err := git.Init(t.TempDir(), git.SHA1, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 256*1024)
	var state uint32 = 1
	for i := range payload {
		state = state*1664525 + 1013904223
		payload[i] = byte(state >> 24)
	}
	handler := api.New(repo, "", logger.New(logger.Error, io.Discard)).Handler()
	const requests = 16
	for round := byte(0); round < 5; round++ {
		payload[0] = round
		if err := repo.WriteWorktree("pending.bin", string(payload)); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		results := make(chan *httptest.ResponseRecorder, requests)
		var wg sync.WaitGroup
		for i := 0; i < requests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
				<-start
				handler.ServeHTTP(rec, req)
				results <- rec
			}()
		}
		close(start)
		wg.Wait()
		close(results)

		for rec := range results {
			if rec.Code != http.StatusOK {
				t.Fatalf("concurrent status returned %d: %s", rec.Code, rec.Body.String())
			}
		}
	}
}
