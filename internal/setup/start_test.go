package setup

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPortIsFree(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if PortIsFree(port) {
		t.Fatalf("port %d is in use but was reported free", port)
	}
	listener.Close()
	if !PortIsFree(port) {
		t.Fatalf("port %d was released but is still reported busy", port)
	}
}

func TestWaitForOdoo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err := WaitForOdoo(context.Background(), server.URL, 10*time.Second); err != nil {
		t.Fatalf("a running server was not detected: %v", err)
	}
	server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := WaitForOdoo(ctx, server.URL, 10*time.Second); err == nil {
		t.Fatal("a stopped server must not count as running")
	}
}
