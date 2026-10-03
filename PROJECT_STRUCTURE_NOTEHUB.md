# NoteHub project structure (Qt + Go)

```text
NoteHub/
├── CMakeLists.txt
├── frontend/
│   ├── CMakeLists.txt
│   ├── resources/
│   │   ├── NoteHub.png
│   │   ├── NoteHub.ico
│   │   ├── notehub.rc
│   │   └── notehub.qrc
│   └── src/
│       ├── main.cpp
│       ├── IpcClient.{h,cpp}
│       ├── MainWindow.{h,cpp}
│       ├── MemoCard.{h,cpp}
│       ├── MemoDialog.{h,cpp}
│       ├── ShareDialog.{h,cpp}
│       └── Theme.{h,cpp}
├── cmd/
│   └── notehub-backend/main.go
├── internal/
│   ├── app/                 # composition root
│   ├── ipc/                 # JSON/stdio adapter for Qt
│   ├── domain/              # pure models
│   ├── service/             # business logic
│   ├── repository/          # interfaces
│   │   └── sqlite/          # persistence + migrations + FTS
│   ├── storage/             # attachment filesystem
│   ├── backup/              # ZIP + safe extraction
│   ├── share/               # local read-only HTTP share server
│   ├── tagparse/
│   ├── identity/
│   └── platform/            # AppData / OS helpers
├── assets/icons/
├── scripts/
│   ├── build-windows.ps1
│   ├── build-linux.sh
│   └── build-macos.sh
├── tests/integration/
├── docs/
└── .github/workflows/
```

Production dependency direction:

```text
Qt GUI → IPC → app → service → repository interface → SQLite
                           ├→ storage
                           ├→ backup
                           └→ share
```
