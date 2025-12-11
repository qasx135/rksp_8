package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"orderservice/internal/userclient"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	db         *sql.DB
	userClient *userclient.Client
}

type WatchedAnime struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Anime      string `json:"anime"`
	FolderName string `json:"folder_name"`
}

type AddAnimeRequest struct {
	Anime      string `json:"anime"`
	FolderName string `json:"folder_name"`
}

func main() {
	dbPath := getEnv("DB_PATH", "watched.db")
	userServiceURL := getEnv("USER_SERVICE_URL", "http://user-service:8080")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatalf("init db: %v", err)
	}

	uc, err := userclient.New(userServiceURL)
	if err != nil {
		log.Fatalf("userclient init: %v", err)
	}

	srv := &Server{
		db:         db,
		userClient: uc,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", srv.handleAddAnime).Methods(http.MethodPost)
	r.HandleFunc("/my", srv.handleMyAnimes).Methods(http.MethodGet)

	addr := ":8080"
	log.Printf("anime-service listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("anime-service failed: %v", err)
	}
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS watched (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			anime TEXT NOT NULL,
			folder_name TEXT NOT NULL
		);
	`)
	return err
}

func (s *Server) handleAddAnime(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing user id", http.StatusUnauthorized)
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	// проверяем, что пользователь существует (взаимодействие между микросервисами)
	_, err = s.userClient.GetUserByID(userIDStr, r.Header)
	if err != nil {
		http.Error(w, "user does not exist", http.StatusBadRequest)
		return
	}

	var req AddAnimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Anime == "" || req.FolderName == "" {
		http.Error(w, "Anime and folder name required", http.StatusBadRequest)
		return
	}

	res, err := s.db.Exec("INSERT INTO watched(user_id, anime, folder_name) VALUES(?, ?, ?)", userID, req.Anime, req.FolderName)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	watchedAnime := WatchedAnime{
		ID:         id,
		UserID:     userID,
		Anime:      req.Anime,
		FolderName: req.FolderName,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(watchedAnime)
}

func (s *Server) handleMyAnimes(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing user id", http.StatusUnauthorized)
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Query("SELECT id, user_id, anime, folder_name FROM watched WHERE user_id = ?", userID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []WatchedAnime
	for rows.Next() {
		var o WatchedAnime
		if err := rows.Scan(&o.ID, &o.UserID, &o.Anime, &o.FolderName); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		orders = append(orders, o)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
