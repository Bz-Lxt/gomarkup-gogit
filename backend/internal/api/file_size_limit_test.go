package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestAddAcceptsFileAtSizeLimit(t *testing.T) {
	log := logger.New(logger.Error, io.Discard)
	repo, err := git.Init(t.TempDir(), git.SHA1, log)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(repo, "", log).Handler()

	putBody, err := json.Marshal(map[string]string{
		"path":    "limit.txt",
		"content": strings.Repeat("x", 2*1024*1024),
	})
	if err != nil {
		t.Fatal(err)
	}
	put := httptest.NewRequest(http.MethodPut, "/api/v1/files", bytes.NewReader(putBody))
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, put)
	if putRec.Code != http.StatusOK {
		t.Fatalf("write at documented size limit returned %d: %s", putRec.Code, putRec.Body.String())
	}

	add := httptest.NewRequest(http.MethodPost, "/api/v1/index/add", strings.NewReader(`{"paths":["limit.txt"]}`))
	addRec := httptest.NewRecorder()
	handler.ServeHTTP(addRec, add)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add at documented size limit returned %d: %s", addRec.Code, addRec.Body.String())
	}
}
