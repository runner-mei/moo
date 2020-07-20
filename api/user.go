package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/runner-mei/errors"
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

// 常用的错误
var (
	ErrUnauthorized       = errors.ErrUnauthorized
	ErrCacheInvalid       = errors.New("permission cache is invald")
	ErrTagNotFound        = errors.New("permission tag is not found")
	ErrPermissionNotFound = errors.New("permission is not found")
	ErrAlreadyClosed      = errors.New("server is closed")
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
	// Users(ctx context.Context, opts ...Option) ([]User, error)

	UserByName(ctx context.Context, username string, opts ...Option) (User, error)
	UserByID(ctx context.Context, userID int64, opts ...Option) (User, error)
}

// User 用户信息
type User interface {
	ID() int64

	// 用户登录名
	Name() string

	// // 是不是有一个管理员角色
	// HasAdminRole() bool

	// // 是不是有一个 Guest 角色
	// IsGuest() bool

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
	HasPermission(ctx context.Context, permissionID string) (bool, error)

	// 是不是有一个指定的角色
	HasRole(string) bool

	// // 本用户是不是指定的用户组的成员
	// IsMemberOf(int64) bool

	// 用户属性
	ForEach(func(string, interface{}))
}

// Usergroup 用户组信息
type Usergroup interface {
	ID() int64

	// 用户登录名
	Name() string

	// 父用户组 ID
	ParentID() int64

	// 父用户组
	Parent(ctx context.Context) Usergroup

	// 组中是不是有这个用户
	HasUser(ctx context.Context, userID int64) bool

	// 用户成员
	Users(ctx context.Context, opts ...Option) ([]User, error)
}

// UsergroupManager 用户管理
type UsergroupManager interface {
	UsergroupsByUserID(ctx context.Context, userID int64, opts ...Option) ([]Usergroup, error)
	UsergroupByName(ctx context.Context, username string, opts ...Option) (Usergroup, error)
	UsergroupByID(ctx context.Context, groupID int64, opts ...Option) (Usergroup, error)
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

type InternalOptions struct {
	IncludeDisabled bool
}

func InternalApply(opts ...Option) InternalOptions {
	var o InternalOptions
	for _, opt := range opts {
		switch opt.(type) {
		case userIncludeDisabled:
			o.IncludeDisabled = true
		}
	}
	return o
}

func MakeMockUser(id int64, name string) *mockUser {
	return &mockUser{id: id, name: name}
}

// User 用户信息
type mockUser struct {
	id   int64
	name string
}

func (u *mockUser) ID() int64 {
	return u.id
}

func (u *mockUser) Name() string {
	return u.name
}

// 是不是有一个管理员角色
func (u *mockUser) HasAdminRole() bool {
	return false
}

// 是不是有一个 Guest 角色
func (u *mockUser) IsGuest() bool {
	return false
}

// 呢称
func (u *mockUser) Nickname() string {
	return u.name
}

func (u *mockUser) WriteProfile(key, value string) error {
	return nil
}

func (u *mockUser) ReadProfile(key string) (string, error) {
	return "", nil
}

func (u *mockUser) Data(key string) interface{} {
	switch key {
	case "id":
		return u.id
	case "name", "nickname":
		return u.name
	}
	return nil
}

func (u *mockUser) Roles() []string {
	return nil
}

func (u *mockUser) HasPermission(ctx context.Context, permissionID string) (bool, error) {
	return false, nil
}

func (u *mockUser) HasRole(string) bool {
	return false
}
func (u *mockUser) HasRoleID(roleid int64) bool {
	return false
}

func (u *mockUser) IsMemberOf(id int64) bool {
	return false
}

func (u *mockUser) ForEach(cb func(string, interface{})) {
	cb("id", u.id)
	cb("name", u.name)
	cb("nickname", u.name)
}
