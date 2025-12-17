package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type Gateway struct {
	authProxy  *httputil.ReverseProxy
	userProxy  *httputil.ReverseProxy
	animeProxy *httputil.ReverseProxy
	jwtSecret  []byte
}

func newReverseProxy(target string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid proxy target %s: %v", target, err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

func NewGateway() *Gateway {
	authURL := getEnv("AUTH_SERVICE_URL", "http://auth-service:8080")
	userURL := getEnv("USER_SERVICE_URL", "http://user-service:8080")
	animeURL := getEnv("ANIME_SERVICE_URL", "http://anime-service:8080")

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	return &Gateway{
		authProxy:  newReverseProxy(authURL),
		userProxy:  newReverseProxy(userURL),
		animeProxy: newReverseProxy(animeURL),
		jwtSecret:  []byte(secret),
	}
}

func (g *Gateway) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractBearerToken(r.Header.Get("Authorization"))
		if tokenStr == "" {
			http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return g.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		sub, _ := claims["sub"].(float64)
		email, _ := claims["email"].(string)
		userID := fmt.Sprintf("%.0f", sub)

		// Прокидываем данные пользователя дальше
		r.Header.Set("X-User-ID", userID)
		r.Header.Set("X-User-Email", email)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func (g *Gateway) authHandler(w http.ResponseWriter, r *http.Request) {
	// /auth/... -> auth-service
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/auth")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	g.authProxy.ServeHTTP(w, r)
}

func (g *Gateway) userHandler(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/users")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	g.userProxy.ServeHTTP(w, r)
}

func (g *Gateway) animeHandler(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/anime")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	g.animeProxy.ServeHTTP(w, r)
}

func main() {
	gw := NewGateway()

	r := mux.NewRouter()

	// healthcheck
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// публичные эндпоинты авторизации
	r.PathPrefix("/auth").Handler(http.HandlerFunc(gw.authHandler)).Methods(http.MethodPost, http.MethodGet)

	// защищённые маршруты
	r.PathPrefix("/users").Handler(gw.authMiddleware(http.HandlerFunc(gw.userHandler)))
	r.PathPrefix("/anime").Handler(gw.authMiddleware(http.HandlerFunc(gw.animeHandler)))

	addr := ":8080"
	log.Printf("Gateway listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("gateway failed: %v", err)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
