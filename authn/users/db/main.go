package dbusers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/authz"
	"github.com/runner-mei/moo/users/usermodels"
	"go.uber.org/atomic"
)

type userManager struct {
	logger            log.Logger

	signingMethod authn.SigningMethod
	secretKey     string
	lockedTimeExpires time.Duration
	users             *usermodels.Users
	authorizer        authz.Authorizer
	userByName        *cache.Cache
	userByID          *cache.Cache
	lastErr           atomic.Error

	superRole   usermodels.Role
	adminRole   usermodels.Role
	visitorRole usermodels.Role
	guestRole   usermodels.Role
}

func (um *userManager) Users(ctx context.Context, opts ...api.Option) ([]api.User, error) {
	options := InternalApply(opts)
	if options.IncludeDisabled {
		if o, found := um.userByName.Get("____all____"); found && o != nil {
			if ugArray, ok := o.([]api.User); ok && ugArray != nil {
				return ugArray, nil
			}
		}
	} else {
		if o, found := um.userByName.Get("____all_enabled____"); found && o != nil {
			if ugArray, ok := o.([]api.User); ok && ugArray != nil {
				return ugArray, nil
			}
		}
	}

	innerList, err := um.users.GetUsers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "query all usergroup fail")
	}

	um.ensureRoles(ctx)

	var uList = make([]api.User, 0, len(innerList))
	var enabledList = make([]api.User, 0, len(innerList))

	for idx := range innerList {
		u := &user{um: um, u: innerList[idx]}
		if err := um.load(ctx, u); err != nil {
			return nil, err
		}
		uList = append(uList, u)
		if !u.IsDisabled() {
			enabledList = append(enabledList, u)
		}
		um.usercacheIt(u)
	}

	um.userByName.SetDefault("____all____", uList)
	um.userByName.SetDefault("____all_enabled____", enabledList)

	if options.IncludeDisabled {
		return uList, nil
	}
	return enabledList, nil
}

func (um *userManager) usercacheIt(u api.User) {
	um.userByName.SetDefault(u.Name(), u)
	um.userByID.SetDefault(strconv.FormatInt(u.ID(), 10), u)
}

func (um *userManager) ensureRoles(ctx context.Context) {
	for _, data := range []struct {
		role *usermodels.Role
		name string
	}{
		{role: &um.superRole, name: api.RoleSuper},
		{role: &um.adminRole, name: api.RoleAdministrator},
		{role: &um.visitorRole, name: api.RoleVisitor},
		{role: &um.guestRole, name: api.RoleGuest},
	} {
		if data.role.ID != 0 {
			continue
		}

		err := um.users.UserDao.GetRoleByName(ctx, data.name)(data.role)
		if err != nil {
			data.role.Name = data.name
			um.logger.Warn("role isnot found", log.String("role", data.name), log.Error(err))
		} else {
			um.userByID.Flush()
			um.userByName.Flush()
		}
	}
}

func (um *userManager) UserByName(ctx context.Context, userName string, opts ...api.Option) (api.User, error) {
	options := InternalApply(opts)

	if o, found := um.userByName.Get(userName); found && o != nil {
		if u, ok := o.(api.User); ok && u != nil {
			if options.IncludeDisabled {
				return u, nil
			}

			if u.(*user).IsDisabled() {
				return nil, errors.New("user with name is '" + userName + "' is disabled")
			}
			return u, nil
		}
	}

	um.ensureRoles(ctx)

	var u = &user{um: um}
	err := um.users.UserDao.GetUserByName(ctx, userName)(&u.u)
	if err != nil {
		switch userName {
		case api.UserAdmin, api.UserBgOperator:
			u.u.Name = userName
			u.roleNames = []string{api.RoleAdministrator}
			u.roles = []usermodels.Role{um.adminRole}

			um.usercacheIt(u)
			return u, nil
		case api.UserGuest:
			u.u.Name = userName
			u.roleNames = []string{api.RoleGuest}
			u.roles = []usermodels.Role{um.guestRole}
			um.usercacheIt(u)
			return u, nil
		default:
			return nil, errors.Wrap(err, "query user with name is '"+userName+"' fail")
		}
	}

	if !options.IncludeDisabled {
		if u.IsDisabled() {
			return nil, errors.New("user with name is '" + userName + "' is disabled")
		}
	}

	err = um.load(ctx, u)
	if err != nil {
		return nil, err
	}
	switch userName {
	case api.UserAdmin, api.UserBgOperator:
		u.roleNames = []string{api.RoleAdministrator}
		u.roles = []usermodels.Role{um.adminRole}
	case api.UserGuest:
		u.roleNames = []string{api.RoleGuest}
		u.roles = []usermodels.Role{um.guestRole}
	}
	um.usercacheIt(u)
	return u, nil
}

func (um *userManager) UserByID(ctx context.Context, userID int64, opts ...api.Option) (api.User, error) {
	options := InternalApply(opts)

	if o, found := um.userByID.Get(strconv.FormatInt(userID, 10)); found && o != nil {
		if u, ok := o.(api.User); ok && u != nil {
			if options.IncludeDisabled {
				return u, nil
			}

			if u.(*user).IsDisabled() {
				return nil, errors.New("user with name is " + u.Name() + " is disabled")
			}
			return u, nil
		}
	}

	um.ensureRoles(ctx)

	var u = &user{um: um}
	err := um.users.UserDao.GetUserByID(ctx, userID)(&u.u)
	if err != nil {
		return nil, errors.Wrap(err, "query user with id is "+fmt.Sprint(userID)+"fail")
	}

	if !options.IncludeDisabled {
		if u.IsDisabled() {
			return nil, errors.New("user with name is " + u.Name() + " is disabled")
		}
	}

	err = um.load(ctx, u)
	if err != nil {
		return nil, err
	}
	um.usercacheIt(u)
	return u, nil
}

func (um *userManager) load(ctx context.Context, u *user) error {
	var err error
	u.roles, err = um.users.UserDao.GetRolesByUser(ctx, u.ID())
	if err != nil {
		return errors.Wrap(err, "query permissions and roles with user is "+u.Name()+" fail")
	}

	u.roleNames = nil
	u.Roles() // 缓存 roleNames

	if um.superRole.ID != 0 {
		for _, role := range u.roles {
			if role.ID == um.superRole.ID {
				return nil
			}
		}
	}

	if um.adminRole.ID != 0 {
		for _, role := range u.roles {
			if role.ID == um.adminRole.ID {
				return nil
			}
		}

		if u.u.Name == api.UserAdmin {
			u.roles = append(u.roles, um.adminRole)

			u.roleNames = nil
			u.Roles() // 缓存 roleNames

			return nil
		}
	}


	return nil
}

type user struct {
	um                  *userManager
	u                   usermodels.User
	roles               []usermodels.Role
	roleNames           []string
	profiles            map[string]string
}

func (u *user) IsDisabled() bool {
	return u.u.IsDisabled()
}

func (u *user) ID() int64 {
	return u.u.ID
}

func (u *user) Name() string {
	return u.u.Name
}

func (u *user) Nickname() string {
	return u.u.Nickname
}

func (u *user) HasAdminRole() bool {
	return u.hasRoleID(u.um.adminRole.ID)
}

func (u *user) IsGuest() bool {
	return len(u.roles) == 1 && u.roles[0].ID == u.um.guestRole.ID
}

func (u *user) hasRoleID(id int64) bool {
	for idx := range u.roles {
		if u.roles[idx].ID == id {
			return true
		}
	}
	return false
}

func (u *user) HasRole(role string) bool {
	for _, name := range u.roleNames {
		if name == role {
			return true
		}
	}
	return false
}

func (u *user) WriteProfile(key, value string) error {
	if value == "" {
		_, err := u.um.users.DeleteProfile(context.Background(), u.ID(), key)
		if err != nil {
			return errors.Wrap(err, "DeleteProfile")
		}
		if u.profiles != nil {
			delete(u.profiles, key)
		}
		return nil
	}

	err := u.um.users.WriteProfile(context.Background(), u.ID(), key, value)
	if err != nil {
		return errors.Wrap(err, "WriteProfile")
	}

	if u.profiles != nil {
		u.profiles[key] = value
	}
	return nil
}

func (u *user) ReadProfile(key string) (string, error) {
	if u.profiles != nil {
		value, ok := u.profiles[key]
		if ok {
			return value, nil
		}
	}
	value, err := u.um.users.ReadProfile(context.Background(), u.ID(), key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, "ReadProfile")
	}
	if u.profiles != nil {
		u.profiles[key] = value
	} else {
		u.profiles = map[string]string{key: value}
	}
	return value, nil
}

func (u *user) Roles() []string {
	if len(u.roleNames) != 0 {
		return u.roleNames
	}
	if len(u.roles) == 0 {
		return nil
	}

	roleNames := make([]string, 0, len(u.roles))
	for idx := range u.roles {
		roleNames = append(roleNames, u.roles[idx].Name)
	}

	u.roleNames = roleNames
	return u.roleNames
}

func (u *user) Data(key string) interface{} {
	switch key {
	case "id":
		return u.u.ID
	case "name":
		return u.u.Name
	case "nickname":
		return u.u.Nickname
	case "description":
		return u.u.Description
	case "attributes":
		return u.u.Attributes
	case "source":
		return u.u.Source
	case "created_at":
		return u.u.CreatedAt
	case "updated_at":
		return u.u.UpdatedAt
	default:
		if u.u.Attributes != nil {
			return u.u.Attributes[key]
		}
	}
	return nil
}

// 用户属性
func (u *user) ForEach(cb func(string, interface{})) {
	cb("id", u.u.ID)
	cb("name", u.u.Name)
	cb("nickname", u.u.Nickname)
	cb("description", u.u.Description)
	// cb("attributes", u.u.Attributes)
	cb("source", u.u.Source)
	cb("created_at", u.u.CreatedAt)
	cb("updated_at", u.u.UpdatedAt)

	if u.u.Attributes != nil {
		for k, v := range u.u.Attributes {
			cb(k, v)
		}
	}
}


func (u *user) HasPermission(permissionID string) bool {
	if u.Name() == api.UserAdmin {
		return true
	}

	if u.um.superRole.ID != 0 {
		for _, role := range u.roles {
			if role.ID == u.um.superRole.ID {
				return true
			}
		}
	}

	if u.um.adminRole.ID != 0 {
		for _, role := range u.roles {
			if role.ID == u.um.adminRole.ID {
				return true
			}
		}
	}
	// if u.um.visitorRole.ID != 0 && QUERY == op {
	// 	for _, role := range u.roles {
	// 		if role.ID == u.um.visitorRole.ID {
	// 			return true
	// 		}
	// 	}
	// }

	// for _, perm := range u.permissionsAndRoles {
	// 	if perm == permissionID {
	// 		return true
	// 	}
	// }

	// parents := u.um.permissionQueryer.Parents(permissionID)
	// for _, parent := range parents {
	// 	for _, perm := range u.permissionsAndRoles {
	// 		if perm == parent {
	// 			return true
	// 		}
	// 	}
	// }

	return false
}

type InternalOptions struct {
	IncludeDisabled bool
}

func InternalApply(opts []api.Option) InternalOptions {
	var o InternalOptions
	for _, opt := range opts {
		if _, ok := opt.(interface {
			IsIncludeDisabled()
		}); ok {
			o.IncludeDisabled = true
		}
	}
	return o
}
