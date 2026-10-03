# NoteHub 0.3.0 — C++/Qt GUI + Go Backend

NoteHub đã được tái cấu trúc từ **Go + Fyne một process** sang kiến trúc desktop
**hai process**:

```text
┌───────────────────────────────────────┐
│ NoteHub.exe                           │
│ C++20 + Qt 6 Widgets                 │
│ MainWindow / Editor / Calendar / ... │
└──────────────────┬────────────────────┘
                   │ JSON-RPC / stdio
                   │ QProcess
┌──────────────────▼────────────────────┐
│ notehub-backend(.exe)                 │
│ Go                                    │
│ Service → Repository → SQLite         │
│ Backup / Search / Share / Storage     │
└──────────────────┬────────────────────┘
                   ├── notehub.db
                   └── attachments/
```

GUI **không truy cập SQLite trực tiếp**. Toàn bộ business logic cũ (memo, tag,
search, calendar, attachment, backup, share, migration) vẫn ở Go. Qt chỉ làm
presentation, chọn file, mở file, theme và điều phối thao tác người dùng.

## Điểm chính của bản Qt

- Qt Widgets native, giao diện 3 vùng responsive, tự ẩn panel phải khi cửa sổ hẹp.
- Cửa sổ chính được đặt trạng thái **Maximized trước frame đầu tiên được hiển thị**,
  để hệ điều hành tự fit vào work area (không che taskbar, không còn hiệu ứng mở
  cửa sổ nhỏ ở góc rồi mới phóng to như Fyne trước đây).
- Timeline, favorite, shared, tags, calendar, Unicode/FTS search.
- Editor Markdown + preview, import/export `.md`, attachment, reorder attachment.
- Attachment browser, mở file/thư mục, đi tới note.
- ZIP backup: Skip / Replace / Duplicate.
- Local share server vẫn chỉ bind `127.0.0.1` và mặc định **tắt sau mỗi lần mở app**.
- Theme System/Light/Dark, font và cỡ chữ được lưu bằng `QSettings`.
- IPC dùng JSON qua stdin/stdout: không mở cổng cho GUI↔backend, không bị firewall,
  không cần DLL bridge/CGO, backend có thể crash/restart độc lập với Qt.

## Build Windows

Yêu cầu:

- Go 1.23+
- CMake 3.21+
- Qt 6.5+ (khuyến nghị Qt 6.11.x)
- Compiler tương thích với bộ Qt đã cài (MSVC 2022 hoặc MinGW tương ứng)

Ví dụ Qt MSVC:

```powershell
cd E:\ALL_PROJECTS\GIT_REPOSITORY\OFFICE_EDITOR\NoteHub
.\scripts\build-windows.ps1 -QtDir "C:\Qt\6.11.2\msvc2022_64"
.\dist\windows\NoteHub.exe
```

Ví dụ Qt MinGW:

```powershell
.\scripts\build-windows.ps1 -QtDir "C:\Qt\6.11.2\mingw_64"
```

Script build tạo cùng một thư mục:

```text
dist/windows/
├── NoteHub.exe              # C++/Qt GUI
├── notehub-backend.exe      # Go backend
├── Qt6*.dll + plugins/...   # windeployqt
└── NoteHub.png
```

`NoteHub.exe` luôn tìm `notehub-backend.exe` cạnh chính nó. Có thể override khi
debug bằng biến môi trường `NOTEHUB_BACKEND` hoặc `--backend <path>`.

## Build Linux / macOS

```bash
# Linux
QT_ROOT=/path/to/Qt/6.x/gcc_64 bash scripts/build-linux.sh
./dist/linux/NoteHub

# macOS
QT_ROOT=/path/to/Qt/6.x/macos bash scripts/build-macos.sh
open dist/macos/NoteHub.app
```

## Chạy với data directory riêng

```powershell
.\dist\windows\NoteHub.exe --data-dir "D:\My Notes\NoteHub"
```

Mặc định:

| Hệ điều hành | Data directory |
|---|---|
| Windows | `%LOCALAPPDATA%\NoteHub` |
| macOS | `~/Library/Application Support/NoteHub` |
| Linux | `$XDG_DATA_HOME/NoteHub` hoặc `~/.local/share/NoteHub` |

Dữ liệu hiện có từ bản Fyne **không cần convert** vì schema SQLite và attachment
store được giữ nguyên. Migration hiện tại vẫn chạy additive qua Go backend.

## Development

Backend:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./cmd/notehub-backend
```

Frontend:

```bash
cmake -S . -B build/qt -DCMAKE_PREFIX_PATH=/path/to/Qt
cmake --build build/qt --config Release --parallel
```

Tài liệu kỹ thuật:

- [Kiến trúc](docs/ARCHITECTURE.md)
- [IPC protocol](docs/IPC_PROTOCOL.md)
- [Quá trình chuyển Fyne → Qt](docs/MIGRATION_QT_GO.md)
- [Phân tích source gốc](docs/SOURCE_ANALYSIS.md)
- [Hướng dẫn build](docs/BUILD.md)
- [Cấu trúc project](PROJECT_STRUCTURE_NOTEHUB.md)

## License

MIT — xem [LICENSE](LICENSE).
