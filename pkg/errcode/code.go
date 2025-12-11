package errcode

// 错误码枚举
// 错误码规则:
//   - 0: 成功
//   - 1xxxx: 客户端错误（参数校验、认证授权等）
//   - 2xxxx: 业务错误（用户、应用等业务逻辑错误）
//   - 3xxxx: 系统错误（数据库、K8s、外部服务等）

type Code int

const (
	// 成功
	Success Code = 0

	// 客户端错误 1xxxx
	ErrBadRequest   Code = 10001 // 请求参数错误
	ErrUnauthorized Code = 10002 // 未登录或 Token 无效
	ErrForbidden    Code = 10003 // 无权限访问
	ErrNotFound     Code = 10004 // 资源不存在

	// 用户相关错误 2xxxx
	ErrUserExists       Code = 20001 // 用户已存在
	ErrUserNotFound     Code = 20002 // 用户不存在
	ErrPasswordWrong    Code = 20003 // 密码错误
	ErrEmailExists      Code = 20004 // 邮箱已被使用
	ErrUserDisabled     Code = 20005 // 用户已被禁用
	ErrInvalidUsername  Code = 20006 // 用户名格式无效
	ErrInvalidPassword  Code = 20007 // 密码格式无效
	ErrInvalidEmail     Code = 20008 // 邮箱格式无效
	ErrLoginFailed      Code = 20009 // 登录失败
	ErrRegisterFailed   Code = 20010 // 注册失败
	ErrTokenExpired     Code = 20011 // Token 已过期
	ErrTokenInvalid     Code = 20012 // Token 无效

	// 应用相关错误 21xxx
	ErrAppNotFound    Code = 21001 // 应用不存在
	ErrAppExists      Code = 21002 // 应用已存在
	ErrAppCreateFail  Code = 21003 // 创建应用失败
	ErrAppUpdateFail  Code = 21004 // 更新应用失败
	ErrAppDeleteFail  Code = 21005 // 删除应用失败
	ErrAppStartFail   Code = 21006 // 启动应用失败
	ErrAppStopFail    Code = 21007 // 停止应用失败
	ErrAppRestartFail Code = 21008 // 重启应用失败

	// 系统错误 3xxxx
	ErrInternal   Code = 30001 // 服务器内部错误
	ErrDatabase   Code = 30002 // 数据库错误
	ErrK8s        Code = 30003 // K8s 操作错误
	ErrK8sConnect Code = 30004 // K8s 连接失败
)

// codeMessages 错误码对应的默认消息
var codeMessages = map[Code]string{
	Success: "成功",

	// 客户端错误
	ErrBadRequest:   "请求参数错误",
	ErrUnauthorized: "未登录或 Token 无效",
	ErrForbidden:    "无权限访问",
	ErrNotFound:     "资源不存在",

	// 用户相关错误
	ErrUserExists:      "用户已存在",
	ErrUserNotFound:    "用户不存在",
	ErrPasswordWrong:   "密码错误",
	ErrEmailExists:     "邮箱已被使用",
	ErrUserDisabled:    "用户已被禁用",
	ErrInvalidUsername: "用户名格式无效",
	ErrInvalidPassword: "密码格式无效",
	ErrInvalidEmail:    "邮箱格式无效",
	ErrLoginFailed:     "登录失败",
	ErrRegisterFailed:  "注册失败",
	ErrTokenExpired:    "Token 已过期",
	ErrTokenInvalid:    "Token 无效",

	// 应用相关错误
	ErrAppNotFound:    "应用不存在",
	ErrAppExists:      "应用已存在",
	ErrAppCreateFail:  "创建应用失败",
	ErrAppUpdateFail:  "更新应用失败",
	ErrAppDeleteFail:  "删除应用失败",
	ErrAppStartFail:   "启动应用失败",
	ErrAppStopFail:    "停止应用失败",
	ErrAppRestartFail: "重启应用失败",

	// 系统错误
	ErrInternal:   "服务器内部错误",
	ErrDatabase:   "数据库错误",
	ErrK8s:        "K8s 操作错误",
	ErrK8sConnect: "K8s 连接失败",
}

// Int 返回错误码的整数值
func (c Code) Int() int {
	return int(c)
}

// Message 返回错误码的默认消息
func (c Code) Message() string {
	if msg, ok := codeMessages[c]; ok {
		return msg
	}
	return "未知错误"
}

// Error 带错误码的错误类型
type Error struct {
	Code Code
	Msg  string
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return e.Code.Message()
}

// New 创建带错误码的错误
func New(code Code) *Error {
	return &Error{Code: code, Msg: code.Message()}
}

// NewWithMsg 创建带自定义消息的错误
func NewWithMsg(code Code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

// FromError 从 error 中提取错误码，如果不是 Error 类型则返回 ErrInternal
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	return &Error{Code: ErrInternal, Msg: err.Error()}
}
