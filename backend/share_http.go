package backend

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ShareHandler returns a small read-only HTTP handler for share tokens.
// It is optional: a desktop app can mount it on localhost/LAN only when the
// user explicitly enables sharing. No HTML/JS/CSS frontend is required.
//
// Routes:
//
//	GET /s/{token}                         -> shared memo JSON
//	GET /s/{token}/attachments/{uid}       -> attachment bytes
func (s *Service) ShareHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 2 || parts[0] != "s" || parts[1] == "" {
			http.NotFound(w, r)
			return
		}
		shared, err := s.ResolveShare(r.Context(), parts[1])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if len(parts) == 2 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_ = json.NewEncoder(w).Encode(shared)
			return
		}
		if len(parts) == 4 && parts[2] == "attachments" {
			uid := parts[3]
			for _, a := range shared.Memo.Attachments {
				if a.UID != uid {
					continue
				}
				full, err := ensureInside(s.attachmentRoot, filepath.FromSlash(a.RelativePath))
				if err != nil {
					http.Error(w, "invalid attachment path", http.StatusInternalServerError)
					return
				}
				f, err := os.Open(full)
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
				w.Header().Set("X-Content-Type-Options", "nosniff")
				http.ServeContent(w, r, a.Filename, st.ModTime(), f)
				return
			}
			http.NotFound(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

// ShareURL joins an externally chosen base URL with a share token.
// Example baseURL: http://192.168.1.10:8787
func ShareURL(baseURL, token string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(token) == "" {
		return "", errors.New("base URL and token are required")
	}
	return baseURL + "/s/" + token, nil
}
