package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HNodeland/cloud-native-reference-app/internal/store"
	"github.com/HNodeland/cloud-native-reference-app/internal/telemetry"
	"go.uber.org/zap"
)

func TestTodoCRUDFlow(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, store.New(), zap.NewNop(), &telemetry.Instrumentation{})

	created := doJSONRequest(t, mux, http.MethodPost, "/todos", map[string]any{"title": "learn otel"}, http.StatusCreated)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("expected id")
	}

	got := doJSONRequest(t, mux, http.MethodGet, "/todos/"+id, nil, http.StatusOK)
	if got["title"] != "learn otel" {
		t.Fatalf("unexpected title: %v", got["title"])
	}

	updated := doJSONRequest(t, mux, http.MethodPut, "/todos/"+id, map[string]any{"completed": true, "title": "done"}, http.StatusOK)
	if updated["completed"] != true {
		t.Fatalf("expected completed true, got: %v", updated["completed"])
	}

	doJSONRequest(t, mux, http.MethodDelete, "/todos/"+id, nil, http.StatusNoContent)

	req := httptest.NewRequest(http.MethodGet, "/todos/"+id, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rr.Code)
	}
}

func TestValidation(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, store.New(), zap.NewNop(), &telemetry.Instrumentation{})

	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(`{"title":"   "}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", rr.Code)
	}
}

func TestRegisterRoutesWithNilDependencies(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, nil, nil)

	doJSONRequest(t, mux, http.MethodGet, "/todos", nil, http.StatusOK)

	created := doJSONRequest(t, mux, http.MethodPost, "/todos", map[string]any{"title": "from nil deps"}, http.StatusCreated)
	id, ok := created["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected id")
	}
}

func doJSONRequest(t *testing.T, handler http.Handler, method, path string, body any, expectedStatus int) map[string]any {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != expectedStatus {
		t.Fatalf("%s %s: expected %d, got %d (body=%s)", method, path, expectedStatus, rr.Code, rr.Body.String())
	}

	if rr.Body.Len() == 0 {
		return nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		return nil
	}
	return decoded
}
