# NoteHub – Project Structure đề xuất

## Mục tiêu

NoteHub là phần mềm desktop ghi chú nhanh theo dạng timeline, chạy trên:

- Windows
- macOS
- Linux

Stack đề xuất:

- **Go**: toàn bộ ứng dụng và business logic.
- **Fyne v2**: GUI desktop thuần Go.
- **SQLite**: database local.
- **Filesystem**: lưu file đính kèm.
- Không dùng React, TypeScript, HTML hoặc CSS.

## Kiến trúc

```text
UI (Fyne)
   ↓
Service
   ↓
Repository / Storage
   ↓
SQLite + Filesystem
```

Nguyên tắc quan trọng: **UI không truy cập SQLite trực tiếp**.

## Cấu trúc thư mục chuẩn

```text
NoteHub/
├── cmd/
│   └── notehub/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── startup.go
│   │   └── shutdown.go
│   ├── domain/
│   │   ├── memo.go
│   │   ├── attachment.go
│   │   ├── tag.go
│   │   ├── calendar.go
│   │   └── share.go
│   ├── service/
│   │   ├── memo_service.go
│   │   ├── attachment_service.go
│   │   ├── timeline_service.go
│   │   ├── calendar_service.go
│   │   ├── tag_service.go
│   │   ├── search_service.go
│   │   ├── share_service.go
│   │   └── backup_service.go
│   ├── repository/
│   │   ├── memo_repository.go
│   │   ├── attachment_repository.go
│   │   ├── tag_repository.go
│   │   ├── share_repository.go
│   │   └── sqlite/
│   │       ├── database.go
│   │       ├── schema.go
│   │       ├── migration.go
│   │       ├── memo_repository.go
│   │       ├── attachment_repository.go
│   │       ├── tag_repository.go
│   │       └── share_repository.go
│   ├── storage/
│   │   ├── attachment_store.go
│   │   ├── paths.go
│   │   └── checksum.go
│   ├── backup/
│   │   ├── export.go
│   │   ├── import.go
│   │   ├── manifest.go
│   │   └── safezip.go
│   ├── share/
│   │   ├── server.go
│   │   └── handler.go
│   ├── tagparse/
│   │   ├── tags.go
│   │   └── tags_test.go
│   ├── platform/
│   │   ├── paths.go
│   │   ├── paths_windows.go
│   │   ├── paths_darwin.go
│   │   ├── paths_linux.go
│   │   ├── opener.go
│   │   ├── opener_windows.go
│   │   ├── opener_darwin.go
│   │   └── opener_linux.go
│   └── ui/
│       ├── window.go
│       ├── navigation.go
│       ├── theme.go
│       ├── rails.go
│       ├── typography/
│       │   └── fonts.go
│       ├── screens/
│       │   ├── timeline.go
│       │   ├── calendar.go
│       │   ├── search.go
│       │   ├── attachments.go
│       │   └── settings.go
│       ├── components/
│       │   ├── sidebar.go
│       │   ├── memo_editor.go
│       │   ├── memo_card.go
│       │   ├── timeline_list.go
│       │   ├── tag_list.go
│       │   ├── search_bar.go
│       │   ├── attachment_card.go
│       │   ├── image_gallery.go
│       │   ├── surface.go
│       │   └── calendar_widget.go
│       └── dialogs/
│           ├── attachment.go
│           ├── share.go
│           ├── export.go
│           ├── import.go
│           └── confirm.go
├── assets/
│   ├── icons/
│   └── images/
├── packaging/
│   ├── windows/
│   ├── macos/
│   └── linux/
├── scripts/
│   ├── build-windows.ps1
│   ├── build-macos.sh
│   └── build-linux.sh
├── .github/
│   └── workflows/
│       ├── test.yml
│       └── release.yml
├── docs/
│   ├── ARCHITECTURE.md
│   ├── DATABASE.md
│   └── BUILD.md
├── tests/
│   └── integration/
├── go.mod
├── go.sum
├── LICENSE
├── NOTICE.md
├── README.md
└── .gitignore

```

## Trách nhiệm từng tầng

### `cmd/notehub`
Entry point duy nhất của chương trình. `main.go` chỉ khởi tạo application và gọi `Run()`.

### `internal/app`
Composition root: xác định AppData, mở SQLite, chạy migration, khởi tạo repository/service/UI, quản lý startup và shutdown.

### `internal/domain`
Các model thuần nghiệp vụ: Memo, Attachment, Tag, Calendar, Share. Không phụ thuộc GUI, SQLite hay HTTP.

### `internal/service`
Business logic chính của NoteHub:
- Memo CRUD
- Attachment
- Timeline
- Calendar
- Tags
- Search
- Share
- Import/Export

### `internal/repository`
Chứa interface persistence. GUI và service chỉ phụ thuộc interface.

### `internal/repository/sqlite`
SQLite implementation, schema và migration.

### `internal/storage`
Quản lý file attachment trên filesystem, checksum, đường dẫn an toàn.

### `internal/backup`
Export/Import ZIP, manifest version và chống ZIP path traversal.

### `internal/share`
HTTP server chia sẻ ghi chú. Chỉ bật khi người dùng kích hoạt Share.

### `internal/tagparse`
Parser cho `#tag` và tag phân cấp như `#project/fpga`.

### `internal/platform`
Tách code đặc thù theo Windows/macOS/Linux bằng Go build tags.

### `internal/ui`
GUI Fyne. Chia rõ `screens`, `components`, `dialogs`. `rails.go` quản lý bố cục
3 vùng responsive, kéo thay đổi độ rộng và collapse sidebar mà không refresh
lại toàn bộ cây widget trong lúc kéo. `components/surface.go` tạo panel bo góc:
sidebar dùng nền phủ nhẹ, workspace dùng nền trung tính và khoảng trống giữa các
panel thay cho vạch phân cách cố định.

`components/image_gallery.go` hiển thị ảnh theo đúng tỉ lệ (`ImageFillContain`),
hỗ trợ kéo đổi thứ tự và xóa ảnh trong editor. Grid ảnh 2 cột co theo workspace
nhưng giới hạn tối đa 960 logical px và tự căn giữa khi hai sidebar được thu gọn,
tránh kéo tile quá rộng. `screens/previews.go` giải mã ảnh ngoài UI thread, giới
hạn ảnh nguồn 12 MP, tạo preview tối đa 640 × 420 và giữ cache hữu hạn để tránh
tăng RAM không kiểm soát.

`ui/typography/fonts.go` phát hiện các font phổ biến đã cài trên Windows/macOS/
Linux và tải lazy bằng Fyne resource. Font không tồn tại tự fallback về System;
NoteHub không đóng gói hoặc phân phối file font. Text size cho phép 10–32 logical
px và tiếp tục tôn trọng DPI scale của hệ điều hành.

## Dữ liệu người dùng

Không lưu database cạnh file thực thi.

Windows:
```text
%LOCALAPPDATA%\NoteHub\
├── notehub.db
└── attachments\
```

macOS:
```text
~/Library/Application Support/NoteHub/
├── notehub.db
└── attachments/
```

Linux:
```text
~/.local/share/NoteHub/
├── notehub.db
└── attachments/
```

## Quy tắc phụ thuộc

```text
ui → service → repository interface
                    ↓
               sqlite implementation

service → storage
service → backup
service → share
```

Không cho phép:
```text
ui → sqlite
domain → fyne
domain → sqlite
repository → ui
```

## Mapping từ source NoteHub hiện tại

| Source hiện tại | Vị trí mới |
|---|---|
| `backend/model.go` | `internal/domain/` |
| `backend/memo.go` | `internal/service/memo_service.go` + `internal/repository/sqlite/memo_repository.go` |
| `backend/attachment.go` | `internal/service/attachment_service.go` + `internal/storage/` + repository |
| `backend/calendar.go` | `internal/service/calendar_service.go` |
| `backend/tag.go` | `internal/service/tag_service.go` |
| `backend/search.go` | `internal/service/search_service.go` |
| `backend/share.go` | `internal/service/share_service.go` |
| `backend/share_http.go` | `internal/share/` |
| `backend/export.go` | `internal/backup/export.go` |
| `backend/import.go` | `internal/backup/import.go` |
| `internal/safezip/` | `internal/backup/safezip.go` |
| `internal/tagparse/` | giữ trong `internal/tagparse/` |
| `backend/schema.go` | `internal/repository/sqlite/schema.go` |
| `backend/sqlite_driver.go` | `internal/repository/sqlite/database.go` |
| `backend/os_helpers.go` | `internal/platform/` |

## Module path

Nên đổi `go.mod` sang:

```go
module github.com/NguyenHien-8/NoteHub
```

## Release

Một source code, build native theo từng hệ điều hành:

```text
Git tag vX.Y.Z
      │
      ├── Windows runner → NoteHub.exe / installer
      ├── macOS runner   → NoteHub.app / DMG
      └── Linux runner   → binary / AppImage / DEB
```

## Kết luận

Đây là cấu trúc nên dùng làm kiến trúc chuẩn cho NoteHub từ giai đoạn hiện tại. Nó giữ backend độc lập với GUI, hỗ trợ mở rộng lâu dài và tránh phải viết lại business logic nếu sau này thay đổi framework GUI.
