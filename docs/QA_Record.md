# QA Record

## Round 1 — 2026-08-20 18:43 (GMT+8)

**Cost**: ¥0（无外部计费 API，全程离线/本地）

### 环境
- `docker compose up --build -d`（宿主端口 41783）
- 镜像内 `go test ./...` 于 Dockerfile backend 阶段执行
- `docker compose run --rm qa` 执行 `tests/api_smoke.py`

### 结果

| 检查 | 结果 |
|---|---|
| Docker Build | PASS（含镜像内 `go test ./...` ok gogit/internal/git） |
| Health Check | PASS `{"status":"ok"}` 时间字段为 GMT+8 |
| SPA 壳 | PASS `id="app"` + 构建后 assets |
| Blob SHA-1 向量 | PASS `src/hello.txt` oid = `ce013625030ba8dba906f756967f9e9ca394464a`（与 `git hash-object` 的 `hello\n` 一致） |
| API Smoke（4 tests） | PASS `.... [100%]` |
| 浏览器抽检 | PASS 总览 / 工作区 / 分支 三视图可渲染，数据来自真实仓库 |

### 单元测试覆盖（Docker build 日志）
- Blob 哈希向量（hello / empty）
- zlib 往返与魔数
- add → commit → checkout 工作区还原
- 三方合并无冲突 + 冲突标记
- SHA-256 闭环
- 路径穿越拒绝
- 损坏对象 / 损坏 index 拒绝

### 备注
- qa 卷只读时 pytest 默认 cache 会告警；compose 已改为 `-p no:cacheprovider`。
- 未调用任何按量 API。

**判定：PASS，进入 Phase 5。**
