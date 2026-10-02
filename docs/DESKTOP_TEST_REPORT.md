# NoteHub Desktop — kết quả triển khai và kiểm thử

Ngày kiểm tra: **02/10/2026**, Windows amd64.

Source được chỉnh trực tiếp trong project NoteHub. Theo yêu cầu mới nhất của
người dùng, không tạo hoặc bàn giao `NoteHub_Desktop_Build.zip`.

## Phạm vi đã triển khai

- Cửa sổ native Go/Fyne: Home, Calendar, Search, Attachments, Tags, Settings;
  bố cục ba cột, logo nhúng, composer, timeline, mini calendar và số liệu thật.
- Tạo/sửa/xóa ghi chú, đính kèm theo staging, xem file bằng ứng dụng hệ thống,
  thumbnail ảnh và thẻ PDF; phản hồi riêng khi lưu ghi chú thành công nhưng một
  file đính kèm thất bại.
- Phân trang keyset 40 ghi chú, tự tải thêm khi cuộn, nhóm theo ngày địa phương,
  lọc tag/ngày/favorite/shared; tìm kiếm Unicode với debounce 300 ms.
- Favorites được lưu trong SQLite bằng migration v2; giữ nguyên schema v1.
  Shared tính ghi chú có ít nhất một share còn hiệu lực, không đếm trùng grant.
- Share có thời hạn/thu hồi; server loopback chỉ bật từ Settings, tắt khi khởi
  động. Token gốc chỉ hiện khi tạo. Export/import giữ các policy hiện có và
  tương thích backup cũ, bổ sung favorite dạng trường tùy chọn.
- Light/Dark/System, Ctrl+K, lưu preferences, đường dẫn dữ liệu tùy chọn,
  lifecycle đóng worker/server/database an toàn.
- Script build native và workflow CI Windows/macOS/Ubuntu có lệnh thực thi.

## Kiến trúc và lỗi đã sửa

GUI tiếp tục gọi `app.Backend` → service → repository/storage. Không đưa SQL vào
UI và không thay backend bằng một triển khai mới. `internal/ui/work` quản lý
context, worker và callback qua `fyne.Do`; generation của truy vấn loại bỏ kết
quả cũ. Thumbnail được giải mã ngoài UI thread với giới hạn kích thước/cache.

Các lỗi được phát hiện và sửa trong quá trình kiểm tra:

1. SQLite mở sai URI ổ đĩa Windows; đã kiểm tra đường dẫn có khoảng trắng,
   Unicode và ký tự đặc biệt, cùng khả năng đóng/mở lại database.
2. Migration có thể áp dụng v2 trùng khi nhiều instance khởi động; kiểm tra và
   thay đổi schema nay cùng nằm trong IMMEDIATE transaction.
3. Reload editor có thể ghi đè nội dung người dùng đang nhập khi query chưa
   xong; editor khóa đúng thời điểm, giữ bản nháp khi conflict.
4. Shutdown có thể chờ vô hạn nếu driver thoát mà không gọi OnClosed;
   `Wait` chủ động hủy worker qua thao tác dừng idempotent.
5. Share hoàn tất sau khi dialog đóng vẫn cần cập nhật số đếm; callback giờ
   cập nhật dữ liệu màn hình trước khi bỏ qua phần dialog đã đóng.
6. Mini calendar vượt chiều rộng rail; đã kiểm tra với theme thực tế, đủ bảy
   cột. Refresh định kỳ của share không reset timeline thông thường.
7. Native renderer có thể mất chữ/icon menu khi đổi theme; `SetAppearance`
   refresh toàn bộ content sau khi đặt theme. Đã quan sát ảnh native trước và
   sau sửa, rồi kiểm tra lặp Light → Dark → Light → Dark. Không khẳng định đã
   xác định chính xác lỗi cache bên trong Fyne.

## Môi trường và lệnh đã chạy

- Go **1.26.2 windows/amd64**.
- Fyne **v2.7.4**, modernc SQLite **v1.34.5**.
- MinGW-w64 GCC **13.1.0**, `CGO_ENABLED=1`.
- Compiler: `C:\Qt\Tools\mingw1310_64\bin\gcc.exe`.

Các lệnh Go native dùng cấu hình PowerShell:

```powershell
$env:PATH = 'C:\Qt\Tools\mingw1310_64\bin;' + $env:PATH
$env:CC = 'gcc'
$env:CGO_ENABLED = '1'
```

| Kiểm tra | Kết quả quan sát |
| --- | --- |
| `gofmt -l` trên toàn bộ file Go của project | Không có file chưa format |
| `git diff --check` | Không có lỗi whitespace |
| `go mod tidy` | Exit 0; không thay đổi go.mod/go.sum |
| `go test -count=1 ./...` | PASS, exit 0, chạy lại sau sửa theme cuối cùng |
| `go vet ./...` | PASS, exit 0 |
| `go build ./cmd/notehub` | PASS, exit 0 |
| `go run -tags desktopsmoke ./tests/native` | PASS, exit 0; sáu màn hình, đổi theme lặp lại, đóng sạch |
| `./scripts/build-windows.ps1 -Version 0.2.0 -Compiler 'C:\Qt\Tools\mingw1310_64\bin\gcc.exe'` | PASS, exit 0; tạo bản release Windows |
| `./dist/windows/NoteHub.exe --version` | In `NoteHub 0.2.0` |

Ngoài ra đã chạy riêng `TestDesktopReferenceRendering` với
`NOTEHUB_TEST_SCREENSHOTS` trỏ tới `build/visual-qa`, xuất ảnh software renderer
ở 1450 × 900 và cửa sổ hẹp. Test xuất ảnh này mặc định được skip để các lần
`go test ./...` bình thường không ghi ảnh vào source.

Các regression test bao phủ migration v1 → v2 và migration đồng thời,
favorite/revision/persistence, shared expiry/revoke/count, phân trang có bộ lọc,
attachment listing/cascade, backup Skip/Replace/Duplicate và record cũ,
calendar/DST, composer, navigation, search không dấu, conflict/reload editor,
callback sau khi đóng dialog và hủy worker khi shutdown. Các test backend có
sẵn tiếp tục PASS.

Native smoke dùng database tạm riêng, không mở dữ liệu NoteHub của người dùng.
Ảnh Light/Dark nằm trong `build/native-smoke/`. Cửa sổ QA dùng 1280 × 730 để vừa
màn hình 1080p khi Windows scale 130%; cửa sổ sản phẩm vẫn mặc định 1450 × 900.
Harness chờ framebuffer ổn định, không capture khi framebuffer bằng 0 và xuất
RGB đã composite thành PNG opaque. Lỗi alpha khi capture được mô tả trong
[release notes Fyne 2.8.1](https://github.com/fyne-io/fyne/releases/tag/v2.8.1);
đây là xử lý ảnh kiểm thử, không thay đổi renderer của ứng dụng.

## Bản Windows đã tạo

Đường dẫn: `dist/windows/NoteHub.exe`.

- Version: **0.2.0**.
- Kích thước: **31,343,616 bytes**.
- SHA-256: `7802BE2888B7B974D0F6A9BDF414178417984448A1EF96667A8E8C3D5A56459C`.
- Kiểm tra PE import bằng `objdump -p`: GDI32, KERNEL32, msvcrt, OPENGL32,
  SHELL32, USER32. Không thấy phụ thuộc DLL runtime MinGW riêng trong bản này.

## Phạm vi chưa xác minh

- Chưa chạy native build/test trên macOS hoặc Linux trong phiên này. Workflow
  đã được triển khai, nhưng chưa dispatch/push và chưa có kết quả remote CI
  cho toàn bộ source cuối cùng. **Acceptance criterion 23 còn cần xác nhận
  bằng một Actions run thành công trên macOS/Linux.**
- Chưa kiểm tra thủ công trọn luồng native file chooser → ứng dụng PDF/ảnh
  bên ngoài trên từng hệ điều hành. Dịch vụ attachment, wiring UI và native
  compile đã được kiểm tra.
- Chưa tạo installer Windows, ký mã hoặc notarize macOS; đây là các bước phân
  phối riêng. Không thực hiện kiểm thử tải dữ liệu lớn kéo dài hoặc race suite
  toàn project trong lượt kiểm tra cuối.

## Danh sách file

Danh sách dưới đây mô tả trạng thái source hiện tại so với baseline `6f88a8a`,
kể cả các cập nhật CI/logo được ghi nhận vào project trong thời gian làm việc;
không quy mọi thay đổi đó cho cùng một tác giả. Các thư mục build/dist không
được đưa vào danh sách source.

### File mới

```text
assets/assets.go
assets/icons/NoteHub_White.png
docs/DESKTOP_TEST_REPORT.md
docs/PROMPT_BUILD_NOTEHUB_DESKTOP.md
docs/superpowers/plans/2026-10-02-notehub-desktop.md
internal/backup/favorite_test.go
internal/platform/chooser.go
internal/repository/sqlite/database_test.go
internal/repository/sqlite/migration_test.go
internal/ui/components/components_test.go
internal/ui/components/helpers_test.go
internal/ui/components/helpers.go
internal/ui/components/surface.go
internal/ui/dialogs/manager.go
internal/ui/dialogs/memo_test.go
internal/ui/dialogs/memo.go
internal/ui/layout_test.go
internal/ui/layout.go
internal/ui/screens/environment.go
internal/ui/screens/filters_test.go
internal/ui/screens/filters.go
internal/ui/screens/previews.go
internal/ui/screens/tags.go
internal/ui/window_test.go
internal/ui/work/runner_test.go
internal/ui/work/runner.go
tests/integration/desktop_backend_test.go
tests/native/main.go
```

### File đã sửa

```text
.github/workflows/release.yml
.github/workflows/test.yml
.gitignore
assets/icons/NoteHub.png
cmd/notehub/main.go
docs/ARCHITECTURE.md
docs/BUILD.md
go.mod
go.sum
internal/app/startup.go
internal/backup/manifest.go
internal/domain/memo.go
internal/repository/attachment_repository.go
internal/repository/memo_repository.go
internal/repository/sqlite/attachment_repository.go
internal/repository/sqlite/database.go
internal/repository/sqlite/memo_repository.go
internal/repository/sqlite/migration.go
internal/service/attachment_service.go
internal/service/backup_service.go
internal/service/calendar_service.go
internal/service/memo_service.go
internal/service/timeline_service.go
internal/ui/components/attachment_card.go
internal/ui/components/calendar_widget.go
internal/ui/components/memo_card.go
internal/ui/components/memo_editor.go
internal/ui/components/search_bar.go
internal/ui/components/sidebar.go
internal/ui/components/tag_list.go
internal/ui/components/timeline_list.go
internal/ui/dialogs/attachment.go
internal/ui/dialogs/confirm.go
internal/ui/dialogs/export.go
internal/ui/dialogs/import.go
internal/ui/dialogs/share.go
internal/ui/navigation.go
internal/ui/screens/attachments.go
internal/ui/screens/calendar.go
internal/ui/screens/search.go
internal/ui/screens/settings.go
internal/ui/screens/timeline.go
internal/ui/theme.go
internal/ui/window.go
README.md
scripts/build-linux.sh
scripts/build-macos.sh
scripts/build-windows.ps1
```

