# Astro

Astro 是一个面向 C 端用户的容器即服务 (CaaS) 平台，提供简单易用的容器管理界面，让非技术用户也能轻松部署和管理容器化应用。

## 项目愿景

降低容器技术的使用门槛，让每个人都能享受云原生技术带来的便利。

## 核心原则

- **简单至上**: 隐藏复杂的技术细节，提供直观的操作界面
- **用户友好**: 面向非技术用户设计，无需了解 Docker/Kubernetes 知识
- **开箱即用**: 提供预配置的应用模板，一键部署常用服务
- **安全可靠**: 内置安全最佳实践，保护用户数据和应用

## 目标用户

- 个人开发者和爱好者
- 小型团队和初创公司
- 需要快速部署应用的非技术人员

## 技术栈

- **语言**: Go 1.25+
- **Web 框架**: Gin
- **K8s 客户端**: client-go
- **数据库**: Mariadb + GORM
- **认证**: JWT
- **配置管理**: Viper
- **权限鉴定**: Casbin
- **日志管理**: Zap

## 项目结构

```
.
├── cmd/
│   └── server/         # 主程序入口
├── internal/
│   ├── handler/        # HTTP 处理器
│   ├── service/        # 业务逻辑层
│   ├── repository/     # 数据访问层
│   ├── model/          # 数据模型
│   ├── middleware/     # 中间件（认证等）
│   └── k8s/            # K8s 客户端封装
├── pkg/
│   ├── config/         # 配置加载
│   ├── errcode/        # 错误码定义
│   └── logger/         # 日志封装（基于 Zap）
├── configs/            # 配置文件
└── docs/               # Swagger API 文档
```

## 开发规范

### 代码风格

- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行静态检查
- 错误必须处理，不允许使用 `_` 忽略
- 包名使用小写单词，避免下划线
- 导出函数和类型必须有注释

### 提交规范

使用语义化提交信息:
- `feat`: 新功能
- `fix`: 修复问题
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具相关

## 核心功能

对 Kubernetes API 的二次封装，提供简化的容器管理能力：

- **应用部署**: 一键部署容器应用，无需编写 YAML ✅
- **服务管理**: 启动、停止、重启、删除应用 ✅
- **日志查看**: 实时查看容器日志 ✅
- **资源监控**: 查看 CPU、内存使用情况 🔄
- **应用模板**: 预置常用应用模板（数据库、Web 服务等）🔄

## 当前进度（2026-08-24）

### MVP 第一阶段 ✅ 已完成
- 用户注册/登录/JWT 认证
- 应用 CRUD（创建/查询/删除）
- 应用启动/停止/重启
- 应用日志查看
- 项目管理与项目级多租户隔离（每项目独立命名空间）

### 可用 API
| 方法 | 路由 | 功能 |
|-----|------|-----|
| POST | /api/v1/register | 用户注册 |
| POST | /api/v1/login | 用户登录 |
| POST | /api/v1/projects | 创建项目 |
| GET | /api/v1/projects | 项目列表 |
| GET | /api/v1/projects/:project_id | 项目详情 |
| DELETE | /api/v1/projects/:project_id | 删除空项目 |
| POST | /api/v1/projects/:project_id/apps | 创建应用 |
| GET | /api/v1/projects/:project_id/apps | 应用列表 |
| GET | /api/v1/projects/:project_id/apps/:id | 应用详情 |
| DELETE | /api/v1/projects/:project_id/apps/:id | 删除应用 |
| POST | /api/v1/projects/:project_id/apps/:id/start | 启动应用 |
| POST | /api/v1/projects/:project_id/apps/:id/stop | 停止应用 |
| POST | /api/v1/projects/:project_id/apps/:id/restart | 重启应用 |
| GET | /api/v1/projects/:project_id/apps/:id/logs | 查看日志 |

# 注意（必须遵循，绝不能违反）

## 设计原则
1. **小步快跑**: 坚决杜绝过度设计，每次功能开发必须"小而美"，越简洁、越直接越好
2. **KISS 原则**: 优先选择最简单的实现方案，避免引入不必要的抽象层
3. **YAGNI 原则**: 不实现当前不需要的功能，不为假设的未来需求预留接口

## 代码规范
4. **错误处理**: 所有错误必须显式处理，禁止使用 `_` 忽略错误返回值
5. **命名清晰**: 变量、函数、类型命名必须准确表达其用途，禁止使用无意义的缩写
6. **单一职责**: 每个函数只做一件事，每个文件只处理一类逻辑

## API 设计
7. **RESTful 风格**: API 设计遵循 RESTful 规范，路径使用名词复数形式
8. **统一响应**: 所有 API 返回统一的响应格式，包含 code、message、data 字段
9. **参数校验**: 所有外部输入必须进行有效性校验，在 handler 层完成

## 安全要求
10. **权限检查**: 所有涉及资源操作的接口必须进行权限校验
11. **敏感信息**: 禁止在日志中输出密码、Token 等敏感信息
12. **SQL 注入**: 禁止拼接 SQL 语句，必须使用 GORM 的参数化查询

## 错误码规范
13. **统一错误码**: 所有 API 响应的 code 字段必须使用 `pkg/errcode` 包中定义的错误码枚举，禁止使用硬编码数字
14. **错误码分段**: 错误码遵循以下分段规则：
    - `0`: 成功
    - `1xxxx`: 客户端错误（参数校验、认证授权等）
    - `2xxxx`: 业务错误（用户 20xxx、应用 21xxx 等）
    - `3xxxx`: 系统错误（数据库、K8s、外部服务等）
15. **新增错误码**: 添加新错误码时必须在对应分段内添加，并在 `codeMessages` 中配置默认消息

## 文档与注释
16. **中文优先**: 代码注释、文档、提交信息统一使用中文
17. **必要注释**: 只在逻辑复杂或不直观的地方添加注释，避免注释显而易见的代码
18. **同步更新**: 修改代码时必须同步更新相关文档和注释

19. **命令使用**: 必须使用 make 中存在的指令，不得随意单独执行

## 日志规范
20. **统一日志**: 禁止使用标准库 `log`，必须使用 `pkg/logger` 包
21. **结构化日志**: 使用 `zap.Field` 记录上下文信息，避免字符串拼接
22. **日志级别**: Debug 用于调试、Info 用于常规、Warn 用于异常、Error 用于错误、Fatal 用于致命错误

## GitHub 与 Trellis 工作治理

23. **权威流程**: 必须遵循 `docs/development-workflow.md`；纯问答和只读调查可不建工作项，一旦实施代码、文档、配置或流程变更则不可跳过
24. **先建工作项**: Feature、Bug、Maintenance 必须先使用对应 GitHub Issue Form，并加入唯一的 [Astro Development](https://github.com/users/cuihe500/projects/6) Project，达到 `Ready` 后才可创建 Trellis 任务
25. **唯一映射**: 一个可独立验收的 Trellis 任务只能对应一个 GitHub Issue 和一个 Project 条目；实现步骤留在 `implement.md`，不得滥建子任务
26. **任务关联**: 创建 Trellis 任务时必须同时写入 `meta.github_issue` 和 `meta.github_project`；禁止使用空值、编号猜测或事后补录
27. **开始门禁**: `task.py start` 前必须确认 Work Type、Priority、Assignee、Status、Start date、Trellis Task 完整，Project Status 已改为 `In Progress`，且用户已批准规划
28. **权限失败即停止**: 无 GitHub 权限、无法验证 Issue/Project 或关联不一致时，只报告缺失项并等待处理，禁止先实施后补录
29. **交付关联**: PR 必须关联 Issue 和 Trellis 路径；完整交付使用 `Fixes #<编号>`，部分交付使用 `Refs #<编号>`，并将 Project Status 更新为 `In Review`
30. **完成定义**: PR 合并且验收通过后才能关闭 Issue、将 Project 改为 `Done` 并归档 Trellis；`task.py finish` 仅清除会话指针，不代表完成
31. **例外处理**: 紧急修复只能由用户明确授权，且最迟在 PR 审查或合并前补齐关联；安全漏洞必须使用 Security Advisory 或受限私有工作项，不得公开敏感信息
32. **命令入口**: 日常 GitHub 与 Trellis 操作分别使用 Makefile 的 `github` 和 `trellis` 等目标，禁止直接调用 `gh` 或 `.trellis/scripts/*.py`；一次性 Project 初始化可在明确授权下例外执行
