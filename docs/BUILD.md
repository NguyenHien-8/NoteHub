# Build NoteHub Qt + Go

## Windows (khuyến nghị)

Cài:

1. Go 1.23+
2. CMake 3.21+
3. Qt 6.5+ qua Qt Online Installer
4. Một kit desktop 64-bit, ví dụ `MSVC 2022 64-bit`

Ví dụ:

```powershell
.\scripts\build-windows.ps1 -QtDir "C:\Qt\6.11.2\msvc2022_64"
```

Nếu dùng `mingw_64`, script tìm Ninja/MinGW trong `C:\Qt\Tools`. Không nên dùng
một `gcc.exe` MSYS2 bất kỳ để link với Qt được build bởi compiler ABI khác; hãy
build bằng compiler đi cùng Qt kit.

## Chỉ build backend

```powershell
go mod tidy
go test ./...
go build -o notehub-backend.exe ./cmd/notehub-backend
```

## Chỉ build frontend

```powershell
cmake -S . -B build/qt -DCMAKE_PREFIX_PATH="C:\Qt\6.11.2\msvc2022_64"
cmake --build build/qt --config Release --parallel
```

## Deployment

Windows phải chạy `windeployqt` để copy Qt DLL và platform plugin `qwindows`.
Build script đã thực hiện bước này. `notehub-backend.exe` phải nằm cạnh
`NoteHub.exe`.
