# NoteHub

Đây là **backend thuần Go + SQLite** được tách/rút gọn theo kiến trúc và thuật toán đã đọc từ `https://github.com/usememos/memos`, dành cho một ứng dụng desktop timeline-note không dùng React/TypeScript/CSS.

## Chức năng có sẵn

- Tạo, đọc, chỉnh sửa, xóa ghi chú.
- Timeline theo `created_ts DESC, id DESC`.
- File đính kèm lưu trên local filesystem, metadata trong SQLite, có SHA-256.
- Calendar theo tháng và truy vấn ghi chú theo ngày, tôn trọng timezone desktop.
- Tags dạng `#tag`, hỗ trợ phân cấp `#work/project` => `work`, `work/project`.
- Search nội dung + tag + khoảng thời gian.
- Chia sẻ bằng bearer token có thể hết hạn; backend cung cấp `CreateShare`, `ResolveShare`, `RevokeShare` và HTTP read-only handler thuần Go (`ShareHandler`).
- Export/Import ZIP, gồm memo + attachment; active share token **không được export** vì đây là credential.

## Điều quan trọng

Đây **không phải** là bản Memos upstream đã xóa frontend rồi compile lại. Memos upstream có dependency graph khá lớn (user/space/auth/protobuf/CEL/filter/API/server...). Gói này là một **standalone desktop subset** được viết lại theo các phần backend liên quan trong Memos để có API nhỏ, dễ gắn với Fyne/Gio/Qt binding hoặc một GUI Go khác.

Nó cũng **không tương thích nhị phân** với file export chính thức của Memos. Format export được cố ý đơn giản hóa cho ứng dụng desktop single-user.

## Mapping từ Memos upstream

Xem `UPSTREAM_MAPPING.md` để biết các file Memos đã được đọc và phần nào được giữ lại/đơn giản hóa.

## Cấu trúc

```text
backend/
  service.go       Open/Close SQLite
  schema.go        schema desktop rút gọn
  memo.go          CRUD + timeline
  attachment.go    attachment local
  calendar.go      calendar
  tag.go           tag queries
  search.go        search
  share.go         share token
  export.go        export ZIP
  import.go        import ZIP
internal/tagparse/ parser hashtag thuần Go
internal/safezip/  chống ZIP path traversal
cmd/demo/          demo tối thiểu
```

## SQLite

Project dùng driver pure-Go:

```text
modernc.org/sqlite
```

Vì vậy không cần cài SQLite server, MySQL hay PostgreSQL. Lần build đầu Go cần tải dependency từ Go module proxy (hoặc bạn có thể vendor dependency trong CI/release).

## Ví dụ dùng từ GUI Go

```go
ctx := context.Background()
svc, err := backend.Open(ctx, backend.Config{
    DataDir: `C:\Users\me\AppData\Local\MyTimelineNotes`,
})
if err != nil { panic(err) }
defer svc.Close()

memo, err := svc.CreateMemo(ctx, "Test FPGA #FPGA")

memo, err = svc.UpdateMemo(ctx, memo.ID, "Đã chỉnh sửa #FPGA #done")

items, err := svc.ListTimeline(ctx, backend.TimelineQuery{Limit: 50})

results, err := svc.Search(ctx, backend.SearchQuery{
    Text: "FPGA",
    Tags: []string{"done"},
    Limit: 50,
})
```

Đính kèm file:

```go
att, err := svc.AddAttachmentFromFile(ctx, memo.ID, `D:\Docs\circuit.pdf`)
```

Calendar:

```go
days, err := svc.CalendarMonth(ctx, 2026, time.October, time.Local)
notes, err := svc.MemosForDate(ctx, "2026-10-02", time.Local)
```

Share core:

```go
share, err := svc.CreateShare(ctx, memo.ID, nil)
shared, err := svc.ResolveShare(ctx, share.Token)
```

> `CreateShare` tạo quyền đọc bằng token ở tầng backend. Nếu muốn chia sẻ qua LAN/Internet, dùng `svc.ShareHandler()` với `http.Server`; route `/s/<token>` trả JSON và `/s/<token>/attachments/<uid>` trả file. Hãy chỉ bind ra mạng khi người dùng chủ động bật chia sẻ.

Export/import:

```go
err := svc.Export(ctx, `D:\Backup\notes.zip`)
report, err := svc.Import(ctx, `D:\Backup\notes.zip`, backend.ImportOverwrite)
```

## Build

```bash
go mod tidy
go test ./...
go build ./cmd/demo
```

Trong môi trường không có Internet, có thể type-check phần code thuần stdlib bằng:

```bash
go test ./internal/...
```

File `sqlite_driver_stub.go` chỉ dùng cho kiểm tra offline với build tag `teststub`; **không dùng tag này cho bản release**.

## Gợi ý GUI Go

Nếu bạn muốn tuyệt đối không dùng HTML/CSS/TypeScript, backend này có thể kết hợp với:

- **Fyne**: Go-native API, cross-platform, phù hợp để làm desktop nhanh.
- **Gio**: thuần Go, nhiều quyền kiểm soát rendering hơn nhưng công sức UI cao hơn.

Backend không phụ thuộc framework GUI nên bạn có thể đổi UI sau này mà không phải viết lại database/service.
