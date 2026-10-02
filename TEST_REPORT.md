# Test report

Prepared from the uploaded `memos-main.zip`.

Checks performed in the build sandbox:

1. Extracted the Memos archive and inspected the relevant Go backend files for memo CRUD/timeline, SQLite, attachments, tags, share, export and import.
2. Ran `gofmt` on all Go sources in this package.
3. Ran offline type-check/tests with the provided `teststub` build tag:

```text
GOPROXY=off GOTOOLCHAIN=local go test -tags teststub ./...
PASS
```

This compiled every package and passed the unit tests for tag extraction and ZIP path validation.

4. Executed the generated SQLite schema using the sandbox's SQLite runtime and checked:
   - memo insert,
   - Unicode-lowercased text search query,
   - tag aggregation,
   - foreign-key cascade from memo -> share.

## Limitation of sandbox verification

The sandbox has no outbound network access, and `modernc.org/sqlite` is not pre-cached. Therefore a normal runtime build using the real pure-Go SQLite driver could not be downloaded/executed here. The release source pins the dependency in `go.mod`; on a normal development machine, run:

```bash
go mod tidy
go test ./...
```

before integrating the package into the desktop GUI.
