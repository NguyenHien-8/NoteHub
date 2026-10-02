# Backend test report

## Đã chạy

### 1. Go type-check + unit tests không cần driver SQLite

```bash
go test -tags teststub ./...
```

Kết quả: **PASS**.

Bao gồm test:

- hierarchical tag parser;
- tag bỏ qua code/link/image;
- UID/token generation;
- attachment atomic local store + size limit + traversal rejection;
- backup writer/reader round-trip;
- unsafe ZIP path rejection;
- attachment SHA-256 tamper detection.

### 2. Go vet

```bash
go vet -tags teststub ./...
```

Kết quả: **PASS**.

### 3. SQLite schema/FTS behavior

Schema được chạy bằng SQLite runtime có FTS5 để kiểm tra:

- tạo schema thành công;
- FTS trigger insert/update/delete hoạt động;
- tìm `ghi chu` khớp `Ghi chú` với `unicode61 remove_diacritics 2`;
- tag filter kết hợp search hoạt động;
- foreign-key cascade xóa attachment khi memo bị xóa.

Kết quả: **PASS**.

## Test integration thực tế có sẵn

`tests/integration/backend_test.go` kiểm tra end-to-end bằng driver
`modernc.org/sqlite`:

- Memo create/update + revision conflict;
- Tags;
- Attachment;
- Timeline;
- accent-insensitive FTS search;
- Calendar;
- Share token hash;
- Export -> Import -> attachment checksum.

Chạy trên máy có Internet/dependency cache:

```bash
go mod tidy
go test ./...
```

## Giới hạn của môi trường kiểm thử hiện tại

Môi trường tạo artifact không thể kết nối `proxy.golang.org`, nên không tải được
`modernc.org/sqlite v1.34.5`. Vì vậy integration test dùng driver Go thật chưa
thể chạy trong môi trường này. Lệnh download đã được thử và lỗi ở DNS/network,
không phải lỗi compile của source.

`teststub` chỉ dùng cho kiểm tra offline; **không dùng `-tags teststub` khi build
NoteHub thật**.
