package users

import (
	"context"
	"fmt"
	"time"

	"github.com/runner-mei/goutils/as"
	"github.com/runner-mei/goutils/netutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/authn/services"
	"github.com/runner-mei/moo/authz"
	"github.com/runner-mei/moo/users/usermodels"
	"go.uber.org/fx"
)

// func Create(env *moo.Environment, users *usermodels.Users, authorizer authz.Authorizer, logger log.Logger) (authn.UserManager, error) {
// 	if authorizer == nil {
// 		authorizer = authz.EmptyAuthorizer{}
// 	}

// 	signingMethod := env.Config.StringWithDefault("users.signing.method", "default")
// 	um := &UserManager{
// 		logger:            logger,
// 		signingMethod:     authn.GetSigningMethod(signingMethod),
// 		secretKey:     	   env.Config.StringWithDefault("users.signing.secret_key", ""),
// 		users:             users,
// 		authorizer: 	   authorizer,
// 		userByName:        cache.New(5*time.Minute, 10*time.Minute),
// 		userByID:          cache.New(5*time.Minute, 10*time.Minute),
// 		lockedTimeExpires: env.Config.DurationWithDefault("users.locked_time_expires", 0),
// 	}
// 	if um.signingMethod == nil {
// 		return nil, errors.New("users.signing.method '"+signingMethod+"' is missing")
// 	}

// 	um.ensureRoles(context.Background())
// 	return um, nil
// }

func (um *UserManager) Create(ctx context.Context, name, nickname, source, password string, fields map[string]interface{}, roles []string) (interface{}, error) {
	user := &usermodels.User{
		Name:       name,
		Nickname:   nickname,
		Password:   password,
		Source:     source,
		Attributes: fields,
	}
	id, err := um.Users.CreateUserWithRoleNames(ctx, user, roles)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func (um *UserManager) Read(ctx *services.AuthContext) (interface{}, services.User, error) {
	var user = &userInfo{
		um:   um,
		user: &usermodels.User{},
	}
	err := um.Users.UserDao.GetUserByName(ctx.Ctx, ctx.Request.Username)(user.user)
	if err != nil {
		return nil, nil, err
	}
	return user.user.ID, user, err
}

func (um *UserManager) Unlock(ctx *services.AuthContext) error {
	return um.Users.UserDao.UnlockUserByUsername(ctx.Ctx, ctx.Request.Username)
}

func (um *UserManager) Lock(ctx *services.AuthContext) error {
	return um.Users.UserDao.LockUserByUsername(ctx.Ctx, ctx.Request.Username)
}

var _ services.User = &userInfo{}
var _ services.Authorizer = &userInfo{}

type userInfo struct {
	um   *UserManager
	user *usermodels.User

	ingressIPList []netutil.IPChecker
}

func (u *userInfo) Data(ctx context.Context, name string) interface{} {
	if u.user.Attributes == nil {
		return nil
	}

	return u.user.Attributes[name]
}

func (u *userInfo) Roles() []string {
	o := u.Data(context.Background(), "roles")
	if o == nil {
		return nil
	}
	switch vv := o.(type) {
	case []string:
		return vv
	case []interface{}:
		ss := make([]string, 0, len(vv))
		for _, v := range vv {
			ss = append(ss, fmt.Sprint(v))
		}
		return ss
	}
	return nil
}

func (u *userInfo) Auth(ctx *services.AuthContext) (bool, error) {
	err := u.um.signingMethod.Verify(ctx.Request.Password, u.user.Password, u.um.secretKey)
	if err != nil {
		if err == authn.ErrSignatureInvalid {
			return true, services.ErrPasswordNotMatch
		}
		return true, err
	}
	return true, nil
}

func (u *userInfo) IsLocked() bool {
	if u.user.LockedAt == nil || u.user.LockedAt.IsZero() || u.user.Name == "admin" {
		return false
	}

	if u.um.lockedTimeExpires == 0 {
		return true
	}
	if time.Now().Before(u.user.LockedAt.Add(u.um.lockedTimeExpires)) {
		return true
	}
	return false
}

func (u *userInfo) Source() string {
	return u.user.Source
}

const WhiteIPListFieldName = "white_address_list"

func (u *userInfo) IngressIPList() ([]netutil.IPChecker, error) {
	if len(u.ingressIPList) > 0 {
		return u.ingressIPList, nil
	}

	if o := u.Data(context.Background(), WhiteIPListFieldName); o != nil {
		var err error
		var ipList []string
		switch v := o.(type) {
		case []string:
			ipList = v
		case []interface{}:
			ipList = make([]string, 0, len(v))
			for idx := range v {
				if s := as.StringWithDefault(v[idx], ""); s != "" {
					ipList = append(ipList, s)
				}
			}
		case string:
			ipList, err = as.SplitStrings([]byte(v))
			if err != nil {
				return nil, fmt.Errorf("value of '"+WhiteIPListFieldName+"' isn't []string - %s", o)
			}
		default:
			return nil, fmt.Errorf("value of '"+WhiteIPListFieldName+"' isn't []string - %T: %s", o, o)
		}

		u.ingressIPList, err = netutil.ToCheckers(ipList)
		if err != nil {
			return nil, fmt.Errorf("value of '"+WhiteIPListFieldName+"' isn't invalid ip range - %v", ipList)
		}
	}
	return u.ingressIPList, nil
}

func init() {
	moo.On(func() moo.Option {
		return fx.Provide(func(env *moo.Environment, users *usermodels.Users, logger log.Logger) (authn.UserManager, api.UserManager, error) {
			um, err := Create(env, users, authz.EmptyAuthorizer{}, logger)
			return um, um, err
		})
	})
}
