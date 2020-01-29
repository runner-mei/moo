package users

import (
	"context"

	"github.com/runner-mei/moo/users/usermodels"
)

// Action
const (
	// UserAdmin admin 用户名
	UserAdmin = usermodels.UserAdmin

	// UserGuest guest 用户名
	UserGuest = usermodels.UserGuest


	// RoleSuper super 角色名
	RoleSuper = usermodels.RoleSuper

	// RoleAdministrator administrator 角色名
	RoleAdministrator = usermodels.RoleAdministrator

	// RoleVisitor visitor 角色名
	RoleVisitor = usermodels.RoleVisitor

	// RoleGuest guest 角色名
	RoleGuest = usermodels.RoleGuest
)

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
}

type ReadCurrentUserFunc func(context.Context) (User, error)
