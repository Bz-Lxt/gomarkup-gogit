package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestAddBatchFailureDoesNotChangeIndex(t *testing.T) {
	repo, err := git.Init(t.TempDir(), git.SHA1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteWorktree("kept.txt", "content\n"); err != nil {
		t.Fatal(err)
	}

	server := New(repo, "", logger.New(logger.Error, io.Discard)).Handler()
	body := bytes.NewBufferString(`{"paths":["kept.txt","removed.txt"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/index/add", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("batch add status = %d, want %d; body: %s", res.Code, http.StatusNotFound, res.Body.String())
	}

	indexReq := httptest.NewRequest(http.MethodGet, "/api/v1/index", nil)
	indexRes := httptest.NewRecorder()
	server.ServeHTTP(indexRes, indexReq)
	if indexRes.Code != http.StatusOK {
		t.Fatalf("index status = %d, want %d; body: %s", indexRes.Code, http.StatusOK, indexRes.Body.String())
	}
	var response struct {
		Data struct {
			Entries []git.IndexEntry `json:"entries"`
		} `json:"data"`
	}
	if err := json.NewDecoder(indexRes.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Entries) != 0 {
		t.Fatalf("index changed after failed batch add: %+v", response.Data.Entries)
	}
}
