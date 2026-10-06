package httpapi

import (
	app "github.com/own/mock-ai/apps/api/internal/exam/application"
	identity "github.com/own/mock-ai/apps/api/internal/identity/httpapi"
	"net/http"
)

func (h *Handler) registerLearning(mux *http.ServeMux) {
	handle := func(path string, fn http.HandlerFunc) { mux.Handle(path, h.auth.RequireUser(fn)) }
	handle("GET /api/v1/attempts/{attempt_id}/workspace", func(w http.ResponseWriter, r *http.Request) {
		u, _ := identity.UserFromContext(r.Context())
		v, e := h.service.GetWorkspace(r.Context(), u.ID, r.PathValue("attempt_id"))
		if e != nil {
			h.fail(w, e)
			return
		}
		writeJSON(w, 200, v)
	})
	handle("PUT /api/v1/attempts/{attempt_id}/workspace", func(w http.ResponseWriter, r *http.Request) {
		var v app.Workspace
		if e := decodeJSON(w, r, &v); e != nil {
			writeError(w, 400, "invalid_request", e.Error())
			return
		}
		u, _ := identity.UserFromContext(r.Context())
		if e := h.service.SaveWorkspace(r.Context(), u.ID, r.PathValue("attempt_id"), v); e != nil {
			h.fail(w, e)
			return
		}
		w.WriteHeader(204)
	})
	handle("GET /api/v1/me/mistakes", func(w http.ResponseWriter, r *http.Request) {
		u, _ := identity.UserFromContext(r.Context())
		v, e := h.service.ListMistakes(r.Context(), u.ID)
		if e != nil {
			h.fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"mistakes": v})
	})
	handle("POST /api/v1/me/mistakes/{question_id}/retry", func(w http.ResponseWriter, r *http.Request) {
		var v struct {
			Values []string `json:"values"`
		}
		if e := decodeJSON(w, r, &v); e != nil {
			writeError(w, 400, "invalid_request", e.Error())
			return
		}
		u, _ := identity.UserFromContext(r.Context())
		result, e := h.service.RetryMistake(r.Context(), u.ID, r.PathValue("question_id"), v.Values)
		if e != nil {
			h.fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"feedback": result})
	})
	handle("GET /api/v1/me/reminder", func(w http.ResponseWriter, r *http.Request) {
		u, _ := identity.UserFromContext(r.Context())
		v, e := h.service.GetReminder(r.Context(), u.ID)
		if e != nil {
			h.fail(w, e)
			return
		}
		writeJSON(w, 200, v)
	})
	handle("PUT /api/v1/me/reminder", func(w http.ResponseWriter, r *http.Request) {
		var v app.Reminder
		if e := decodeJSON(w, r, &v); e != nil {
			writeError(w, 400, "invalid_request", e.Error())
			return
		}
		u, _ := identity.UserFromContext(r.Context())
		if e := h.service.SaveReminder(r.Context(), u.ID, v); e != nil {
			h.fail(w, e)
			return
		}
		w.WriteHeader(204)
	})
	admin := func(fn http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			u, _ := identity.UserFromContext(r.Context())
			for _, role := range u.Roles {
				if role == "admin" {
					fn(w, r)
					return
				}
			}
			writeError(w, 403, "forbidden", "Administrator access required")
		}
	}
	handle("GET /api/v1/admin/content", admin(func(w http.ResponseWriter, r *http.Request) {
		v, e := h.service.ListDrafts(r.Context())
		if e != nil {
			h.fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"drafts": v})
	}))
	save := admin(func(w http.ResponseWriter, r *http.Request) {
		var v app.Content
		if e := decodeJSON(w, r, &v); e != nil {
			writeError(w, 400, "invalid_request", e.Error())
			return
		}
		u, _ := identity.UserFromContext(r.Context())
		id, e := h.service.SaveDraft(r.Context(), u.ID, r.PathValue("draft_id"), v)
		if e != nil {
			h.fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]string{"id": id})
	})
	handle("POST /api/v1/admin/content", save)
	handle("PUT /api/v1/admin/content/{draft_id}", save)
	handle("POST /api/v1/admin/content/{draft_id}/publish", admin(func(w http.ResponseWriter, r *http.Request) {
		id, e := h.service.PublishDraft(r.Context(), r.PathValue("draft_id"))
		if e != nil {
			h.fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]string{"exam_id": id})
	}))
}
