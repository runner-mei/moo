//go:generate gobatis dao.go

package usermodels

import (
	"context"
	"time"

	"github.com/runner-mei/moo/api"
)

const (
	// UserAdmin admin 用户名
	UserAdmin = api.UserAdmin

	// UserGuest guest 用户名
	UserGuest = api.UserGuest

	// UserBgOperator background operator 用户名
	UserBgOperator = api.UserBgOperator

	// RoleSuper super 角色名
	RoleSuper = api.RoleSuper

	// RoleAdministrator administrator 角色名
	RoleAdministrator = api.RoleAdministrator

	// RoleVisitor visitor 角色名
	RoleVisitor = api.RoleVisitor

	// RoleGuest guest 角色名
	RoleGuest = api.RoleGuest
)

type OnlineUser struct {
	TableName struct{}  `json:"-" xorm:"moo_online_users"`
	UserID    int64     `json:"user_id" xorm:"user_id pk"`
	Uuid      string    `json:"uuid,omitempty" xorm:"uuid unique"`
	Address   string    `json:"address" xorm:"address"`
	CreatedAt time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

type User struct {
	TableName   struct{}               `json:"-" xorm:"moo_users"`
	ID          int64                  `json:"id" xorm:"id pk autoincr"`
	Name        string                 `json:"name" xorm:"name unique notnull"`
	Nickname    string                 `json:"nickname" xorm:"nickname unique notnull"`
	Password    string                 `json:"password,omitempty" xorm:"password null"`
	Description string                 `json:"description,omitempty" xorm:"description null"`
	Attributes  map[string]interface{} `json:"attributes" xorm:"attributes jsonb null"`
	Source      string                 `json:"source,omitempty" xorm:"source null"`
	Signature   string                 `json:"signature,omitempty" xorm:"signature null"`
	Disabled    bool                   `json:"disabled,omitempty" xorm:"disabled null"`
	LockedAt    *time.Time             `json:"locked_at,omitempty" xorm:"locked_at null"`
	CreatedAt   time.Time              `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

func (user *User) IsDisabled() bool {
	return user.Disabled // || user.Type == ItsmReporter
}

func (user *User) IsBuiltin() bool {
	return user.Name == UserAdmin ||
		user.Name == UserGuest
}

type UserProfile struct {
	TableName struct{} `json:"-" xorm:"moo_user_profiles"`
	ID        int64    `json:"id" xorm:"id pk unique(a)"`
	Name      string   `json:"name" xorm:"name pk unique(a) notnull"`
	Value     string   `json:"value,omitempty" xorm:"value"`
}

type Role struct {
	TableName   struct{}  `json:"-" xorm:"moo_roles"`
	ID          int64     `json:"id" xorm:"id pk autoincr"`
	Name        string    `json:"name" xorm:"name unique notnull"`
	Description string    `json:"description,omitempty" xorm:"description null"`
	CreatedAt   time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

func (role *Role) IsBuiltin() bool {
	return role.Name == RoleSuper ||
		role.Name == RoleAdministrator ||
		role.Name == RoleVisitor ||
		role.Name == RoleGuest
}

type UserAndRole struct {
	TableName struct{} `json:"-" xorm:"moo_users_and_roles"`
	UserID    int64    `json:"user_id" xorm:"user_id unique(user_role)"`
	RoleID    int64    `json:"role_id" xorm:"role_id unique(user_role) notnull"`
}

type PermissionAndRole struct {
	TableName  struct{} `json:"-" xorm:"moo_permission_and_roles"`
	RoleID     int64    `json:"role_id" xorm:"role_id unique(permission_role) notnull"`
	Permission string   `json:"permission" xorm:"permission unique(permission_role) notnull"`
}

type UserQueryer interface {
	// @type select
	// @default SELECT count(*) > 0 FROM <tablename type="User" /> WHERE nickname = #{name}
	NicknameExists(ctx context.Context, name string) (bool, error)

	// @record_type Role
	GetRoleByName(ctx context.Context, name string) func(*Role) error

	// @record_type User
	GetUserByID(ctx context.Context, id int64) func(*User) error

	// @record_type User
	GetUserByName(ctx context.Context, name string) func(*User) error

	// @record_type User
	GetUsers(ctx context.Context) ([]User, error)

	// @default SELECT FROM <tablename type="PermissionAndRole" /> WHERE role_id in <foreach collection="roleIDs" open="(" separator="," close=")">#{item}</foreach>
	GetPermissionsByRoleIDs(ctx context.Context, roleIDs []int64) ([]string, error)

	// @default SELECT * FROM <tablename type="Role" as="roles" /> WHERE
	//  exists (select * from <tablename type="UserAndRole" /> as users_roles
	//     where users_roles.role_id = roles.id and users_roles.user_id = #{userID})
	GetRolesByUser(ctx context.Context, userID int64) ([]Role, error)

	// @default SELECT value FROM <tablename type="UserProfile" /> WHERE id = #{userID} AND name = #{name}
	ReadProfile(ctx context.Context, userID int64, name string) (string, error)

	// @default INSERT INTO <tablename type="UserProfile" /> (id, name, value) VALUES(#{userID}, #{name}, #{value})
	//     ON CONFLICT (id, name) DO UPDATE SET value = excluded.value
	WriteProfile(ctx context.Context, userID int64, name, value string) error

	// @type delete
	// @default DELETE FROM <tablename type="UserProfile" /> WHERE id=#{userID} AND name=#{name}
	DeleteProfile(ctx context.Context, userID int64, name string) (int64, error)
}

type UserDao interface {
	UserQueryer

	// @type update
	// @default UPDATE <tablename type="User"/>
	//       SET locked_at = now() WHERE lower(name) = lower(#{username})
	Lock(ctx context.Context, username string) error

	// @type update
	// @default UPDATE <tablename type="User"/>
	//       SET locked_at = NULL WHERE lower(name) = lower(#{username})
	Unlock(ctx context.Context, username string) error

	CreateUser(ctx context.Context, user *User) (int64, error)

	// @type update
	// @default UPDATE <tablename type="User"/>
	//       SET disabled = true WHERE id=#{id}
	DisableUser(ctx context.Context, id int64) error

	// @type update
	// @default UPDATE <tablename type="User"/>
	//       SET disabled = false WHERE id=#{id}
	EnableUser(ctx context.Context, id int64) error

	UpdateUser(ctx context.Context, id int64, user *User) (int64, error)

	// @default INSERT INTO <tablename type="UserAndRole"/>(user_id, role_id)
	//       VALUES(#{userid}, #{roleid})
	//       ON CONFLICT (user_id, role_id)
	//       DO UPDATE SET user_id=EXCLUDED.user_id, role_id=EXCLUDED.role_id
	AddRoleToUser(ctx context.Context, userid, roleid int64) error

	// @default DELETE FROM <tablename type="UserAndRole"/> WHERE user_id = #{userid} and role_id = #{roleid}
	RemoveRoleFromUser(ctx context.Context, userid, roleid int64) error
}
