package dialogs

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/share"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

type Manager struct {
	Backend    *app.Backend
	Window     fyne.Window
	Jobs       *work.Runner
	Changed    func()
	drafts     map[*widget.Entry]string
	serverMu   sync.Mutex
	server     *share.Server
	backupBusy bool
	preview    func(context.Context, domain.Attachment) image.Image
}

func New(b *app.Backend, w fyne.Window, j *work.Runner, changed func()) *Manager {
	return &Manager{Backend: b, Window: w, Jobs: j, Changed: changed, drafts: make(map[*widget.Entry]string)}
}

func (m *Manager) SetPreviewLoader(loader func(context.Context, domain.Attachment) image.Image) {
	m.preview = loader
}
func (m *Manager) Error(err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		dialog.ShowError(err, m.Window)
	}
}
func (m *Manager) Dirty() bool {
	for entry, original := range m.drafts {
		if entry.Text != original {
			return true
		}
	}
	return false
}

func (m *Manager) SharingURL() string {
	m.serverMu.Lock()
	defer m.serverMu.Unlock()
	if m.server == nil {
		return ""
	}
	return "http://" + m.server.Addr().String()
}

// SetSharing is called by a worker. Starting the application never invokes it.
func (m *Manager) SetSharing(enabled bool, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	m.serverMu.Lock()
	defer m.serverMu.Unlock()
	if !enabled {
		if m.server == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.server.Shutdown(ctx)
		if err == nil {
			m.server = nil
		}
		return err
	}
	address := "127.0.0.1:" + strconv.Itoa(port)
	if m.server != nil {
		if m.server.Addr().String() == address {
			return nil
		}
		return fmt.Errorf("disable sharing before changing the port")
	}
	server, err := share.Start(address, share.Handler(m.Backend.Shares))
	if err != nil {
		return err
	}
	m.server = server
	return nil
}

// Close runs after the job runner has stopped and waited for all operations.
func (m *Manager) Close() error {
	m.serverMu.Lock()
	defer m.serverMu.Unlock()
	if m.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := m.server.Shutdown(ctx)
	m.server = nil
	return err
}
