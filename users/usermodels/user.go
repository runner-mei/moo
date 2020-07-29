//go:generate gobatis user.go

package usermodels

import (
	"context"
	"database/sql"
	"io"
	"time"

	"github.com/runner-mei/goutils/as"
	"github.com/runner-mei/goutils/netutil"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/validation"
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
	UserID    int64     `json:"user_id" xorm:"user_id pk(user_address)"`
	Address   string    `json:"address" xorm:"address pk(user_address)"`
	Uuid      string    `json:"uuid,omitempty" xorm:"uuid unique"`
	CreatedAt time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

// @gobatis.ignore
type OnlineUsers interface {
	List(ctx context.Context, interval string) ([]OnlineUser, error)
}

type OnlineUserDao interface {
	// @default SELECT * FROM <tablename /> <if test="isNotEmpty(interval)">WHERE (updated_at + #{interval}::INTERVAL) &gt; now() </if>
	List(ctx context.Context, interval string) ([]OnlineUser, error)
	Create(ctx context.Context, userID int64, address, uuid string) (int64, error)

	// @type insert
	// @default INSERT INTO <tablename type="OnlineUser" />(user_id, address, uuid, created_at, updated_at)
	//          VALUES(#{userID}, #{address}, #{uuid}, now(), now())  ON CONFLICT (user_id, address)
	//          DO UPDATE SET updated = now()
	CreateOrTouch(ctx context.Context, userID int64, address, uuid string) (int64, error)

	// @type update
	// @default UPDATE <tablename type="OnlineUser" /> SET updated_at = now() WHERE user_id = #{userID} AND address = #{address}
	Touch(ctx context.Context, userID int64, address, uuid string) (int64, error)

	// @default DELETE FROM <tablename type="OnlineUser" /> WHERE user_id = #{userID} AND address = #{address}
	Delete(ctx context.Context, userID int64, address string) (int64, error)
}

type User struct {
	TableName   struct{}               `json:"-" xorm:"moo_users"`
	ID          int64                  `json:"id" xorm:"id pk autoincr"`
	Name        string                 `json:"name" xorm:"name unique notnull"`
	Nickname    string                 `json:"nickname" xorm:"nickname unique notnull"`
	Password    string                 `json:"password,omitempty" xorm:"password null"`
	Description string                 `json:"description,omitempty" xorm:"description null"`
	CanLogin    bool                   `json:"can_login,omitempty" xorm:"can_login null"`
	IsDefault   bool                   `json:"is_default,omitempty" xorm:"is_default null"`
	Attributes  map[string]interface{} `json:"attributes" xorm:"attributes jsonb null"`
	Source      string                 `json:"source,omitempty" xorm:"source null"`
	Signature   string                 `json:"signature,omitempty" xorm:"signature null"`
	Disabled    bool                   `json:"disabled,omitempty" xorm:"disabled null"`
	LockedAt    *time.Time             `json:"locked_at,omitempty" xorm:"locked_at null"`
	CreatedAt   time.Time              `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty" xorm:"updated_at updated"`

	// Type        int                    `json:"type,omitempty" xorm:"type"`
	Reserved1 map[string]string `json:"profiles" xorm:"profiles <- null"`
}

func (user *User) Default() *User {
	user.CanLogin = true
	return user
}

func (user *User) Validate(validation *validation.Validation) bool {
	validation.Required("Name", user.Name)
	validation.Required("Nickname", user.Nickname)
	if user.Source != "ldap" && user.Source != "cas" {
		validation.MinSize("Password", user.Password, 8)
		validation.MaxSize("Password", user.Password, 250)
	}

	o := user.Attributes["white_address_list"]
	if o != nil {
		var ss = as.ToStrings(o)
		if len(ss) != 0 {
			_, err := netutil.ToCheckers(ss)
			if err != nil {
				validation.Error("Attributes[white_address_list]", err.Error())
			}
		}
	}
	return validation.HasErrors()
}

func (user *User) IsDisabled() bool {
	return user.Disabled // || user.Type == ItsmReporter
}

func (user *User) IsBuiltin() bool {
	if user.IsDefault {
		return true
	}

	return user.Name == UserAdmin ||
		user.Name == UserGuest ||
		user.Name == UserBgOperator
}

func (user *User) IsHidden() bool {
	return user.Name == UserBgOperator // || user.Type == ItsmReporter
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
	Type        int64     `json:"type" xorm:"type null"`
	Name        string    `json:"name" xorm:"name unique notnull"`
	Description string    `json:"description,omitempty" xorm:"description null"`
	IsDefault   bool      `json:"is_default,omitempty" xorm:"is_default null"`
	CreatedAt   time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

func (role *Role) Validate(v *validation.Validation) bool {
	v.Required("Name", role.Name)
	return v.HasErrors()
}

func (role *Role) IsBuiltin() bool {
	if role.IsDefault {
		return true
	}
	return role.Name == RoleSuper ||
		role.Name == RoleAdministrator ||
		role.Name == RoleVisitor ||
		role.Name == RoleGuest
}

type UserAndRole struct {
	TableName struct{} `json:"-" xorm:"moo_users_and_roles"`
	Reserve1  int64    `json:"id" xorm:"id <-"`
	UserID    int64    `json:"user_id" xorm:"user_id unique(user_role)"`
	RoleID    int64    `json:"role_id" xorm:"role_id unique(user_role) notnull"`
}

type UserQueryParams struct {
	NameLike  string
	CanLogin  sql.NullBool
	Enabled   sql.NullBool
	Source    sql.NullString
	Roles     []int64
	Rolenames []string

	UsergroupIDs       []int64
	UsergroupRecursive bool
}

type UserQueryer interface {
	// @type select
	// @default SELECT count(*) > 0 FROM <tablename type="Role" /> WHERE name = #{name}
	RolenameExists(ctx context.Context, name string) (bool, error)

	// @type select
	// @default SELECT count(*) > 0 FROM <tablename type="User" /> WHERE lower(name) = lower(#{name})
	UsernameExists(ctx context.Context, name string) (bool, error)

	// @type select
	// @default SELECT count(*) > 0 FROM <tablename type="User" /> WHERE nickname = #{name}
	NicknameExists(ctx context.Context, name string) (bool, error)

	// @record_type Role
	GetRoleCount(ctx context.Context, nameLike string) (int64, error)

	// @record_type Role
	GetRoles(ctx context.Context, nameLike string, _type sql.NullInt64, offset, limit int64) (func(*Role) (bool, error), io.Closer)

	// @default SELECT * FROM <tablename name="Role" as="roles" /> WHERE roles.id in
	//  (SELECT role_id from <tablename type="UserAndUsergroup" /> WHERE group_id = #{usergroupID} and user_id = #{userID})
	// GetRoleByUsergroupIDAndUserID(ctx context.Context, usergroupID, userID int64) ([]Role, error)

	// @record_type Role
	GetRoleByID(ctx context.Context, id int64) func(*Role) error

	// @record_type Role
	GetRoleByName(ctx context.Context, name string) func(*Role) error

	// @record_type Role
	GetRolesByNames(ctx context.Context, name []string) (func(*Role) (bool, error), io.Closer)

	// @record_type User
	GetUserByID(ctx context.Context, id int64) func(*User) error

	// @default SELECT * FROM <tablename type="User" /> WHERE lower(name) = lower(#{name})
	GetUserByName(ctx context.Context, name string) func(*User) error

	// @default SELECT * FROM <tablename type="User" /> WHERE lower(nickname) = lower(#{nickname})
	GetUserByNickname(ctx context.Context, nickname string) func(*User) error

	// @default SELECT * FROM <tablename type="User" /> WHERE lower(name) = lower(#{name}) OR lower(nickname) = lower(#{nickname})
	GetUserByNameOrNickname(ctx context.Context, name, nickname string) func(*User) error

	// @default SELECT count(*) FROM <tablename type="User" as="users" /> <where>
	//  <if test="len(params.Roles) &gt; 0">EXISTS(SELECT * FROM <tablename type="UserAndRole" as="u2r" /> where u2r.role_id in (<foreach collection="params.Roles" separator=",">#{item}</foreach>) AND u2r.user_id = users.id) AND</if>
	//  <if test="len(params.Rolenames) &gt; 0">EXISTS(SELECT * FROM <tablename type="UserAndRole" as="u2r" /> JOIN <tablename type="Role" as="r" /> ON u2r.role_id = r.id
	//      WHERE r.name in (<foreach collection="params.Rolenames" separator=",">#{item}</foreach>) AND u2r.user_id = users.id) AND
	//  </if>
	//  <if test="isNotEmpty(params.NameLike)"> (users.name like <like value="params.NameLike" /> OR users.nickname like <like value="params.NameLike" />) AND</if>
	//  <if test="params.CanLogin.Valid"> users.can_login = #{params.CanLogin} AND </if>
	//  <if test="params.Enabled.Valid"> (<if test="!params.Enabled.Bool"> NOT </if> ( users.disabled IS NULL OR users.disabled = false )) AND </if>
	//  <if test="len(params.UsergroupIDs) &gt; 0">
	//   <if test="params.UsergroupRecursive">
	//     exists (select * from <tablename type="UserAndUsergroup" /> as u2g where u2g.user_id = users.id and u2g.group_id in (
	//         WITH RECURSIVE ALLGROUPS (ID)  AS (
	//           SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH
	//             FROM <tablename type="Usergroup" as="ug" /> WHERE <if test="len(params.UsergroupIDs) == 1"> ug.id = <foreach collection="params.UsergroupIDs" separator=",">#{item}</foreach></if>
	//             <if test="len(params.UsergroupIDs) &gt; 1"> ug.id in (<foreach collection="params.UsergroupIDs" separator=",">#{item}</foreach>)</if>
	//             UNION ALL
	//           SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH
	//             FROM <tablename type="Usergroup" as="D" /> JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)
	//         SELECT ID FROM ALLGROUPS))
	//   </if>
	//   <if test="!params.UsergroupRecursive">
	//      exists (select * from <tablename type="UserAndUsergroup" as="u2g" />
	//       where u2g.user_id = users.id and <if test="len(params.UsergroupIDs) == 1"> u2g.group_id = <foreach collection="params.UsergroupIDs" separator=",">#{item}</foreach></if>
	//            <if test="len(params.UsergroupIDs) &gt; 1"> u2g.group_id in (<foreach collection="params.UsergroupIDs" separator=",">#{item}</foreach>)</if>)
	//   </if>
	//  </if>
	//  </where>
	GetUserCount(ctx context.Context, params *UserQueryParams) (int64, error)

	// @default SELECT * FROM <tablename type="User" as="users" /> <where>
	//  <if test="len(params.Roles) &gt; 0">EXISTS(SELECT * FROM <tablename type="UserAndRole" as="u2r" /> where u2r.role_id in (<foreach collection="params.Roles" separator=",">#{item}</foreach>) AND u2r.user_id = users.id) AND</if>
	//  <if test="len(params.Rolenames) &gt; 0">EXISTS(SELECT * FROM <tablename type="UserAndRole" as="u2r" /> JOIN <tablename type="Role" as="r" /> ON u2r.role_id = r.id
	//      WHERE r.name in (<foreach collection="params.Rolenames" separator=",">#{item}</foreach>) AND u2r.user_id = users.id) AND
	//  </if>
	//  <if test="isNotEmpty(params.NameLike)"> (users.name like <like value="params.NameLike" /> OR users.nickname like <like value="params.NameLike" />) AND</if>
	//  <if test="params.CanLogin.Valid"> users.can_login = #{params.CanLogin} AND </if>
	//  <if test="params.Enabled.Valid"> (<if test="!params.Enabled.Bool"> NOT </if> ( users.disabled IS NULL OR users.disabled = false )) AND </if>
	//  <if test="len(params.UsergroupIDs) &gt; 0">
	//   <if test="params.UsergroupRecursive">
	//     exists (select * from <tablename type="UserAndUsergroup" as="u2g" />
	//        where u2g.user_id = users.id and u2g.group_id in (
	//         WITH RECURSIVE ALLGROUPS (ID)  AS (
	//           SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH
	//             FROM <tablename type="Usergroup" as="ug" /> WHERE <if test="len(params.UsergroupIDs) == 1"> ug.id = <foreach collection="params.UsergroupIDs" separator=",">#{item}</foreach></if>
	//             <if test="len(params.UsergroupIDs) &gt; 1"> ug.id in (<foreach collection="params.UsergroupIDs" separator=",">#{item}</foreach>)</if>
	//             UNION ALL
	//           SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH
	//             FROM <tablename type="Usergroup" as="D" /> JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)
	//         SELECT ID FROM ALLGROUPS))
	//   </if>
	//   <if test="!params.UsergroupRecursive">
	//      exists (select * from <tablename type="UserAndUsergroup" as="u2g" />
	//       where u2g.user_id = users.id and <if test="len(params.UsergroupIDs) == 1"> u2g.group_id = <foreach collection="params.UsergroupIDs" separator=",">#{item}</foreach></if>
	//            <if test="len(params.UsergroupIDs) &gt; 1"> u2g.group_id in (<foreach collection="params.UsergroupIDs" separator=",">#{item}</foreach>)</if>)
	//   </if>
	//  </if>
	//  </where>
	//  <pagination />
	//  <order_by />
	GetUsers(ctx context.Context, params *UserQueryParams, offset, limit int64, sort string) (func(*User) (bool, error), io.Closer)

	// @default SELECT * FROM <tablename type="Role" as="roles" /> WHERE
	//  exists (select * from <tablename type="UserAndRole" /> as users_roles
	//     where users_roles.role_id = roles.id and users_roles.user_id = #{userID})
	//  OR exists (select * from <tablename type="UserAndUsergroup" /> as u2g
	//     where u2g.role_id = roles.id and u2g.user_id = #{userID})
	GetRolesByUserID(ctx context.Context, userID int64) ([]Role, error)

	// @default SELECT value FROM <tablename type="UserProfile" /> WHERE id = #{userID} AND name = #{name}
	ReadProfile(ctx context.Context, userID int64, name string) (string, error)

	// @default INSERT INTO <tablename type="UserProfile" /> (id, name, value) VALUES(#{userID}, #{name}, #{value})
	//     ON CONFLICT (id, name) DO UPDATE SET value = excluded.value
	WriteProfile(ctx context.Context, userID int64, name, value string) error

	// @type delete
	// @default DELETE FROM <tablename type="UserProfile" /> WHERE id=#{userID} AND name=#{name}
	DeleteProfile(ctx context.Context, userID int64, name string) (int64, error)

	// @record_type UserAndRole
	GetUserAndRoleList(ctx context.Context) (func(*UserAndRole) (bool, error), io.Closer)

	// @default SELECT id, name FROM <tablename type="User" as="users" /> <where>
	//  <if test="!isJobPostion.Valid">
	//       <if test="groupID.Valid">EXISTS(SELECT * FROM <tablename type="UserAndUsergroup" as="u2g" /> WHERE u2g.user_id = users.id AND u2g.group_id = #{groupID})</if>
	//       <if test="roleID.Valid"> AND
	//            (EXISTS(SELECT * FROM <tablename type="UserAndRole" as="u2r" /> WHERE u2r.user_id = users.id AND u2r.role_id = #{roleID}) OR
	//             EXISTS(SELECT * FROM <tablename type="UserAndUsergroup" as="u2g" /> WHERE u2g.user_id = users.id AND u2g.role_id = #{roleID}))
	//       </if>
	//  </if>
	//  <if test="isJobPostion.Valid">
	//    <if test="groupID.Valid">EXISTS(SELECT * FROM <tablename type="UserAndUsergroup" as="u2g" /> WHERE u2g.user_id = users.id AND u2g.group_id = #{groupID}
	//        <if test="roleID.Valid">AND u2g.role_id = #{roleID}</if>)</if>
	//    <if test="roleID.Valid">AND EXISTS(SELECT * FROM <tablename type="UserAndUsergroup" as="u2g" /> WHERE u2g.user_id = users.id AND u2g.role_id = #{roleID})</if>
	//  </if>
	// </where>
	GetUsernames(ctx context.Context, groupID, roleID sql.NullInt64, isJobPostion sql.NullBool) (map[int64]string, error)

	// @default SELECT id, name FROM <tablename type="Role" /> <if test="type.Valid"> WHERE type = #{type} </if>
	GetRolenames(ctx context.Context, _type sql.NullInt64) (map[int64]string, error)
}

type UserDao interface {
	UserQueryer

	// @type update
	// @default UPDATE <tablename type="User"/> SET locked_at = now() WHERE id=#{id}
	LockUser(ctx context.Context, id int64) error

	// @type update
	// @default UPDATE <tablename type="User"/> SET locked_at = null WHERE id=#{id}
	UnlockUser(ctx context.Context, id int64) error

	// @type update
	// @default UPDATE <tablename type="User"/>
	//       SET locked_at = now() WHERE lower(name) = lower(#{username})
	LockUserByUsername(ctx context.Context, username string) error

	// @type update
	// @default UPDATE <tablename type="User"/>
	//       SET locked_at = NULL WHERE lower(name) = lower(#{username})
	UnlockUserByUsername(ctx context.Context, username string) error

	CreateUser(ctx context.Context, user *User) (int64, error)

	UpdateUser(ctx context.Context, id int64, user *User) (int64, error)

	// @record_type User
	UpdateUserPassword(ctx context.Context, id int64, password string) (int64, error)

	// @record_type User
	DeleteUser(ctx context.Context, id int64) (int64, error)

	// @type update
	// @default UPDATE <tablename type="User"/>
	//       SET disabled = true <if test="name.Valid">, name= #{name} </if>
	//           <if test="nickname.Valid">, nickname= #{nickname} </if>
	//       WHERE id=#{id}
	DisableUser(ctx context.Context, id int64, name, nickname sql.NullString) error

	// @type update
	// @default UPDATE <tablename type="User"/>
	//       SET disabled = false <if test="name.Valid">, name= #{name} </if>
	//           <if test="nickname.Valid">, nickname= #{nickname} </if>
	//       WHERE id=#{id}
	EnableUser(ctx context.Context, id int64, name, nickname sql.NullString) error

	// @type select
	// @default SELECT count(*) > 0 FROM <tablename type="UserAndRole" />
	//          WHERE user_id = #{userid} AND role_id = #{roleid}
	HasRoleForUser(ctx context.Context, userid, roleid int64) (bool, error)

	// @default INSERT INTO <tablename type="UserAndRole"/>(user_id, role_id)
	//       VALUES(#{userid}, #{roleid})
	//       ON CONFLICT (user_id, role_id)
	//       DO UPDATE SET user_id=EXCLUDED.user_id, role_id=EXCLUDED.role_id
	AddRoleToUser(ctx context.Context, userid, roleid int64) error

	// @record_type UserAndRole
	RemoveRoleFromUser(ctx context.Context, userid, roleid int64) error

	// @record_type UserAndRole
	RemoveRolesFromUser(ctx context.Context, userid int64) error

	CreateRole(ctx context.Context, role *Role) (int64, error)

	UpdateRole(ctx context.Context, id int64, role *Role) (int64, error)

	// @record_type Role
	DeleteRole(ctx context.Context, id int64) (int64, error)
}
