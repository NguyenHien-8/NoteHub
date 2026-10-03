# Phân tích source NoteHub trước khi chuyển Fyne → Qt

Baseline phân tích: `NguyenHien-8/NoteHub`, branch `master`, commit
`ed994a19aad2728efdf67343e2299bd3a430c98f` (2026-10-03). Các blob quan trọng
trong file ZIP người dùng cung cấp (`cmd/notehub/main.go`, `go.mod`,
`internal/repository/sqlite/memo_repository.go`, `internal/ui/window.go`) khớp
SHA Git với commit này.

## 1. Quy mô và điểm tách phù hợp

Source gốc có 129 file Go, khoảng 12.3k dòng. Riêng `internal/ui` Fyne khoảng
7.3k dòng, tức gần 60% tổng Go source. Trong khi đó domain/service/repository và
persistence đã được tách tương đối sạch. Vì vậy điểm refactor ít rủi ro nhất là:

```text
Fyne UI (thay hoàn toàn)
       ↓
Go service/repository/storage (giữ làm source of truth)
```

Không nên viết lại SQLite/business logic bằng C++, vì như vậy vừa tăng phạm vi
thay đổi vừa tạo nguy cơ khác hành vi với dữ liệu cũ.

## 2. Thuật toán/backend đang có

### Memo + concurrency

- `MemoService` chịu trách nhiệm validate nội dung, parse `#tag`, hydrate tag và
  attachment.
- Update dùng `expectedRevision`. SQL chỉ update khi `revision` hiện tại trùng
  revision GUI đang giữ; nếu memo còn tồn tại nhưng revision khác thì trả
  `domain.ErrConflict`.
- Favorite chỉ tăng revision/updated time khi giá trị thực sự đổi.

Qt giữ nguyên quy tắc này; IPC map conflict sang HTTP-like code `409`, editor hỏi
người dùng tải lại phiên bản mới thay vì ghi đè im lặng.

### Timeline

Timeline không dùng OFFSET cho luồng chính. Nó dùng keyset cursor:

```text
(created_ts < cursor.created_ts)
OR (created_ts = cursor.created_ts AND id < cursor.id)
ORDER BY created_ts DESC, id DESC
```

Cách này ổn định hơn khi có memo mới được thêm trong lúc người dùng đang cuộn.
Qt tiếp tục gửi/nhận cursor qua `timeline.list`.

### Search

SQLite FTS5 dùng:

```text
tokenize = 'unicode61 remove_diacritics 2'
```

và `bm25(memo_fts)` để xếp hạng khi có text. Tag filter vẫn là `EXISTS` trên
`memo_tag`; khoảng ngày dùng `[from, to)` để tránh lỗi ranh giới ngày. Qt chỉ
chuyển ngày local thành RFC3339 rồi để Go xử lý query.

### Calendar

Go tính ranh giới tháng/ngày theo timezone/offset của GUI, sau đó query timestamp
UTC. Qt truyền offset và highlight những ngày có memo; việc xác định memo thuộc
ngày nào vẫn thuộc Go backend.

### Attachment storage

- Bytes không nhét vào SQLite; SQLite chỉ giữ metadata.
- File được ghi vào temp file, giới hạn kích thước, tính SHA-256, `fsync`, rồi
  rename atomically.
- Path luôn được kiểm tra nằm dưới attachment root để chặn path traversal.
- Reorder yêu cầu một snapshot ID đầy đủ; snapshot stale trả conflict.

Qt dùng native file picker và gửi source path sang backend. Backend vẫn là nơi duy
nhất copy/validate file.

### Backup

Backup ZIP giữ format cũ. Export kiểm tra SHA-256/size trước khi hoàn tất. Import
có giới hạn archive, safe ZIP boundary, kiểm tra hash/size và ba policy:
`skip`, `replace`, `duplicate`.

### Share

Bearer token chỉ trả plaintext đúng lúc tạo grant; DB lưu hash token. Resolve
share luôn kiểm tra expiry và attachment phải thuộc đúng memo được grant. Local
HTTP server mặc định tắt và Qt chỉ bind `127.0.0.1` khi người dùng bật.

## 3. Nguyên nhân kiến trúc GUI cũ khó tối ưu desktop

Fyne đang gánh cả widget/layout/theme lẫn native window lifecycle. Source cũ đã
phải thêm lớp Windows-specific:

- `window_size*` gọi Win32 work area/DPI để đo kích thước trước show;
- `window_startup_windows.go` chạm trực tiếp GLFW `WindowHint(Maximized)`;
- `window.go` phải phối hợp Fyne logical size, monitor, taskbar và timing của
  native HWND.

Đó là dấu hiệu presentation framework đang buộc project xử lý chi tiết native
window ngoài abstraction của chính nó. Với Qt, `QMainWindow`/window manager xử lý
DPI, frame, taskbar/work area và native dialogs trực tiếp. Bản refactor tạo
window khi còn hidden, chờ backend `ready`, set `Qt::WindowMaximized`, rồi mới
`show()`, vì vậy first visible frame đã ở đúng work area.

## 4. Kiến trúc mới

```text
NoteHub.exe (C++20 + Qt Widgets)
        │
        │ JSON request/response, stdin/stdout
        │ QProcess
        ▼
notehub-backend.exe (Go)
        │
        ├─ domain
        ├─ service
        ├─ repository → SQLite
        ├─ storage → filesystem
        ├─ backup
        └─ share
```

### Vì sao chọn stdio IPC thay TCP/gRPC/c-shared

- Không mở port cho GUI↔backend, không dính firewall.
- Không cần protobuf/toolchain chỉ để gọi local service.
- Không cần CGO/C ABI và không phải quản lý ownership Go/C++ trong cùng process.
- `QProcess` sở hữu child process tự nhiên; frontend có thể phát hiện crash và
  shutdown child có kiểm soát.
- JSON đủ cho workload note app; attachment bytes vẫn đi thẳng filesystem qua
  Go backend, không base64 qua IPC.

stdout của backend được dành riêng cho protocol; log đi stderr. Protocol có
version để tránh chạy nhầm GUI/backend khác đời.

## 5. Mapping tính năng Fyne → Qt

| Chức năng | Bản Qt |
|---|---|
| Main shell / rails | `QMainWindow`, left rail responsive, right quick filter |
| Timeline | `MemoCard` + keyset paging |
| Composer | `QTextEdit`, attachment staging, tag insert |
| Markdown editor | `MemoDialog`, edit/preview tabs |
| Image/file preview | Qt pixmap/file tiles; mở bằng `QDesktopServices` |
| Calendar | `QCalendarWidget`, month summary, marked dates, date timeline |
| Search | FTS text + tag + inclusive local date inputs |
| Tags | `QListWidget` → timeline filter |
| Attachment manager | `QTreeWidget`, open/folder/note/delete |
| Backup | native `QFileDialog`, Skip/Replace/Duplicate |
| Share | local server, create/list/revoke token, one-time link/token + clipboard |
| Theme/font | `QSettings`, System/Light/Dark, font family/size |
| Window startup | maximize before first visible frame |

## 6. Compatibility

Không thay schema/migrations v1→v3, UID, backup format hay attachment directory.
Data directory vẫn là cùng quy ước `NoteHub`, nên database Fyne hiện tại có thể
được Go backend mới mở trực tiếp. Preferences giao diện không migrate vì Fyne
Preferences và QSettings là hai hệ khác nhau; đây không phải dữ liệu người dùng.

## 7. Validation đã thực hiện trong môi trường tạo source

- `gofmt` toàn bộ Go source.
- `GOPROXY=off go test -tags teststub ./...`: PASS cho các package không cần
  tải SQLite driver.
- Normal `go test ./...` chưa thể chạy trong sandbox vì module cache không có
  `modernc.org/sqlite v1.34.5` và network module download bị tắt.
- CMake parse được đến `find_package(Qt6 ...)`; sandbox không cài Qt 6 SDK nên
  không thể link/build GUI tại đây.
- GitHub Actions trong project mới build Go + Qt trên Windows/macOS/Linux để khi
  push lên repository, compiler thật sẽ kiểm tra frontend trên cả ba OS.

Trên máy Windows có Qt SDK, dùng `scripts/build-windows.ps1` để build và chạy
`windeployqt`, tạo `dist/windows/NoteHub.exe` cùng `notehub-backend.exe`.
