# NoteHub Architecture — Qt + Go, two processes

## Kiến trúc mục tiêu

```text
                              NoteHub
                 ┌──────────────┴──────────────┐
                 │                             │
                 ▼                             ▼
       C++ / Qt GUI process              Go Backend process
       ──────────────────                ──────────────────
       Qt Widgets                        Domain
       MainWindow                        Service
       MemoCard / MemoDialog    IPC      Repository interfaces
       Calendar / Search       ◄────►     SQLite implementation
       Attachments / Tags      JSON      Backup / Share / Storage
       Settings                stdio
                                              │
                                  ┌───────────┴───────────┐
                                  ▼                       ▼
                               SQLite                Filesystem
```

## Vì sao dùng 2 process

1. **Không trộn ABI C++/Go.** Không cần `c-shared`, CGO wrapper hoặc quản lý ownership
   giữa Qt object và Go runtime.
2. **Backend giữ nguyên kiến trúc đã kiểm thử.** Service/repository/SQLite không bị
   viết lại bằng C++.
3. **GUI native đẹp hơn Fyne.** Qt xử lý DPI, work area, font, native dialogs và
   resize tốt hơn cho desktop Windows.
4. **Crash isolation.** GUI và backend có vòng đời tách biệt; Qt phát hiện backend
   dừng bất thường và đóng có kiểm soát.
5. **IPC không mở network port.** QProcess nối stdin/stdout trực tiếp. Local share
   HTTP server là chức năng khác và chỉ bật khi người dùng yêu cầu.

## Dependency rules

```text
frontend/ (C++/Qt)
       │ JSON-RPC
       ▼
internal/ipc
       │
       ▼
internal/app
       │
       ├── service ──► repository interfaces ──► repository/sqlite
       ├── storage
       ├── backup
       └── share
```

Không cho phép:

```text
Qt GUI ─X─► SQLite
Qt GUI ─X─► attachment storage internals
Domain ─X─► Qt/Fyne
Repository ─X─► GUI
Go backend stdout ─X─► log text (stdout dành riêng cho JSON IPC)
```

## Startup sequence

```text
1. User launches NoteHub.exe
2. Qt creates MainWindow but keeps it hidden
3. QProcess starts notehub-backend.exe
4. Go resolves AppData → opens SQLite → runs migrations → constructs services
5. Go sends {"event":"ready", ...}
6. Qt initializes pages/data
7. Qt sets WindowMaximized BEFORE show()
8. First visible frame is already fitted to OS work area
```

Điểm 7–8 thay thế toàn bộ workaround GLFW/Fyne trước đây (`window_size*`,
`window_startup*`). Qt/window manager sở hữu native maximize và DPI.

## Shutdown sequence

Qt gửi `app.shutdown`, backend dừng local share server (nếu đang chạy), đóng
SQLite, sau đó exit. Nếu backend không dừng trong timeout, QProcess terminate/kill
để không để process mồ côi.

## Data compatibility

Các package sau được giữ làm source of truth:

- `internal/domain`
- `internal/service`
- `internal/repository` + `internal/repository/sqlite`
- `internal/storage`
- `internal/backup`
- `internal/share`
- `internal/tagparse`
- `internal/identity`
- `internal/platform` (data path / OS helpers còn phù hợp)

Schema `memo`, `memo_tag`, `attachment`, `memo_share`, `memo_fts` và migration
v1→v3 không thay đổi, nên database từ NoteHub 0.2/Fyne mở trực tiếp được.
