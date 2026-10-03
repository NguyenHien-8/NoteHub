# Migration: Go/Fyne → C++/Qt + Go backend

## Phần bị loại khỏi production build

- `internal/ui/**` (Fyne widgets/screens/dialogs/theme)
- `cmd/notehub` Fyne entry point
- `internal/platform/window_size*`
- `internal/platform/window_startup*`
- `internal/platform/chooser.go` (Zenity file dialogs)
- Fyne/OpenGL/GLFW/Zenity dependencies trong `go.mod`

Qt sử dụng native `QFileDialog`, `QDesktopServices`, `QSettings`, `QCalendarWidget`
và `QMainWindow` thay cho các lớp trên.

## Phần được giữ nguyên

Business logic và persistence của bản mới nhất được giữ ở Go. Việc chuyển GUI
không thay schema, attachment layout, token hashing, backup format, FTS search,
optimistic revision hay safe ZIP checks.

## Lớp mới

### `cmd/notehub-backend`
Process Go headless. Chỉ khởi tạo backend và phục vụ IPC.

### `internal/ipc`
Adapter giữa DTO JSON và service methods. Đây là anti-corruption layer để Qt
không biết repository/SQLite.

### `frontend/`
C++20 + Qt 6 Widgets:

- `IpcClient`: QProcess + request ID + callback mapping.
- `MainWindow`: responsive shell, pages, filters, settings.
- `MemoCard`: timeline card và attachment previews.
- `MemoDialog`: Markdown editor/preview + attachment management.
- `Theme`: System/Light/Dark + font scaling.

## Startup window fix

Fyne cần GLFW creation hint vì native window chỉ xuất hiện sau `Show()`. Qt không
cần workaround đó. Bản mới xây toàn bộ widget tree khi còn hidden, nhận `ready`
từ backend, gọi `setWindowState(Qt::WindowMaximized)` rồi mới `show()`. Vì vậy
không có frame trung gian ở góc phải hoặc frame vượt work area.
