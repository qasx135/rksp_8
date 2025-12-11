package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	db *sql.DB
}

type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func main() {
	dbPath := getEnv("DB_PATH", "user.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatalf("init db: %v", err)
	}

	srv := &Server{db: db}

	r := mux.NewRouter()
	r.HandleFunc("/me", srv.handleMe).Methods(http.MethodGet)
	// внутренний эндпоинт для anime-service
	r.HandleFunc("/internal/users/{id}", srv.handleGetUser).Methods(http.MethodGet)

	addr := ":8080"
	log.Printf("user-service listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("user-service failed: %v", err)
	}
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL
		);
	`)
	return err
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	email := r.Header.Get("X-User-Email")
	if userIDStr == "" || email == "" {
		http.Error(w, "missing user headers", http.StatusUnauthorized)
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	u, err := s.findOrCreateUser(userID, email)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var u User
	err = s.db.QueryRow("SELECT id, email, name FROM users WHERE id = ?", userID).
		Scan(&u.ID, &u.Email, &u.Name)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func (s *Server) findOrCreateUser(id int64, email string) (*User, error) {
	var u User
	err := s.db.QueryRow("SELECT id, email, name FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Email, &u.Name)
	if err == nil {
		return &u, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	name := email // простое имя по умолчанию
	_, err = s.db.Exec("INSERT INTO users(id, email, name) VALUES(?, ?, ?)", id, email, name)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:    id,
		Email: email,
		Name:  name,
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
