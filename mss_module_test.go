package mssmodule

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestMSSHandler_ServeHTTP(t *testing.T) {
	// 1. Setup the handler
	handler := MSSMiddleware{}

	// 2. Create a mock next handler in the chain
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})

	// 3. Create a request with our custom MSS value in the context
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), mssKey, 1460) // Inject mock MSS
	req = req.WithContext(ctx)

	// 4. Record the response
	rr := httptest.NewRecorder()

	// 5. Execute
	err := handler.ServeHTTP(rr, req, next)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	// 6. Assertions
	gotMss := req.Header.Get("X-Client-MSS")
	if gotMss != "1460" {
		t.Errorf("expected header X-Client-MSS to be 1460, got %s", gotMss)
	}
}
