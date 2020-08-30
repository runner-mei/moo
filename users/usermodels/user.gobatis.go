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
		{ //// OnlineUserDao.List
			if _, exists := ctx.Statements["OnlineUserDao.List"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&OnlineUser{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" <if test=\"isNotEmpty(interval)\">WHERE (updated_at + #{interval}::INTERVAL) &gt; now() </if>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "OnlineUserDao.List",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OnlineUserDao.List"] = stmt
			}
		}
		{ //// OnlineUserDao.Create
			if _, exists := ctx.Statements["OnlineUserDao.Create"]; !exists {

				sqlStr, err := gobatis.GenerateInsertSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&OnlineUser{}),
					[]string{
						"userID",
						"address",
						"uuid",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
						reflect.TypeOf(new(string)).Elem(),
						reflect.TypeOf(new(string)).Elem(),
					}, false)
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate OnlineUserDao.Create error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "OnlineUserDao.Create",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OnlineUserDao.Create"] = stmt
			}
		}
		{ //// OnlineUserDao.CreateOrTouch
			if _, exists := ctx.Statements["OnlineUserDao.CreateOrTouch"]; !exists {
				var sb strings.Builder
				sb.WriteString("INSERT INTO ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&OnlineUser{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("(user_id, address, uuid, created_at, updated_at)\r\n          VALUES(#{userID}, #{address}, #{uuid}, now(), now())  ON CONFLICT (user_id, address)\r\n          DO UPDATE SET updated = now()")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "OnlineUserDao.CreateOrTouch",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OnlineUserDao.CreateOrTouch"] = stmt
			}
		}
		{ //// OnlineUserDao.Touch
			if _, exists := ctx.Statements["OnlineUserDao.Touch"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&OnlineUser{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" SET updated_at = now() WHERE user_id = #{userID} AND address = #{address}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "OnlineUserDao.Touch",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OnlineUserDao.Touch"] = stmt
			}
		}
		{ //// OnlineUserDao.Delete
			if _, exists := ctx.Statements["OnlineUserDao.Delete"]; !exists {
				var sb strings.Builder
				sb.WriteString("DELETE FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&OnlineUser{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE user_id = #{userID} AND address = #{address}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "OnlineUserDao.Delete",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OnlineUserDao.Delete"] = stmt
			}
		}
		return nil
	})
}

func NewOnlineUserDao(ref gobatis.SqlSession) OnlineUserDao {
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
	return &OnlineUserDaoImpl{session: ref}
}

type OnlineUserDaoImpl struct {
	session gobatis.SqlSession
}

func (impl *OnlineUserDaoImpl) List(ctx context.Context, interval string) ([]OnlineUser, error) {
	var instances []OnlineUser
	results := impl.session.Select(ctx, "OnlineUserDao.List",
		[]string{
			"interval",
		},
		[]interface{}{
			interval,
		})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (impl *OnlineUserDaoImpl) Create(ctx context.Context, userID int64, address string, uuid string) (int64, error) {
	return impl.session.Insert(ctx, "OnlineUserDao.Create",
		[]string{
			"userID",
			"address",
			"uuid",
		},
		[]interface{}{
			userID,
			address,
			uuid,
		})
}

func (impl *OnlineUserDaoImpl) CreateOrTouch(ctx context.Context, userID int64, address string, uuid string) (int64, error) {
	return impl.session.Insert(ctx, "OnlineUserDao.CreateOrTouch",
		[]string{
			"userID",
			"address",
			"uuid",
		},
		[]interface{}{
			userID,
			address,
			uuid,
		})
}

func (impl *OnlineUserDaoImpl) Touch(ctx context.Context, userID int64, address string, uuid string) (int64, error) {
	return impl.session.Update(ctx, "OnlineUserDao.Touch",
		[]string{
			"userID",
			"address",
			"uuid",
		},
		[]interface{}{
			userID,
			address,
			uuid,
		})
}

func (impl *OnlineUserDaoImpl) Delete(ctx context.Context, userID int64, address string) (int64, error) {
	return impl.session.Delete(ctx, "OnlineUserDao.Delete",
		[]string{
			"userID",
			"address",
		},
		[]interface{}{
			userID,
			address,
		})
}

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
		{ //// UserQueryer.RolenameExists
			if _, exists := ctx.Statements["UserQueryer.RolenameExists"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT count(*) > 0 FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE name = #{name}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.RolenameExists",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.RolenameExists"] = stmt
			}
		}
		{ //// UserQueryer.UsernameExists
			if _, exists := ctx.Statements["UserQueryer.UsernameExists"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT count(*) > 0 FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE lower(name) = lower(#{name})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.UsernameExists",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.UsernameExists"] = stmt
			}
		}
		{ //// UserQueryer.NicknameExists
			if _, exists := ctx.Statements["UserQueryer.NicknameExists"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT count(*) > 0 FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE nickname = #{name}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.NicknameExists",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.NicknameExists"] = stmt
			}
		}
		{ //// UserQueryer.GetRoleCount
			if _, exists := ctx.Statements["UserQueryer.GetRoleCount"]; !exists {
				sqlStr, err := gobatis.GenerateCountSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Role{}),
					[]string{
						"nameLike",
					},
					[]reflect.Type{
						reflect.TypeOf(new(string)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetRoleCount error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetRoleCount",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetRoleCount"] = stmt
			}
		}
		{ //// UserQueryer.GetRoles
			if _, exists := ctx.Statements["UserQueryer.GetRoles"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Role{}),
					[]string{
						"nameLike",
						"type",
						"offset",
						"limit",
					},
					[]reflect.Type{
						reflect.TypeOf(new(string)).Elem(),
						reflect.TypeOf(&sql.NullInt64{}).Elem(),
						reflect.TypeOf(new(int64)).Elem(),
						reflect.TypeOf(new(int64)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetRoles error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetRoles",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetRoles"] = stmt
			}
		}
		{ //// UserQueryer.GetRoleByID
			if _, exists := ctx.Statements["UserQueryer.GetRoleByID"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Role{}),
					[]string{
						"id",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetRoleByID error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetRoleByID",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetRoleByID"] = stmt
			}
		}
		{ //// UserQueryer.GetRoleByName
			if _, exists := ctx.Statements["UserQueryer.GetRoleByName"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Role{}),
					[]string{
						"name",
					},
					[]reflect.Type{
						reflect.TypeOf(new(string)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetRoleByName error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetRoleByName",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetRoleByName"] = stmt
			}
		}
		{ //// UserQueryer.GetRolesByNames
			if _, exists := ctx.Statements["UserQueryer.GetRolesByNames"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Role{}),
					[]string{
						"name",
					},
					[]reflect.Type{
						reflect.TypeOf([]string{}),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetRolesByNames error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetRolesByNames",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetRolesByNames"] = stmt
			}
		}
		{ //// UserQueryer.GetUserByID
			if _, exists := ctx.Statements["UserQueryer.GetUserByID"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&User{}),
					[]string{
						"id",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetUserByID error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUserByID",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUserByID"] = stmt
			}
		}
		{ //// UserQueryer.GetUserByName
			if _, exists := ctx.Statements["UserQueryer.GetUserByName"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE lower(name) = lower(#{name})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUserByName",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUserByName"] = stmt
			}
		}
		{ //// UserQueryer.GetUserByNickname
			if _, exists := ctx.Statements["UserQueryer.GetUserByNickname"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE lower(nickname) = lower(#{nickname})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUserByNickname",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUserByNickname"] = stmt
			}
		}
		{ //// UserQueryer.GetUserByNameOrNickname
			if _, exists := ctx.Statements["UserQueryer.GetUserByNameOrNickname"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE lower(name) = lower(#{name}) OR lower(nickname) = lower(#{nickname})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUserByNameOrNickname",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUserByNameOrNickname"] = stmt
			}
		}
		{ //// UserQueryer.GetUserCount
			if _, exists := ctx.Statements["UserQueryer.GetUserCount"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT count(*) FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("users")
				sb.WriteString(" <where>\r\n  <if test=\"len(params.Roles) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" where u2r.role_id in (<foreach collection=\"params.Roles\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND</if>\r\n  <if test=\"len(params.ExcludeRoles) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" where u2r.role_id not in (<foreach collection=\"params.ExcludeRoles\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND</if>\r\n  <if test=\"len(params.Rolenames) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" JOIN ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("r")
				sb.WriteString(" ON u2r.role_id = r.id\r\n      WHERE r.name in (<foreach collection=\"params.Rolenames\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND\r\n  </if>\r\n  <if test=\"len(params.ExcludeRolenames) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" JOIN ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("r")
				sb.WriteString(" ON u2r.role_id = r.id\r\n      WHERE r.name not in (<foreach collection=\"params.ExcludeRolenames\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND\r\n  </if>\r\n  <if test=\"isNotEmpty(params.NameLike)\"> (users.name like <like value=\"params.NameLike\" /> OR users.nickname like <like value=\"params.NameLike\" />) AND</if>\r\n  <if test=\"params.CanLogin.Valid\"> users.can_login = #{params.CanLogin} AND </if>\r\n  <if test=\"params.Enabled.Valid\"> (<if test=\"!params.Enabled.Bool\"> NOT </if> ( users.disabled IS NULL OR users.disabled = false )) AND </if>\r\n  <if test=\"len(params.UsergroupIDs) &gt; 0 || len(params.JobPositions) &gt; 0\">\r\n     exists (select * from ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" as u2g where u2g.user_id = users.id\r\n         <if test=\"len(params.JobPositions) == 1\"><foreach collection=\"params.JobPositions\" open=\" AND u2g.role_id = \" separator=\",\" close=\")\">#{item}</foreach></if>\r\n         <if test=\"len(params.JobPositions) &gt; 1\"><foreach collection=\"params.JobPositions\" open=\" AND u2g.role_id in (\" separator=\",\" close=\")\">#{item}</foreach></if>\r\n         <if test=\"len(params.UsergroupIDs) &gt; 0\">\r\n           <if test=\"params.UsergroupRecursive\">\r\n             AND u2g.group_id in (WITH RECURSIVE ALLGROUPS (ID)  AS (\r\n                SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH\r\n                FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("ug")
				sb.WriteString(" WHERE\r\n                   <if test=\"len(params.UsergroupIDs) == 1\"> ug.id = <foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach></if>\r\n                   <if test=\"len(params.UsergroupIDs) &gt; 1\"> ug.id in (<foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach>)</if>\r\n                   UNION ALL\r\n                SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH\r\n                   FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("D")
				sb.WriteString(" JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)\r\n                SELECT ID FROM ALLGROUPS))\r\n           </if>\r\n           <if test=\"!params.UsergroupRecursive\">\r\n                  <if test=\"len(params.UsergroupIDs) == 1\"> and u2g.group_id = <foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach></if>\r\n                  <if test=\"len(params.UsergroupIDs) &gt; 1\"> and u2g.group_id in (<foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach>)</if>)\r\n           </if>\r\n         </if>\r\n  </if>\r\n  </where>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUserCount",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUserCount"] = stmt
			}
		}
		{ //// UserQueryer.GetUsers
			if _, exists := ctx.Statements["UserQueryer.GetUsers"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("users")
				sb.WriteString(" <where>\r\n  <if test=\"len(params.Roles) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" where u2r.role_id in (<foreach collection=\"params.Roles\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND</if>\r\n  <if test=\"len(params.ExcludeRoles) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" where u2r.role_id not in (<foreach collection=\"params.ExcludeRoles\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND</if>\r\n  <if test=\"len(params.Rolenames) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" JOIN ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("r")
				sb.WriteString(" ON u2r.role_id = r.id\r\n      WHERE r.name in (<foreach collection=\"params.Rolenames\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND\r\n  </if>\r\n  <if test=\"len(params.ExcludeRolenames) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" JOIN ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("r")
				sb.WriteString(" ON u2r.role_id = r.id\r\n      WHERE r.name not in (<foreach collection=\"params.ExcludeRolenames\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND\r\n  </if>\r\n  <if test=\"isNotEmpty(params.NameLike)\"> (users.name like <like value=\"params.NameLike\" /> OR users.nickname like <like value=\"params.NameLike\" />) AND</if>\r\n  <if test=\"params.CanLogin.Valid\"> users.can_login = #{params.CanLogin} AND </if>\r\n  <if test=\"params.Enabled.Valid\"> (<if test=\"!params.Enabled.Bool\"> NOT </if> ( users.disabled IS NULL OR users.disabled = false )) AND </if>\r\n  <if test=\"len(params.UsergroupIDs) &gt; 0 || len(params.JobPositions) &gt; 0\">\r\n     exists (select * from ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" as u2g where u2g.user_id = users.id\r\n         <if test=\"len(params.JobPositions) == 1\"><foreach collection=\"params.JobPositions\" open=\" AND u2g.role_id = \" separator=\",\" close=\")\">#{item}</foreach></if>\r\n         <if test=\"len(params.JobPositions) &gt; 1\"><foreach collection=\"params.JobPositions\" open=\" AND u2g.role_id in (\" separator=\",\" close=\")\">#{item}</foreach></if>\r\n         <if test=\"len(params.UsergroupIDs) &gt; 0\">\r\n           <if test=\"params.UsergroupRecursive\">\r\n             AND u2g.group_id in (WITH RECURSIVE ALLGROUPS (ID)  AS (\r\n                SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH\r\n                FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("ug")
				sb.WriteString(" WHERE\r\n                   <if test=\"len(params.UsergroupIDs) == 1\"> ug.id = <foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach></if>\r\n                   <if test=\"len(params.UsergroupIDs) &gt; 1\"> ug.id in (<foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach>)</if>\r\n                   UNION ALL\r\n                SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH\r\n                   FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("D")
				sb.WriteString(" JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)\r\n                SELECT ID FROM ALLGROUPS))\r\n           </if>\r\n           <if test=\"!params.UsergroupRecursive\">\r\n                  <if test=\"len(params.UsergroupIDs) == 1\"> and u2g.group_id = <foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach></if>\r\n                  <if test=\"len(params.UsergroupIDs) &gt; 1\"> and u2g.group_id in (<foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach>)</if>)\r\n           </if>\r\n         </if>\r\n  </if>\r\n  </where>\r\n  <pagination />\r\n  <order_by />")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUsers",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUsers"] = stmt
			}
		}
		{ //// UserQueryer.GetUserIDs
			if _, exists := ctx.Statements["UserQueryer.GetUserIDs"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT users.id FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("users")
				sb.WriteString(" <where>\r\n  <if test=\"len(params.Roles) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" where u2r.role_id in (<foreach collection=\"params.Roles\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND</if>\r\n  <if test=\"len(params.ExcludeRoles) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" where u2r.role_id not in (<foreach collection=\"params.ExcludeRoles\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND</if>\r\n  <if test=\"len(params.Rolenames) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" JOIN ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("r")
				sb.WriteString(" ON u2r.role_id = r.id\r\n      WHERE r.name in (<foreach collection=\"params.Rolenames\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND\r\n  </if>\r\n  <if test=\"len(params.ExcludeRolenames) &gt; 0\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" JOIN ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("r")
				sb.WriteString(" ON u2r.role_id = r.id\r\n      WHERE r.name not in (<foreach collection=\"params.ExcludeRolenames\" separator=\",\">#{item}</foreach>) AND u2r.user_id = users.id) AND\r\n  </if>\r\n  <if test=\"isNotEmpty(params.NameLike)\"> (users.name like <like value=\"params.NameLike\" /> OR users.nickname like <like value=\"params.NameLike\" />) AND</if>\r\n  <if test=\"params.CanLogin.Valid\"> users.can_login = #{params.CanLogin} AND </if>\r\n  <if test=\"params.Enabled.Valid\"> (<if test=\"!params.Enabled.Bool\"> NOT </if> ( users.disabled IS NULL OR users.disabled = false )) AND </if>\r\n  <if test=\"len(params.UsergroupIDs) &gt; 0 || len(params.JobPositions) &gt; 0\">\r\n     exists (select * from ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" as u2g where u2g.user_id = users.id\r\n         <if test=\"len(params.JobPositions) == 1\"><foreach collection=\"params.JobPositions\" open=\" AND u2g.role_id = \" separator=\",\" close=\")\">#{item}</foreach></if>\r\n         <if test=\"len(params.JobPositions) &gt; 1\"><foreach collection=\"params.JobPositions\" open=\" AND u2g.role_id in (\" separator=\",\" close=\")\">#{item}</foreach></if>\r\n         <if test=\"len(params.UsergroupIDs) &gt; 0\">\r\n           <if test=\"params.UsergroupRecursive\">\r\n             AND u2g.group_id in (WITH RECURSIVE ALLGROUPS (ID)  AS (\r\n                SELECT ID, name, PARENT_ID, ARRAY[ID] AS PATH, 1 AS DEPTH\r\n                FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("ug")
				sb.WriteString(" WHERE\r\n                   <if test=\"len(params.UsergroupIDs) == 1\"> ug.id = <foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach></if>\r\n                   <if test=\"len(params.UsergroupIDs) &gt; 1\"> ug.id in (<foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach>)</if>\r\n                   UNION ALL\r\n                SELECT  D.ID, D.NAME, D.PARENT_ID, ALLGROUPS.PATH || D.ID, ALLGROUPS.DEPTH + 1 AS DEPTH\r\n                   FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Usergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("D")
				sb.WriteString(" JOIN ALLGROUPS ON D.PARENT_ID = ALLGROUPS.ID)\r\n                SELECT ID FROM ALLGROUPS))\r\n           </if>\r\n           <if test=\"!params.UsergroupRecursive\">\r\n                  <if test=\"len(params.UsergroupIDs) == 1\"> and u2g.group_id = <foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach></if>\r\n                  <if test=\"len(params.UsergroupIDs) &gt; 1\"> and u2g.group_id in (<foreach collection=\"params.UsergroupIDs\" separator=\",\">#{item}</foreach>)</if>)\r\n           </if>\r\n         </if>\r\n  </if>\r\n  </where>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUserIDs",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUserIDs"] = stmt
			}
		}
		{ //// UserQueryer.GetRolesByUserID
			if _, exists := ctx.Statements["UserQueryer.GetRolesByUserID"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("roles")
				sb.WriteString(" WHERE\r\n  exists (select * from ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" as users_roles\r\n     where users_roles.role_id = roles.id and users_roles.user_id = #{userID})\r\n  OR exists (select * from ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" as u2g\r\n     where u2g.role_id = roles.id and u2g.user_id = #{userID})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetRolesByUserID",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetRolesByUserID"] = stmt
			}
		}
		{ //// UserQueryer.ReadProfile
			if _, exists := ctx.Statements["UserQueryer.ReadProfile"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT value FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserProfile{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE id = #{userID} AND name = #{name}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.ReadProfile",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.ReadProfile"] = stmt
			}
		}
		{ //// UserQueryer.WriteProfile
			if _, exists := ctx.Statements["UserQueryer.WriteProfile"]; !exists {
				var sb strings.Builder
				sb.WriteString("INSERT INTO ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserProfile{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" (id, name, value) VALUES(#{userID}, #{name}, #{value})\r\n     ON CONFLICT (id, name) DO UPDATE SET value = excluded.value")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.WriteProfile",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.WriteProfile"] = stmt
			}
		}
		{ //// UserQueryer.DeleteProfile
			if _, exists := ctx.Statements["UserQueryer.DeleteProfile"]; !exists {
				var sb strings.Builder
				sb.WriteString("DELETE FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserProfile{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE id=#{userID} AND name=#{name}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.DeleteProfile",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.DeleteProfile"] = stmt
			}
		}
		{ //// UserQueryer.GetUserAndRoleList
			if _, exists := ctx.Statements["UserQueryer.GetUserAndRoleList"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&UserAndRole{}),
					[]string{},
					[]reflect.Type{},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetUserAndRoleList error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUserAndRoleList",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUserAndRoleList"] = stmt
			}
		}
		{ //// UserQueryer.GetUsernames
			if _, exists := ctx.Statements["UserQueryer.GetUsernames"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT id, name FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("users")
				sb.WriteString(" <where>\r\n  <if test=\"!isJobPostion.Valid\">\r\n       <if test=\"groupID.Valid\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2g")
				sb.WriteString(" WHERE u2g.user_id = users.id AND u2g.group_id = #{groupID})</if>\r\n       <if test=\"roleID.Valid\"> AND\r\n            (EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2r")
				sb.WriteString(" WHERE u2r.user_id = users.id AND u2r.role_id = #{roleID}) OR\r\n             EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2g")
				sb.WriteString(" WHERE u2g.user_id = users.id AND u2g.role_id = #{roleID}))\r\n       </if>\r\n  </if>\r\n  <if test=\"isJobPostion.Valid\">\r\n    <if test=\"groupID.Valid\">EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2g")
				sb.WriteString(" WHERE u2g.user_id = users.id AND u2g.group_id = #{groupID}\r\n        <if test=\"roleID.Valid\">AND u2g.role_id = #{roleID}</if>)</if>\r\n    <if test=\"roleID.Valid\">AND EXISTS(SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndUsergroup{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" AS ")
				sb.WriteString("u2g")
				sb.WriteString(" WHERE u2g.user_id = users.id AND u2g.role_id = #{roleID})</if>\r\n  </if>\r\n </where>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetUsernames",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetUsernames"] = stmt
			}
		}
		{ //// UserQueryer.GetRolenames
			if _, exists := ctx.Statements["UserQueryer.GetRolenames"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT id, name FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&Role{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" <if test=\"type.Valid\"> WHERE type = #{type} </if>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetRolenames",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetRolenames"] = stmt
			}
		}
		return nil
	})
}

func NewUserQueryer(ref gobatis.SqlSession) UserQueryer {
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
	return &UserQueryerImpl{session: ref}
}

type UserQueryerImpl struct {
	session gobatis.SqlSession
}

func (impl *UserQueryerImpl) RolenameExists(ctx context.Context, name string) (bool, error) {
	var instance bool
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UserQueryer.RolenameExists",
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

func (impl *UserQueryerImpl) UsernameExists(ctx context.Context, name string) (bool, error) {
	var instance bool
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UserQueryer.UsernameExists",
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

func (impl *UserQueryerImpl) NicknameExists(ctx context.Context, name string) (bool, error) {
	var instance bool
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UserQueryer.NicknameExists",
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

func (impl *UserQueryerImpl) GetRoleCount(ctx context.Context, nameLike string) (int64, error) {
	var instance int64
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UserQueryer.GetRoleCount",
		[]string{
			"nameLike",
		},
		[]interface{}{
			nameLike,
		}).Scan(&nullable)
	if err != nil {
		return 0, err
	}
	if !nullable.Valid {
		return 0, sql.ErrNoRows
	}

	return instance, nil
}

func (impl *UserQueryerImpl) GetRoles(ctx context.Context, nameLike string, _type sql.NullInt64, offset int64, limit int64) (func(*Role) (bool, error), io.Closer) {
	results := impl.session.Select(ctx, "UserQueryer.GetRoles",
		[]string{
			"nameLike",
			"type",
			"offset",
			"limit",
		},
		[]interface{}{
			nameLike,
			_type,
			offset,
			limit,
		})
	return func(value *Role) (bool, error) {
		if !results.Next() {
			if results.Err() == sql.ErrNoRows {
				return false, nil
			}
			return false, results.Err()
		}
		return true, results.Scan(value)
	}, results
}

func (impl *UserQueryerImpl) GetRoleByID(ctx context.Context, id int64) func(*Role) error {
	result := impl.session.SelectOne(ctx, "UserQueryer.GetRoleByID",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
	return func(value *Role) error {
		return result.Scan(value)
	}
}

func (impl *UserQueryerImpl) GetRoleByName(ctx context.Context, name string) func(*Role) error {
	result := impl.session.SelectOne(ctx, "UserQueryer.GetRoleByName",
		[]string{
			"name",
		},
		[]interface{}{
			name,
		})
	return func(value *Role) error {
		return result.Scan(value)
	}
}

func (impl *UserQueryerImpl) GetRolesByNames(ctx context.Context, name []string) (func(*Role) (bool, error), io.Closer) {
	results := impl.session.Select(ctx, "UserQueryer.GetRolesByNames",
		[]string{
			"name",
		},
		[]interface{}{
			name,
		})
	return func(value *Role) (bool, error) {
		if !results.Next() {
			if results.Err() == sql.ErrNoRows {
				return false, nil
			}
			return false, results.Err()
		}
		return true, results.Scan(value)
	}, results
}

func (impl *UserQueryerImpl) GetUserByID(ctx context.Context, id int64) func(*User) error {
	result := impl.session.SelectOne(ctx, "UserQueryer.GetUserByID",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
	return func(value *User) error {
		return result.Scan(value)
	}
}

func (impl *UserQueryerImpl) GetUserByName(ctx context.Context, name string) func(*User) error {
	result := impl.session.SelectOne(ctx, "UserQueryer.GetUserByName",
		[]string{
			"name",
		},
		[]interface{}{
			name,
		})
	return func(value *User) error {
		return result.Scan(value)
	}
}

func (impl *UserQueryerImpl) GetUserByNickname(ctx context.Context, nickname string) func(*User) error {
	result := impl.session.SelectOne(ctx, "UserQueryer.GetUserByNickname",
		[]string{
			"nickname",
		},
		[]interface{}{
			nickname,
		})
	return func(value *User) error {
		return result.Scan(value)
	}
}

func (impl *UserQueryerImpl) GetUserByNameOrNickname(ctx context.Context, name string, nickname string) func(*User) error {
	result := impl.session.SelectOne(ctx, "UserQueryer.GetUserByNameOrNickname",
		[]string{
			"name",
			"nickname",
		},
		[]interface{}{
			name,
			nickname,
		})
	return func(value *User) error {
		return result.Scan(value)
	}
}

func (impl *UserQueryerImpl) GetUserCount(ctx context.Context, params *UserQueryParams) (int64, error) {
	var instance int64
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UserQueryer.GetUserCount",
		[]string{
			"params",
		},
		[]interface{}{
			params,
		}).Scan(&nullable)
	if err != nil {
		return 0, err
	}
	if !nullable.Valid {
		return 0, sql.ErrNoRows
	}

	return instance, nil
}

func (impl *UserQueryerImpl) GetUsers(ctx context.Context, params *UserQueryParams, offset int64, limit int64, sort string) (func(*User) (bool, error), io.Closer) {
	results := impl.session.Select(ctx, "UserQueryer.GetUsers",
		[]string{
			"params",
			"offset",
			"limit",
			"sort",
		},
		[]interface{}{
			params,
			offset,
			limit,
			sort,
		})
	return func(value *User) (bool, error) {
		if !results.Next() {
			if results.Err() == sql.ErrNoRows {
				return false, nil
			}
			return false, results.Err()
		}
		return true, results.Scan(value)
	}, results
}

func (impl *UserQueryerImpl) GetUserIDs(ctx context.Context, params *UserQueryParams) ([]int64, error) {
	var instances []int64
	results := impl.session.Select(ctx, "UserQueryer.GetUserIDs",
		[]string{
			"params",
		},
		[]interface{}{
			params,
		})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (impl *UserQueryerImpl) GetRolesByUserID(ctx context.Context, userID int64) ([]Role, error) {
	var instances []Role
	results := impl.session.Select(ctx, "UserQueryer.GetRolesByUserID",
		[]string{
			"userID",
		},
		[]interface{}{
			userID,
		})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (impl *UserQueryerImpl) ReadProfile(ctx context.Context, userID int64, name string) (string, error) {
	var instance string
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UserQueryer.ReadProfile",
		[]string{
			"userID",
			"name",
		},
		[]interface{}{
			userID,
			name,
		}).Scan(&nullable)
	if err != nil {
		return "", err
	}
	if !nullable.Valid {
		return "", sql.ErrNoRows
	}

	return instance, nil
}

func (impl *UserQueryerImpl) WriteProfile(ctx context.Context, userID int64, name string, value string) error {
	_, err := impl.session.Update(ctx, "UserQueryer.WriteProfile",
		[]string{
			"userID",
			"name",
			"value",
		},
		[]interface{}{
			userID,
			name,
			value,
		})
	return err
}

func (impl *UserQueryerImpl) DeleteProfile(ctx context.Context, userID int64, name string) (int64, error) {
	return impl.session.Delete(ctx, "UserQueryer.DeleteProfile",
		[]string{
			"userID",
			"name",
		},
		[]interface{}{
			userID,
			name,
		})
}

func (impl *UserQueryerImpl) GetUserAndRoleList(ctx context.Context) (func(*UserAndRole) (bool, error), io.Closer) {
	results := impl.session.Select(ctx, "UserQueryer.GetUserAndRoleList",
		[]string{},
		[]interface{}{})
	return func(value *UserAndRole) (bool, error) {
		if !results.Next() {
			if results.Err() == sql.ErrNoRows {
				return false, nil
			}
			return false, results.Err()
		}
		return true, results.Scan(value)
	}, results
}

func (impl *UserQueryerImpl) GetUsernames(ctx context.Context, groupID sql.NullInt64, roleID sql.NullInt64, isJobPostion sql.NullBool) (map[int64]string, error) {
	var instances = map[int64]string{}

	results := impl.session.Select(ctx, "UserQueryer.GetUsernames",
		[]string{
			"groupID",
			"roleID",
			"isJobPostion",
		},
		[]interface{}{
			groupID,
			roleID,
			isJobPostion,
		})
	err := results.ScanBasicMap(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (impl *UserQueryerImpl) GetRolenames(ctx context.Context, _type sql.NullInt64) (map[int64]string, error) {
	var instances = map[int64]string{}

	results := impl.session.Select(ctx, "UserQueryer.GetRolenames",
		[]string{
			"type",
		},
		[]interface{}{
			_type,
		})
	err := results.ScanBasicMap(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
		{ //// UserDao.LockUser
			if _, exists := ctx.Statements["UserDao.LockUser"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" SET locked_at = now() WHERE id=#{id}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.LockUser",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.LockUser"] = stmt
			}
		}
		{ //// UserDao.UnlockUser
			if _, exists := ctx.Statements["UserDao.UnlockUser"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" SET locked_at = null WHERE id=#{id}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.UnlockUser",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.UnlockUser"] = stmt
			}
		}
		{ //// UserDao.LockUserByUsername
			if _, exists := ctx.Statements["UserDao.LockUserByUsername"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n       SET locked_at = now() WHERE lower(name) = lower(#{username})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.LockUserByUsername",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.LockUserByUsername"] = stmt
			}
		}
		{ //// UserDao.UnlockUserByUsername
			if _, exists := ctx.Statements["UserDao.UnlockUserByUsername"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n       SET locked_at = NULL WHERE lower(name) = lower(#{username})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.UnlockUserByUsername",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.UnlockUserByUsername"] = stmt
			}
		}
		{ //// UserDao.CreateUser
			if _, exists := ctx.Statements["UserDao.CreateUser"]; !exists {
				sqlStr, err := gobatis.GenerateInsertSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&User{}),
					[]string{
						"user",
					},
					[]reflect.Type{
						reflect.TypeOf((*User)(nil)),
					}, false)
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.CreateUser error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.CreateUser",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.CreateUser"] = stmt
			}
		}
		{ //// UserDao.UpdateUser
			if _, exists := ctx.Statements["UserDao.UpdateUser"]; !exists {
				sqlStr, err := gobatis.GenerateUpdateSQL(ctx.Dialect, ctx.Mapper,
					"user.", reflect.TypeOf(&User{}),
					[]string{
						"id",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.UpdateUser error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.UpdateUser",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.UpdateUser"] = stmt
			}
		}
		{ //// UserDao.UpdateUserPassword
			if _, exists := ctx.Statements["UserDao.UpdateUserPassword"]; !exists {
				sqlStr, err := gobatis.GenerateUpdateSQL2(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&User{}), reflect.TypeOf(new(int64)), "id", []string{
						"password",
					})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.UpdateUserPassword error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.UpdateUserPassword",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.UpdateUserPassword"] = stmt
			}
		}
		{ //// UserDao.DeleteUser
			if _, exists := ctx.Statements["UserDao.DeleteUser"]; !exists {
				sqlStr, err := gobatis.GenerateDeleteSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&User{}),
					[]string{
						"id",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.DeleteUser error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.DeleteUser",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.DeleteUser"] = stmt
			}
		}
		{ //// UserDao.DisableUser
			if _, exists := ctx.Statements["UserDao.DisableUser"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n       SET disabled = true <if test=\"name.Valid\">, name= #{name} </if>\r\n           <if test=\"nickname.Valid\">, nickname= #{nickname} </if>\r\n       WHERE id=#{id}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.DisableUser",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.DisableUser"] = stmt
			}
		}
		{ //// UserDao.EnableUser
			if _, exists := ctx.Statements["UserDao.EnableUser"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n       SET disabled = false <if test=\"name.Valid\">, name= #{name} </if>\r\n           <if test=\"nickname.Valid\">, nickname= #{nickname} </if>\r\n       WHERE id=#{id}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.EnableUser",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.EnableUser"] = stmt
			}
		}
		{ //// UserDao.HasRoleForUser
			if _, exists := ctx.Statements["UserDao.HasRoleForUser"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT count(*) > 0 FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n          WHERE user_id = #{userid} AND role_id = #{roleid}")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.HasRoleForUser",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.HasRoleForUser"] = stmt
			}
		}
		{ //// UserDao.AddRoleToUser
			if _, exists := ctx.Statements["UserDao.AddRoleToUser"]; !exists {
				var sb strings.Builder
				sb.WriteString("INSERT INTO ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("(user_id, role_id)\r\n       VALUES(#{userid}, #{roleid})\r\n       ON CONFLICT (user_id, role_id)\r\n       DO UPDATE SET user_id=EXCLUDED.user_id, role_id=EXCLUDED.role_id")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.AddRoleToUser",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.AddRoleToUser"] = stmt
			}
		}
		{ //// UserDao.RemoveRoleFromUser
			if _, exists := ctx.Statements["UserDao.RemoveRoleFromUser"]; !exists {
				sqlStr, err := gobatis.GenerateDeleteSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&UserAndRole{}),
					[]string{
						"userid",
						"roleid",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
						reflect.TypeOf(new(int64)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.RemoveRoleFromUser error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.RemoveRoleFromUser",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.RemoveRoleFromUser"] = stmt
			}
		}
		{ //// UserDao.RemoveRolesFromUser
			if _, exists := ctx.Statements["UserDao.RemoveRolesFromUser"]; !exists {
				sqlStr, err := gobatis.GenerateDeleteSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&UserAndRole{}),
					[]string{
						"userid",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.RemoveRolesFromUser error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.RemoveRolesFromUser",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.RemoveRolesFromUser"] = stmt
			}
		}
		{ //// UserDao.CreateRole
			if _, exists := ctx.Statements["UserDao.CreateRole"]; !exists {
				sqlStr, err := gobatis.GenerateInsertSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Role{}),
					[]string{
						"role",
					},
					[]reflect.Type{
						reflect.TypeOf((*Role)(nil)),
					}, false)
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.CreateRole error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.CreateRole",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.CreateRole"] = stmt
			}
		}
		{ //// UserDao.UpdateRole
			if _, exists := ctx.Statements["UserDao.UpdateRole"]; !exists {
				sqlStr, err := gobatis.GenerateUpdateSQL(ctx.Dialect, ctx.Mapper,
					"role.", reflect.TypeOf(&Role{}),
					[]string{
						"id",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.UpdateRole error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.UpdateRole",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.UpdateRole"] = stmt
			}
		}
		{ //// UserDao.DeleteRole
			if _, exists := ctx.Statements["UserDao.DeleteRole"]; !exists {
				sqlStr, err := gobatis.GenerateDeleteSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&Role{}),
					[]string{
						"id",
					},
					[]reflect.Type{
						reflect.TypeOf(new(int64)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserDao.DeleteRole error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.DeleteRole",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.DeleteRole"] = stmt
			}
		}
		return nil
	})
}

func NewUserDao(ref gobatis.SqlSession, userQueryer UserQueryer) UserDao {
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
	return &UserDaoImpl{UserQueryer: userQueryer,
		session: ref}
}

type UserDaoImpl struct {
	UserQueryer
	session gobatis.SqlSession
}

func (impl *UserDaoImpl) LockUser(ctx context.Context, id int64) error {
	_, err := impl.session.Update(ctx, "UserDao.LockUser",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
	return err
}

func (impl *UserDaoImpl) UnlockUser(ctx context.Context, id int64) error {
	_, err := impl.session.Update(ctx, "UserDao.UnlockUser",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
	return err
}

func (impl *UserDaoImpl) LockUserByUsername(ctx context.Context, username string) error {
	_, err := impl.session.Update(ctx, "UserDao.LockUserByUsername",
		[]string{
			"username",
		},
		[]interface{}{
			username,
		})
	return err
}

func (impl *UserDaoImpl) UnlockUserByUsername(ctx context.Context, username string) error {
	_, err := impl.session.Update(ctx, "UserDao.UnlockUserByUsername",
		[]string{
			"username",
		},
		[]interface{}{
			username,
		})
	return err
}

func (impl *UserDaoImpl) CreateUser(ctx context.Context, user *User) (int64, error) {
	return impl.session.Insert(ctx, "UserDao.CreateUser",
		[]string{
			"user",
		},
		[]interface{}{
			user,
		})
}

func (impl *UserDaoImpl) UpdateUser(ctx context.Context, id int64, user *User) (int64, error) {
	return impl.session.Update(ctx, "UserDao.UpdateUser",
		[]string{
			"id",
			"user",
		},
		[]interface{}{
			id,
			user,
		})
}

func (impl *UserDaoImpl) UpdateUserPassword(ctx context.Context, id int64, password string) (int64, error) {
	return impl.session.Update(ctx, "UserDao.UpdateUserPassword",
		[]string{
			"id",
			"password",
		},
		[]interface{}{
			id,
			password,
		})
}

func (impl *UserDaoImpl) DeleteUser(ctx context.Context, id int64) (int64, error) {
	return impl.session.Delete(ctx, "UserDao.DeleteUser",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
}

func (impl *UserDaoImpl) DisableUser(ctx context.Context, id int64, name sql.NullString, nickname sql.NullString) error {
	_, err := impl.session.Update(ctx, "UserDao.DisableUser",
		[]string{
			"id",
			"name",
			"nickname",
		},
		[]interface{}{
			id,
			name,
			nickname,
		})
	return err
}

func (impl *UserDaoImpl) EnableUser(ctx context.Context, id int64, name sql.NullString, nickname sql.NullString) error {
	_, err := impl.session.Update(ctx, "UserDao.EnableUser",
		[]string{
			"id",
			"name",
			"nickname",
		},
		[]interface{}{
			id,
			name,
			nickname,
		})
	return err
}

func (impl *UserDaoImpl) HasRoleForUser(ctx context.Context, userid int64, roleid int64) (bool, error) {
	var instance bool
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "UserDao.HasRoleForUser",
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

func (impl *UserDaoImpl) AddRoleToUser(ctx context.Context, userid int64, roleid int64) error {
	_, err := impl.session.Insert(ctx, "UserDao.AddRoleToUser",
		[]string{
			"userid",
			"roleid",
		},
		[]interface{}{
			userid,
			roleid,
		},
		true)
	return err
}

func (impl *UserDaoImpl) RemoveRoleFromUser(ctx context.Context, userid int64, roleid int64) error {
	_, err := impl.session.Delete(ctx, "UserDao.RemoveRoleFromUser",
		[]string{
			"userid",
			"roleid",
		},
		[]interface{}{
			userid,
			roleid,
		})
	return err
}

func (impl *UserDaoImpl) RemoveRolesFromUser(ctx context.Context, userid int64) error {
	_, err := impl.session.Delete(ctx, "UserDao.RemoveRolesFromUser",
		[]string{
			"userid",
		},
		[]interface{}{
			userid,
		})
	return err
}

func (impl *UserDaoImpl) CreateRole(ctx context.Context, role *Role) (int64, error) {
	return impl.session.Insert(ctx, "UserDao.CreateRole",
		[]string{
			"role",
		},
		[]interface{}{
			role,
		})
}

func (impl *UserDaoImpl) UpdateRole(ctx context.Context, id int64, role *Role) (int64, error) {
	return impl.session.Update(ctx, "UserDao.UpdateRole",
		[]string{
			"id",
			"role",
		},
		[]interface{}{
			id,
			role,
		})
}

func (impl *UserDaoImpl) DeleteRole(ctx context.Context, id int64) (int64, error) {
	return impl.session.Delete(ctx, "UserDao.DeleteRole",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
}
