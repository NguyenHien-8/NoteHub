# Backend NoteHub – phân tích và chọn lọc thuật toán từ Memos

## Phạm vi đã đọc

Bản Memos được cung cấp trong `memos-main.zip` đã được đối chiếu tập trung vào
những phần có giá trị trực tiếp cho NoteHub desktop:

- `store/db/sqlite/sqlite.go`
- `store/db/sqlite/memo.go`
- `store/memo.go`
- `store/db/sqlite/attachment.go`
- `store/attachment.go`
- `store/db/sqlite/memo_share.go`
- `store/memo_share.go`
- `markdown/markdown.go`
- `core/memoexport/{format,path,validate,reader,writer}.go`
- `server/api/v1/user_service_memo_export.go`
- `server/api/v1/user_service_memo_import.go`

Không copy ngẫu nhiên toàn bộ Memos. Mỗi phần dưới đây chỉ được giữ khi phù hợp
với mục tiêu NoteHub: ứng dụng desktop cá nhân, Go + SQLite, timeline, file local.

## Thuật toán/ý tưởng được giữ và lý do

### 1. SQLite WAL + busy timeout + giao dịch ghi IMMEDIATE

Từ cách Memos cấu hình SQLite, NoteHub giữ các ý tưởng:

- `journal_mode(WAL)` để đọc và ghi ít khóa nhau hơn.
- `busy_timeout(10000)` để chờ writer khác thay vì lỗi `SQLITE_BUSY` ngay.
- `_txlock=immediate` để transaction dự kiến ghi lấy quyền ghi từ đầu, tránh lỗi
  nâng cấp read transaction -> write transaction.
- `mmap_size(0)` để tránh các vấn đề memory mapping không cần thiết trên desktop.

NoteHub **bật `foreign_keys(ON)`**, khác Memos hiện tại, vì schema NoteHub nhỏ và
được thiết kế có cascade rõ ràng.

### 2. Timeline có thứ tự ổn định

Memos luôn dùng `id` làm tie-breaker sau timestamp. NoteHub giữ nguyên nguyên
tắc này:

```sql
ORDER BY created_ts DESC, id DESC
```

Ngoài ra NoteHub nâng thành **keyset pagination** thay vì phụ thuộc OFFSET cho
timeline:

```sql
created_ts < cursorTime
OR (created_ts = cursorTime AND id < cursorID)
```

Điều này ổn định hơn khi timeline lớn hoặc có memo mới được chèn trong lúc cuộn.

### 3. Tag phân cấp

Memos mở rộng:

```text
#Research/FPGA
```

thành:

```text
Research
Research/FPGA
```

NoteHub giữ hành vi này, deduplicate không phân biệt hoa thường nhưng giữ cách
viết đầu tiên. Parser NoteHub bỏ tag nằm trong fenced code, inline code, Markdown
link/image và hashtag đã escape.

Không copy nguyên `markdown` package của Memos vì package đó phụ thuộc Goldmark
extensions riêng, payload protobuf và nhiều logic web/server ngoài phạm vi
NoteHub. NoteHub dùng parser nhỏ, độc lập và có test riêng.

### 4. Attachment: metadata tách khỏi bytes

Giữ mô hình Memos:

- SQLite chỉ giữ metadata.
- File thật nằm trong filesystem.
- Mỗi file có UID, size và SHA-256.
- Xóa memo cascade metadata attachment.

Bổ sung cho desktop:

- upload vào file tạm -> `fsync` -> atomic rename;
- giới hạn kích thước mỗi file;
- path containment chống `../`;
- DB insert lỗi thì file vừa tạo được cleanup;
- khi DB delete đã commit, xóa file theo best-effort để không biến lỗi cleanup
  thành một lần retry destructive operation.

### 5. Search

Memos có CEL/filter engine rất mạnh, phù hợp server nhiều người dùng. Với NoteHub
nó quá nặng. NoteHub chỉ giữ tư tưởng query có filter tag/date và thay phần text
search bằng SQLite **FTS5 `unicode61`**.

Lợi ích:

- tìm kiếm Unicode tốt hơn `LOWER/NOCASE` mặc định của SQLite;
- `remove_diacritics 2` hỗ trợ tìm `ghi chu` ra `ghi chú`;
- truy vấn được quote thành literal terms, không đưa thẳng cú pháp FTS do user
  nhập vào.

### 6. Chia sẻ read-only bằng bearer token

Memos dùng một share UID để cấp read-only access. NoteHub giữ:

- share riêng cho từng memo;
- có thời hạn hoặc không hết hạn;
- HTTP endpoint read-only;
- attachment chỉ mở sau khi token được xác thực lại.

NoteHub tăng an toàn bằng cách **không lưu token plaintext**. SQLite chỉ lưu
`SHA-256(token)`; token thật chỉ trả về khi tạo share.

### 7. Export / Import an toàn

Đây là phần được chọn lọc nhiều nhất từ `core/memoexport` của Memos:

- manifest có format/version/generator/counts;
- memo content và record tách thành entry riêng;
- attachment có SHA-256 + size;
- reject ZIP absolute path, `..`, backslash, colon, control characters;
- reject duplicate entries;
- chỉ nhận Store/Deflate, không nhận encrypted ZIP;
- giới hạn số entry;
- giới hạn từng JSON/content entry;
- kiểm tra expansion ratio chống ZIP/decompression bomb;
- filename được làm an toàn cho cả Windows/macOS/Linux;
- kiểm tra SHA-256 attachment khi import;
- 3 conflict policy: **Skip / Replace / Duplicate**.

Import NoteHub xử lý theo từng memo:

1. đọc và validate archive trước;
2. stream attachment từ ZIP, không cần load media lớn toàn bộ vào RAM;
3. ghi file mới với UID mới;
4. kiểm tra size + SHA-256;
5. transaction SQLite ghi memo + tags + attachment metadata;
6. nếu transaction lỗi, xóa các file mới;
7. Replace chỉ xóa file cũ sau khi DB transaction đã commit.

Cách này ưu tiên **không mất dữ liệu** khi xảy ra lỗi giữa filesystem và SQLite.

### 8. Optimistic revision

NoteHub thêm `revision` vào memo. Khi edit:

```text
UPDATE ... WHERE id=? AND revision=?
```

Nếu GUI đang sửa một bản cũ, backend trả `domain.ErrConflict` thay vì âm thầm
ghi đè thay đổi mới hơn. Đây là bổ sung desktop quan trọng khi sau này có
autosave/nhiều cửa sổ.

## Những phần Memos đã chủ động KHÔNG lấy

- React / TypeScript / CSS / web frontend.
- Connect/gRPC/protobuf API.
- User/role/SSO/OAuth.
- Spaces và membership policy.
- Reactions, inbox, comments, relation graph.
- S3/external storage registry.
- AI/LLM features.
- CEL filter engine đầy đủ.
- Map/location.
- Webhook/email.
- MySQL/PostgreSQL drivers.

Lý do: những phần này làm dependency graph lớn và không phục vụ bản NoteHub
desktop cá nhân hiện tại.

## Cấu trúc backend sau khi tích hợp

```text
internal/
├── app/                 # composition root backend
├── domain/              # model + typed errors
├── identity/            # UID + bearer token
├── service/             # business logic
├── repository/          # interfaces
│   └── sqlite/          # SQLite + FTS5 + migrations
├── storage/             # attachment local filesystem
├── tagparse/            # hashtag parser
├── backup/              # validated ZIP export/import format
├── share/               # optional read-only HTTP share server
└── platform/            # Windows/macOS/Linux paths/open external
```

GUI Fyne chưa được triển khai trong thay đổi backend này.
