package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteList(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeList(recorder, []string{"a", "b"}, listMeta{
		Page:     2,
		PageSize: 10,
		Total:    12,
		Sort:     "title",
		Order:    "asc",
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload listResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unexpected json: %v", err)
	}
	if payload.Meta.Page != 2 || payload.Meta.Sort != "title" || payload.Meta.Order != "asc" {
		t.Fatalf("unexpected list payload: %+v", payload)
	}
}
