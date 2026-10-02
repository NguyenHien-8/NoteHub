# Prompt Build Project NoteHub Desktop

Bạn là Senior Go Desktop Engineer và Software Architect.

Nhiệm vụ của bạn là đọc, phân tích, chỉnh sửa và hoàn thiện Project NoteHub hiện tại thành một phần mềm desktop native chạy trên Windows, macOS và Linux.

============================================================
I. SOURCE CODE VÀ NGUYÊN TẮC LÀM VIỆC
============================================================

TRƯỚC KHI VIẾT CODE:
1. Đọc source code hiện tại.
2. Đọc tối thiểu các khu vực:

cmd/notehub/

internal/app/
internal/domain/
internal/service/
internal/repository/
internal/repository/sqlite/
internal/storage/
internal/tagparse/
internal/backup/
internal/share/
internal/platform/
internal/ui/

docs/ARCHITECTURE.md
docs/BACKEND_FROM_MEMOS.md
docs/BACKEND_TEST_REPORT.md
BACKEND_CHANGE_SUMMARY.md
go.mod

4. Hiểu kiến trúc trước khi sửa.
5. Không được viết lại backend chỉ vì muốn đơn giản hóa.
6. Không được copy bừa source từ Memos.
7. Phải tận dụng các service và repository hiện có của NoteHub.

Backend hiện tại đã được thiết kế:

UI
 ↓
Service
 ↓
Repository / Storage
 ↓
SQLite + Filesystem

GUI tuyệt đối KHÔNG truy cập SQLite trực tiếp.

Mọi thao tác phải đi thông qua:

internal/app.Backend

và:

MemoService
AttachmentService
TimelineService
CalendarService
TagService
SearchService
ShareService
BackupService

============================================================
II. CÔNG NGHỆ BẮT BUỘC
============================================================

Phần mềm desktop sử dụng:

Go
Fyne v2
SQLite

Không sử dụng:

React
TypeScript
JavaScript
HTML
CSS
Electron
Tauri
Wails

Toàn bộ giao diện phải được viết bằng Go/Fyne.

Database:

modernc.org/sqlite

Giữ SQLite hiện tại.

Không thay SQLite bằng MySQL hoặc PostgreSQL.

Ứng dụng phải chạy native trên:

Windows
macOS
Linux

============================================================
III. MỤC TIÊU SẢN PHẨM
============================================================

NoteHub là phần mềm ghi chú nhanh theo dạng timeline.

Triết lý sử dụng:

Ghi nhanh
   ↓
Timeline
   ↓
Tag / Calendar / Search
   ↓
Tìm lại nội dung sau này

Không thiết kế NoteHub thành ứng dụng chat.

Không biến giao diện thành Telegram/Discord/Messenger.

Phong cách sử dụng phải gần Memos:

- tạo ghi chú nhanh;
- timeline;
- tag;
- calendar;
- search;
- attachment;
- thao tác rất ít bước.

============================================================
IV. GIAO DIỆN THAM CHIẾU
============================================================

Hình ảnh tôi gửi là UI reference chính.

Giao diện hoàn thiện phải giống hình ảnh này về:

- bố cục;
- tỷ lệ;
- khoảng trắng;
- sidebar;
- card;
- typography;
- màu sắc;
- kích thước vùng nội dung;
- cách hiển thị timeline.

Không cần pixel-perfect 100%, nhưng phải tạo cảm giác cùng một sản phẩm.

Không tự thiết kế một giao diện hoàn toàn khác.

============================================================
V. BỐ CỤC CỬA SỔ
============================================================

Giao diện chia thành 3 cột:

┌───────────────┬────────────────────────────────┬──────────────────┐
│               │                                │                  │
│ Left Sidebar  │        Main Content            │ Right Sidebar    │
│               │                                │                  │
│ khoảng 250 px │         flexible               │ khoảng 280 px    │
│               │                                │                  │
└───────────────┴────────────────────────────────┴──────────────────┘

Kích thước mặc định gần:

1450 x 900

Minimum window:

1100 x 700

Main Content phải co giãn theo cửa sổ.

Không hard-code toàn bộ giao diện theo một độ phân giải duy nhất.

Khi cửa sổ hẹp:
- ưu tiên giữ Sidebar trái;
- Right Sidebar có thể collapse/hide;
- Main Content tiếp tục sử dụng bình thường.

============================================================
VI. HEADER
============================================================

Phía trên:

Logo NoteHub
+
NoteHub

Sử dụng logo:

assets/icons/NoteHub.png

Ở vùng giữa/phải có Global Search:

[Tìm kiếm ghi chú, tag, nội dung...] [Ctrl + K]

Ctrl+K phải:

- focus vào ô Search;
- nếu đang ở màn hình khác thì có thể chuyển sang Search hoặc mở search overlay.

Không fake các nút minimize/maximize/close nếu framework không quản lý chúng an toàn.

Ưu tiên sử dụng native window chrome của Windows/macOS/Linux.

============================================================
VII. SIDEBAR TRÁI
============================================================

Menu chính phải đúng thứ tự:

Home
Calendar
Search
Attachments
Tags
Settings

LƯU Ý:

Không dùng chữ "Timeline" trong Sidebar.

Phải dùng:

Home

Home thực chất hiển thị timeline.

Selected item:

- nền xanh rất nhạt;
- icon xanh;
- text xanh;
- rounded rectangle.

Màu gợi ý:

Primary blue:
#087BFF

Selected background:
#EAF3FF

Text:
#0F172A

Secondary text:
#64748B

Border:
#E5EAF0

Background:
#F8FAFC

============================================================
VIII. MY TAGS
============================================================

Dưới Sidebar:

--------------------
My Tags           +
--------------------

Ví dụ:

● #FPGA
● #ESP32
● #Research
● #Study
● #Project/NoteHub

Dữ liệu thật phải lấy từ:

backend.Tags.List(ctx)

Không hard-code tag demo.

Số lượng và nội dung tag phải cập nhật sau khi:

- tạo memo;
- sửa memo;
- xóa memo.

Màu tag nên sinh deterministic từ tên tag.

Ví dụ cùng "#FPGA" thì mỗi lần mở app vẫn cùng màu.

============================================================
IX. HOME / TIMELINE
============================================================

Home là màn hình quan trọng nhất.

Phía trên timeline là Memo Composer.

Thiết kế:

┌─────────────────────────────────────────────┐
│ Có suy nghĩ gì...                           │
│                                             │
│                                             │
│  📎 Đính kèm     🏷 Thêm tag        Save   │
└─────────────────────────────────────────────┘

Sử dụng MultiLineEntry.

Save:

backend.Memos.Create(...)

Khi tạo thành công:

- clear editor;
- clear attachment staging;
- refresh timeline;
- refresh tags;
- refresh Calendar;
- refresh counts.

Không restart ứng dụng.

============================================================
X. TAG TRONG EDITOR
============================================================

NoteHub backend hiện parse tag từ nội dung memo.

Vì vậy nút:

Thêm tag

không tạo database tag riêng.

Khi user chọn tag:

#FPGA

hãy chèn:

#FPGA

vào nội dung memo đúng vị trí phù hợp.

Backend tiếp tục dùng tagparse.Extract() hiện tại.

Không tạo một hệ thống tag thứ hai song song.

============================================================
XI. ATTACHMENT
============================================================

Nút:

Đính kèm

mở native file chooser.

Sau khi chọn file:

- hiển thị attachment staging trong editor;
- chỉ lưu attachment sau khi memo được tạo;
- gọi backend.Attachments.AddFromFile().

Nếu memo tạo thành công nhưng attachment upload có lỗi:

- memo vẫn tồn tại;
- báo attachment nào thất bại;
- không crash ứng dụng.

Attachment card:

IMAGE:
hiển thị thumbnail.

PDF:
hiển thị PDF icon + filename + size.

Các file khác:
generic file icon + filename + size.

Double click hoặc Open:

sử dụng:

platform.OpenExternal(...)

Không tự viết PDF viewer/video player trong giai đoạn đầu.

============================================================
XII. MEMO CARD
============================================================

Memo card giống hình tham chiếu:

┌──────────────────────────────────────────────┐
│ [thumbnail]  Kiểm tra FPGA Cyclone IV       │
│              Nội dung ghi chú...       14:32 │
│              #FPGA                     ...   │
└──────────────────────────────────────────────┘

Không thêm trường "title" vào database chỉ để giống screenshot.

NoteHub hiện chỉ có:

Memo.Content

Hãy derive UI title:

- dòng không rỗng đầu tiên = title hiển thị;
- các dòng còn lại = preview body.

Nếu memo chỉ có một dòng:
hiển thị chính dòng đó.

Không làm thay đổi dữ liệu gốc.

============================================================
XIII. TIMELINE GROUPING
============================================================

Timeline sử dụng:

backend.Timeline.List()

Không tự viết SELECT trong UI.

Phải sử dụng keyset pagination hiện có.

Nhóm memo thành:

Today
Yesterday
Earlier

theo timezone local của user.

Ví dụ:

Today
Thứ 5, 24 Tháng 4, 2025

Yesterday
Thứ 4, 23 Tháng 4, 2025

Earlier
...

Hỗ trợ:

scroll
load more

Không load toàn bộ database vào RAM khi app mở.

Page đầu khoảng:

30-50 memo.

Khi user scroll gần cuối:
load next page bằng TimelineCursor.

============================================================
XIV. MENU "..."
============================================================

Mỗi memo có menu:

...

Bao gồm:

Edit
Favorite
Share
Delete

Delete phải có confirm dialog.

Edit:

backend.Memos.Get()
backend.Memos.Update()

Phải truyền đúng Revision.

Nếu:

domain.ErrConflict

thì không ghi đè âm thầm.

Hiển thị dialog:

"Ghi chú đã được thay đổi ở nơi khác. Vui lòng tải lại."

============================================================
XV. FAVORITES
============================================================

Reference UI có:

Favorites

nhưng backend hiện tại chưa có Favorite.

Hãy triển khai chức năng này theo cách tối thiểu và sạch.

Không hack local state trong GUI.

Thêm migration mới, KHÔNG sửa migration v1 đã phát hành.

Ví dụ:

migration v2

thêm:

favorite INTEGER NOT NULL DEFAULT 0

vào memo.

Cập nhật:

domain.Memo

repository

SQLite repository

service

để hỗ trợ:

SetFavorite()

FavoriteOnly filter

CountFavorites()

Việc thêm Favorite phải không phá database cũ.

============================================================
XVI. RIGHT SIDEBAR
============================================================

Right Sidebar chỉ gồm:

1. Mini Calendar
2. Quick Filters

KHÔNG có:

Recent Tags

Reference image mới nhất đã bỏ Recent Tags.

============================================================
XVII. MINI CALENDAR
============================================================

Calendar panel:

Tháng 4, 2025

<               >

T2 T3 T4 T5 T6 T7 CN

Các ngày có memo:
hiển thị dot xanh nhỏ.

Ngày đang chọn:
circle blue.

Dữ liệu phải lấy từ:

backend.Calendar.Month()

Khi click ngày:

backend.Calendar.MemosForDate()

và timeline Home chỉ hiển thị memo ngày đó.

Có nút clear để quay lại toàn timeline.

============================================================
XVIII. QUICK FILTERS
============================================================

Khung:

Quick Filters

All Notes       count
Favorites       count
Shared          count

All Notes:
toàn bộ timeline.

Favorites:
memo Favorite=true.

Shared:
memo đang có active share.

Không hard-code:

24
3
1

Các số phải lấy từ database/service.

Nếu backend thiếu API count/filter cần thiết:
hãy mở rộng repository/service tối thiểu.

Không query SQL từ UI.

Shared phải tính active share:
- chưa hết hạn;
- hoặc không có expiration.

============================================================
XIX. SEARCH
============================================================

Search phải dùng:

backend.Search.Search()

Không viết search engine mới.

Backend hiện đã dùng:

SQLite FTS5
unicode61
remove_diacritics 2

Do đó phải giữ khả năng:

"ghi chu"

tìm được:

"Ghi chú"

Search UI hỗ trợ:

- text;
- tag;
- date from;
- date to.

Global Search nên debounce khoảng:

250-350 ms

để tránh query mỗi phím quá mức.

Không block UI thread trong lúc search.

============================================================
XX. CALENDAR SCREEN
============================================================

Menu Calendar mở màn hình Calendar đầy đủ.

Hiển thị:

Month View

Ngày nào có memo:
dot/count.

Click ngày:
hiển thị memo ngày đó.

Sử dụng lại CalendarService.

Không tạo logic date query riêng trong GUI.

============================================================
XXI. ATTACHMENTS SCREEN
============================================================

Hiển thị tất cả attachment theo grid/list:

- image;
- PDF;
- file khác.

Cho phép:

Open
Reveal/Open
Delete
Go to Memo

Nếu repository hiện chưa có:

ListAllAttachments()

hãy bổ sung repository/service API nhỏ, không query SQL trực tiếp trong UI.

============================================================
XXII. TAGS SCREEN
============================================================

Tags screen:

#FPGA       12
#ESP32       8
#Study       5

Sử dụng:

backend.Tags.List()

Click tag:
mở Home và filter theo tag.

Hỗ trợ tag phân cấp:

#Research
#Research/FPGA

Giữ thuật toán tag hiện tại.

Không rewrite tag parser.

============================================================
XXIII. SHARE
============================================================

Share sử dụng backend hiện tại:

backend.Shares.Create()
backend.Shares.List()
backend.Shares.Revoke()

Cho phép chọn:

Never expires
1 day
7 days
30 days

Token thật chỉ hiển thị khi vừa tạo.

Không thay đổi cơ chế backend đang lưu SHA-256(token).

Nếu cần HTTP server để share:
sử dụng internal/share hiện tại.

Server share chỉ chạy khi user bật sharing.

Không tự động mở HTTP port khi app khởi động.

============================================================
XXIV. SETTINGS
============================================================

Settings gồm tối thiểu:

Appearance
- System
- Light
- Dark

Data
- Data directory
- Open data folder

Backup
- Export Backup
- Import Backup

Sharing
- Enable/Disable local share server
- Port

About
- NoteHub version

Export:

backend.Backup.ExportFile()

Import:

backend.Backup.ImportFile()

Hiển thị policy:

Skip
Replace
Duplicate

Import hoàn tất:
refresh toàn bộ UI.

============================================================
XXV. BACKEND PHẢI ĐƯỢC GIỮ
============================================================

Không được phá các thuật toán hiện tại:

SQLite:
- WAL
- busy_timeout
- foreign_keys
- IMMEDIATE transaction

Timeline:
- created_ts DESC
- id DESC
- keyset pagination

Search:
- FTS5
- unicode61
- remove_diacritics 2

Memo:
- optimistic revision

Attachment:
- local filesystem
- SHA-256
- temp → fsync → rename
- path containment

Share:
- token hash
- expiration

Backup:
- format version
- checksum
- path traversal protection
- ZIP bomb protection
- Skip / Replace / Duplicate

Đây là các thuật toán đã chọn lọc từ Memos.

KHÔNG rewrite chúng nếu không có lỗi thực tế.

============================================================
XXVI. GUI ARCHITECTURE
============================================================

Giữ structure:

internal/ui/
│
├── window.go
├── navigation.go
├── theme.go
│
├── screens/
│   ├── timeline.go
│   ├── calendar.go
│   ├── search.go
│   ├── attachments.go
│   └── settings.go
│
├── components/
│   ├── sidebar.go
│   ├── memo_editor.go
│   ├── memo_card.go
│   ├── timeline_list.go
│   ├── tag_list.go
│   ├── search_bar.go
│   ├── attachment_card.go
│   └── calendar_widget.go
│
└── dialogs/
    ├── attachment.go
    ├── share.go
    ├── export.go
    ├── import.go
    └── confirm.go

Không gom toàn bộ GUI vào:

main.go

hoặc:

window.go

============================================================
XXVII. APP ENTRY POINT
============================================================

Hoàn thiện:

cmd/notehub/main.go

Luồng startup:

main
 ↓
Fyne App
 ↓
app.OpenBackend()
 ↓
ui.NewWindow(...)
 ↓
window.ShowAndRun()

Khi application đóng:

backend.Close()

phải được gọi.

Nếu OpenBackend lỗi:
hiển thị error rõ ràng và thoát an toàn.

============================================================
XXVIII. ASSETS
============================================================

Logo:

assets/icons/NoteHub.png

Hãy embed asset vào binary bằng:

go:embed

nếu phù hợp.

Không phụ thuộc đường dẫn working-directory để load logo.

Khi user chạy:

NoteHub.exe

ở bất kỳ thư mục nào, logo vẫn phải hoạt động.

============================================================
XXIX. UI PERFORMANCE
============================================================

Không chạy database operation nặng trên UI thread.

Dùng goroutine cho:

- search;
- load timeline;
- import;
- export;
- attachment hashing/copy;
- calendar query lớn.

Mọi thay đổi widget phải marshal an toàn trở lại Fyne UI thread theo API Fyne tương ứng.

Phải có:

loading state

và:

error feedback

nhưng giao diện không được rối.

============================================================
XXX. STYLE
============================================================

Thiết kế Light Theme giống reference.

Phong cách:

modern
minimal
clean
Windows 11 inspired
soft rounded cards
thin border
very subtle shadow
ample whitespace

Không làm:

Material Design quá nặng
gradient khắp giao diện
glassmorphism quá mức
sidebar tối
card quá nhiều màu

Logo mới là nơi gradient nổi bật nhất.

Tag có màu nhưng pastel.

============================================================
XXXI. KHOẢNG CÁCH GỢI Ý
============================================================

Sidebar:
240-260 px

Right Sidebar:
270-300 px

Outer padding:
16-20 px

Card padding:
12-16 px

Card corner radius:
10-14 px

Gap:
8 / 12 / 16 px

Primary Save button:
height khoảng 40 px

============================================================
XXXII. INTERNATIONALIZATION
============================================================

UI ban đầu ưu tiên English giống hình:

Home
Calendar
Search
Attachments
Tags
Settings
My Tags
Today
Yesterday
Earlier
Quick Filters
All Notes
Favorites
Shared
Save

Nhưng nội dung memo hỗ trợ Unicode đầy đủ:

Tiếng Việt
English
ký hiệu kỹ thuật

Date có thể hiển thị theo locale hệ điều hành.

============================================================
XXXIII. KHÔNG ĐƯỢC LÀM
============================================================

Không:

- xóa backend hiện tại;
- đổi database nếu không cần;
- copy nguyên Memos;
- đưa SQL vào UI;
- dùng React/TypeScript/CSS;
- tạo mock data cho ứng dụng thật;
- hard-code count;
- hard-code tags;
- hard-code calendar;
- tạo nút nhưng không hoạt động;
- sửa thuật toán backup an toàn mà không có lý do;
- bỏ test hiện tại;
- xóa documentation hiện tại;
- làm lại toàn bộ project từ đầu.

Chỉ sửa những phần liên quan.

============================================================
XXXIV. TEST
============================================================

Sau khi code:

gofmt toàn bộ file Go thay đổi.

Chạy:

go mod tidy
go test ./...
go vet ./...

Sau đó build:

go build ./cmd/notehub

Không báo thành công nếu chưa kiểm tra compile.

Bổ sung test cho backend extension:

Favorite migration
Favorite filter
Favorite count
Shared filter/count
ListAllAttachments nếu thêm.

Các test backend hiện tại phải tiếp tục PASS.

============================================================
XXXV. CROSS-PLATFORM
============================================================

Project phải build trên:

Windows
macOS
Linux

Không dùng Windows-only API bên ngoài:

internal/platform/*

Platform specific logic phải có build tags:

//go:build windows
//go:build darwin
//go:build linux

GitHub Actions phải thực sự chạy:

Windows runner
macOS runner
Ubuntu runner

Không để workflow chỉ còn comment placeholder.

Linux runner phải cài dependencies Fyne cần thiết trước khi build.

============================================================
XXXVI. BUILD / RELEASE
============================================================

Mục tiêu cuối:

Windows:
NoteHub.exe
và sau này NoteHub-Setup.exe

macOS:
NoteHub.app
NoteHub.dmg

Linux:
NoteHub binary
và sau này AppImage / DEB

Không yêu cầu cross-compile macOS từ Windows.

Dùng native GitHub Actions runner cho từng OS.

============================================================
XXXVII. KẾ HOẠCH TRIỂN KHAI
============================================================

Thực hiện theo thứ tự:

PHASE 1
Audit source hiện tại.

PHASE 2
Thêm Fyne dependency và App entry point.

PHASE 3
Xây custom theme và main window.

PHASE 4
Xây Sidebar + Navigation.

PHASE 5
Xây Memo Editor.

PHASE 6
Xây Memo Card + Timeline + pagination.

PHASE 7
Xây tag list/filter.

PHASE 8
Xây mini Calendar + Calendar screen.

PHASE 9
Xây Search.

PHASE 10
Xây Attachment UI.

PHASE 11
Xây Share dialogs.

PHASE 12
Xây Settings + Export/Import.

PHASE 13
Bổ sung Favorites và Quick Filters.

PHASE 14
Responsive layout + polish UI.

PHASE 15
Test backend + GUI compile.

PHASE 16
Cross-platform CI.

Không nhảy thẳng vào viết toàn bộ code trước khi hiểu source.

============================================================
XXXVIII. ACCEPTANCE CRITERIA
============================================================

Project chỉ được xem là hoàn thành khi:

1. NoteHub mở thành cửa sổ desktop thật.
2. Không cần Docker.
3. Không mở browser localhost.
4. Home giống bố cục reference:
   - sidebar trái;
   - editor phía trên;
   - timeline ở giữa;
   - mini calendar bên phải;
   - Quick Filters bên phải.
5. Sidebar hiển thị "Home", không phải "Timeline".
6. Không có Recent Tags panel bên phải.
7. Tạo memo hoạt động.
8. Edit memo hoạt động.
9. Delete hoạt động.
10. Attachment hoạt động.
11. Timeline phân trang hoạt động.
12. Calendar hoạt động.
13. Tags hoạt động.
14. Search hoạt động.
15. Favorite hoạt động.
16. Share hoạt động.
17. Export/Import hoạt động.
18. Dữ liệu tồn tại sau khi đóng/mở app.
19. Database cũ migration được an toàn.
20. go test ./... PASS.
21. go vet ./... PASS.
22. Windows build thành công.
23. macOS/Linux được kiểm tra qua CI.

============================================================
XXXIX. CÁCH TRẢ KẾT QUẢ
============================================================

Không chỉ đưa code snippet.

Hãy chỉnh trực tiếp Project.

Sau khi hoàn thành:

1. Liệt kê file đã sửa.
2. Liệt kê file mới.
3. Giải thích ngắn gọn thay đổi kiến trúc.
4. Báo chính xác các command test/build đã chạy.
5. Báo lỗi hoặc phần chưa test nếu có.
6. Không nói "đã test" nếu thực tế chưa chạy.
7. Không suy đoán kết quả.
8. Đóng gói và gửi lại file ZIP chứa Project NoteHub đã chỉnh sửa.

Tên ZIP:

NoteHub_Desktop_Build.zip

============================================================
XL. ƯU TIÊN
============================================================

Thứ tự ưu tiên:

1. Không làm hỏng dữ liệu.
2. Không phá backend hiện tại.
3. Chức năng hoạt động thật.
4. Kiến trúc sạch.
5. Cross-platform.
6. UI giống reference.
7. Tối ưu hiệu năng.
8. Polish nhỏ.

Nếu có xung đột giữa "giống hình" và "an toàn/cross-platform",
ưu tiên an toàn và cross-platform nhưng giữ bố cục/thẩm mỹ gần reference nhất.

Bắt đầu bằng việc audit toàn bộ source NoteHub hiện tại và mô tả ngắn gọn kế hoạch file-level trước khi sửa code.
Sau đó triển khai lần lượt, test và gửi lại Project hoàn chỉnh.
