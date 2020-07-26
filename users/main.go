package users

import (
	"context"
	"database/sql"
	"time"

	// gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/syncx"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/authz"
	"github.com/runner-mei/moo/users/usermodels"
)

type Authorizer interface {
	Refresh() error

	authz.Authorizer
}

func Create(env *moo.Environment, users *usermodels.Users, authorizer authz.Authorizer, logger log.Logger) (authn.UserManager, error) {
	if authorizer == nil {
		return nil, errors.New("authorizer is nil")
	}

	signingMethod := env.Config.StringWithDefault("users.signing.method", "default")
	um := &UserManager{
		logger:            logger,
		Users:             users,
		authorizer:        authorizer,
		signingMethod:     authn.GetSigningMethod(signingMethod),
		secretKey:         env.Config.StringWithDefault("users.signing.secret_key", ""),
		lockedTimeExpires: env.Config.DurationWithDefault("users.locked_time_expires", 0),
	}
	if um.signingMethod == nil {
		return nil, errors.New("users.signing.method '" + signingMethod + "' is missing")
	}
	// um.userCache.logger = logger
	// um.userCache.loadUsers = um.loadUsers

	um.userCache.defaultExpiration = um.lockedTimeExpires
	um.userCache.items = map[int64]userCacheItem{}
	um.userCache.findByName = users.GetUserByName
	um.userCache.findByID = users.GetUserByID
	um.userCache.load = um.loadUser2

	if refresher, ok := authorizer.(Authorizer); ok {
		refresh := func() {
			um.lastErr.Set(refresher.Refresh())
		}
		refresh()
		um.authorizeTicker.Init(5*time.Minute, refresh)
	}

	// time.AfterFunc(1*time.Minute, func() {
	// 	um.Users(context.Background())
	// })
	return um, nil
}

type UserManager struct {
	InnerUsers []string

	logger          log.Logger
	Users           *usermodels.Users
	authorizer      authz.Authorizer
	authorizeTicker syncx.Tickable

	signingMethod     authn.SigningMethod
	secretKey         string
	lockedTimeExpires time.Duration

	userCache UserCache
	lastErr   syncx.ErrorValue
}

// func (um *UserManager) Users(ctx context.Context, opts ...api.Option) ([]api.User, error) {
// 	if e := um.lastErr.Get(); e != nil {
// 		return nil, e
// 	}
// 	return um.userCache.Users(ctx, opts...)
// }

func (um *UserManager) UserByName(ctx context.Context, username string, opts ...api.Option) (api.User, error) {
	if e := um.lastErr.Get(); e != nil {
		return nil, e
	}
	u, err := um.userCache.UserByName(ctx, username, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			for _, name := range um.InnerUsers {
				if username == name {
					var u = &user{um: um}
					u.u.Name = username
					return u, um.loadUser(ctx, u)
				}
			}
		}
		return nil, err
	}

	return u, nil
}

func (um *UserManager) UserByID(ctx context.Context, userID int64, opts ...api.Option) (api.User, error) {
	if e := um.lastErr.Get(); e != nil {
		return nil, e
	}
	return um.userCache.UserByID(ctx, userID, opts...)
}

func (um *UserManager) loadUsers(ctx context.Context) ([]api.User, error) {
	roleList, err := um.Users.GetRoles(context.Background(), "", 0, 0)
	if err != nil {
		return nil, errors.Wrap(err, "read roles fail")
	}

	next, closer := um.Users.UserDao.GetUsers(ctx, &usermodels.UserQueryParams{Enabled: sql.NullBool{Valid: true, Bool: true}}, 0, 0, "")
	defer util.CloseWith(closer)

	var allList = make([]api.User, 0, 64)
	for {
		u := &user{um: um}
		ok, err := next(&u.u)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, errors.Wrap(err, "read users fail")
		}

		if !ok {
			break
		}
		allList = append(allList, u)
	}

	urnext, urcloser := um.Users.UserDao.GetUserAndRoleList(context.Background())
	defer util.CloseWith(urcloser)
	for {
		r := usermodels.UserAndRole{}
		ok, err := urnext(&r)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, errors.Wrap(err, "read userroles fail")
		}

		if !ok {
			break
		}

		uidx := -1
		for i := range allList {
			if allList[i].ID() == r.UserID {
				uidx = i
				allList[i].(*user).roleIDs = append(allList[i].(*user).roleIDs, r.RoleID)
				break
			}
		}

		if uidx >= 0 {
			for i := range roleList {
				if roleList[i].ID == r.RoleID {
					allList[uidx].(*user).roles = append(allList[uidx].(*user).roles, roleList[i])
					break
				}
			}
		}
	}

	for idx := range allList {
		if err := um.loadUser(ctx, allList[idx].(*user)); err != nil {
			return nil, err
		}
	}

	return allList, nil
}

func (um *UserManager) loadUser2(ctx context.Context, u *usermodels.User) (api.User, error) {
	var au user
	au.um = um
	au.u = *u
	return &au, um.loadUser(ctx, &au)
}

func (um *UserManager) loadUser(ctx context.Context, u *user) (err error) {
	// um.ensureRoles(ctx)

	u.um = um
	err = um.loadRolesForUser(ctx, u)
	if err != nil {
		return err
	}

	// switch u.Name() {
	// case UserAdmin, UserTPTNetwork:
	// 	u.roleNames = []string{RoleAdministrator}
	// 	u.roles = []usermodels.Role{um.adminRole}
	// 	u.roleIDs = []int64{um.adminRole.ID}
	// case UserGuest:
	// 	u.um = um
	// 	u.roleNames = []string{RoleGuest}
	// 	u.roles = []usermodels.Role{um.guestRole}
	// 	u.roleIDs = []int64{um.guestRole.ID}
	// default:
	// 	u.um = um
	// 	err = um.loadRolesForUser(ctx, u)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// if u.ID() > 0 {
	// 	u.groups, err = um.Users.GetGroupIDsByUserID(ctx, u.ID())
	// 	if err != nil {
	// 		return errors.Wrap(err, "query user and usergroup with user is "+u.Name()+" fail")
	// 	}
	// }
	return nil
}

func (um *UserManager) loadRolesForUser(ctx context.Context, u *user) (err error) {
	if len(u.roles) == 0 {
		u.roles, err = um.Users.UserDao.GetRolesByUserID(ctx, u.ID())
		if err != nil {
			return errors.Wrap(err, "query permissions and roles with user is "+u.Name()+" fail")
		}
	}

	u.roleNames = nil
	u.Roles() // 缓存 roleNames

	// if um.superRole.ID != 0 {
	// 	for _, role := range u.roles {
	// 		if role.ID == um.superRole.ID {
	// 			return nil
	// 		}
	// 	}
	// }

	// if um.adminRole.ID != 0 {
	// 	for _, role := range u.roles {
	// 		if role.ID == um.adminRole.ID {
	// 			return nil
	// 		}
	// 	}

	// 	if u.u.Name == UserAdmin {
	// 		u.roles = append(u.roles, um.adminRole)

	// 		u.roleNames = nil
	// 		u.Roles() // 缓存 roleNames
	// 		return nil
	// 	}
	// }
	return nil
}

// type InternalOptions struct {
// 	IncludeDisabled bool
// }

// func InternalApply(opts ...api.Option) InternalOptions {
// 	var o InternalOptions
// 	for _, opt := range opts {
// 		switch opt.(type) {
// 		case userIncludeDisabled:
// 			o.IncludeDisabled = true
// 		}
// 	}
// 	return o
// }
