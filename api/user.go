package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/runner-mei/errors"
)

// 常用的错误
var (
	ErrUnauthorized = errors.ErrUnauthorized
)

const (
	// UserAdmin admin 用户名
	UserAdmin = "admin"

	// UserGuest guest 用户名
	UserGuest = "guest"

	// UserBgOperator background operator 用户名
	UserBgOperator = "background_operator"

	// RoleSuper super 角色名
	RoleSuper = "super"

	// RoleAdministrator administrator 角色名
	RoleAdministrator = "administrator"

	// RoleVisitor visitor 角色名
	RoleVisitor = "visitor"

	// RoleGuest guest 角色名
	RoleGuest = "guest"
)

type userIncludeDisabled struct{}

func (u userIncludeDisabled) IsIncludeDisabled() {}

func (u userIncludeDisabled) apply() {}

// Option 用户选项
type Option interface {
	apply()
}

// UserIncludeDisabled 禁用的用户也返回
func UserIncludeDisabled() Option {
	return userIncludeDisabled{}
}

// UserManager 用户管理
type UserManager interface {
	Users(ctx context.Context, opts ...Option) ([]User, error)

	UserByName(ctx context.Context, username string, opts ...Option) (User, error)
	UserByID(ctx context.Context, userID int64, opts ...Option) (User, error)
}

// User 用户信息
type User interface {
	ID() int64

	// 用户登录名
	Name() string

	// 是不是有一个管理员角色
	HasAdminRole() bool

	// 是不是有一个 Guest 角色
	IsGuest() bool

	// 呢称
	Nickname() string

	// Profile 是用于保存用户在界面上的一些个性化数据

	// WriteProfile 保存 profiles
	WriteProfile(key, value string) error

	// Profile 是用于保存用户在界面上的一些个性化数据
	// ReadProfile 读 profiles
	ReadProfile(key string) (string, error)

	// 用户扩展属性
	Data(key string) interface{}

	// 用户角色列表
	Roles() []string

	// 用户是否有指定的权限
	HasPermission(permissionName string) bool

	// 是不是有一个指定的角色
	HasRole(string) bool

	// 用户属性
	ForEach(func(string, interface{}))
}

type userKey struct{}

func (*userKey) String() string {
	return "moo-user-key"
}

var UserKey = &userKey{}

type ReadCurrentUserFunc func(context.Context) (User, error)

func ContextWithUser(ctx context.Context, u ReadCurrentUserFunc) context.Context {
	return context.WithValue(ctx, UserKey, u)
}

func UserFromContext(ctx context.Context) ReadCurrentUserFunc {
	o := ctx.Value(UserKey)
	if o == nil {
		return nil
	}
	f, _ := o.(ReadCurrentUserFunc)
	return f
}

func ReadUserFromContext(ctx context.Context) (User, error) {
	o := ctx.Value(UserKey)
	if o == nil {
		return nil, errors.NewError(http.StatusUnauthorized, "user isnot exists because session is unauthorized")
	}
	f, ok := o.(ReadCurrentUserFunc)
	if ok {
		return f(ctx)
	}
	u, ok := o.(User)
	if ok {
		return u, nil
	}
	return nil, errors.NewError(http.StatusInternalServerError, fmt.Sprintf("user is unknown type - %T", o))
}
