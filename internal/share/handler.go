package share

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

type Resolver interface {
	Resolve(context.Context, string) (*domain.SharedMemo, error)
	OpenSharedAttachment(context.Context, string, string) (*os.File, *domain.Attachment, error)
}

// Handler exposes a deliberately small, read-only sharing surface:
//
//	GET /s/{token}
//	GET /s/{token}/attachments/{attachmentUID}
//
// No desktop database path or filesystem path is serialized.
func Handler(resolver Resolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 2 || parts[0] != "s" || parts[1] == "" {
			http.NotFound(w, r)
			return
		}
		token := parts[1]
		if len(parts) == 2 {
			shared, err := resolver.Resolve(r.Context(), token)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(shared)
			return
		}
		if len(parts) == 4 && parts[2] == "attachments" && parts[3] != "" {
			f, a, err := resolver.OpenSharedAttachment(r.Context(), token, parts[3])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			st, err := f.Stat()
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if a.MIMEType != "" {
				w.Header().Set("Content-Type", a.MIMEType)
			}
			if cd := mime.FormatMediaType("inline", map[string]string{"filename": a.Filename}); cd != "" {
				w.Header().Set("Content-Disposition", cd)
			}
			http.ServeContent(w, r, a.Filename, st.ModTime(), f)
			return
		}
		http.NotFound(w, r)
	})
}

func URL(baseURL, token string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("base URL and token are required")
	}
	return baseURL + "/s/" + token, nil
}
