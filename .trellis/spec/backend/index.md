# Backend 开发规范

> Astro 后端（Go + Gin + GORM + client-go）的项目级编码规范，供 AI 子代理与新成员加载。

---

## 规范索引

| 文档 | 内容 | 状态 |
|------|------|------|
| [目录结构](./directory-structure.md) | 分层布局、调用方向、命名约定 | ✅ 已填写 |
| [数据库规范](./database-guidelines.md) | GORM 模型、Repository 模式、AutoMigrate | ✅ 已填写 |
| [错误处理](./error-handling.md) | errcode 分段、分层错误转换、统一响应 | ✅ 已填写 |
| [质量规范](./quality-guidelines.md) | Makefile 命令、校验/权限分层、禁止模式 | ✅ 已填写 |
| [日志规范](./logging-guidelines.md) | pkg/logger 用法、级别、敏感信息 | ✅ 已填写 |
| [认证规范](./auth-guidelines.md) | JWT 与 OAuth2/OIDC 登录契约、状态校验、身份映射 | ✅ 已填写 |

---

## 维护约定

- 文档描述**代码的真实现状**，不写理想化模式；发现规范与代码不符时以代码为准并更新文档。
- 根目录 `AGENTS.md` 是最高优先级项目约束，本目录是其落地细化。
- 按项目约定，规范文档统一使用**中文**。
