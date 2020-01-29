package users

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	cache "github.com/patrickmn/go-cache"
	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/users/usermodels"
	"go.uber.org/atomic"
)

type PermissionCache interface {
	Parents(groupID string) []string
}

func CreateUserManager(driverName string, db *sql.DB, permCache PermissionCache, logger log.Logger) UserManager {
	factory, err := gobatis.New(&gobatis.Config{
		Tracer:     log.NewSQLTracer(logger),
		TagPrefix:  "xorm",
		TagMapper:  gobatis.TagSplitForXORM,
		DriverName: driverName,
		DB:         db,
		XMLPaths: []string{
			"gobatis",
		},
	})
	if err != nil {
		panic(err)
	}
	return Create(factory, permCache, logger)
}

func Create(factory *gobatis.SessionFactory, permCache PermissionCache, logger log.Logger) UserManager {
	reference := factory.SessionReference()
	userDao := usermodels.NewUserQueryer(reference)

	um := &userManager{
		logger:               logger,
		userDao:              userDao,
		permissionGroupCache: permCache,
		userByName:           cache.New(5*time.Minute, 10*time.Minute),
		userByID:             cache.New(5*time.Minute, 10*time.Minute),
	}
	um.ensureRoles(context.Background())
	return um
}

type userManager struct {
	logger               log.Logger
	userDao              usermodels.UserQueryer
	permissionGroupCache PermissionCache
	userByName           *cache.Cache
	userByID             *cache.Cache
	lastErr              atomic.Error

	superRole   usermodels.Role
	adminRole   usermodels.Role
	visitorRole usermodels.Role
	guestRole   usermodels.Role
}

func (um *userManager) Users(ctx context.Context, opts ...Option) ([]User, error) {
	if e := um.lastErr.Load(); e != nil {
		return nil, e
	}

	var includeDisabled bool
	for _, opt := range opts {
		switch opt.(type) {
		case userIncludeDisabled:
			includeDisabled = true
		}
	}

	if includeDisabled {
		if o, found := um.userByName.Get("____all____"); found && o != nil {
			if ugArray, ok := o.([]User); ok && ugArray != nil {
				return ugArray, nil
			}
		}
	} else {
		if o, found := um.userByName.Get("____all_enabled____"); found && o != nil {
			if ugArray, ok := o.([]User); ok && ugArray != nil {
				return ugArray, nil
			}
		}
	}

	innerList, err := um.userDao.GetUsers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "query all usergroup fail")
	}

	um.ensureRoles(ctx)

	var uList = make([]User, 0, len(innerList))
	var enabledList = make([]User, 0, len(innerList))

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

	if includeDisabled {
		return uList, nil
	}
	return enabledList, nil
}

func (um *userManager) usercacheIt(u User) {
	um.userByName.SetDefault(u.Name(), u)
	um.userByID.SetDefault(strconv.FormatInt(u.ID(), 10), u)
}

func (um *userManager) ensureRoles(ctx context.Context) {
	for _, data := range []struct {
		role *usermodels.Role
		name string
	}{
		{role: &um.superRole, name: RoleSuper},
		{role: &um.adminRole, name: RoleAdministrator},
		{role: &um.visitorRole, name: RoleVisitor},
		{role: &um.guestRole, name: RoleGuest},
	} {
		if data.role.ID != 0 {
			continue
		}

		err := um.userDao.GetRoleByName(ctx, data.name)(data.role)
		if err != nil {
			data.role.Name = data.name
			um.logger.Warn("role isnot found", log.String("role", data.name), log.Error(err))
		} else {
			um.userByID.Flush()
			um.userByName.Flush()
		}
	}
}

func (um *userManager) UserByName(ctx context.Context, userName string, opts ...Option) (User, error) {
	if e := um.lastErr.Load(); e != nil {
		return nil, e
	}

	var includeDisabled bool
	for _, opt := range opts {
		switch opt.(type) {
		case userIncludeDisabled:
			includeDisabled = true
		}
	}

	if o, found := um.userByName.Get(userName); found && o != nil {
		if u, ok := o.(User); ok && u != nil {
			if includeDisabled {
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
	err := um.userDao.GetUserByName(ctx, userName)(&u.u)
	if err != nil {
		switch userName {
		case UserAdmin:
			u.u.Name = userName
			u.roleNames = []string{RoleAdministrator}
			u.roles = []usermodels.Role{um.adminRole}

			um.usercacheIt(u)
			return u, nil
		case UserGuest:
			u.u.Name = userName
			u.roleNames = []string{RoleGuest}
			u.roles = []usermodels.Role{um.guestRole}
			um.usercacheIt(u)
			return u, nil
		default:
			return nil, errors.Wrap(err, "query user with name is '"+userName+"' fail")
		}
	}

	if !includeDisabled {
		if u.IsDisabled() {
			return nil, errors.New("user with name is '" + userName + "' is disabled")
		}
	}

	err = um.load(ctx, u)
	if err != nil {
		return nil, err
	}
	switch userName {
	case UserAdmin:
		u.roleNames = []string{RoleAdministrator}
		u.roles = []usermodels.Role{um.adminRole}
	case UserGuest:
		u.roleNames = []string{RoleGuest}
		u.roles = []usermodels.Role{um.guestRole}
	}
	um.usercacheIt(u)
	return u, nil
}

func (um *userManager) UserByID(ctx context.Context, userID int64, opts ...Option) (User, error) {
	if e := um.lastErr.Load(); e != nil {
		return nil, e
	}

	var includeDisabled bool
	for _, opt := range opts {
		switch opt.(type) {
		case userIncludeDisabled:
			includeDisabled = true
		}
	}

	if o, found := um.userByID.Get(strconv.FormatInt(userID, 10)); found && o != nil {
		if u, ok := o.(User); ok && u != nil {
			if includeDisabled {
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
	err := um.userDao.GetUserByID(ctx, userID)(&u.u)
	if err != nil {
		return nil, errors.Wrap(err, "query user with id is "+fmt.Sprint(userID)+"fail")
	}

	if !includeDisabled {
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
	u.roles, err = um.userDao.GetRolesByUser(ctx, u.ID())
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

		if u.u.Name == UserAdmin {
			u.roles = append(u.roles, um.adminRole)

			u.roleNames = nil
			u.Roles() // 缓存 roleNames

			return nil
		}
	}

	var roleIDs = make([]int64, len(u.roles))
	for idx := range u.roles {
		roleIDs[idx] = u.roles[idx].ID
	}

	if len(roleIDs) > 0 {
		u.permissionsAndRoles, err = um.userDao.GetPermissionsByRoleIDs(ctx, roleIDs)
		//err = um.db.PermissionGroupsAndRoles().Where(orm.Cond{"role_id IN": roleIDs}).All(&u.permissionsAndRoles)
		if err != nil {
			return errors.Wrap(err, "query permissions and roles with user is "+u.Name()+" fail")
		}
	}

	return nil
}

type user struct {
	um                  *userManager
	permissionsAndRoles []string
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
		_, err := u.um.userDao.DeleteProfile(context.Background(), u.ID(), key)
		if err != nil {
			return errors.Wrap(err, "DeleteProfile")
		}
		if u.profiles != nil {
			delete(u.profiles, key)
		}
		return nil
	}

	err := u.um.userDao.WriteProfile(context.Background(), u.ID(), key, value)
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
	value, err := u.um.userDao.ReadProfile(context.Background(), u.ID(), key)
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

func (u *user) HasPermission(permissionID string) bool {
	if u.Name() == UserAdmin {
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

	for _, perm := range u.permissionsAndRoles {
		if perm == permissionID {
			return true
		}
	}

	parents := u.um.permissionGroupCache.Parents(permissionID)
	for _, parent := range parents {
		for _, perm := range u.permissionsAndRoles {
			if perm == parent {
				return true
			}
		}
	}

	return false
}

type userIncludeDisabled struct{}

func (u userIncludeDisabled) apply() {}

type InternalOptions struct {
	IncludeDisabled bool
}

func InternalApply(opts ...Option) InternalOptions {
	var o InternalOptions
	for _, opt := range opts {
		switch opt.(type) {
		case userIncludeDisabled:
			o.IncludeDisabled = true
		}
	}
	return o
}
