// Please don't edit this file!
package usermodels

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"reflect"
	"strings"

	gobatis "github.com/runner-mei/GoBatis"
)

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
		{ //// UsergroupQueryer.UsergroupnameExists
			if _, exists := ctx.Statements["UsergroupQueryer.UsergroupnameExists"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT count(*) > 0 FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE name = #{name}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupQueryer.UsergroupnameExists",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupQueryer.UsergroupnameExists"] = stmt
			}
		}
		{ //// UsergroupQueryer.GetUsergroupByID
			if _, exists := ctx.Statements["UsergroupQueryer.GetUsergroupByID"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Usergroup{}),
					[]string{
						"id",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UsergroupQueryer.GetUsergroupByID error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupQueryer.GetUsergroupByID",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupQueryer.GetUsergroupByID"] = stmt
			}
		}
		{ //// UsergroupQueryer.GetUsergroupsByRecursive
			if _, exists := ctx.Statements["UsergroupQueryer.GetUsergroupsByRecursive"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" where id in (\r\n   WITH RECURSIVE ALLGROUPS (ID)  AS (\r\n     SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH\r\n        FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("ug")
				sb.WriteString(" WHERE id = #{id} <foreach collection=\"list\" open=\"AND id in (\" close=\")\">#{item}</foreach>\r\n     UNION ALL\r\n     SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH\r\n        FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("D")
				sb.WriteString(" JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)\r\n   SELECT ID FROM ALLGROUPS ORDER BY PATH)")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupQueryer.GetUsergroupsByRecursive",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupQueryer.GetUsergroupsByRecursive"] = stmt
			}
		}
		{ //// UsergroupQueryer.GetUsergroupByName
			if _, exists := ctx.Statements["UsergroupQueryer.GetUsergroupByName"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Usergroup{}),
					[]string{
						"name",
					},
					[]reflect.Type{
						reflect.TypeOf(new(string)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UsergroupQueryer.GetUsergroupByName error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupQueryer.GetUsergroupByName",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupQueryer.GetUsergroupByName"] = stmt
			}
		}
		{ //// UsergroupQueryer.GetUsergroups
			if _, exists := ctx.Statements["UsergroupQueryer.GetUsergroups"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Usergroup{}),
					[]string{},
					[]reflect.Type{},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UsergroupQueryer.GetUsergroups error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupQueryer.GetUsergroups",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupQueryer.GetUsergroups"] = stmt
			}
		}
		{ //// UsergroupQueryer.GetUserIDsByGroupID
			if _, exists := ctx.Statements["UsergroupQueryer.GetUserIDsByGroupID"]; !exists {
				var sb strings.Builder
				sb.WriteString("<if test=\"recursive\">\r\n SELECT user_id FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("uug")
				sb.WriteString(" where uug.group_id in (\r\n WITH RECURSIVE ALLGROUPS (ID)  AS (\r\n   SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH\r\n      FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("ug")
				sb.WriteString(" WHERE id=#{groupID}\r\n   UNION ALL\r\n   SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH\r\n      FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("D")
				sb.WriteString(" JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)\r\n  SELECT ID FROM ALLGROUPS ORDER BY PATH)\r\n  <if test=\"userEnabled.Valid\"> AND EXISTS (SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u")
				sb.WriteString(" WHERE <if test=\"!userEnabled.Bool\"> NOT </if> ( disabled IS NULL or disabled = false ) AND uug.user_id = u.id) </if>\r\n </if>\r\n <if test=\"!recursive\">\r\n    SELECT user_id FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("uug")
				sb.WriteString(" where uug.group_id = #{groupID}\r\n       <if test=\"userEnabled.Valid\">\r\n         AND EXISTS (\r\n           SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u")
				sb.WriteString("\r\n           WHERE uug.user_id = u.id AND <if test=\"!userEnabled.Bool\"> NOT </if> ( disabled IS NULL or disabled = false )\r\n         )\r\n       </if>\r\n </if>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupQueryer.GetUserIDsByGroupID",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupQueryer.GetUserIDsByGroupID"] = stmt
			}
		}
		{ //// UsergroupQueryer.GetUsersAndGroups
			if _, exists := ctx.Statements["UsergroupQueryer.GetUsersAndGroups"]; !exists {
				var sb strings.Builder
				sb.WriteString("<if test=\"recursive\">\r\n SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("uug")
				sb.WriteString(" where uug.group_id in (\r\n WITH RECURSIVE ALLGROUPS (ID)  AS (\r\n   SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH\r\n      FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("ug")
				sb.WriteString(" WHERE id=#{groupID}\r\n   UNION ALL\r\n   SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH\r\n      FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("D")
				sb.WriteString(" JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)\r\n  SELECT ID FROM ALLGROUPS ORDER BY PATH)\r\n  <if test=\"userEnabled\"> AND EXISTS (SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u")
				sb.WriteString(" WHERE ( disabled IS NULL or disabled = false ) AND uug.user_id = u.id) </if>\r\n </if>\r\n <if test=\"!recursive\">\r\n    SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("uug")
				sb.WriteString(" where uug.group_id = #{groupID}\r\n      <if test=\"userEnabled\"> AND EXISTS (SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u")
				sb.WriteString(" WHERE ( disabled IS NULL or disabled = false ) AND uug.user_id = u.id) </if>\r\n </if>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupQueryer.GetUsersAndGroups",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupQueryer.GetUsersAndGroups"] = stmt
			}
		}
		{ //// UsergroupQueryer.GetUserAndGroupList
			if _, exists := ctx.Statements["UsergroupQueryer.GetUserAndGroupList"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("uug")
				sb.WriteString(" <where>\r\n   <if test=\"groupEnabled\"> EXISTS (SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("g")
				sb.WriteString(" WHERE ( disabled IS NULL or disabled = false ) AND uug.group_id = g.id) </if>\r\n   <if test=\"userid.Valid\"> AND EXISTS (SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u")
				sb.WriteString(" WHERE uug.user_id = #{userid}) </if>\r\n  </where>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupQueryer.GetUserAndGroupList",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupQueryer.GetUserAndGroupList"] = stmt
			}
		}
		return nil
	})
}

func NewUsergroupQueryer(ref gobatis.SqlSession) UsergroupQueryer {
	if ref == nil {
		panic(errors.New("param 'ref' is nil"))
	}
	if reference, ok := ref.(*gobatis.Reference); ok {
		if reference.SqlSession == nil {
			panic(errors.New("param 'ref.SqlSession' is nil"))
		}
	} else if valueReference, ok := ref.(gobatis.Reference); ok {
		if valueReference.SqlSession == nil {
			panic(errors.New("param 'ref.SqlSession' is nil"))
		}
	}
	return &UsergroupQueryerImpl{session: ref}
}

type UsergroupQueryerImpl struct {
	session gobatis.SqlSession
}

func (impl *UsergroupQueryerImpl) UsergroupnameExists(ctx context.Context, name string) (bool, error) {
	var instance bool
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UsergroupQueryer.UsergroupnameExists",
		[]string{
			"name",
		},
		[]interface{}{
			name,
		}).Scan(&nullable)
	if err != nil {
		return false, err
	}
	if !nullable.Valid {
		return false, sql.ErrNoRows
	}

	return instance, nil
}

func (impl *UsergroupQueryerImpl) GetUsergroupByID(ctx context.Context, id int64) func(*Usergroup) error {
	result := impl.session.SelectOne(ctx, "UsergroupQueryer.GetUsergroupByID",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
	return func(value *Usergroup) error {
		return result.Scan(value)
	}
}

func (impl *UsergroupQueryerImpl) GetUsergroupsByRecursive(ctx context.Context, id int64, list ...int64) (func(*Usergroup) (bool, error), io.Closer) {
	results := impl.session.Select(ctx, "UsergroupQueryer.GetUsergroupsByRecursive",
		[]string{
			"id",
			"list",
		},
		[]interface{}{
			id,
			list,
		})
	return func(value *Usergroup) (bool, error) {
		if !results.Next() {
			if results.Err() == sql.ErrNoRows {
				return false, nil
			}
			return false, results.Err()
		}
		return true, results.Scan(value)
	}, results
}

func (impl *UsergroupQueryerImpl) GetUsergroupByName(ctx context.Context, name string) func(*Usergroup) error {
	result := impl.session.SelectOne(ctx, "UsergroupQueryer.GetUsergroupByName",
		[]string{
			"name",
		},
		[]interface{}{
			name,
		})
	return func(value *Usergroup) error {
		return result.Scan(value)
	}
}

func (impl *UsergroupQueryerImpl) GetUsergroups(ctx context.Context) (func(*Usergroup) (bool, error), io.Closer) {
	results := impl.session.Select(ctx, "UsergroupQueryer.GetUsergroups",
		[]string{},
		[]interface{}{})
	return func(value *Usergroup) (bool, error) {
		if !results.Next() {
			if results.Err() == sql.ErrNoRows {
				return false, nil
			}
			return false, results.Err()
		}
		return true, results.Scan(value)
	}, results
}

func (impl *UsergroupQueryerImpl) GetUserIDsByGroupID(ctx context.Context, groupID int64, recursive bool, userEnabled sql.NullBool) ([]int64, error) {
	var instances []int64
	results := impl.session.Select(ctx, "UsergroupQueryer.GetUserIDsByGroupID",
		[]string{
			"groupID",
			"recursive",
			"userEnabled",
		},
		[]interface{}{
			groupID,
			recursive,
			userEnabled,
		})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (impl *UsergroupQueryerImpl) GetUsersAndGroups(ctx context.Context, groupID int64, recursive bool, userEnabled bool) ([]UserAndUsergroup, error) {
	var instances []UserAndUsergroup
	results := impl.session.Select(ctx, "UsergroupQueryer.GetUsersAndGroups",
		[]string{
			"groupID",
			"recursive",
			"userEnabled",
		},
		[]interface{}{
			groupID,
			recursive,
			userEnabled,
		})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (impl *UsergroupQueryerImpl) GetUserAndGroupList(ctx context.Context, userid sql.NullInt64, groupEnabled bool) (func(*UserAndUsergroup) (bool, error), io.Closer) {
	results := impl.session.Select(ctx, "UsergroupQueryer.GetUserAndGroupList",
		[]string{
			"userid",
			"groupEnabled",
		},
		[]interface{}{
			userid,
			groupEnabled,
		})
	return func(value *UserAndUsergroup) (bool, error) {
		if !results.Next() {
			if results.Err() == sql.ErrNoRows {
				return false, nil
			}
			return false, results.Err()
		}
		return true, results.Scan(value)
	}, results
}

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
		{ //// UsergroupDao.CreateUsergroup
			if _, exists := ctx.Statements["UsergroupDao.CreateUsergroup"]; !exists {
				sqlStr, err := gobatis.GenerateInsertSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Usergroup{}),
					[]string{
						"usergroup",
					},
					[]reflect.Type{
						reflect.TypeOf((*Usergroup)(nil)),
					}, false)
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UsergroupDao.CreateUsergroup error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupDao.CreateUsergroup",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupDao.CreateUsergroup"] = stmt
			}
		}
		{ //// UsergroupDao.UpdateUsergroup
			if _, exists := ctx.Statements["UsergroupDao.UpdateUsergroup"]; !exists {
				sqlStr, err := gobatis.GenerateUpdateSQL(ctx.Dialect, ctx.Mapper,
					"usergroup.", reflect.TypeOf(&Usergroup{}),
					[]string{
						"id",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UsergroupDao.UpdateUsergroup error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupDao.UpdateUsergroup",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupDao.UpdateUsergroup"] = stmt
			}
		}
		{ //// UsergroupDao.DeleteUsergroup
			if _, exists := ctx.Statements["UsergroupDao.DeleteUsergroup"]; !exists {
				var sb strings.Builder
				sb.WriteString("<if test=\"recursive\">\r\n SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" where id in (\r\n   WITH RECURSIVE ALLGROUPS (ID)  AS (\r\n     SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH\r\n        FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("ug")
				sb.WriteString(" WHERE id=#{groupID}\r\n     UNION ALL\r\n     SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH\r\n        FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("D")
				sb.WriteString(" JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)\r\n   SELECT ID FROM ALLGROUPS ORDER BY PATH)\r\n </if>\r\n <if test=\"!recursive\">\r\n    SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" where id = #{id}\r\n </if>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupDao.DeleteUsergroup",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupDao.DeleteUsergroup"] = stmt
			}
		}
		{ //// UsergroupDao.HasUserForGroup
			if _, exists := ctx.Statements["UsergroupDao.HasUserForGroup"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT count(*) > 0 FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n           WHERE group_id = #{groupid} and user_id = #{userid}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupDao.HasUserForGroup",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupDao.HasUserForGroup"] = stmt
			}
		}
		{ //// UsergroupDao.AddUserToGroup
			if _, exists := ctx.Statements["UsergroupDao.AddUserToGroup"]; !exists {
				var sb strings.Builder
				sb.WriteString("INSERT INTO ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("(group_id, user_id)\r\n       VALUES(#{groupid}, #{userid})\r\n       ON CONFLICT (group_id, user_id)\r\n       DO NOTHING")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupDao.AddUserToGroup",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupDao.AddUserToGroup"] = stmt
			}
		}
		{ //// UsergroupDao.RemoveUserFromGroup
			if _, exists := ctx.Statements["UsergroupDao.RemoveUserFromGroup"]; !exists {
				var sb strings.Builder
				sb.WriteString("DELETE FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n           WHERE group_id = #{groupid} and user_id = #{userid}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupDao.RemoveUserFromGroup",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupDao.RemoveUserFromGroup"] = stmt
			}
		}
		{ //// UsergroupDao.RemoveUserFromAllGroups
			if _, exists := ctx.Statements["UsergroupDao.RemoveUserFromAllGroups"]; !exists {
				var sb strings.Builder
				sb.WriteString("DELETE FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n           WHERE user_id = #{userid}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UsergroupDao.RemoveUserFromAllGroups",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UsergroupDao.RemoveUserFromAllGroups"] = stmt
			}
		}
		return nil
	})
}

func NewUsergroupDao(ref gobatis.SqlSession, usergroupQueryer UsergroupQueryer) UsergroupDao {
	if ref == nil {
		panic(errors.New("param 'ref' is nil"))
	}
	if reference, ok := ref.(*gobatis.Reference); ok {
		if reference.SqlSession == nil {
			panic(errors.New("param 'ref.SqlSession' is nil"))
		}
	} else if valueReference, ok := ref.(gobatis.Reference); ok {
		if valueReference.SqlSession == nil {
			panic(errors.New("param 'ref.SqlSession' is nil"))
		}
	}
	return &UsergroupDaoImpl{UsergroupQueryer: usergroupQueryer,
		session: ref}
}

type UsergroupDaoImpl struct {
	UsergroupQueryer
	session gobatis.SqlSession
}

func (impl *UsergroupDaoImpl) CreateUsergroup(ctx context.Context, usergroup *Usergroup) (int64, error) {
	return impl.session.Insert(ctx, "UsergroupDao.CreateUsergroup",
		[]string{
			"usergroup",
		},
		[]interface{}{
			usergroup,
		})
}

func (impl *UsergroupDaoImpl) UpdateUsergroup(ctx context.Context, id int64, usergroup *Usergroup) (int64, error) {
	return impl.session.Update(ctx, "UsergroupDao.UpdateUsergroup",
		[]string{
			"id",
			"usergroup",
		},
		[]interface{}{
			id,
			usergroup,
		})
}

func (impl *UsergroupDaoImpl) DeleteUsergroup(ctx context.Context, id int64, recursive bool) (int64, error) {
	return impl.session.Delete(ctx, "UsergroupDao.DeleteUsergroup",
		[]string{
			"id",
			"recursive",
		},
		[]interface{}{
			id,
			recursive,
		})
}

func (impl *UsergroupDaoImpl) HasUserForGroup(ctx context.Context, userid int64, roleid int64) (bool, error) {
	var instance bool
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UsergroupDao.HasUserForGroup",
		[]string{
			"userid",
			"roleid",
		},
		[]interface{}{
			userid,
			roleid,
		}).Scan(&nullable)
	if err != nil {
		return false, err
	}
	if !nullable.Valid {
		return false, sql.ErrNoRows
	}

	return instance, nil
}

func (impl *UsergroupDaoImpl) AddUserToGroup(ctx context.Context, groupid int64, userid int64) error {
	_, err := impl.session.Insert(ctx, "UsergroupDao.AddUserToGroup",
		[]string{
			"groupid",
			"userid",
		},
		[]interface{}{
			groupid,
			userid,
		},
		true)
	return err
}

func (impl *UsergroupDaoImpl) RemoveUserFromGroup(ctx context.Context, groupid int64, userid int64) error {
	_, err := impl.session.Delete(ctx, "UsergroupDao.RemoveUserFromGroup",
		[]string{
			"groupid",
			"userid",
		},
		[]interface{}{
			groupid,
			userid,
		})
	return err
}

func (impl *UsergroupDaoImpl) RemoveUserFromAllGroups(ctx context.Context, userid int64) error {
	_, err := impl.session.Delete(ctx, "UsergroupDao.RemoveUserFromAllGroups",
		[]string{
			"userid",
		},
		[]interface{}{
			userid,
		})
	return err
}
