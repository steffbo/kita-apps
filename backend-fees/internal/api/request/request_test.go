package request

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestDecodeJSON_RejectsOversizedBody(t *testing.T) {
	body := `{"note":"` + string(bytes.Repeat([]byte("a"), MaxJSONBodyBytes)) + `"}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	var v map[string]string
	if err := DecodeJSON(req, &v); err == nil {
		t.Fatal("expected error for body larger than MaxJSONBodyBytes")
	}

	req = httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"note":"ok"}`))
	if err := DecodeJSON(req, &v); err != nil || v["note"] != "ok" {
		t.Fatalf("small body: err=%v v=%v", err, v)
	}
}
