# GoGit — 实施路线图

> 版本：v1.0
> 日期：2026-08-20 (GMT+8)
> 规模：预估 6,000–8,000 LoC（< 10,000，无需分 MVP/V1/V2 边界）

---

## 阶段顺序决策

**选择：UI-First（默认）**

理由：管理页是仪表盘 / CRUD 型界面（总览、文件树、暂存、历史、分支），组件结构可依据 `docs/API.md` 的资源契约先行搭建，不依赖运行时数据模型推导。Git 引擎在 Phase 3 按同一契约接线。

---

## 目录结构

```
GoGit/
├── backend/                 # Go 引擎 + REST API
│   ├── cmd/server/
│   └── internal/{git,api,logger}
├── frontend-user/           # Vue 3 管理页（唯一前端）
├── tests/                   # API Smoke（Pytest）
├── docs/
├── docker-compose.yml
└── Dockerfile
```

`frontend-admin` / `frontend-mp`：本项目为单机免登录本地工具，无管理后台与微信小程序，不创建空壳目录。

---

## Phase 1 — 架构 [x]

- [x] `git init` + `.gitignore`
- [x] `docs/Roadmap.md`
- [x] `docs/API.md` 契约草案
- [x] `docker-compose.yml` 骨架（开发随机端口 **41783**）
- [x] 目录占位

## Phase 2 — UI [x]

- [x] `docs/DesignSpec.md`（Carbon Archive 美学）
- [x] Vue 3 + Tailwind 管理页六视图
- [x] Toast / Modal / 表单校验 / 响应式断点

## Phase 3 — Logic [x]

- [x] 对象存储（Blob/Tree/Commit + SHA-1/SHA-256 + zlib）
- [x] index / add / commit / status
- [x] branch 创建、切换、三方 merge
- [x] REST API 接线 + 统一 Logger
- [x] Dockerfile 多阶段（ARM64 + AMD64）
- [x] 演示仓库种子数据

## Phase 4 — QA [x]

- [x] 引擎单元测试（哈希向量、zlib 往返、三方合并、切换还原）
- [x] `tests/api_smoke.py` 容器内 Mock/离线（本项目无外部 API，¥0）
- [x] `docs/QA_Record.md`

## 扩展 — 2026-08-21 [x]

- [x] `.gogitignore` / unified diff / rev-parse / unstage / restore / fsck / config
- [x] Go 源文件 > 20，总行数 ≥ 2000（实测 33 文件 / 4501 行）

## Phase 5 — Audit [x]

- [x] `docs/AuditReport.md`
- [x] Knowledge Harvest

---

## 开发端口

| 服务 | 宿主端口 | 容器端口 |
|---|---|---|
| Web + API | **41783**（探测未占用） | 8080 |
