# 数据库规范

> GORM + MariaDB（mysql 驱动），无独立迁移工具。

---

## 连接与迁移

- `internal/repository/db.go` 的 `Init()` 建立连接并持有包级变量 `repository.DB`。
- **迁移 = AutoMigrate**：新模型必须加进 `db.go` 的 `AutoMigrate(&model.User{}, &model.App{})` 列表，没有 SQL 迁移文件。

## 模型（internal/model/model.go）

- 所有模型嵌入 `BaseModel`（ID/CreatedAt/UpdatedAt/DeletedAt 软删除）。
- 字段用 gorm tag 声明约束，敏感字段 json 隐藏：

```go
type User struct {
    BaseModel
    UUID     string `gorm:"type:char(36);uniqueIndex;not null" json:"uuid"`
    Username string `gorm:"size:64;uniqueIndex;not null" json:"username"`
    Password string `gorm:"size:128;not null" json:"-"`   // json:"-" 不外泄
}
```

- 自动生成字段用 GORM Hook（如 `User.BeforeCreate` 生成 UUID）。

## Repository 模式（internal/repository/）

- 每个领域一个空结构体 + 构造函数：`AppRepository` / `NewAppRepository()`，方法内直接用包级 `DB`。
- 方法命名：`Create` / `Update` / `Delete` / `GetByID` / `GetByUserID` / `GetByUserAndName` / `UpdateStatus`。
- 单字段更新用列更新，不整行 Save：

```go
func (r *AppRepository) UpdateStatus(id uint, status string) error {
    return DB.Model(&model.App{}).Where("id = ?", id).Update("status", status).Error
}
```

- 查询必须参数化占位符：`DB.Where("user_id = ? AND name = ?", userID, name)`。**禁止拼接 SQL 字符串。**

## Not-Found 约定

repository 直接返回 GORM 原始错误；由 **service 层** 用 `errors.Is(err, gorm.ErrRecordNotFound)` 判断并转成业务错误码（见 `internal/service/app.go` 的 `DeleteApp`）。

## 禁止

- 拼接 SQL / `Raw()` 裸 SQL。
- 在 handler 或 service 里直接使用 `repository.DB`。
- 忽略 `.Error` 返回值。
