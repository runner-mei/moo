package users

import (
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/users/usermodels"
)

// Action
const (
	// UserAdmin admin 用户名
	UserAdmin = usermodels.UserAdmin

	// UserGuest guest 用户名
	UserGuest = usermodels.UserGuest

	// UserBgOperator background operator 用户名
	UserBgOperator = usermodels.UserBgOperator

	// RoleSuper super 角色名
	RoleSuper = usermodels.RoleSuper

	// RoleAdministrator administrator 角色名
	RoleAdministrator = usermodels.RoleAdministrator

	// RoleVisitor visitor 角色名
	RoleVisitor = usermodels.RoleVisitor

	// RoleGuest guest 角色名
	RoleGuest = usermodels.RoleGuest
)

type ReadCurrentUserFunc = api.ReadCurrentUserFunc

// type UserManager = api.UserManager
type User = api.User
type Option = api.Option
