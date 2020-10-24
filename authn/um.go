package authn

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/netutil"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn/services"
)

var _ api.User = &tokenUser{}
var _ services.User = &tokenUser{}

type UserManager interface {
	api.UserManager
	services.UserManager

	Create(ctx context.Context, name, nickname, source, password string, fields map[string]interface{}, roles []string) (interface{}, error)
}

type tokenUser struct {
	api.User

	Username string
	token    *jwt.Token
}

func (u *tokenUser) Name() string {
	return u.Username
}

// 呢称
func (u *tokenUser) Nickname() string {
	return u.Username
}

func (u *tokenUser) IsLocked() bool {
	return false
}

func (u *tokenUser) Source() string {
	return "api"
}

func (u *tokenUser) HasRole(name string) bool {
	if name == api.RoleBgOperator {
		return true
	}
	return u.User.HasRole(name)
}

func (u *tokenUser) Roles() []string {
	roles := u.User.Roles()
	if len(roles) == 0 {
		return []string{
			api.RoleBgOperator,
		}
	}
	return append(roles, api.RoleBgOperator)
}

func (u *tokenUser) IngressIPList() ([]netutil.IPChecker, error) {
	return []netutil.IPChecker{}, nil
}

func tokenToUser(um api.UserManager, cb loong.TokenCheckFunc) loong.TokenCheckFunc {
	if um == nil {
		return cb
	}

	tptUser, err := um.UserByName(context.Background(), api.UserBgOperator, api.UserIncludeDisabled())
	if err != nil {
		panic(err)
	}

	return func(ctx context.Context, req *http.Request, tokenStr string) (context.Context, error) {
		ctx, err := cb(ctx, req, tokenStr)
		if err != nil {
			return ctx, err
		}

		return api.ContextWithReadCurrentUser(ctx, api.ReadCurrentUserFunc(func(ctx context.Context) (api.User, error) {
			o := loong.TokenFromContext(ctx)
			if o == nil {
				return nil, loong.ErrTokenNotFound
			}

			token, ok := o.(*jwt.Token)
			if !ok {
				return nil, errors.New("token isnot jwt.Token")
			}

			claims, ok := token.Claims.(*jwt.StandardClaims)
			if !ok {
				return nil, errors.New("claims isnot jwt.StandardClaims")
			}

			ss := strings.SplitN(claims.Audience, " ", 2)
			if len(ss) < 2 {
				return nil, errors.New("Audience '" + claims.Audience + "' is invalid")
			}

			userid, cerr := strconv.ParseInt(ss[0], 10, 64)
			if cerr != nil {
				return nil, errors.New("Audience '" + claims.Audience + "' is invalid")
			}

			if userid == 0 {
				return &tokenUser{
					User:     tptUser,
					Username: ss[1],
					token:    token,
				}, nil
			}
			return um.UserByID(ctx, userid)
		})), nil
	}
}
