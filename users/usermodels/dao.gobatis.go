// Please don't edit this file!
package usermodels

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"

	gobatis "github.com/runner-mei/GoBatis"
)

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
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
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&User{}),
					[]string{
						"name",
					},
					[]reflect.Type{
						reflect.TypeOf(new(string)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetUserByName error")
				}
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
		{ //// UserQueryer.GetUsers
			if _, exists := ctx.Statements["UserQueryer.GetUsers"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&User{}),
					[]string{},
					[]reflect.Type{},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate UserQueryer.GetUsers error")
				}
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
		{ //// UserQueryer.GetPermissionsByRoleIDs
			if _, exists := ctx.Statements["UserQueryer.GetPermissionsByRoleIDs"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&PermissionAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE role_id in <foreach collection=\"roleIDs\" open=\"(\" separator=\",\" close=\")\">#{item}</foreach>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetPermissionsByRoleIDs",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetPermissionsByRoleIDs"] = stmt
			}
		}
		{ //// UserQueryer.GetRolesByUser
			if _, exists := ctx.Statements["UserQueryer.GetRolesByUser"]; !exists {
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
				sb.WriteString(" as users_roles\r\n     where users_roles.role_id = roles.id and users_roles.user_id = #{userID})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserQueryer.GetRolesByUser",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserQueryer.GetRolesByUser"] = stmt
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

func (impl *UserQueryerImpl) GetUsers(ctx context.Context) ([]User, error) {
	var instances []User
	results := impl.session.Select(ctx, "UserQueryer.GetUsers",
		[]string{},
		[]interface{}{})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (impl *UserQueryerImpl) GetPermissionsByRoleIDs(ctx context.Context, roleIDs []int64) ([]string, error) {
	var instances []string
	results := impl.session.Select(ctx, "UserQueryer.GetPermissionsByRoleIDs",
		[]string{
			"roleIDs",
		},
		[]interface{}{
			roleIDs,
		})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (impl *UserQueryerImpl) GetRolesByUser(ctx context.Context, userID int64) ([]Role, error) {
	var instances []Role
	results := impl.session.Select(ctx, "UserQueryer.GetRolesByUser",
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

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
		{ //// UserDao.Lock
			if _, exists := ctx.Statements["UserDao.Lock"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n       SET locked_at = now() WHERE lower(name) = lower(#{username})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.Lock",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.Lock"] = stmt
			}
		}
		{ //// UserDao.Unlock
			if _, exists := ctx.Statements["UserDao.Unlock"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n       SET locked_at = NULL WHERE lower(name) = lower(#{username})")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "UserDao.Unlock",
					gobatis.StatementTypeUpdate,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["UserDao.Unlock"] = stmt
			}
		}
		{ //// UserDao.CreateUser
			if _, exists := ctx.Statements["UserDao.CreateUser"]; !exists {
				sqlStr, err := gobatis.GenerateInsertSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&User{}),
					[]string{"user"},
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
		{ //// UserDao.DisableUser
			if _, exists := ctx.Statements["UserDao.DisableUser"]; !exists {
				var sb strings.Builder
				sb.WriteString("UPDATE ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&User{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString("\r\n       SET disabled = true WHERE id=#{id}")
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
				sb.WriteString("\r\n       SET disabled = false WHERE id=#{id}")
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
				var sb strings.Builder
				sb.WriteString("DELETE FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&UserAndRole{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" WHERE user_id = #{userid} and role_id = #{roleid}")
				sqlStr := sb.String()

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

func (impl *UserDaoImpl) Lock(ctx context.Context, username string) error {
	_, err := impl.session.Update(ctx, "UserDao.Lock",
		[]string{
			"username",
		},
		[]interface{}{
			username,
		})
	return err
}

func (impl *UserDaoImpl) Unlock(ctx context.Context, username string) error {
	_, err := impl.session.Update(ctx, "UserDao.Unlock",
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

func (impl *UserDaoImpl) DisableUser(ctx context.Context, id int64) error {
	_, err := impl.session.Update(ctx, "UserDao.DisableUser",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
	return err
}

func (impl *UserDaoImpl) EnableUser(ctx context.Context, id int64) error {
	_, err := impl.session.Update(ctx, "UserDao.EnableUser",
		[]string{
			"id",
		},
		[]interface{}{
			id,
		})
	return err
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
