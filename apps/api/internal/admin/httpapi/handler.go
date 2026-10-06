package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	app "github.com/own/mock-ai/apps/api/internal/admin/application"
	"net/http"
	"strings"
)

type contextKey struct{}

const CookieName = "mock_ai_admin_session"

type Handler struct {
	service *app.Service
	origin  string
	secure  bool
	slots   chan struct{}
}

func New(s *app.Service, origin string, secure bool) *Handler {
	return &Handler{s, strings.TrimRight(origin, "/"), secure, make(chan struct{}, 4)}
}
func AdminFromContext(ctx context.Context) app.Admin {
	a, _ := ctx.Value(contextKey{}).(app.Admin)
	return a
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}
func fail(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]any{"error": map[string]string{"message": message}})
}
func (h *Handler) sameOrigin(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == "GET" || r.Method == "HEAD" {
		return true
	}
	if r.Header.Get("Origin") != h.origin {
		fail(w, 403, "Admin requests must originate from the admin panel")
		return false
	}
	return true
}
func (h *Handler) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.sameOrigin(w, r) {
			return
		}
		cookie, e := r.Cookie(CookieName)
		if e != nil {
			fail(w, 401, "Admin sign-in required")
			return
		}
		a, e := h.service.Session(r.Context(), cookie.Value)
		if e != nil {
			fail(w, 401, "Admin sign-in required")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, a)))
	})
}
func (h *Handler) cookie(w http.ResponseWriter, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: value, Path: "/api/v1/admin", MaxAge: age, HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteStrictMode})
}
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/admin/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !h.sameOrigin(w, r) {
			return
		}
		select {
		case h.slots <- struct{}{}:
			defer func() { <-h.slots }()
		default:
			fail(w, 429, "Please try again shortly")
			return
		}
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input); e != nil {
			fail(w, 400, "Invalid login request")
			return
		}
		a, token, e := h.service.Login(r.Context(), input.Email, input.Password)
		if e != nil {
			if errors.Is(e, app.ErrUnauthorized) {
				fail(w, 401, e.Error())
			} else {
				fail(w, 500, "Admin login unavailable")
			}
			return
		}
		// Signing in again replaces the current browser's admin session.
		if previous, e := r.Cookie(CookieName); e == nil {
			_ = h.service.Logout(r.Context(), previous.Value)
		}
		h.cookie(w, token, 8*60*60)
		respond(w, 200, map[string]any{"admin": a})
	})
	mux.Handle("GET /api/v1/admin/me", h.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]any{"admin": AdminFromContext(r.Context())})
	})))
	mux.HandleFunc("POST /api/v1/admin/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if !h.sameOrigin(w, r) {
			return
		}
		if c, e := r.Cookie(CookieName); e == nil {
			if e = h.service.Logout(r.Context(), c.Value); e != nil {
				fail(w, 500, "Could not sign out")
				return
			}
		}
		h.cookie(w, "", -1)
		respond(w, 204, nil)
	})
}
