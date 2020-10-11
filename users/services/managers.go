package services

import (
	"context"
	"database/sql"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/as"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/db"
	"github.com/runner-mei/moo/users/usermodels"
	"github.com/runner-mei/moo/users/welcome"
	"github.com/runner-mei/validation"
)

type RequestContext struct {
	Ctx         context.Context
	Factory     *gobatis.SessionFactory
	Tx          *gobatis.Tx
	CurrentUser api.User
	Locale      string
	OpLogger    api.OperationLogger
	OnlineUsers usermodels.OnlineUsers
	Users       *usermodels.Users
	Usergroups  usermodels.UsergroupDao
}

func (req *RequestContext) Commit() error {
	if req.Tx == nil {
		return nil
	}
	return req.Tx.Commit()
}

func (req *RequestContext) Rollback() error {
	if req.Tx == nil {
		return nil
	}
	return req.Tx.Rollback()
}

func (req *RequestContext) Begin(nativeTx ...gobatis.DBRunner) (*RequestContext, error) {
	if req.Tx != nil {
		newReq := &RequestContext{}
		*newReq = *req
		newReq.Tx = nil
		return newReq, nil
	}
	tx, err := req.Factory.Begin(nativeTx...)
	if err != nil {
		return nil, err
	}

	newReq := &RequestContext{}
	*newReq = *req
	newReq.Tx = tx

	session := tx.SessionReference()
	newReq.OnlineUsers = usermodels.NewOnlineUserDao(session)
	newReq.OpLogger = req.OpLogger.Tx(tx)
	newReq.Users = req.Users.Tx(session)
	newReq.Usergroups = usermodels.NewUsergroupDao(session, usermodels.NewUsergroupQueryer(session))
	return newReq, nil
}

func (req *RequestContext) InTransaction(cb func(*RequestContext) error) error {
	ctx, err := req.Begin()
	if err != nil {
		return err
	}
	defer util.RollbackWith(ctx)

	err = cb(ctx)
	if err != nil {
		return err
	}
	return ctx.Commit()
}

type Service struct {
	Env     *moo.Environment
	Factory *gobatis.SessionFactory

	OpLogger    api.OperationLogger
	OnlineUsers usermodels.OnlineUsers
	Users       *usermodels.Users
	Usergroups  usermodels.UsergroupDao

	OnlineExpired  string
	WelcomeRootURL string
	Validator      *validation.Validation
}

func (svc *Service) NewContext(ctx context.Context, currentUser api.User, locale string) *RequestContext {
	return &RequestContext{
		Ctx:     ctx,
		Factory: svc.Factory,

		// Tx          *gobatis.Tx
		CurrentUser: currentUser,
		Locale:      locale,
		OpLogger:    svc.OpLogger,
		OnlineUsers: svc.OnlineUsers,
		Users:       svc.Users,
		Usergroups:  svc.Usergroups,
	}
}

func (svc *Service) NewContextWithTx(ctx context.Context, nativeTx gobatis.DBRunner, currentUser api.User, locale string) (*RequestContext, error) {
	req := &RequestContext{
		Ctx:     ctx,
		Factory: svc.Factory,

		// Tx          *gobatis.Tx
		CurrentUser: currentUser,
		Locale:      locale,
		OpLogger:    svc.OpLogger,
		OnlineUsers: svc.OnlineUsers,
		Users:       svc.Users,
		Usergroups:  svc.Usergroups,
	}

	tx, err := req.Factory.Begin(nativeTx)
	if err != nil {
		return nil, err
	}

	req.Tx = tx

	session := tx.SessionReference()
	req.OnlineUsers = usermodels.NewOnlineUserDao(session)
	req.OpLogger = req.OpLogger.Tx(tx)
	req.Users = req.Users.Tx(session)
	req.Usergroups = usermodels.NewUsergroupDao(session, usermodels.NewUsergroupQueryer(session))
	return req, nil
}

type UserQueryOptions struct {
	HasOnlineInfo    bool
	HasRoleInfo      bool
	HasWelcomeInfo   bool
	HasUsergroupInfo bool
}

func covertWelcomePages(env *moo.Environment, userList []usermodels.User, rootURL string) {
	choices, err := welcome.ReadURLs(env, rootURL)
	if err != nil {
		log.Println("covertWelcomePages:", err)
		return
	}
	if len(choices) == 0 {
		return
	}

	toWelcomePageName := func(s string) string {
		for idx := range choices {
			for _, choice := range choices[idx].Children {
				if s == choice.Value {
					return choice.Label
				}
			}
		}
		return s
	}

	for idx := range userList {
		if userList[idx].Attributes == nil {
			continue
		}
		v := userList[idx].Attributes[welcome.FieldName]

		if v == nil {
			break
		}

		s := as.StringWithDefault(v, "")
		if s == "" {
			break
		}
		userList[idx].Attributes[welcome.FieldName] = toWelcomePageName(s)
	}
}

func (svc *Service) GetUsers(ctx *RequestContext, query *usermodels.UserQueryParams, opts *UserQueryOptions,
	offset, limit int64, sort string) ([]usermodels.User, error) {
	userList, err := ctx.Users.GetUsers(ctx.Ctx, query, offset, limit, sort)
	if err != nil {
		return nil, err
	}

	usergroupCache := map[int64]*usermodels.Usergroup{}
	for i := range userList {
		_, err := svc.loadUser(ctx, &userList[i], opts, usergroupCache)
		if err != nil {
			return nil, err
		}
	}

	return userList, nil
}

func (svc *Service) GetUserByID(ctx *RequestContext, id int64, opts *UserQueryOptions) (*usermodels.User, error) {
	u, err := ctx.Users.GetUserByID(ctx.Ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFoundWithText("该用户不存在!")
			// return nil, ErrUserIDNotFound(id)
		}
		return nil, err
	}
	return svc.loadUser(ctx, u, opts, map[int64]*usermodels.Usergroup{})
}

func (svc *Service) loadUser(ctx *RequestContext, u *usermodels.User, opts *UserQueryOptions, usergroupCache map[int64]*usermodels.Usergroup) (*usermodels.User, error) {
	if opts.HasOnlineInfo {
		onlines, err := ctx.OnlineUsers.List(ctx.Ctx, svc.OnlineExpired)
		if err != nil {
			return nil, errors.Wrap(err, "获取在线用户失败")
		}
		isOnline := func(userID int64) bool {
			for _, online := range onlines {
				if online.UserID == userID {
					return true
				}
			}
			return false
		}
		u.SetOnlineToExtensions(isOnline(u.ID))
	}

	if opts.HasWelcomeInfo {
		users := []usermodels.User{*u}
		covertWelcomePages(svc.Env, users, svc.WelcomeRootURL)
	}
	if opts.HasRoleInfo {
		roles, err := ctx.Users.UserDao.GetRolesByUserID(ctx.Ctx, u.ID)
		if err != nil {
			return nil, errors.Wrap(err, "读用户的角色信息失败")
		}
		u.SetRolesToExtensions(roles)
	}

	if opts.HasUsergroupInfo {
		var usergroups []usermodels.Usergroup

		next, closer := ctx.Usergroups.GetUserAndGroupList(ctx.Ctx, sql.NullInt64{Valid: true, Int64: u.ID}, true)
		defer util.CloseWith(closer)

		for {
			var u2g usermodels.UserAndUsergroup
			ok, err := next(&u2g)
			if err != nil {
				return nil, errors.Wrap(err, "读用户的用户组信息失败")
			}
			if !ok {
				break
			}

			ug := usergroupCache[u2g.GroupID]
			if ug != nil {
				if ug.Disabled {
					continue
				}
				//u.Usergroups = append(u.Usergroups, *ug)
				continue
			}
			var group usermodels.Usergroup
			err = ctx.Usergroups.GetUsergroupByID(ctx.Ctx, u2g.GroupID)(&group)
			if err != nil {
				return nil, errors.Wrap(err, "读用户的用户组信息失败")
			}
			usergroupCache[group.ID] = &group
			if group.Disabled {
				continue
			}
			usergroups = append(usergroups, group)
		}

		u.SetUsergroupsToExtensions(usergroups)
	}
	return u, nil
}

func (svc *Service) CreateUserWithRoleNames(ctx *RequestContext, user *usermodels.User, roles []string) (int64, error) {
	var roleIDs []int64
	for _, name := range roles {
		role, err := svc.Users.GetRoleByName(ctx.Ctx, name)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.New("角色 '"+ name +"' 不存在")
			}
			return 0, err
		}
		roleIDs = append(roleIDs, role.ID)
	}

	return svc.CreateUser(ctx, user, roleIDs)
}

func (svc *Service) CreateUser(ctx *RequestContext, user *usermodels.User, selectedRoles []int64) (int64, error) {
	oldUser1, err := ctx.Users.GetUserByName(ctx.Ctx, user.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return 0, errors.Wrap(err, "查询该用户是否存在失败")
		}
	} else {
		if oldUser1.IsDisabled() {
			return 0, validation.NewValidationError("Name", "该用户名 '"+user.Name+"' 已存在, 但被删除，你可以请管理员恢复它!")
		}
		return 0, validation.NewValidationError("Name", "该用户名 '"+user.Name+"' 已存在!")
	}
	oldUser2, err := ctx.Users.GetUserByNickname(ctx.Ctx, user.Nickname)
	if err != nil {
		if !errors.IsNotFound(err) {
			return 0, errors.Wrap(err, "查询该用户是否存在失败")
		}
	} else {
		if oldUser2.IsDisabled() {
			return 0, validation.NewValidationError("Nickname", "该用户姓名 '"+user.Nickname+"' 已存在, 但被删除，你可以请管理员恢复它!")
		}
		return 0, validation.NewValidationError("Nickname", "该用户姓名 '"+user.Nickname+"' 已存在!")
	}

	ctx, err = ctx.Begin()
	if err != nil {
		return 0, err
	}
	defer util.RollbackWith(ctx)

	userID, err := ctx.Users.CreateUser(ctx.Ctx, user, selectedRoles)
	if err != nil {
		return 0, err
	}

	if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
		UserID:     ctx.CurrentUser.ID(),
		Username:   ctx.CurrentUser.Name(),
		Type:       "add_user",
		Successful: true,
		Content:    "创建用户: " + user.Name,
	}); err != nil {
		return 0, errors.Wrap(err, "添加操作日志失败")
	}

	if err := ctx.Commit(); err != nil {
		return 0, err
	}
	return userID, nil
}

func (svc *Service) UpdateUserFields(ctx *RequestContext, userID int64, values map[string]interface{}) error {
	oldUser, err := ctx.Users.GetUserByID(ctx.Ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}

	for k, v := range values {
		oldUser.Attributes[k] = v
	}

	return ctx.InTransaction(func(ctx *RequestContext) error {
		err = ctx.Users.UpdateUser(ctx.Ctx, userID, oldUser)
		if err != nil {
			return errors.Wrap(err, "更新用户失败")
		}

		content := "更新用户: " + oldUser.Name

		if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
			UserID:     ctx.CurrentUser.ID(),
			Username:   ctx.CurrentUser.Name(),
			Type:       "update_user",
			Successful: true,
			Content:    content,
		}); err != nil {
			return errors.Wrap(err, "添加操作日志失败")
		}
		return nil
	})
}

func isStars(s string) bool {
	for _, c := range s {
		if !unicode.IsSpace(c) && c != '*' {
			return false
		}
	}
	return false
}

type UpdateRoleMode int

const (
	RoleUpdateModeSkip UpdateRoleMode = iota
	RoleUpdateModeAdd
	RoleUpdateModeUpdate
)

func (svc *Service) UpdateUser(ctx *RequestContext, userID int64, user *usermodels.User, updateRole UpdateRoleMode, selectedRoles []int64) error {
	oldUser, err := ctx.Users.GetUserByID(ctx.Ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}

	if oldUser.Name != user.Name {
		exists, err := ctx.Users.UsernameExists(ctx.Ctx, user.Name)
		if err != nil {
			return errors.Wrap(err, "查询用户名是否已存在失败")
		}

		if exists {
			return validation.NewValidationError("Name", "该用户名 '"+user.Name+"' 已存在!")
		}
	}

	if oldUser.Nickname != user.Nickname {
		exists, err := ctx.Users.NicknameExists(ctx.Ctx, user.Nickname)
		if err != nil {
			return errors.Wrap(err, "查询用户名是否已存在失败")
		}
		if exists {
			return validation.NewValidationError("Nickname", "该用户名 '"+user.Nickname+"' 已存在!")
		}
	}

	if len(user.Attributes) > 0 && len(oldUser.Attributes) > 0 {
		fields, _ := ReadFieldsFromDir(svc.Env)
		for k, v := range oldUser.Attributes {
			_, exists := user.Attributes[k]
			if exists {
				continue
			}
			if len(fields) > 0 {
				for idx := range fields {
					for j := range fields[idx].Fields {
						if fields[idx].Prefix+fields[idx].Fields[j].ID == k {
							exists = true
							break
						}
					}

					if exists {
						break
					}
				}
			}
			if exists {
				continue
			}
			user.Attributes[k] = v
		}
	}

	hasPassword := user.Password != ""
	if user.Source == "cas" || user.Source == "ldap" {
		hasPassword = false
	}
	if oldUser.Password == user.Password || isStars(user.Password) {
		hasPassword = false
	}

	if !hasPassword {
		user.Password = "Validate_$2dfg&123_Is_Ok"
	}

	validator := svc.Validator.New()
	if user.Validate(validator) {
		return validator.ToError()
	}

	if hasPassword {
		user.Password = user.Password
	} else {
		user.Password = oldUser.Password
	}

	return ctx.InTransaction(func(ctx *RequestContext) error {
		err = ctx.Users.UpdateUser(ctx.Ctx, userID, user)
		if err != nil {
			return errors.Wrap(err, "更新用户失败")
		}

		content := "更新用户: " + user.Name
		switch updateRole {
		case RoleUpdateModeSkip:
		case RoleUpdateModeAdd:
			for _, roleID := range selectedRoles {
				err := ctx.Users.UserDao.AddRoleToUser(ctx.Ctx, user.ID, roleID)
				if err != nil {
					return errors.Wrap(err, "授于角色失败")
				}
			}

			var created []string
			for _, roleID := range selectedRoles {
				var r usermodels.Role
				err = ctx.Users.UserDao.GetRoleByID(ctx.Ctx, roleID)(&r)
				if err != nil {
					return errors.Wrap(err, "查询新增角色失败")
				}
				created = append(created, r.Name)
			}
			content = content + ", 新增角色 '" + strings.Join(created, ",") + "'"
		case RoleUpdateModeUpdate:
			//更新用户与角色关系
			created, deleted, err := svc.updateUserRoles(ctx.Ctx, ctx.Users.UserDao, user.ID, selectedRoles)
			if err != nil {
				return err
			}
			if len(created) > 0 {
				content = content + ", 新增角色 '" + strings.Join(created, ",") + "'"
			}
			if len(deleted) > 0 {
				content = content + ", 删除角色 '" + strings.Join(deleted, ",") + "'"
			}
		default:
			return errors.Wrap(err, "不支持的角色更新模式 '"+strconv.FormatInt(int64(updateRole), 10)+"'")
		}

		if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
			UserID:     ctx.CurrentUser.ID(),
			Username:   ctx.CurrentUser.Name(),
			Type:       "update_user",
			Successful: true,
			Content:    content,
		}); err != nil {
			return errors.Wrap(err, "添加操作日志失败")
		}
		return nil
	})
}

func (svc *Service) UpdateUserRoles(ctx *RequestContext, userID int64, newRoles []int64) error {
	oldUser, err := ctx.Users.GetUserByID(ctx.Ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}

	return ctx.InTransaction(func(ctx *RequestContext) error {
		created, deleted, err := svc.updateUserRoles(ctx.Ctx, ctx.Users.UserDao, userID, newRoles)
		if err != nil {
			return err
		}

		content := "修改用户 '" + oldUser.Name + "' 的角色"
		if len(created) > 0 {
			content = content + ", 新增角色 '" + strings.Join(created, ",") + "'"
		}
		if len(deleted) > 0 {
			content = content + ", 删除角色 '" + strings.Join(deleted, ",") + "'"
		}
		if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
			UserID:     ctx.CurrentUser.ID(),
			Username:   ctx.CurrentUser.Name(),
			Type:       "change_user_roles",
			Successful: true,
			Content:    content,
		}); err != nil {
			return errors.Wrap(err, "添加操作日志失败")
		}
		return nil
	})
}

func (svc *Service) UpdateUserPassword(ctx *RequestContext, userID int64, newPassword string) error {
	oldUser, err := ctx.Users.GetUserByID(ctx.Ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.updateUserPassword(ctx, oldUser, newPassword)
}

func (svc *Service) UpdateUserPasswordByName(ctx *RequestContext, name string, newPassword string) error {
	oldUser, err := ctx.Users.GetUserByName(ctx.Ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.updateUserPassword(ctx, oldUser, newPassword)
}

func (svc *Service) updateUserPassword(ctx *RequestContext, user *usermodels.User, newPassword string) error {
	return ctx.InTransaction(func(ctx *RequestContext) error {
		err := ctx.Users.UpdateUserPassword(ctx.Ctx, user.ID, newPassword)
		if err != nil {
			return errors.Wrap(err, "更改用户密码失败")
		}

		if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
			UserID:     ctx.CurrentUser.ID(),
			Username:   ctx.CurrentUser.Name(),
			Type:       "change_user_password",
			Successful: true,
			Content:    "修改用户 '" + user.Name + "' 的密码",
		}); err != nil {
			return errors.Wrap(err, "添加操作日志失败")
		}
		return nil
	})
}

func (svc *Service) EnableUser(ctx *RequestContext, userID int64) error {
	oldUser, err := ctx.Users.GetUserByID(ctx.Ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.enableUser(ctx, oldUser)
}

func (svc *Service) EnableUserByName(ctx *RequestContext, name string) error {
	oldUser, err := ctx.Users.GetUserByName(ctx.Ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.enableUser(ctx, oldUser)
}

func (svc *Service) enableUser(ctx *RequestContext, user *usermodels.User) error {
	return ctx.InTransaction(func(ctx *RequestContext) error {
		err := ctx.Users.UserDao.EnableUser(ctx.Ctx, user.ID, nullString(user.Name), nullString(user.Nickname))
		if err != nil {
			return errors.Wrap(err, "启用用户 '"+user.Name+"' 失败")
		}

		if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
			UserID:     ctx.CurrentUser.ID(),
			Username:   ctx.CurrentUser.Name(),
			Type:       "enable_user",
			Successful: true,
			Content:    "启用用户 '" + user.Name + "'",
		}); err != nil {
			return errors.Wrap(err, "添加操作日志失败")
		}
		return nil
	})
}

func (svc *Service) DisableUser(ctx *RequestContext, userID int64) error {
	oldUser, err := ctx.Users.GetUserByID(ctx.Ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.disableUser(ctx, oldUser)
}

func (svc *Service) DisableUserByName(ctx *RequestContext, name string) error {
	oldUser, err := ctx.Users.GetUserByName(ctx.Ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.disableUser(ctx, oldUser)
}

func (svc *Service) disableUser(ctx *RequestContext, user *usermodels.User) error {
	return ctx.InTransaction(func(ctx *RequestContext) error {
		err := ctx.Users.UserDao.DisableUser(ctx.Ctx, user.ID, nullString(user.Name), nullString(user.Nickname))
		if err != nil {
			return errors.Wrap(err, "启用用户 '"+user.Name+"' 失败")
		}

		if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
			UserID:     ctx.CurrentUser.ID(),
			Username:   ctx.CurrentUser.Name(),
			Type:       "enable_user",
			Successful: true,
			Content:    "启用用户 '" + user.Name + "'",
		}); err != nil {
			return errors.Wrap(err, "添加操作日志失败")
		}
		return nil
	})
}

func (svc *Service) UpdateUserRolesNoLog(ctx context.Context, userDao usermodels.UserDao, userID int64, newRoles []int64) ([]string, []string, error) {
	return svc.updateUserRoles(ctx, userDao, userID, newRoles)
}

func (svc *Service) updateUserRoles(ctx context.Context, userDao usermodels.UserDao, userID int64, newRoles []int64) ([]string, []string, error) {
	var created, deleted []string
	oldRoles, err := userDao.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	for _, r := range oldRoles {
		found := false
		for _, roleID := range newRoles {
			if r.ID == roleID {
				found = true
				break
			}
		}

		if !found {
			err = userDao.RemoveRoleFromUser(ctx, userID, r.ID)
			if err != nil {
				return nil, nil, errors.Wrap(err, "收回角色失败")
			}
			deleted = append(deleted, r.Name)
		}
	}
	for _, roleID := range newRoles {
		found := false
		for _, r := range oldRoles {
			if r.ID == roleID {
				found = true
				break
			}
		}

		if !found {
			err = userDao.AddRoleToUser(ctx, userID, roleID)
			if err != nil {
				return nil, nil, errors.Wrap(err, "授于角色失败")
			}

			var r usermodels.Role
			err = userDao.GetRoleByID(ctx, roleID)(&r)
			if err != nil {
				return nil, nil, errors.Wrap(err, "查询新增角色失败")
			}
			created = append(created, r.Name)
		}
	}
	return created, deleted, nil
}

func nullString(name string) sql.NullString {
	return sql.NullString{Valid: true, String: name}
}

const deleteTag = "(deleted:"

// DeleteUser 删除用户
func (svc *Service) DeleteUser(ctx *RequestContext, userID int64, notDelete bool) error {
	oldUser, err := ctx.Users.GetUserByID(ctx.Ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.deleteUser(ctx, oldUser, notDelete)
}

// DeleteUserByName 删除用户
func (svc *Service) DeleteUserByName(ctx *RequestContext, name string, notDelete bool) error {
	oldUser, err := ctx.Users.GetUserByName(ctx.Ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.deleteUser(ctx, oldUser, notDelete)
}

func (svc *Service) deleteUser(ctx *RequestContext, user *usermodels.User, notDelete bool) error {
	return ctx.InTransaction(func(ctx *RequestContext) error {
		var err error
		username := user.Name
		if notDelete {
			suffix := deleteTag + " " + time.Now().Format(time.RFC3339) + ")"
			if !strings.Contains(user.Name, deleteTag) {
				user.Name = user.Name + suffix
			}
			if !strings.Contains(user.Nickname, deleteTag) {
				user.Nickname = user.Nickname + suffix
			}

			err = ctx.Users.UserDao.DisableUser(ctx.Ctx, user.ID,
				nullString(user.Name), nullString(user.Nickname))
			if err == nil {
				err = ctx.Usergroups.RemoveUserFromAllGroups(ctx.Ctx, user.ID)
			}
		} else {
			_, err = ctx.Users.UserDao.DeleteUser(ctx.Ctx, user.ID)
		}
		if err != nil {
			return errors.Wrap(err, "删除用户失败")
		}

		if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
			UserID:     ctx.CurrentUser.ID(),
			Username:   ctx.CurrentUser.Name(),
			Type:       "delete_user",
			Successful: true,
			Content:    "删除用户: " + username,
		}); err != nil {
			return errors.Wrap(err, "添加操作日志失败")
		}
		return nil
	})
}

// Recovery 按 id 删除记录
func (svc *Service) RecoveryUser(ctx *RequestContext, userID int64) error {
	oldUser, err := ctx.Users.GetUserByID(ctx.Ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.recoveryUser(ctx, oldUser)
}

// Recovery 按 id 删除记录
func (svc *Service) RecoveryUserByName(ctx *RequestContext, name string) error {
	oldUser, err := ctx.Users.GetUserByName(ctx.Ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.ErrNotFoundWithText("该用户不存在!")
		}
		return errors.Wrap(err, "查询用户失败")
	}
	return svc.recoveryUser(ctx, oldUser)
}

// Recovery 按 id 删除记录
func (svc *Service) recoveryUser(ctx *RequestContext, user *usermodels.User) error {
	if idx := strings.Index(user.Name, deleteTag); idx >= 0 {
		newName := user.Name[:idx]
		if exists, err := ctx.Users.UserDao.UsernameExists(ctx.Ctx, newName); err != nil {
			return errors.Wrap(err, "查询用户名是否同名失败")
		} else if !exists {
			user.Name = newName
		}
	}
	if idx := strings.Index(user.Nickname, deleteTag); idx >= 0 {
		newName := user.Nickname[:idx]
		if exists, err := ctx.Users.UserDao.NicknameExists(ctx.Ctx, newName); err != nil {
			return errors.Wrap(err, "查询用户呢称是否同名失败")
		} else if !exists {
			user.Nickname = newName
		}
	}

	return ctx.InTransaction(func(ctx *RequestContext) error {
		err := ctx.Users.UserDao.EnableUser(ctx.Ctx, user.ID,
			nullString(user.Name), nullString(user.Nickname))
		if nil != err {
			return errors.Wrap(err, "恢复用户失败")
		}

		if err := ctx.OpLogger.Tx(ctx.Tx).LogRecord(ctx.Ctx, &api.OperationLog{
			UserID:     ctx.CurrentUser.ID(),
			Username:   ctx.CurrentUser.Name(),
			Type:       "recovery_user",
			Successful: true,
			Content:    "恢复用户: " + user.Name,
		}); err != nil {
			return errors.Wrap(err, "添加操作日志失败")
		}
		return nil
	})
}

func NewService(env *moo.Environment,
	factory *gobatis.SessionFactory,
	users *usermodels.Users,
	opLogger api.OperationLogger,
	validator *validation.Validation) (*Service, error) {

	welcomeRootURL := env.Config.StringWithDefault(api.CfgRootEndpoint, env.DaemonUrlPath)
	if welcomeRootURL == "" {
		return nil, errors.New("初始用户服务失败： 缺少参数 '" + api.CfgRootEndpoint + "'")
	}
	session := factory.SessionReference()
	return &Service{
		Env:            env,
		Factory:        factory,
		OpLogger:       opLogger,
		OnlineUsers:    usermodels.NewOnlineUserDao(session),
		Users:          users,
		Usergroups:     usermodels.NewUsergroupDao(session, usermodels.NewUsergroupQueryer(session)),
		OnlineExpired:  env.Config.StringWithDefault(api.CfgUserOnlineExpired, "30 MINUTE"),
		WelcomeRootURL: welcomeRootURL,
		Validator:      validator,
	}, nil
}

type OptValidation struct {
	moo.In

	Validator *validation.Validation `optional:"true"`
}

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Provide(func(env *moo.Environment, model db.InModelFactory, users *usermodels.Users, opLogger api.OperationLogger, optValidator OptValidation) (*Service, error) {
			validator := optValidator.Validator
			if validator == nil {
				validator = validation.Default
			}
			return NewService(env, model.Factory, users, opLogger, validator)
		})
	})
}
