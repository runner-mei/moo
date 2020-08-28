package users

import (
	"context"
	"strconv"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/users/usermodels"
)

func ErrUsergroupnameNotFound(groupname string) error {
	return errors.ErrNotFoundWithText("usergroup with id is '" + groupname + "' isnot found")
}

func ErrUsergroupIDNotFound(userid int64) error {
	return errors.ErrNotFoundWithText("usergroup with id is '" + strconv.FormatInt(userid, 10) + "' isnot found")
}

type errUsergroupDisabled struct {
	name string
}

func (e errUsergroupDisabled) Error() string {
	return "usergroup with name is '" + e.name + "' is disabled"
}

func ErrUsergroupDisabled(groupname string) error {
	return errUsergroupDisabled{name: groupname}
}

func IsUsergroupDisabled(err error) bool {
	_, ok := err.(errUsergroupDisabled)
	return ok
}

type usergroup struct {
	userManager      api.UserManager
	usergroupManager api.UsergroupManager
	ug               usermodels.Usergroup
	userids          []int64
}

func (ug *usergroup) IsDisabled() bool {
	return ug.ug.Disabled
}

func (ug *usergroup) ID() int64 {
	return ug.ug.ID
}

func (ug *usergroup) Name() string {
	return ug.ug.Name
}

func (ug *usergroup) ParentID() int64 {
	return ug.ug.ParentID
}

func (ug *usergroup) Users(ctx context.Context, opts ...Option) ([]User, error) {
	var list = make([]User, 0, 8)
	for _, userID := range ug.userids {
		u, err := ug.userManager.UserByID(ctx, userID, opts...)
		if err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, nil
}

func (ug *usergroup) HasUserID(ctx context.Context, userID int64) bool {
	for _, u := range ug.userids {
		if u == userID {
			return true
		}
	}
	return false
}

func (ug *usergroup) HasUser(ctx context.Context, user User) bool {
	return ug.HasUserID(ctx, user.ID())
}

func (ug *usergroup) Exists(ctx context.Context, user User) bool {
	return ug.HasUser(ctx, user)
}

// 父用户组
func (ug *usergroup) Parent(ctx context.Context) Usergroup {
	parent, err := ug.usergroupManager.UsergroupByID(ctx, ug.ug.ParentID)
	if err != nil {
		return nil
	}
	return parent
}

func NewUsergroupManager(env *moo.Environment, userManager api.UserManager, queryer usermodels.UsergroupQueryer) (api.UsergroupManager, error) {
	return &UsergroupCache{
		userManager: userManager,
		queryer:     queryer,
		name2id:     map[string]int64{},
	}, nil
}
