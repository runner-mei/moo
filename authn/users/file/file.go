package fileusers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/runner-mei/goutils/as"
	"github.com/runner-mei/goutils/netutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/authn/services"
	"go.uber.org/fx"
)

func init() {
	moo.On(func() moo.Option {
		return fx.Provide(func(env *moo.Environment, logger log.Logger) (authn.UserManager, api.UserManager, error) {
			fum, err := NewFileUserManager(env, logger)
			return fum, fum, err
		})
	})
}

type FileUserManager struct {
	env           *moo.Environment
	logger        log.Logger
	SigningMethod authn.SigningMethod
	SecretKey     string
}

func (h *FileUserManager) Create(ctx context.Context, name, nickname, source, password string, fields map[string]interface{}, roles []string) (interface{}, error) {
	var users, err = h.All()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var maxID int64
	var oldID int64
	var old map[string]interface{}

	if len(users) == 0 {

		adm := h.createAdminUser()
		op := h.createBgOpUser()

		if name == api.UserAdmin {
			oldID = adm.id
			old = adm.data
		}
		if name == api.UserBgOperator {
			oldID = op.id
			old = op.data
		}

		maxID = 10
		users = []map[string]interface{}{adm.data, op.data}
	}

	if old == nil {
		for _, user := range users {
			id := as.Int64WithDefault(user["id"], 0)
			if id != 0 && id > maxID {
				maxID = id
			}

			uname := as.StringWithDefault(user["username"], "")
			if uname == name {
				for k, v := range fields {
					user[k] = v
				}
				user["username"] = name
				user["nickname"] = nickname
				user["password"] = password
				user["source"] = source
				user["roles"] = roles

				oldID = id
				old = user
				break
			}
		}
	}

	var id int64 = maxID + 1
	if old == nil {
		if fields == nil {
			fields = map[string]interface{}{}
		}
		fields["username"] = name
		fields["nickname"] = nickname
		fields["password"] = password
		fields["source"] = source
		fields["roles"] = roles
		fields["id"] = id

		users = append(users, fields)
	} else {
		if oldID == 0 {
			oldID = id
			fields["id"] = id
		} else {
			id = oldID
		}
	}

	file := h.env.Fs.FromDataConfig("moo_users.json")
	out, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	err = json.NewEncoder(out).Encode(users)
	if err != nil {
		return nil, err
	}
	return id, out.Close()
}

func (h *FileUserManager) Read(ctx *services.AuthContext) (interface{}, services.User, error) {
	u, err := h.Find(context.Background(), ctx.Request.Username)
	if err != nil {
		return nil, nil, err
	}
	return u.id, u, nil
}

func (h *FileUserManager) Unlock(*services.AuthContext) error {
	return nil
}

func (h *FileUserManager) Lock(*services.AuthContext) error {
	return nil
}

// func (h *FileUserManager) Locked() ([]LockedUser, error) {
// 	return nil, nil
// }

func (h *FileUserManager) createAdminUser() *fileUser {
	return &fileUser{
		fum:      h,
		id:       1,
		name:     api.UserAdmin,
		password: "admin",
		data:     map[string]interface{}{"id": 1, "username": api.UserAdmin, "password": "admin"},
	}
}

func (h *FileUserManager) createBgOpUser() *fileUser {
	return &fileUser{
		fum:      h,
		id:       2,
		name:     api.UserBgOperator,
		password: "admin",
		data:     map[string]interface{}{"id": 2, "username": api.UserBgOperator, "password": "admin"},
	}
}

func (h *FileUserManager) All() ([]map[string]interface{}, error) {
	file := h.env.Fs.FromDataConfig("moo_users.json")
	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var users []map[string]interface{}
	err = json.NewDecoder(reader).Decode(&users)
	return users, err
}

func (h *FileUserManager) toAPIUser(ctx context.Context, userID int64, username string, fields map[string]interface{}) (*fileUser, error) {
	if userID == 0 {
		userID = as.Int64WithDefault(fields["id"], 0)
	}
	if username == "" {
		username = as.StringWithDefault(fields["username"], "")
	}
	return &fileUser{
		fum:      h,
		id:       userID,
		name:     username,
		password: as.StringWithDefault(fields["password"], ""),
		data:     fields,
	}, nil
}

func (h *FileUserManager) Find(ctx context.Context, name string) (*fileUser, error) {
	var users, err = h.All()
	if err != nil {
		if os.IsNotExist(err) {
			if name == api.UserAdmin {
				return h.createAdminUser(), nil
			}

			if name == api.UserBgOperator {
				return h.createBgOpUser(), nil
			}
			return nil, services.ErrUserNotFound
		}
		return nil, err
	}

	for _, user := range users {
		uname := as.StringWithDefault(user["username"], "")
		if uname == name {
			return h.toAPIUser(ctx, 0, uname, user)
		}
	}
	return nil, nil
}

func (h *FileUserManager) Users(ctx context.Context, opts ...api.Option) ([]api.User, error) {
	var users, err = h.All()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, services.ErrUserNotFound
		}
		return []api.User{h.createAdminUser(), h.createBgOpUser()}, nil
	}

	var results = make([]api.User, 0, len(users))
	for _, user := range users {
		u, err := h.toAPIUser(ctx, 0, "", user)
		if err != nil {
			return nil, err
		}
		results = append(results, u)
	}
	return results, nil
}

func (h *FileUserManager) UserByID(ctx context.Context, userID int64, opts ...api.Option) (api.User, error) {
	var users, err = h.All()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, services.ErrUserNotFound
		}
		return nil, err
	}

	for _, user := range users {
		id := as.Int64WithDefault(user["id"], 0)
		if id == userID {
			u, err := h.toAPIUser(ctx, id, "", user)
			if err != nil {
				return nil, err
			}
			return u, nil
		}
	}
	return nil, nil
}

func (h *FileUserManager) UserByName(ctx context.Context, userName string, opts ...api.Option) (api.User, error) {
	user, err := h.Find(ctx, userName)
	if err != nil {
		return nil, err
	}
	return user, nil
}

var _ services.User = &fileUser{}
var _ services.Authorizer = &fileUser{}

type fileUser struct {
	fum           *FileUserManager
	id            int64
	name          string
	password      string
	data          map[string]interface{}
	roles         []string
	ingressIPList []netutil.IPChecker
}

func (u *fileUser) ID() int64 {
	return u.id
}

func (u *fileUser) Name() string {
	return u.name
}

func (u *fileUser) Nickname() string {
	return u.name
}

func (u *fileUser) HasRole(role string) bool {
	for _, name := range u.roles {
		if role == name {
			return true
		}
	}
	return false
}

func (u *fileUser) HasAdminRole() bool {
	return u.HasRole(api.RoleAdministrator)
}

func (u *fileUser) IsGuest() bool {
	return u.HasRole(api.RoleGuest)
}

func (u *fileUser) Data(name string) interface{} {
	if u.data == nil {
		return nil
	}

	return u.data[name]
}
// 用户属性
func (u *fileUser) ForEach(cb func(string, interface{})) {
	cb("id", u.u.ID)
	cb("name", u.u.Name)

	if u.u.Attributes != nil {
		for k, v := range u.u.Attributes {
			cb(k, v)
		}
	}
}

func (u *fileUser) WriteProfile(key, value string) error {
	if value == "" {
		return nil // errors.New("DeleteProfile")
	}

	return nil // errors.New( "WriteProfile")
}

func (u *fileUser) ReadProfile(key string) (string, error) {
	return "", nil // errors.New("ReadProfile")
}

func (u *fileUser) HasPermission(permissionID string) bool {
	return true
}

var _ services.Authorizer = &fileUser{}

func (u *fileUser) Auth(ctx *services.AuthContext) (bool, error) {
	err := u.fum.SigningMethod.Verify(ctx.Request.Password, u.password, u.fum.SecretKey)
	if err != nil {
		if err == authn.ErrSignatureInvalid {
			return true, services.ErrPasswordNotMatch
		}
		return true, err
	}
	return true, nil
}

func (u *fileUser) Roles() []string {
	if u.roles != nil {
		return u.roles
	}

	o := u.Data("roles")
	if o == nil {
		u.roles = []string{}
		return nil
	}

	switch vv := o.(type) {
	case []string:
		u.roles = vv
		return vv
	case []interface{}:
		ss := make([]string, 0, len(vv))
		for _, v := range vv {
			ss = append(ss, fmt.Sprint(v))
		}
		u.roles = ss
		return ss
	}
	return nil
}

func (u *fileUser) IsLocked() bool {
	return false
}

func (u *fileUser) Source() string {
	return as.StringWithDefault(u.Data("source"), "")
}

const WhiteIPListFieldName = "white_address_list"

func (u *fileUser) IngressIPList() ([]netutil.IPChecker, error) {
	if len(u.ingressIPList) > 0 {
		return u.ingressIPList, nil
	}

	if o := u.Data(WhiteIPListFieldName); o != nil {
		s, ok := o.(string)
		if !ok {
			return nil, fmt.Errorf("value of '"+WhiteIPListFieldName+"' isn't string - %T: %s", o, o)
		}
		ipList, err := as.SplitStrings([]byte(s))
		if err != nil {
			return nil, fmt.Errorf("value of '"+WhiteIPListFieldName+"' isn't []string - %s", o)
		}
		u.ingressIPList, err = netutil.ToCheckers(ipList)
		if err != nil {
			return nil, fmt.Errorf("value of '"+WhiteIPListFieldName+"' isn't invalid ip range - %s", s)
		}
	}
	return u.ingressIPList, nil
}

func NewFileUserManager(env *moo.Environment, logger log.Logger) (authn.UserManager, error) {
	signingMethod := env.Config.StringWithDefault("users.signing.method", "default")
	fum := &FileUserManager{
		env:           env,
		logger:        logger,
		SigningMethod: authn.GetSigningMethod(signingMethod),
		SecretKey:     env.Config.StringWithDefault("users.signing.secret_key", ""),
	}
	if fum.SigningMethod == nil {
		return nil, errors.New("users.signing.method '"+signingMethod+"' is missing")
	} 
	return fum, nil
}
