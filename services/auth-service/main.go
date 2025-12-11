package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"authservice/internal/auth"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	db        *sql.DB
	jwtSecret []byte
}

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func main() {
	dbPath := getEnv("DB_PATH", "auth.db")
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatalf("init db: %v", err)
	}

	srv := &Server{
		db:        db,
		jwtSecret: []byte(secret),
	}

	r := mux.NewRouter()
	r.HandleFunc("/register", srv.handleRegister).Methods(http.MethodPost)
	r.HandleFunc("/login", srv.handleLogin).Methods(http.MethodPost)

	addr := ":8080"
	log.Printf("auth-service listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("auth-service failed: %v", err)
	}
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		);
	`)
	return err
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var c Credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if c.Email == "" || c.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "cannot hash password", http.StatusInternalServerError)
		return
	}

	res, err := s.db.Exec("INSERT INTO users(email, password_hash) VALUES(?, ?)", c.Email, string(hash))
	if err != nil {
		http.Error(w, "user already exists?", http.StatusBadRequest)
		return
	}
	id, _ := res.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(User{ID: id, Email: c.Email})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var c Credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if c.Email == "" || c.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	var u User
	if err := s.db.QueryRow("SELECT id, email, password_hash FROM users WHERE email = ?", c.Email).
		Scan(&u.ID, &u.Email, &u.PasswordHash); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(c.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(u.ID, u.Email, s.jwtSecret, 24*time.Hour)
	if err != nil {
		http.Error(w, "cannot generate token", http.StatusInternalServerError)
		return
	}

	resp := TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
