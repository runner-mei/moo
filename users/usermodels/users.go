package usermodels

import (
	"context"
	"database/sql"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/operation_logs"
)

func NewUsers(env *moo.Environment, dbFactory *gobatis.SessionFactory, ologger operation_logs.OperationLogger) *Users {
	sessionRef := dbFactory.SessionReference()
	return &Users{
		env:       env,
		dbFactory: dbFactory,
		userDao:   NewUserDao(sessionRef, NewUserQueryer(sessionRef)),
		ologger:   ologger,
	}
}

type Users struct {
	env       *moo.Environment
	dbFactory *gobatis.SessionFactory
	userDao   UserDao
	ologger   operation_logs.OperationLogger
}

func (c *Users) NicknameExists(ctx context.Context, name string) (bool, error) {
	return c.userDao.NicknameExists(ctx, name)
}

func (c *Users) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	var role Role
	err := c.userDao.GetRoleByName(ctx, name)(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (c *Users) GetUserByName(ctx context.Context, name string) (*User, error) {
	var user User
	err := c.userDao.GetUserByName(ctx, name)(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Users) CreateUserWithRoleNames(ctx context.Context, user *User, roles []string) (int64, error) {
	var roleIDs []int64
	for _, name := range roles {
		var role Role
		err := c.userDao.GetRoleByName(ctx, name)(&role)
		if err != nil {
			return 0, err
		}
		roleIDs = append(roleIDs, role.ID)
	}
	return c.CreateUser(ctx, user, roleIDs)
}

func (c *Users) CreateUser(ctx context.Context, user *User, roles []int64) (int64, error) {
	tx, err := c.dbFactory.DB().(*sql.DB).Begin()
	if err != nil {
		return 0, errors.WithTitle(err, "用户不存在，创建新用户时启动事务失败")
	}
	defer util.RollbackWith(tx)

	ctx = gobatis.WithDbConnection(ctx, tx)

	userid, err := c.userDao.CreateUser(ctx, user)
	if err != nil {
		return 0, errors.WithTitle(err, "用户不存在，创建新用户失败")
	}
	user.ID = userid

	for _, roleid := range roles {
		err = c.userDao.AddRoleToUser(ctx, userid, roleid)
		if err != nil {
			return 0, errors.WithTitle(err, "用户不存在，创建新用户时添加角色失败")
		}
	}

	c.ologger.WithTx(tx).LogRecord(ctx, &operation_logs.OperationLog{
		UserID:     userid,
		Username:   user.Name,
		Successful: true,
		Type:       "add_user",
		Content:    "创建用户: " + user.Name,
		//Fields     &OperationLogRecord{}
	})
	if err := tx.Commit(); err != nil {
		return 0, errors.WithTitle(err, "用户不存在，创建新用户时提交事务失败")
	}
	return userid, nil
}
