package users

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/moo/authz"
	"github.com/runner-mei/moo/users/usermodels"
)

var DefaultTimeout = 10 * time.Minute

func ErrUsernameNotFound(username string) error {
	return errors.ErrNotFoundWithText("user with name is '" + username + "' isnot found")
}

func ErrUserIDNotFound(userid int64) error {
	return errors.ErrNotFoundWithText("user with id is '" + strconv.FormatInt(userid, 10) + "' isnot found")
}

func ErrUserDisabled(username string) error {
	return errors.New("user with name is '" + username + "' is disabled")
}

type user struct {
	um        *UserManager
	u         usermodels.User
	roles     []usermodels.Role
	roleNames []string
	roleIDs   []int64
	profiles  map[string]string
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

// func (u *user) HasAdminRole() bool {
// 	return u.hasRoleID(u.um.adminRole.ID)
// }

// func (u *user) IsGuest() bool {
// 	return len(u.roles) == 1 && u.roles[0].ID == u.um.guestRole.ID
// }

// func (u *user) hasRoleID(id int64) bool {
// 	for idx := range u.roles {
// 		if u.roles[idx].ID == id {
// 			return true
// 		}
// 	}
// 	return false
// }

func (u *user) HasRole(role string) bool {
	for _, name := range u.roleNames {
		if name == role {
			return true
		}
	}
	return false
}

// func (u *user) IsMemberOf(group int64) bool {
// 	for _, id := range u.usergroups {
// 		if id == group {
// 			return true
// 		}
// 	}
// 	return false
// }

func (u *user) WriteProfile(key, value string) error {
	if value == "" {
		_, err := u.um.Users.DeleteProfile(context.Background(), u.ID(), key)
		if err != nil {
			return errors.Wrap(err, "DeleteProfile")
		}
		if u.profiles != nil {
			delete(u.profiles, key)
		}
		return nil
	}

	err := u.um.Users.WriteProfile(context.Background(), u.ID(), key, value)
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
	value, err := u.um.Users.ReadProfile(context.Background(), u.ID(), key)
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

func (u *user) getRoleIDs() []int64 {
	if len(u.roleIDs) != 0 {
		return u.roleIDs
	}
	roleIDs := make([]int64, 0, len(u.roles))
	for idx := range u.roles {
		roleIDs = append(roleIDs, u.roles[idx].ID)
	}

	// if u.Name() == UserAdmin && u.um.adminRole.ID > 0 {
	// 	roleIDs = append(roleIDs, u.um.adminRole.ID)
	// }

	u.roleIDs = roleIDs
	return u.roleIDs
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

func (u *user) HasPermission(ctx context.Context, permissionID string) (bool, error) {
	state, err := u.um.authorizer.IsGranted(ctx, permissionID, u.getRoleIDs())
	if err != nil {
		return false, err
	}
	return state.IsGranted(), nil
}

func (u *user) HasPermissionAny(ctx context.Context, permissionIDs []string) (bool, error) {
	for _, permissionID := range permissionIDs {
		state, err := u.um.authorizer.IsGranted(ctx, permissionID, u.getRoleIDs())
		if err != nil {
			return false, err
		}
		if state.IsGranted() {
			return true, nil
		}
	}
	return false, nil
}

func (u *user) IsGranted(ctx context.Context, permissionID string) (authz.PermissionState, error) {
	return u.um.authorizer.IsGranted(ctx, permissionID, u.getRoleIDs())
}
