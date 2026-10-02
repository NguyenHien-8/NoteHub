# Mapping với mã nguồn Memos đã đọc

Nguồn đầu vào: `memos-main.zip` do người dùng cung cấp.

Các phần upstream quan trọng đã được đọc để hiểu luồng xử lý:

| Chức năng | File/nhóm file Memos tham khảo | Cách backend-lite xử lý |
|---|---|---|
| Memo CRUD / timeline | `store/memo.go`, `store/db/sqlite/memo.go` | SQLite CRUD, order `created_ts DESC, id DESC`, API single-user nhỏ hơn |
| SQLite schema | `store/migration/sqlite/LATEST.sql` | Chỉ giữ memo/tag/attachment/share và bỏ user/space/reaction/inbox |
| Attachment | `store/attachment.go`, `store/db/sqlite/attachment.go`, `store/memo_attachment.go` | File lưu local, DB giữ metadata, SHA-256, FK cascade |
| Tags | `markdown/markdown.go` | Giữ ý tưởng extract tag + hierarchy; dùng parser stdlib nhẹ thay Goldmark/protobuf payload |
| Search/filter | `filter/*`, `store/db/sqlite/memo.go` | Rút gọn thành content/tag/date search; `search_text` được Unicode-lowercase ở Go |
| Share | `store/memo_share.go`, `store/db/sqlite/memo_share.go` | Bearer token read-only + optional expiry |
| Export | `core/memoexport/*`, `server/api/v1/user_service_memo_export.go` | ZIP manifest + memo records + attachment bytes + SHA-256 |
| Import | `server/api/v1/user_service_memo_import.go`, `core/memoexport/reader.go` | Validate ZIP path, validate hashes, skip/overwrite policy |
| Calendar | Memos dùng timestamp của memo và lớp UI/query | Backend-lite thêm API month/day trực tiếp để GUI Go dùng |

## Khác biệt có chủ đích

- Không có React, TypeScript, CSS hay code trong `web/`.
- Không có HTTP/gRPC/protobuf ở lõi desktop.
- Không có multi-user, auth, Spaces, reactions, inbox, map, webhook, AI.
- Không dùng Memos CEL filter engine; desktop API chỉ cung cấp search cần thiết.
- Tags được lưu ở bảng `memo_tag` thay vì payload protobuf để code desktop đơn giản.
- Export format là format riêng `timeline-notes-export/1.0`, không giả vờ tương thích với Memos official export.
- Share token không được export/import để tránh sao chép bearer credential ngoài ý muốn.
