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
	Reserve1  int64    `json:"id" xorm:"id <-"`
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
	//        FROM <tablename type="Usergroup" as="ug" /> WHERE id = #{id} <foreach collection="list" open="AND id in (" close=")" separator=",">#{item}</foreach>
	//     UNION ALL
	//     SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH
	//        FROM <tablename type="Usergroup" as="D" /> JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)
	//   SELECT ID FROM ALLGROUPS ORDER BY PATH)
	GetUsergroupsByRecursive(ctx context.Context, id int64, list ...int64) (func(*Usergroup) (bool, error), io.Closer)

	// @record_type Usergroup
	GetUsergroupByName(ctx context.Context, name string) func(*Usergroup) error

	// @default SELECT * FROM <tablename type="Usergroup" as="g" /> <if test="userid.Valid"> WHERE exists(select * from <tablename type="UserAndUsergroup" as="uug" /> where uug.group_id = g.id and uug.user_id = ${userid.Int64})</if>
	GetUsergroups(ctx context.Context, userid sql.NullInt64) (func(*Usergroup) (bool, error), io.Closer)

	// @default <if test="recursive">
	// SELECT user_id FROM <tablename type="UserAndUsergroup" as="uug" /> where uug.group_id in (
	// WITH RECURSIVE ALLGROUPS (ID)  AS (
	//   SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH
	//      FROM <tablename type="Usergroup" as="ug" />
	//      WHERE <if test="len(groupIDs) == 1"> ug.id = <foreach collection="groupIDs" separator=",">#{item}</foreach></if>
	//             <if test="len(groupIDs) &gt; 1"> ug.id in (<foreach collection="groupIDs" separator=",">#{item}</foreach>)</if>
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
	GetUserIDsByGroupIDs(ctx context.Context, groupIDs []int64, recursive bool, userEnabled sql.NullBool) ([]int64, error)

	// @default SELECT * FROM <tablename type="User" as="users" /> WHERE
	//  <if test="userEnabled.Valid"> (<if test="!userEnabled.Bool"> NOT </if> ( users.disabled IS NULL OR users.disabled = false )) AND </if>
	// <if test="recursive">
	// EXISTS (select * FROM <tablename type="UserAndUsergroup" as="uug" /> where uug.user_id = users.id and uug.group_id in (
	// WITH RECURSIVE ALLGROUPS (ID) AS (
	//     SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH
	//      FROM <tablename type="Usergroup" as="ug" />
	//      WHERE <if test="len(groupIDs) == 1"> ug.id = <foreach collection="groupIDs" separator=",">#{item}</foreach></if>
	//             <if test="len(groupIDs) &gt; 1"> ug.id in (<foreach collection="groupIDs" separator=",">#{item}</foreach>)</if>
	//   UNION ALL
	//     SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH
	//      FROM <tablename type="Usergroup" as="D" /> JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)
	//  SELECT ID FROM ALLGROUPS ORDER BY PATH))
	// </if>
	// <if test="!recursive">
	//  EXISTS (select * from <tablename type="UserAndUsergroup" /> as uug
	//     where uug.user_id = users.id and uug.group_id = #{groupID})
	// </if>
	GetUsersByGroupIDs(ctx context.Context, groupIDs []int64, recursive bool, userEnabled sql.NullBool) ([]User, error)

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

	// @default SELECT * FROM <tablename type="UserAndUsergroup" as="uug" /> <where>
	//   <if test="groupEnabled"> EXISTS (SELECT * FROM <tablename type="Usergroup" as="g" /> WHERE ( g.disabled IS NULL or g.disabled = false ) AND uug.group_id = g.id) </if>
	//   <if test="userid.Valid"> AND EXISTS (SELECT * FROM <tablename type="User" as="u" /> WHERE uug.user_id = #{userid}) </if>
	//  </where>
	GetUserAndGroupList(ctx context.Context, userid sql.NullInt64, groupEnabled bool) (func(*UserAndUsergroup) (bool, error), io.Closer)

	// @default SELECT * FROM <tablename name="Role" as="roles" /> WHERE roles.id in
	//  (SELECT role_id from <tablename type="UserAndUsergroup" /> WHERE group_id = #{usergroupID} and user_id = #{userID})
	GetRoleByUsergroupIDAndUserID(ctx context.Context, usergroupID, userID int64) ([]Role, error)
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
	//           WHERE group_id = #{groupid} and user_id = #{userid}  <if test="roleid &gt; 0"> and role_id = #{roleid} </if> limit 1
	HasUserForGroup(ctx context.Context, groupid, userid, roleid int64) (bool, error)

	// @default INSERT INTO <tablename type="UserAndUsergroup"/>(group_id, user_id, role_id)
	//       VALUES(#{groupid}, #{userid}<if test="roleid &gt; 0">, #{roleid}</if><if test="roleid &lt;= 0">, NULL</if>)
	//       ON CONFLICT (group_id, user_id, role_id) DO NOTHING
	AddUserToGroup(ctx context.Context, groupid, userid, roleid int64) error

	// @default DELETE FROM <tablename type="UserAndUsergroup"/>
	//           WHERE group_id = #{groupid} and user_id = #{userid}
	RemoveUserFromGroup(ctx context.Context, groupid, userid int64) error

	// @default DELETE FROM <tablename type="UserAndUsergroup"/>
	//           WHERE user_id = #{userid}
	RemoveUserFromAllGroups(ctx context.Context, userid int64) error
}

func GetUsergroups(ctx context.Context, next func(*Usergroup) (bool, error)) ([]Usergroup, error) {
	var usergroupList []Usergroup
	for {
		var u Usergroup
		ok, err := next(&u)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, err
		}
		if !ok {
			break
		}
		usergroupList = append(usergroupList, u)
	}

	return usergroupList, nil
}
