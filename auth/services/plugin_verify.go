package services

type Authorizer interface {
	Auth(ctx *AuthContext) (bool, error)
}

func DefaultUserCheck() AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnAuth(func(ctx *AuthContext) (bool, error) {
			au, ok := ctx.Authentication.(Authorizer)
			if !ok {
				return false, nil
			}
			return au.Auth(ctx)
		})
		return nil
	})
}

// func TptUserCheck() AuthOption {
// 	return AuthOptionFunc(func(auth *AuthService) error {
// 		verify, initerr := CreateVerify(tptMethod.Alg(), nil)
// 		if initerr != nil {
// 			return initerr
// 		}

// 		auth.OnAuth(func(ctx *AuthContext) (bool, error) {

// 			u, ok := ctx.Authentication.(*UserInfo)
// 			if !ok {
// 				return false, nil
// 			}

// 			var method string
// 			if o := u.Data["source"]; o != nil {
// 				method = fmt.Sprint(o)
// 			}

// 			if method != "" && method != "builin" {
// 				return false, nil
// 			}

// 			if u.Password == "" {
// 				return true, ErrPasswordEmpty
// 			}

// 			err := verify(ctx.Request.Password, u.Password)
// 			if err != nil {
// 				if err == ErrSignatureInvalid {
// 					return true, ErrPasswordNotMatch
// 				}
// 				return true, err
// 			}
// 			return true, nil
// 		})
// 		return nil
// 	})
// }

// var _ Authorizer = &apiUser{}

// type apiUser struct {
// 	signString string
// }

// func (u *apiUser) Auth(ctx *AuthContext) (bool, error) {
// 	if u.signString == ctx.Request.Password {
// 		return true, nil
// 	}
// 	return true, ErrPasswordNotMatch
// }

// func TptInternalUserCheck(env *moo.Environment) AuthOption {
// 	// internalUserEnabled: env.Config.BoolWithDefault("jwt.internal.users", true),
// 	return AuthOptionFunc(func(auth *AuthService) error {
// 		apiUsers := readUsers(env)

// 		auth.OnLoad(func(ctx *AuthContext) (interface{}, error) {
// 			lowerName := strings.ToLower(ctx.Request.Username)
// 			signString, ok := apiUsers[lowerName]
// 			if !ok {
// 				return nil, nil
// 			}
// 			ctx.Response.UserSource = "api"

// 			return &apiUser{
// 				signString: signString,
// 			}, nil
// 		})
// 		return nil
// 	})
// }
