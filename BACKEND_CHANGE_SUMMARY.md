# NoteHub backend change summary

## Mục tiêu

Bổ sung backend thật vào Project Structure NoteHub bằng cách chọn lọc các thuật
toán phù hợp từ Memos, không kéo nguyên server/web stack của Memos vào desktop.

## Các nhóm file đã triển khai

- `internal/domain/*`: model thật + typed errors.
- `internal/repository/*`: interface persistence.
- `internal/repository/sqlite/*`: SQLite, migration, WAL, FTS5, memo/tag/file/share repositories.
- `internal/service/*`: Memo, Attachment, Timeline, Calendar, Tags, Search, Share, Backup.
- `internal/storage/*`: local attachment store atomic + SHA-256 + size/path safety.
- `internal/tagparse/*`: hierarchical hashtag parser.
- `internal/backup/*`: versioned export/import ZIP + validation + anti ZIP bomb/path traversal.
- `internal/share/*`: optional read-only HTTP share endpoint/server.
- `internal/identity/*`: UID + bearer token + token hashing.
- `internal/platform/*`: Windows/macOS/Linux app data paths/open external.
- `internal/app/*`: composition root để GUI sau này khởi động backend bằng một lệnh.
- `tests/integration/backend_test.go`: end-to-end test dùng SQLite thật khi dependency khả dụng.

## Thay đổi kiến trúc quan trọng

`internal/safezip` cũ được loại bỏ vì chức năng ZIP safety đã được gom vào
`internal/backup/safezip.go`, tránh hai implementation trùng nhau.

GUI Fyne chưa được triển khai trong gói này; các file `internal/ui` vẫn được giữ
nguyên làm skeleton.

Xem chi tiết thuật toán chọn từ Memos tại `docs/BACKEND_FROM_MEMOS.md`.
