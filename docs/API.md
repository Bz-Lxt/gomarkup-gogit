# GoGit REST API

Base URL: `/api/v1`  
时间字段：用户可见值为 `yyyy-MM-dd HH:mm:ss`（GMT+8）；提交对象内部存储 Unix 秒 + `+0800`。  
响应信封：成功 `{ "data": ... }`，失败 `{ "error": { "code", "message", "details?" } }`。

---

## 错误码表

| HTTP | code | 含义 |
|---|---|---|
| 400 | `invalid_json` | 请求体不是合法 JSON |
| 400 | `invalid_path` | 路径为空、含 `..` 或指向 `.gogit` |
| 400 | `validation_error` | 字段缺失或类型/边界不合法 |
| 404 | `not_found` | 对象 / 分支 / 文件不存在 |
| 409 | `conflict` | 切换会覆盖未提交变更，或 merge 产生冲突 |
| 409 | `already_exists` | 分支已存在 |
| 409 | `already_up_to_date` | 目标已包含在当前历史中 |
| 422 | `merge_in_progress` | 存在未完成的 merge |
| 500 | `internal_error` | 未预期的内部错误 |

---

## 端点

### `GET /api/v1/health`

```json
{ "data": { "status": "ok", "time": "2026-08-20 18:30:00" } }
```

### `GET /api/v1/repo`

```json
{
  "data": {
    "path": "/data/repo",
    "hash_algo": "sha1",
    "current_branch": "main",
    "head": "ce013625030ba8dba906f756967f9e9ca394464a",
    "object_count": 12,
    "merge_in_progress": false
  }
}
```

### `POST /api/v1/repo/init`

```json
{ "hash_algo": "sha1" }
```
`hash_algo` 可选 `sha1`（默认）或 `sha256`。仓库已存在返回 409。

### `GET /api/v1/files?path=`

工作区目录列表。`path` 默认为仓库根。

```json
{
  "data": {
    "path": "src",
    "entries": [
      { "name": "hello.txt", "path": "src/hello.txt", "type": "file", "size": 12, "mode": "100644" }
    ]
  }
}
```

### `GET /api/v1/files/content?path=src/hello.txt`

```json
{ "data": { "path": "src/hello.txt", "content": "hello\n", "binary": false, "size": 6 } }
```

### `PUT /api/v1/files`

写工作区文件（便于演示 add/commit）。

```json
{ "path": "src/hello.txt", "content": "hello world\n" }
```

### `DELETE /api/v1/files?path=src/hello.txt`

删除工作区文件。

### `POST /api/v1/index/add`

```json
{ "paths": ["src/hello.txt", "docs"] }
```

目录会被递归加入。响应返回写入 index 的条目。

### `GET /api/v1/index`

```json
{
  "data": {
    "entries": [
      { "path": "src/hello.txt", "mode": "100644", "oid": "...", "size": 6 }
    ]
  }
}
```

### `GET /api/v1/status`

```json
{
  "data": {
    "staged": [{ "path": "a.txt", "status": "added" }],
    "unstaged": [{ "path": "b.txt", "status": "modified" }],
    "untracked": [{ "path": "c.txt", "status": "untracked" }]
  }
}
```

`status` 取值：`added` / `modified` / `deleted` / `untracked`。

### `POST /api/v1/commits`

```json
{ "message": "feat: add hello", "author": "Ada <ada@gogit.local>" }
```

若存在 `MERGE_HEAD`，提交自动成为双亲 merge commit。

```json
{
  "data": {
    "hash": "...",
    "tree": "...",
    "parents": [],
    "author": "Ada <ada@gogit.local>",
    "message": "feat: add hello",
    "committed_at": "2026-08-20 18:30:00"
  }
}
```

### `GET /api/v1/commits?branch=main`

按时间倒序返回提交列表（含 `parents`）。

### `GET /api/v1/commits/:hash`

单次提交详情。

### `GET /api/v1/commits/:hash/tree`

扁平化快照：`{ "data": { "entries": [{ "path", "mode", "oid", "type" }] } }`。

### `GET /api/v1/branches`

```json
{
  "data": [
    { "name": "main", "hash": "...", "current": true }
  ]
}
```

### `POST /api/v1/branches`

```json
{ "name": "feature/docs" }
```
201 Created。

### `POST /api/v1/checkout`

```json
{ "name": "feature/docs" }
```

切换分支并检出工作区。若会覆盖未提交变更 → 409 `conflict`。

### `POST /api/v1/merge`

```json
{ "branch": "feature/docs" }
```

成功：返回 merge commit。  
冲突：409，`details` 为冲突路径列表，工作区写入冲突标记。

### `GET /api/v1/diff?path=&side=`

`side` 为 `unstaged`（工作区 vs index）或 `staged`（index vs HEAD）。返回 unified diff。

```json
{ "data": { "path": "a.txt", "side": "unstaged", "binary": false, "patch": "--- a/a.txt\n+++ b/a.txt\n...", "old_oid": "...", "new_oid": "..." } }
```

### `POST /api/v1/index/unstage`

```json
{ "paths": ["a.txt"] }
```
将路径恢复为 HEAD 中的条目（新文件则从 index 删除）。

### `POST /api/v1/files/restore`

```json
{ "paths": ["a.txt"] }
```
用 HEAD 快照覆盖工作区文件。

### `POST /api/v1/index/reset`

```json
{ "paths": ["a.txt"], "mode": "mixed" }
```
`mixed` = unstage；`worktree` = unstage 并恢复工作区。

### `GET /api/v1/rev-parse?q=HEAD`

解析 `HEAD` / 分支名 / 完整或唯一短哈希。

```json
{ "data": { "q": "HEAD", "oid": "...", "kind": "commit" } }
```

### `GET /api/v1/fsck`

对象完整性：哈希回算、可达性、dangling 列表。

### `GET /api/v1/config` / `PUT /api/v1/config`

```json
{ "user_name": "Ada", "user_email": "ada@gogit.local" }
```
`hash_algo` 初始化后不可改。

### `GET /api/v1/objects/:hash`

按类型返回解析后的对象：

- Blob：`{ type, hash, size, content?, binary }`
- Tree：`{ type, hash, entries: [{ mode, name, oid, type }] }`
- Commit：`{ type, hash, tree, parents, author, message, committed_at }`
