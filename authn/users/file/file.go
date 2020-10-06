package fileusers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/as"
	"github.com/runner-mei/goutils/netutil"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/authn/services"
	"go.uber.org/fx"
)

var _ services.Authenticator = &fileUser{}
var _ services.User = &fileUser{}

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return fx.Provide(func(env *moo.Environment, logger log.Logger) (authn.UserManager, api.UserManager, error) {
			fum, err := NewFileUserManager(env, logger)
			return fum, fum, err
		})
	})
}

type FileUserManager struct {
	env           *moo.Environment
	filename      string
	logger        log.Logger
	SigningMethod authn.SigningMethod
	SecretKey     string
}

func readValue(fields map[string]interface{}, name string) string {
	o := fields[name]
	if o == nil {
		return ""
	}
	return as.StringWithDefault(o, "")
}

func (h *FileUserManager) CreateByDict(ctx context.Context, fields map[string]interface{}) (interface{}, error) {
	username := readValue(fields, "username")
	if username == "" {
		return nil, errors.New("username is missing")
	}
	nickname := readValue(fields, "nickname")
	if nickname == "" {
		nickname = username
	}
	password := readValue(fields, "password")
	if password == "" {
		return nil, errors.New("password is missing")
	}
	return h.Create(ctx, username, nickname, "", password, fields, nil)
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

	return id, h.writeUsers(ctx, users)
}

func (h *FileUserManager) Update(ctx context.Context, name string, values map[string]interface{}) error {
	var users, err = h.All()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	foundIndex := -1
	for idx, user := range users {

		u, err := h.toAPIUser(ctx, 0, "", user)
		if err != nil {
			return err
		}
		if u.name == name {
			foundIndex = idx
			break
		}
	}

	if foundIndex <= 0 {
		return services.ErrUserNotFound
	}

	for key, value := range values {
		if value == nil {
			delete(users[foundIndex], key)
		} else {
			users[foundIndex][key] = value
		}
	}
	return h.writeUsers(ctx, users)
}

func (h *FileUserManager) Delete(ctx context.Context, name string) error {
	var users, err = h.All()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	offset := 0
	for idx, user := range users {
		u, err := h.toAPIUser(ctx, 0, "", user)
		if err != nil {
			return err
		}
		if u.name == name {
			continue
		}

		if offset != idx {
			users[offset] = users[idx]
		}
		offset++
	}
	users = users[:offset]

	return h.writeUsers(ctx, users)
}

func (h *FileUserManager) Read(ctx *services.AuthContext) (interface{}, services.User, error) {
	_, u, err := h.Find(context.Background(), ctx.Request.Username)
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

func (h *FileUserManager) writeUsers(ctx context.Context, users []map[string]interface{}) error {
	if len(users) == 0 {
		filename := h.env.Fs.FromDataConfig(h.filename)
		return os.Remove(filename)
	}

	filename := h.env.Fs.FromDataConfig(h.filename)
	if util.FileExists(filename) {
		os.Remove(filename + ".old")
		os.Rename(filename, filename+".old")
	}

	out, err := os.Create(filename)
	if err != nil {
		os.Rename(filename+".old", filename)
		return err
	}
	defer out.Close()

	err = json.NewEncoder(out).Encode(users)
	if err != nil {
		out.Close()
		os.Remove(filename)
		os.Rename(filename+".old", filename)
	}
	return err
}

func (h *FileUserManager) All() ([]map[string]interface{}, error) {
	file := h.env.Fs.FromDataConfig(h.filename)
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

func (h *FileUserManager) Find(ctx context.Context, name string) ([]map[string]interface{}, *fileUser, error) {
	var users, err = h.All()
	if err != nil {
		if os.IsNotExist(err) {
			if name == api.UserAdmin {
				u := h.createAdminUser()
				return []map[string]interface{}{u.data}, u, nil
			}

			if name == api.UserBgOperator {
				u := h.createBgOpUser()
				return []map[string]interface{}{u.data}, u, nil
			}
			return nil, nil, services.ErrUserNotFound
		}
		return nil, nil, err
	}

	for _, user := range users {
		uname := as.StringWithDefault(user["username"], "")
		if uname == name {
			u, err := h.toAPIUser(ctx, 0, uname, user)
			return users, u, err
		}
	}
	return users, nil, nil
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
	_, user, err := h.Find(ctx, userName)
	if err != nil {
		return nil, err
	}
	return user, nil
}


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

func (u *fileUser) DisplayName(ctx context.Context, s ...string) string {
	if len(s) != 0 {
		if s[0] == "" {
			return u.name
		}

		formatedName := os.Expand(strings.Replace(s[0], "\\$\\{", "${", -1), func(placeholderName string) string {
			value := u.Data(ctx, placeholderName)
			return as.StringWithDefault(value, "")
		})
		return formatedName
	}
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

func (u *fileUser) HasRoleID(int64) bool {
	return false
}

func (u *fileUser) HasAdminRole() bool {
	return u.HasRole(api.RoleAdministrator)
}

func (u *fileUser) IsGuest() bool {
	return u.HasRole(api.RoleGuest)
}

func (u *fileUser) Data(ctx context.Context, name string) interface{} {
	if u.data == nil {
		return nil
	}

	return u.data[name]
}

// 用户属性
func (u *fileUser) ForEach(cb func(string, interface{})) {
	cb("id", u.id)
	cb("name", u.name)

	if u.data != nil {
		for k, v := range u.data {
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

func (u *fileUser) HasPermission(ctx context.Context, permissionID string) (bool, error) {
	return true, nil
}

func (u *fileUser) HasPermissionAny(ctx context.Context, permissionIDs []string) (bool, error) {
	return true, nil
}


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

	o := u.Data(nil, "roles")
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
	return as.StringWithDefault(u.Data(nil, "source"), "")
}

const WhiteIPListFieldName = "white_address_list"

func (u *fileUser) IngressIPList() ([]netutil.IPChecker, error) {
	if len(u.ingressIPList) > 0 {
		return u.ingressIPList, nil
	}

	if o := u.Data(nil, WhiteIPListFieldName); o != nil {
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
	signingMethod := env.Config.StringWithDefault(api.CfgUserSigningMethod, "default")
	fum := &FileUserManager{
		env:           env,
		filename:      env.Config.StringWithDefault(api.CfgUserFilename, "moo_users.json"),
		logger:        logger,
		SigningMethod: authn.GetSigningMethod(signingMethod),
		SecretKey:     env.Config.StringWithDefault(api.CfgUserSigningSecretKey, ""),
	}
	if fum.SigningMethod == nil {
		return nil, errors.New("users.signing.method '" + signingMethod + "' is missing")
	}
	return fum, nil
}
