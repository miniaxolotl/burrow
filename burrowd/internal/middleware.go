package internal

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func loggingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s %s %s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
		})
	}
}

func corsMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

var publicPaths = []string{"/api/auth/", "/api/stripe/webhook"}

func authMiddleware(jwtSecret string, deploymentMode string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if deploymentMode == "oss" {
				next.ServeHTTP(w, r)
				return
			}
			for _, prefix := range publicPaths {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}
			token := tokenFromHeader(r)
			if token == "" {
				jsonError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err := ValidateAccessToken(token, jwtSecret)
			if err != nil {
				jsonError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, contextKey("role"), claims.Role)
			ctx = context.WithValue(ctx, contextKey("plan"), claims.Plan)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func tokenFromHeader(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}