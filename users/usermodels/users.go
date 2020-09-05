package usermodels

import (
	"context"
	"database/sql"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/db"
)

func init() {
	moo.On(func() moo.Option {
		return moo.Provide(func(env *moo.Environment, db db.InModelFactory, signingMethod SigningMethod) *Users {
			sessionRef := db.Factory.SessionReference()
			return NewUsers(env, sessionRef, signingMethod)
		})
	})
}

type SigningMethod interface {
	Sign(context.Context, string) (string, error)
}

type SigningMethodFunc func(context.Context, string) (string, error)

func (f SigningMethodFunc) Sign(ctx context.Context, s string) (string, error) {
	return f(ctx, s)
}

func NewUsers(env *moo.Environment, sessionRef gobatis.SqlSession, signingMethod SigningMethod) *Users {
	return &Users{
		SigningMethod: signingMethod,
		// env:       env,
		// dbFactory: dbFactory,
		UserDao: NewUserDao(sessionRef, NewUserQueryer(sessionRef)),
		// ologger:   ologger,
	}
}

type Users struct {
	SigningMethod SigningMethod
	// env       *moo.Environment
	// dbFactory *gobatis.SessionFactory
	UserDao UserDao
	// ologger   operation_logs.OperationLogger
}

func (c *Users) Tx(sessionRef gobatis.SqlSession) *Users {
	return NewUsers(nil, sessionRef, c.SigningMethod)
}

func (c *Users) UsernameExists(ctx context.Context, name string) (bool, error) {
	return c.UserDao.UsernameExists(ctx, name)
}

func (c *Users) NicknameExists(ctx context.Context, name string) (bool, error) {
	return c.UserDao.NicknameExists(ctx, name)
}

func (c *Users) GetUsers(ctx context.Context, query *UserQueryParams, offset, limit int64, sort string) ([]User, error) {
	return GetUsers(ctx, c.UserDao, query, offset, limit, sort)
}

func GetUsers(ctx context.Context, userDao UserQueryer, query *UserQueryParams, offset, limit int64, sort string) ([]User, error) {
	next, closer := userDao.GetUsers(ctx, query, offset, limit, sort)
	defer util.CloseWith(closer)

	var userList []User
	for {
		var u User
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
		userList = append(userList, u)
	}

	return userList, nil
}

func (c *Users) GetUserByID(ctx context.Context, userid int64) (*User, error) {
	var user User
	err := c.UserDao.GetUserByID(ctx, userid)(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Users) GetUserByName(ctx context.Context, name string) (*User, error) {
	var user User
	err := c.UserDao.GetUserByName(ctx, name)(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Users) GetUserByNickname(ctx context.Context, name string) (*User, error) {
	var user User
	err := c.UserDao.GetUserByNickname(ctx, name)(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Users) GetRoles(ctx context.Context, name string, offset, limit int64) ([]Role, error) {
	return GetRoles(ctx, c.UserDao, name, offset, limit)
}

func GetRoles(ctx context.Context, userDao UserQueryer, name string, offset, limit int64) ([]Role, error) {
	next, closer := userDao.GetRoles(ctx, name, sql.NullInt64{}, offset, limit)
	defer util.CloseWith(closer)

	var roles []Role
	for {
		var role Role
		ok, err := next(&role)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, err
		}
		if !ok {
			break
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (c *Users) GetRoleByID(ctx context.Context, roleid int64) (*Role, error) {
	var role Role
	err := c.UserDao.GetRoleByID(ctx, roleid)(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (c *Users) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	var role Role
	err := c.UserDao.GetRoleByName(ctx, name)(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (c *Users) CreateUserWithRoleNames(ctx context.Context, user *User, roles []string) (int64, error) {
	var roleIDs []int64
	for _, name := range roles {
		var role Role
		err := c.UserDao.GetRoleByName(ctx, name)(&role)
		if err != nil {
			return 0, err
		}
		roleIDs = append(roleIDs, role.ID)
	}
	return c.CreateUser(ctx, user, roleIDs)
}

func (c *Users) CreateUser(ctx context.Context, user *User, roles []int64) (int64, error) {
	// tx, err := c.dbFactory.DB().(*sql.DB).Begin()
	// if err != nil {
	// 	return 0, errors.WithTitle(err, "用户不存在，创建新用户时启动事务失败")
	// }
	// defer util.RollbackWith(tx)

	// ctx = gobatis.WithDbConnection(ctx, tx)

	if user.Password != "" {
		var err error
		user.Password, err = c.SigningMethod.Sign(ctx, user.Password)
		if err != nil {
			return 0, errors.WithTitle(err, "用户密码加密失败")
		}
	}
	userid, err := c.UserDao.CreateUser(ctx, user)
	if err != nil {
		return 0, errors.WithTitle(err, "用户不存在，创建新用户失败")
	}
	user.ID = userid

	for _, roleid := range roles {
		err = c.UserDao.AddRoleToUser(ctx, userid, roleid)
		if err != nil {
			return 0, errors.WithTitle(err, "用户不存在，创建新用户时添加角色失败")
		}
	}

	// c.ologger.WithTx(tx).LogRecord(ctx, &operation_logs.OperationLog{
	// 	UserID:     userid,
	// 	Username:   user.Name,
	// 	Successful: true,
	// 	Type:       "add_user",
	// 	Content:    "创建用户: " + user.Name,
	// 	//Fields     &OperationLogRecord{}
	// })
	// if err := tx.Commit(); err != nil {
	// 	return 0, errors.WithTitle(err, "用户不存在，创建新用户时提交事务失败")
	// }
	return userid, nil
}

func (c *Users) UpdateUserPassword(ctx context.Context, userID int64, password string) error {
	if password != "" {
		newPassword, err := c.SigningMethod.Sign(ctx, password)
		if err != nil {
			return errors.WithTitle(err, "用户密码加密失败")
		}
		password = newPassword
	}
	count, err := c.UserDao.UpdateUserPassword(ctx, userID, password)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}

	if count == 0 {
		return errors.ErrNotFoundWithText("该用户不存在!")
	}
	return nil
}

func (c *Users) UpdateUser(ctx context.Context, userID int64, user *User) error {
	if user.Password != "" {
		var err error
		user.Password, err = c.SigningMethod.Sign(ctx, user.Password)
		if err != nil {
			return errors.WithTitle(err, "用户密码加密失败")
		}
	}

	count, err := c.UserDao.UpdateUser(ctx, userID, user)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}

	if count == 0 {
		return errors.ErrNotFoundWithText("该用户不存在!")
	}
	return nil
}

func (c *Users) ReadProfile(ctx context.Context, userID int64, name string) (string, error) {
	return c.UserDao.ReadProfile(ctx, userID, name)
}

func (c *Users) WriteProfile(ctx context.Context, userID int64, name, value string) error {
	return c.UserDao.WriteProfile(ctx, userID, name, value)
}

func (c *Users) DeleteProfile(ctx context.Context, userID int64, name string) (int64, error) {
	return c.UserDao.DeleteProfile(ctx, userID, name)
}
