package main

import (
	"net/http/httptest"
	"testing"
)

func TestAuthTokenFromRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	if got := authTokenFromHTTPRequest(req); got != "header-token" {
		t.Fatalf("header token = %q", got)
	}

	req = httptest.NewRequest("GET", "/?token=query-token", nil)
	if got := authTokenFromHTTPRequest(req); got != "query-token" {
		t.Fatalf("query token = %q", got)
	}
}
