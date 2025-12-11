package userclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUserByID(t *testing.T) {
	// простой мок-сервер
	handler := http.NewServeMux()
	handler.HandleFunc("/internal/users/1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":1,"email":"test@example.com","name":"test"}`))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	u, err := client.GetUserByID("1", http.Header{})
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}
	if u.Email != "test@example.com" {
		t.Fatalf("unexpected email: %s", u.Email)
	}
}
