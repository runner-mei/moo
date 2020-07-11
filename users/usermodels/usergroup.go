//go:generate gobatis usergroup.go

package usermodels

import (
	"context"
	"database/sql"
	"io"
	"time"

	"github.com/runner-mei/validation"
)

type UserAndUsergroup struct {
	TableName struct{} `json:"-" xorm:"moo_users_and_usergroups"`
	UserID    int64    `json:"user_id" xorm:"user_id notnull"`
	GroupID   int64    `json:"group_id" xorm:"group_id notnull"`
	RoleID    int64    `json:"role_id" xorm:"role_id null"`
}

type Usergroup struct {
	TableName   struct{}  `json:"-" xorm:"moo_usergroups"`
	ID          int64     `json:"id" xorm:"id pk autoincr"`
	Name        string    `json:"name" xorm:"name notnull"`
	Description string    `json:"description" xorm:"description null"`
	ParentID    int64     `json:"parent_id" xorm:"parent_id null"`
	Disabled    bool      `json:"disabled" xorm:"disabled null"`
	CreatedAt   time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

func (group *Usergroup) Validate(v *validation.Validation) bool {
	v.Required("Name", group.Name)
	return v.HasErrors()
}

type UsergroupQueryer interface {
	// @type select
	// @default SELECT count(*) > 0 FROM <tablename type="Usergroup" /> WHERE name = #{name}
	UsergroupnameExists(ctx context.Context, name string) (bool, error)

	// @record_type Usergroup
	GetUsergroupByID(ctx context.Context, id int64) func(*Usergroup) error

	// @default SELECT * FROM <tablename type="Usergroup" /> where id in (
	//   WITH RECURSIVE ALLGROUPS (ID)  AS (
	//     SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH
	//        FROM <tablename type="Usergroup" as="ug" /> WHERE id = #{id} <foreach collection="list" open="AND id in (" close=")">#{item}</foreach>
	//     UNION ALL
	//     SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH
	//        FROM <tablename type="Usergroup" as="D" /> JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)
	//   SELECT ID FROM ALLGROUPS ORDER BY PATH)
	GetUsergroupsByRecursive(ctx context.Context, id int64, list ...int64) (func(*Usergroup) (bool, error), io.Closer)

	// @record_type Usergroup
	GetUsergroupByName(ctx context.Context, name string) func(*Usergroup) error

	// @record_type Usergroup
	GetUsergroups(ctx context.Context) (func(*Usergroup) (bool, error), io.Closer)

	// @default <if test="recursive">
	// SELECT user_id FROM <tablename type="UserAndUsergroup" as="uug" /> where uug.group_id in (
	// WITH RECURSIVE ALLGROUPS (ID)  AS (
	//   SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH
	//      FROM <tablename type="Usergroup" as="ug" /> WHERE id=#{groupID}
	//   UNION ALL
	//   SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH
	//      FROM <tablename type="Usergroup" as="D" /> JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)
	//  SELECT ID FROM ALLGROUPS ORDER BY PATH)
	//  <if test="userEnabled.Valid"> AND EXISTS (SELECT * FROM <tablename type="User" as="u" /> WHERE <if test="!userEnabled.Bool"> NOT </if> ( disabled IS NULL or disabled = false ) AND uug.user_id = u.id) </if>
	// </if>
	// <if test="!recursive">
	//    SELECT user_id FROM <tablename type="UserAndUsergroup" as="uug" /> where uug.group_id = #{groupID}
	//       <if test="userEnabled.Valid">
	//         AND EXISTS (
	//           SELECT * FROM <tablename type="User" as="u" />
	//           WHERE uug.user_id = u.id AND <if test="!userEnabled.Bool"> NOT </if> ( disabled IS NULL or disabled = false )
	//         )
	//       </if>
	// </if>
	GetUserIDsByGroupID(ctx context.Context, groupID int64, recursive bool, userEnabled sql.NullBool) ([]int64, error)

	// @default SELECT group_id FROM <tablename type="UserAndUsergroup" as="u2g" /> WHERE user_id = #{userID}
	GetGroupIDsByUserID(ctx context.Context, userID int64) ([]int64, error)

	// @record_type UserAndUsergroup
	GetUserAndGroupList(ctx context.Context, userid sql.NullInt64) (func(*UserAndUsergroup) (bool, error), io.Closer)

	// @default SELECT id, name FROM <tablename type="User" as="users" /> where
	//  EXISTS(SELECT * FROM <tablename type="UserAndRole" as="u2r" /> WHERE u2r.user_id = users.id AND u2r.role_id = #{roleID})
	//  OR EXISTS(SELECT * FROM <tablename type="UserAndUsergroup" as="u2g" /> WHERE u2g.user_id = users.id AND u2g.role_id = #{roleID})
	GetUsernamesByRoleID(ctx context.Context, roleID int64) (map[int64]string, error)

	// @default SELECT id, name FROM <tablename type="Role" /> <if test="type.Valid"> WHERE type = #{type} </if>
	GetRolenames(ctx context.Context, _type sql.NullInt64) (map[int64]string, error)
}

type UsergroupDao interface {
	UsergroupQueryer

	CreateUsergroup(ctx context.Context, usergroup *Usergroup) (int64, error)

	UpdateUsergroup(ctx context.Context, id int64, usergroup *Usergroup) (int64, error)

	// @default <if test="recursive">
	// SELECT * FROM <tablename type="Usergroup" /> where id in (
	//   WITH RECURSIVE ALLGROUPS (ID)  AS (
	//     SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH
	//        FROM <tablename type="Usergroup" as="ug" /> WHERE id=#{groupID}
	//     UNION ALL
	//     SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH
	//        FROM <tablename type="Usergroup" as="D" /> JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)
	//   SELECT ID FROM ALLGROUPS ORDER BY PATH)
	// </if>
	// <if test="!recursive">
	//    SELECT * FROM <tablename type="Usergroup" /> where id = #{id}
	// </if>
	DeleteUsergroup(ctx context.Context, id int64, recursive bool) (int64, error)

	// @type select
	// @default SELECT count(*) > 0 FROM <tablename type="UserAndUsergroup" />
	//           WHERE group_id = #{groupid} and user_id = #{userid}
	HasUserForGroup(ctx context.Context, userid, roleid int64) (bool, error)

	// @default INSERT INTO <tablename type="UserAndUsergroup"/>(group_id, user_id)
	//       VALUES(#{groupid}, #{userid})
	//       ON CONFLICT (group_id, user_id)
	//       DO NOTHING
	AddUserToGroup(ctx context.Context, groupid, userid int64) error

	// @default DELETE FROM <tablename type="UserAndUsergroup"/>
	//           WHERE group_id = #{groupid} and user_id = #{userid}
	RemoveUserFromGroup(ctx context.Context, groupid, userid int64) error

	// @default DELETE FROM <tablename type="UserAndUsergroup"/>
	//           WHERE user_id = #{userid}
	RemoveUserFromAllGroups(ctx context.Context, userid int64) error

	// @default <if test="recursive">
	// SELECT * FROM <tablename type="UserAndUsergroup" as="uug" /> where uug.group_id in (
	// WITH RECURSIVE ALLGROUPS (ID)  AS (
	//   SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH
	//      FROM <tablename type="Usergroup" as="ug" /> WHERE id=#{groupID}
	//   UNION ALL
	//   SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH
	//      FROM <tablename type="Usergroup" as="D" /> JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)
	//  SELECT ID FROM ALLGROUPS ORDER BY PATH)
	//  <if test="userEnabled"> AND EXISTS (SELECT * FROM <tablename type="User" as="u" /> WHERE ( disabled IS NULL or disabled = false ) AND uug.user_id = u.id) </if>
	// </if>
	// <if test="!recursive">
	//    SELECT * FROM <tablename type="UserAndUsergroup" as="uug" /> where uug.group_id = #{groupID}
	//      <if test="userEnabled"> AND EXISTS (SELECT * FROM <tablename type="User" as="u" /> WHERE ( disabled IS NULL or disabled = false ) AND uug.user_id = u.id) </if>
	// </if>
	GetUsersAndGroups(ctx context.Context, groupID int64, recursive, userEnabled bool) ([]UserAndUsergroup, error)
}
