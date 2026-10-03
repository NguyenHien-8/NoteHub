// Package ipc exposes the desktop backend over the private pipes of its Qt
// parent process. It never opens a desktop API socket or serializes DB IDs.
package ipc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/service"
	"github.com/NguyenHien-8/NoteHub/internal/share"
)

const ProtocolVersion = 1
const MaxRequestBytes = 8 << 20
const MaxResponseBytes = 64 << 20

type request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}
type rpcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type response struct {
	ID     string    `json:"id"`
	Result any       `json:"result,omitempty"`
	Error  *rpcError `json:"error,omitempty"`
}

// Server serializes operations in pipe order. This preserves mutation ordering
// and prevents a backup import racing an edit; the Qt event loop stays async.
type Server struct {
	backend *app.Backend
	version string
	sharing *share.Server
}

func New(b *app.Backend, version string) *Server { return &Server{backend: b, version: version} }
func (s *Server) Close() error {
	if s.sharing == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.sharing.Shutdown(ctx)
	s.sharing = nil
	return err
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), MaxRequestBytes)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		var req request
		res := response{}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil || req.ID == "" || req.Method == "" {
			res.ID = req.ID
			res.Error = &rpcError{"invalid_request", "A string id and method are required in each JSON record."}
		} else {
			res.ID = req.ID
			result, err := s.dispatch(ctx, req.Method, req.Params)
			if err != nil {
				res.Error = classify(err)
			} else {
				res.Result = result
			}
		}
		var encoded bytes.Buffer
		encoder := json.NewEncoder(&encoded)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(res); err != nil {
			return err
		}
		if encoded.Len() > MaxResponseBytes {
			encoded.Reset()
			_ = json.NewEncoder(&encoded).Encode(response{ID: req.ID, Error: &rpcError{"too_large", "Response too large; request a smaller page."}})
		}
		if _, err := out.Write(encoded.Bytes()); err != nil {
			return err
		}
	}
	return scanner.Err() // EOF is the parent's graceful shutdown signal.
}

func classify(err error) *rpcError {
	code := "internal"
	switch {
	case errors.Is(err, domain.ErrConflict):
		code = "conflict"
	case errors.Is(err, domain.ErrNotFound):
		code = "not_found"
	case errors.Is(err, domain.ErrInvalid):
		code = "invalid_params"
	case errors.Is(err, context.Canceled):
		code = "canceled"
	}
	return &rpcError{code, err.Error()}
}

type params struct {
	UID      string   `json:"uid"`
	Content  string   `json:"content"`
	Revision int64    `json:"revision"`
	Favorite bool     `json:"favorite"`
	Shared   bool     `json:"shared"`
	Tags     []string `json:"tags"`
	Text     string   `json:"text"`
	Date     string   `json:"date"`
	Cursor   string   `json:"cursor"`
	Limit    int      `json:"limit"`
	Offset   int      `json:"offset"`
	Year     int      `json:"year"`
	Month    int      `json:"month"`
	Paths    []string `json:"paths"`
	Path     string   `json:"path"`
	Order    []string `json:"order"`
	Policy   string   `json:"policy"`
	Enabled  bool     `json:"enabled"`
	Port     int      `json:"port"`
	Days     int      `json:"days"`
}

type attachmentDTO struct {
	domain.Attachment
	Path    string `json:"path"`
	MemoUID string `json:"memoUid"`
}
type memoDTO struct {
	domain.Memo
	Files []attachmentDTO `json:"attachments"`
}

func (s *Server) attachment(ctx context.Context, a domain.Attachment) attachmentDTO {
	path, _ := s.backend.Attachments.Path(ctx, a.ID)
	memo, _ := s.backend.Store.GetMemoByID(ctx, a.MemoID)
	uid := ""
	if memo != nil {
		uid = memo.UID
	}
	return attachmentDTO{a, path, uid}
}
func (s *Server) memo(ctx context.Context, m *domain.Memo) memoDTO {
	files := make([]attachmentDTO, 0, len(m.Attachments))
	for _, a := range m.Attachments {
		files = append(files, s.attachment(ctx, a))
	}
	return memoDTO{*m, files}
}

func (s *Server) dispatch(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	p := params{}
	if len(raw) > 0 && string(raw) != "null" {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
		}
	}
	b := s.backend
	switch method {
	case "hello":
		return map[string]any{"protocol": ProtocolVersion, "version": s.version, "dataDir": b.Paths.Root}, nil
	case "metadata":
		counts, err := b.Timeline.Counts(ctx)
		if err != nil {
			return nil, err
		}
		tags, err := b.Tags.List(ctx)
		if err != nil {
			return nil, err
		}
		url := ""
		if s.sharing != nil {
			url = "http://" + s.sharing.Addr().String()
		}
		return map[string]any{"all": counts.All, "favorites": counts.Favorites, "shared": counts.Shared, "tags": tags, "sharingUrl": url}, nil
	case "memos.list":
		limit := p.Limit
		if limit <= 0 {
			limit = 20
		}
		limit = min(limit, 50)
		if p.Offset < 0 {
			return nil, fmt.Errorf("%w: negative offset", domain.ErrInvalid)
		}
		var items []domain.Memo
		var next *repository.TimelineCursor
		var err error
		var cursor *repository.TimelineCursor
		if p.Cursor != "" {
			data, e := base64.RawURLEncoding.DecodeString(p.Cursor)
			if e != nil {
				return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
			}
			cursor = &repository.TimelineCursor{}
			if json.Unmarshal(data, cursor) != nil {
				return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
			}
		}
		nextOffset := 0
		if p.Text != "" {
			items, err = b.Search.Search(ctx, repository.SearchQuery{Text: p.Text, Tags: p.Tags, Limit: limit, Offset: p.Offset})
			if len(items) == limit {
				nextOffset = p.Offset + limit
			}
		} else {
			q := repository.TimelineQuery{Tags: p.Tags, FavoriteOnly: p.Favorite, SharedOnly: p.Shared, Limit: limit, Cursor: cursor}
			if p.Date != "" {
				start, e := time.ParseInLocation("2006-01-02", p.Date, time.Local)
				if e != nil {
					return nil, fmt.Errorf("%w: invalid date", domain.ErrInvalid)
				}
				end := start.AddDate(0, 0, 1)
				q.From = &start
				q.To = &end
			}
			items, next, err = b.Timeline.List(ctx, q)
		}
		if err != nil {
			return nil, err
		}
		result := make([]memoDTO, 0, len(items))
		for i := range items {
			result = append(result, s.memo(ctx, &items[i]))
		}
		token := ""
		if next != nil {
			data, _ := json.Marshal(next)
			token = base64.RawURLEncoding.EncodeToString(data)
		}
		return map[string]any{"items": result, "cursor": token, "nextOffset": nextOffset}, nil
	case "memos.create":
		m, err := b.Memos.Create(ctx, p.Content)
		if err != nil {
			return nil, err
		}
		return s.memo(ctx, m), nil
	case "memos.get", "memos.update", "memos.delete", "memos.favorite", "attachments.add", "attachments.reorder", "shares.create", "shares.list":
		m, err := b.Memos.GetByUID(ctx, p.UID)
		if err != nil {
			return nil, err
		}
		switch method {
		case "memos.get":
			return s.memo(ctx, m), nil
		case "memos.update":
			updated, err := b.Memos.Update(ctx, m.ID, p.Revision, p.Content)
			if err != nil {
				return nil, err
			}
			return s.memo(ctx, updated), nil
		case "memos.delete":
			return true, b.Memos.Delete(ctx, m.ID)
		case "memos.favorite":
			return true, b.Memos.SetFavorite(ctx, m.ID, p.Favorite)
		case "attachments.add":
			if len(p.Paths) > 100 {
				return nil, fmt.Errorf("%w: select at most 100 files", domain.ErrInvalid)
			}
			failures := []string{}
			for _, path := range p.Paths {
				if _, err := b.Attachments.AddFromFile(ctx, m.ID, path); err != nil {
					failures = append(failures, path+": "+err.Error())
				}
			}
			updated, err := b.Memos.Get(ctx, m.ID)
			if err != nil {
				return nil, err
			}
			return map[string]any{"memo": s.memo(ctx, updated), "failures": failures}, nil
		case "attachments.reorder":
			byUID := map[string]int64{}
			for _, a := range m.Attachments {
				byUID[a.UID] = a.ID
			}
			ids := make([]int64, len(p.Order))
			for i, uid := range p.Order {
				id, ok := byUID[uid]
				if !ok {
					return nil, domain.ErrConflict
				}
				ids[i] = id
			}
			return true, b.Attachments.Reorder(ctx, m.ID, ids)
		case "shares.list":
			return b.Shares.List(ctx, m.ID)
		case "shares.create":
			if s.sharing == nil {
				return nil, fmt.Errorf("%w: enable local sharing in Settings first", domain.ErrInvalid)
			}
			if p.Days < 0 || p.Days > 3650 {
				return nil, fmt.Errorf("%w: invalid expiry", domain.ErrInvalid)
			}
			var expires *time.Time
			if p.Days > 0 {
				end := time.Now().AddDate(0, 0, p.Days)
				expires = &end
			}
			grant, err := b.Shares.Create(ctx, m.ID, expires)
			if err != nil {
				return nil, err
			}
			return map[string]any{"uid": grant.Share.UID, "url": "http://" + s.sharing.Addr().String() + "/s/" + grant.Token}, nil
		}
	case "attachments.list":
		limit := p.Limit
		if limit <= 0 {
			limit = 40
		}
		if p.Offset < 0 {
			return nil, domain.ErrInvalid
		}
		items, err := b.Attachments.ListAll(ctx, min(limit, 100), p.Offset)
		if err != nil {
			return nil, err
		}
		out := make([]attachmentDTO, 0, len(items))
		for _, a := range items {
			out = append(out, s.attachment(ctx, a))
		}
		return out, nil
	case "attachments.delete":
		a, err := b.Store.GetAttachmentByUID(ctx, p.UID)
		if err != nil {
			return nil, err
		}
		return true, b.Attachments.Delete(ctx, a.ID)
	case "calendar.month":
		if p.Year < 1 || p.Year > 9999 || p.Month < 1 || p.Month > 12 {
			return nil, fmt.Errorf("%w: invalid month", domain.ErrInvalid)
		}
		return b.Calendar.Month(ctx, p.Year, time.Month(p.Month), time.Local)
	case "backup.export":
		if p.Path == "" {
			return nil, domain.ErrInvalid
		}
		return true, b.Backup.ExportFile(ctx, p.Path)
	case "backup.import":
		policies := map[string]service.ImportPolicy{"skip": service.ImportSkip, "replace": service.ImportReplace, "duplicate": service.ImportDuplicate}
		policy, ok := policies[p.Policy]
		if !ok || p.Path == "" {
			return nil, domain.ErrInvalid
		}
		return b.Backup.ImportFile(ctx, p.Path, policy)
	case "sharing.set":
		if !p.Enabled {
			return true, s.Close()
		}
		if s.sharing != nil {
			return true, nil
		}
		if p.Port < 0 || p.Port > 65535 {
			return nil, domain.ErrInvalid
		}
		server, err := share.Start("127.0.0.1:"+strconv.Itoa(p.Port), share.Handler(b.Shares))
		if err != nil {
			return nil, err
		}
		s.sharing = server
		return true, nil
	case "shares.revoke":
		return true, b.Shares.Revoke(ctx, p.UID)
	}
	return nil, fmt.Errorf("%w: unknown method %q", domain.ErrInvalid, method)
}
