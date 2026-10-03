# NoteHub IPC Protocol v1

Transport: UTF-8 JSON objects separated by whitespace/newline over the child
process `stdin`/`stdout`.

Qt launches `notehub-backend(.exe)` using `QProcess`. Backend stdout is reserved
for protocol traffic; logs go to stderr.

## Startup event

```json
{"event":"ready","protocolVersion":1,"version":"0.3.0","dataDir":"...","os":"windows"}
```

Fatal startup error:

```json
{"event":"fatal","message":"..."}
```

## Request / response

```json
{"id":12,"method":"timeline.list","params":{"limit":20}}
```

```json
{"id":12,"result":{"items":[],"next":null}}
```

Error:

```json
{"id":12,"error":{"code":409,"kind":"conflict","message":"conflict"}}
```

Error mapping:

| Go error | IPC code |
|---|---:|
| `domain.ErrInvalid` | 400 |
| `domain.ErrNotFound` | 404 |
| `domain.ErrConflict` | 409 |
| Other | 500 |

## Methods

- `app.ping`, `app.info`, `app.shutdown`
- `memo.create`, `memo.get`, `memo.update`, `memo.delete`, `memo.favorite`
- `timeline.list`, `timeline.counts`
- `search.query`
- `calendar.month`, `calendar.date`
- `tags.list`
- `attachment.add`, `attachment.list`, `attachment.listAll`,
  `attachment.reorder`, `attachment.delete`, `attachment.path`
- `backup.export`, `backup.import`
- `share.create`, `share.list`, `share.revoke`, `share.server`, `share.status`

IPC DTO cố ý có local numeric IDs và `localPath` vì đây là kênh nội bộ giữa hai
process cùng user session. HTTP share API vẫn dùng domain JSON cũ, không serialize
DB ID hoặc filesystem path.
