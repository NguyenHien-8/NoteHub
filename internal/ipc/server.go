package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/service"
	sharehttp "github.com/NguyenHien-8/NoteHub/internal/share"
)

const protocolVersion = 1

type Request struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type Response struct {
	ID     int64     `json:"id"`
	Result any       `json:"result,omitempty"`
	Error  *RPCError `json:"error,omitempty"`
}

type Event struct {
	Event           string `json:"event"`
	ProtocolVersion int    `json:"protocolVersion,omitempty"`
	Version         string `json:"version,omitempty"`
	DataDir         string `json:"dataDir,omitempty"`
	OS              string `json:"os,omitempty"`
	Message         string `json:"message,omitempty"`
}

type Server struct {
	backend *app.Backend
	version string
	sharing *sharehttp.Server
}

func NewServer(backend *app.Backend, version string) *Server {
	return &Server{backend: backend, version: version}
}

func ReadyEvent(backend *app.Backend, version string) Event {
	return Event{Event: "ready", ProtocolVersion: protocolVersion, Version: version, DataDir: backend.Paths.Root, OS: runtime.GOOS}
}

func FatalEvent(message string) Event { return Event{Event: "fatal", Message: message} }

// Serve runs the local desktop IPC protocol over stdin/stdout. Each request and
// response is a single JSON object. Encoding/json also accepts arbitrary
// whitespace, so large memo bodies are not limited by bufio.Scanner token size.
// stdout is reserved exclusively for protocol traffic; diagnostics belong on
// stderr so the C++ frontend can parse responses deterministically.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	dec := json.NewDecoder(bufio.NewReaderSize(in, 128*1024))
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return s.closeSharing()
			}
			return fmt.Errorf("decode IPC request: %w", err)
		}
		result, shutdown, err := s.handle(ctx, req.Method, req.Params)
		resp := Response{ID: req.ID}
		if err != nil {
			resp.Error = rpcError(err)
		} else {
			resp.Result = result
		}
		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("encode IPC response: %w", err)
		}
		if shutdown {
			return s.closeSharing()
		}
	}
}

func (s *Server) closeSharing() error {
	if s.sharing == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := s.sharing.Shutdown(ctx)
	s.sharing = nil
	return err
}

func (s *Server) handle(ctx context.Context, method string, raw json.RawMessage) (any, bool, error) {
	switch method {
	case "app.ping":
		return map[string]any{"ok": true, "protocolVersion": protocolVersion}, false, nil
	case "app.info":
		return map[string]any{"version": s.version, "protocolVersion": protocolVersion, "dataDir": s.backend.Paths.Root, "database": s.backend.Paths.Database, "os": runtime.GOOS}, false, nil
	case "app.shutdown":
		return map[string]any{"ok": true}, true, nil

	case "memo.create":
		var p struct {
			Content string `json:"content"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		m, err := s.backend.Memos.Create(ctx, p.Content)
		if err != nil {
			return nil, false, err
		}
		return s.memoDTO(m), false, nil
	case "memo.get":
		var p idParam
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		m, err := s.backend.Memos.Get(ctx, p.ID)
		if err != nil {
			return nil, false, err
		}
		return s.memoDTO(m), false, nil
	case "memo.update":
		var p struct {
			ID       int64  `json:"id"`
			Revision int64  `json:"revision"`
			Content  string `json:"content"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		m, err := s.backend.Memos.Update(ctx, p.ID, p.Revision, p.Content)
		if err != nil {
			return nil, false, err
		}
		return s.memoDTO(m), false, nil
	case "memo.delete":
		var p idParam
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		return okResult(s.backend.Memos.Delete(ctx, p.ID))
	case "memo.favorite":
		var p struct {
			ID       int64 `json:"id"`
			Favorite bool  `json:"favorite"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		return okResult(s.backend.Memos.SetFavorite(ctx, p.ID, p.Favorite))

	case "timeline.counts":
		counts, err := s.backend.Timeline.Counts(ctx)
		if err != nil {
			return nil, false, err
		}
		return map[string]int{"all": counts.All, "favorites": counts.Favorites, "shared": counts.Shared}, false, nil
	case "timeline.list":
		q, err := decodeTimelineQuery(raw)
		if err != nil {
			return nil, false, err
		}
		items, next, err := s.backend.Timeline.List(ctx, q)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"items": s.memoDTOs(items), "next": cursorDTO(next)}, false, nil

	case "search.query":
		q, err := decodeSearchQuery(raw)
		if err != nil {
			return nil, false, err
		}
		items, err := s.backend.Search.Search(ctx, q)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"items": s.memoDTOs(items)}, false, nil

	case "calendar.month":
		var p struct {
			Year          int `json:"year"`
			Month         int `json:"month"`
			OffsetMinutes int `json:"offsetMinutes"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		if p.Year < 1970 || p.Year > 9999 || p.Month < 1 || p.Month > 12 {
			return nil, false, fmt.Errorf("%w: invalid calendar month", domain.ErrInvalid)
		}
		loc := fixedLocation(p.OffsetMinutes)
		days, err := s.backend.Calendar.Month(ctx, p.Year, time.Month(p.Month), loc)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"days": days}, false, nil
	case "calendar.date":
		var p struct {
			Date          string             `json:"date"`
			Limit         int                `json:"limit"`
			OffsetMinutes int                `json:"offsetMinutes"`
			Cursor        *timelineCursorDTO `json:"cursor"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		cur, err := parseCursor(p.Cursor)
		if err != nil {
			return nil, false, err
		}
		items, next, err := s.backend.Calendar.PageForDate(ctx, p.Date, fixedLocation(p.OffsetMinutes), p.Limit, cur)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"items": s.memoDTOs(items), "next": cursorDTO(next)}, false, nil

	case "tags.list":
		tags, err := s.backend.Tags.List(ctx)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"items": tags}, false, nil

	case "attachment.add":
		var p struct {
			MemoID int64  `json:"memoId"`
			Path   string `json:"path"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		if strings.TrimSpace(p.Path) == "" {
			return nil, false, fmt.Errorf("%w: attachment path is required", domain.ErrInvalid)
		}
		a, err := s.backend.Attachments.AddFromFile(ctx, p.MemoID, p.Path)
		if err != nil {
			return nil, false, err
		}
		return s.attachmentDTO(*a), false, nil
	case "attachment.list":
		var p struct {
			MemoID int64 `json:"memoId"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		items, err := s.backend.Attachments.List(ctx, p.MemoID)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"items": s.attachmentDTOs(items)}, false, nil
	case "attachment.listAll":
		var p struct {
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		items, err := s.backend.Attachments.ListAll(ctx, p.Limit, p.Offset)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"items": s.attachmentDTOs(items)}, false, nil
	case "attachment.reorder":
		var p struct {
			MemoID int64   `json:"memoId"`
			IDs    []int64 `json:"ids"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		return okResult(s.backend.Attachments.Reorder(ctx, p.MemoID, p.IDs))
	case "attachment.delete":
		var p idParam
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		return okResult(s.backend.Attachments.Delete(ctx, p.ID))
	case "attachment.path":
		var p idParam
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		path, err := s.backend.Attachments.Path(ctx, p.ID)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"path": path}, false, nil

	case "backup.export":
		var p struct {
			Path string `json:"path"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		return okResult(s.backend.Backup.ExportFile(ctx, p.Path))
	case "backup.import":
		var p struct {
			Path   string `json:"path"`
			Policy string `json:"policy"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		policy, err := importPolicy(p.Policy)
		if err != nil {
			return nil, false, err
		}
		report, err := s.backend.Backup.ImportFile(ctx, p.Path, policy)
		if err != nil {
			return nil, false, err
		}
		return report, false, nil

	case "share.create":
		var p struct {
			MemoID      int64 `json:"memoId"`
			ExpiresDays int   `json:"expiresDays"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		var expires *time.Time
		if p.ExpiresDays > 0 {
			t := time.Now().UTC().Add(time.Duration(p.ExpiresDays) * 24 * time.Hour)
			expires = &t
		}
		grant, err := s.backend.Shares.Create(ctx, p.MemoID, expires)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"share": shareDTO(grant.Share), "token": grant.Token, "url": s.shareURL(grant.Token)}, false, nil
	case "share.list":
		var p struct {
			MemoID int64 `json:"memoId"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		items, err := s.backend.Shares.List(ctx, p.MemoID)
		if err != nil {
			return nil, false, err
		}
		out := make([]map[string]any, len(items))
		for i := range items {
			out[i] = shareDTO(items[i])
		}
		return map[string]any{"items": out}, false, nil
	case "share.revoke":
		var p struct {
			UID string `json:"uid"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		return okResult(s.backend.Shares.Revoke(ctx, p.UID))
	case "share.server":
		var p struct {
			Enabled bool `json:"enabled"`
			Port    int  `json:"port"`
		}
		if err := decode(raw, &p); err != nil {
			return nil, false, err
		}
		return s.setSharing(p.Enabled, p.Port)
	case "share.status":
		return s.shareStatus(), false, nil
	default:
		return nil, false, fmt.Errorf("%w: unknown IPC method %q", domain.ErrInvalid, method)
	}
}

type idParam struct {
	ID int64 `json:"id"`
}

func decode(raw json.RawMessage, v any) error {
	if len(raw) == 0 || string(raw) == "null" {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%w: invalid IPC parameters: %v", domain.ErrInvalid, err)
	}
	return nil
}

func okResult(err error) (any, bool, error) {
	if err != nil {
		return nil, false, err
	}
	return map[string]any{"ok": true}, false, nil
}

func rpcError(err error) *RPCError {
	out := &RPCError{Code: 500, Kind: "internal", Message: err.Error()}
	switch {
	case errors.Is(err, domain.ErrInvalid):
		out.Code, out.Kind = 400, "invalid"
	case errors.Is(err, domain.ErrNotFound):
		out.Code, out.Kind = 404, "not_found"
	case errors.Is(err, domain.ErrConflict):
		out.Code, out.Kind = 409, "conflict"
	}
	return out
}

type attachmentDTO struct {
	ID        int64  `json:"id"`
	UID       string `json:"uid"`
	MemoID    int64  `json:"memoId"`
	Position  int    `json:"position"`
	CreatedAt string `json:"createdAt"`
	Filename  string `json:"filename"`
	MIMEType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	LocalPath string `json:"localPath,omitempty"`
}

type memoDTO struct {
	ID          int64           `json:"id"`
	UID         string          `json:"uid"`
	Content     string          `json:"content"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
	Revision    int64           `json:"revision"`
	Favorite    bool            `json:"favorite"`
	Tags        []string        `json:"tags"`
	Attachments []attachmentDTO `json:"attachments"`
}

func (s *Server) attachmentDTO(a domain.Attachment) attachmentDTO {
	path, _ := s.backend.Files.Path(a.RelativePath)
	return attachmentDTO{ID: a.ID, UID: a.UID, MemoID: a.MemoID, Position: a.Position, CreatedAt: formatTime(a.CreatedAt), Filename: a.Filename, MIMEType: a.MIMEType, Size: a.Size, SHA256: a.SHA256, LocalPath: path}
}
func (s *Server) attachmentDTOs(items []domain.Attachment) []attachmentDTO {
	out := make([]attachmentDTO, len(items))
	for i := range items {
		out[i] = s.attachmentDTO(items[i])
	}
	return out
}
func (s *Server) memoDTO(m *domain.Memo) memoDTO {
	if m == nil {
		return memoDTO{}
	}
	return memoDTO{ID: m.ID, UID: m.UID, Content: m.Content, CreatedAt: formatTime(m.CreatedAt), UpdatedAt: formatTime(m.UpdatedAt), Revision: m.Revision, Favorite: m.Favorite, Tags: append([]string(nil), m.Tags...), Attachments: s.attachmentDTOs(m.Attachments)}
}
func (s *Server) memoDTOs(items []domain.Memo) []memoDTO {
	out := make([]memoDTO, len(items))
	for i := range items {
		out[i] = s.memoDTO(&items[i])
	}
	return out
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

type timelineCursorDTO struct {
	CreatedAt string `json:"createdAt"`
	ID        int64  `json:"id"`
}

func cursorDTO(c *repository.TimelineCursor) any {
	if c == nil {
		return nil
	}
	return timelineCursorDTO{CreatedAt: formatTime(c.CreatedAt), ID: c.ID}
}
func parseCursor(c *timelineCursorDTO) (*repository.TimelineCursor, error) {
	if c == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
	}
	return &repository.TimelineCursor{CreatedAt: t, ID: c.ID}, nil
}

func parseOptionalTime(v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid time %q", domain.ErrInvalid, v)
	}
	return &t, nil
}

func decodeTimelineQuery(raw json.RawMessage) (repository.TimelineQuery, error) {
	var p struct {
		From         string             `json:"from"`
		To           string             `json:"to"`
		Tags         []string           `json:"tags"`
		Limit        int                `json:"limit"`
		Cursor       *timelineCursorDTO `json:"cursor"`
		FavoriteOnly bool               `json:"favoriteOnly"`
		SharedOnly   bool               `json:"sharedOnly"`
	}
	if err := decode(raw, &p); err != nil {
		return repository.TimelineQuery{}, err
	}
	from, err := parseOptionalTime(p.From)
	if err != nil {
		return repository.TimelineQuery{}, err
	}
	to, err := parseOptionalTime(p.To)
	if err != nil {
		return repository.TimelineQuery{}, err
	}
	cur, err := parseCursor(p.Cursor)
	if err != nil {
		return repository.TimelineQuery{}, err
	}
	return repository.TimelineQuery{From: from, To: to, Tags: p.Tags, Limit: p.Limit, Cursor: cur, FavoriteOnly: p.FavoriteOnly, SharedOnly: p.SharedOnly}, nil
}

func decodeSearchQuery(raw json.RawMessage) (repository.SearchQuery, error) {
	var p struct {
		Text   string   `json:"text"`
		Tags   []string `json:"tags"`
		From   string   `json:"from"`
		To     string   `json:"to"`
		Limit  int      `json:"limit"`
		Offset int      `json:"offset"`
	}
	if err := decode(raw, &p); err != nil {
		return repository.SearchQuery{}, err
	}
	from, err := parseOptionalTime(p.From)
	if err != nil {
		return repository.SearchQuery{}, err
	}
	to, err := parseOptionalTime(p.To)
	if err != nil {
		return repository.SearchQuery{}, err
	}
	return repository.SearchQuery{Text: p.Text, Tags: p.Tags, From: from, To: to, Limit: p.Limit, Offset: p.Offset}, nil
}

func fixedLocation(offsetMinutes int) *time.Location {
	if offsetMinutes < -14*60 || offsetMinutes > 14*60 {
		offsetMinutes = 0
	}
	name := "UTC"
	if offsetMinutes != 0 {
		sign := "+"
		n := offsetMinutes
		if n < 0 {
			sign, n = "-", -n
		}
		name = "UTC" + sign + strconv.Itoa(n/60) + ":" + fmt.Sprintf("%02d", n%60)
	}
	return time.FixedZone(name, offsetMinutes*60)
}

func importPolicy(v string) (service.ImportPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "skip":
		return service.ImportSkip, nil
	case "replace":
		return service.ImportReplace, nil
	case "duplicate":
		return service.ImportDuplicate, nil
	default:
		return service.ImportSkip, fmt.Errorf("%w: unknown import policy %q", domain.ErrInvalid, v)
	}
}

func shareDTO(sh domain.Share) map[string]any {
	out := map[string]any{"uid": sh.UID, "createdAt": formatTime(sh.CreatedAt)}
	if sh.ExpiresAt != nil {
		out["expiresAt"] = formatTime(*sh.ExpiresAt)
	}
	return out
}

func (s *Server) shareStatus() map[string]any {
	if s.sharing == nil {
		return map[string]any{"enabled": false, "url": ""}
	}
	return map[string]any{"enabled": true, "url": "http://" + s.sharing.Addr().String()}
}
func (s *Server) shareURL(token string) string {
	if s.sharing == nil {
		return ""
	}
	u, _ := sharehttp.URL("http://"+s.sharing.Addr().String(), token)
	return u
}
func (s *Server) setSharing(enabled bool, port int) (any, bool, error) {
	if !enabled {
		if err := s.closeSharing(); err != nil {
			return nil, false, err
		}
		return s.shareStatus(), false, nil
	}
	if port < 0 || port > 65535 {
		return nil, false, fmt.Errorf("%w: invalid share port", domain.ErrInvalid)
	}
	if s.sharing != nil {
		if err := s.closeSharing(); err != nil {
			return nil, false, err
		}
	}
	address := "127.0.0.1:" + strconv.Itoa(port)
	server, err := sharehttp.Start(address, sharehttp.Handler(s.backend.Shares))
	if err != nil {
		return nil, false, err
	}
	s.sharing = server
	return s.shareStatus(), false, nil
}

// EncodeEvent is used by the backend process before Serve starts.
func EncodeEvent(w io.Writer, event Event) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(event)
}

// Ensure stdout is not accidentally used for diagnostics in this package.
var _ = os.Stderr
