package services

type Authenticator interface {
	Auth(ctx *AuthContext) (bool, error)
}

func DefaultUserCheck() AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnAuth(func(ctx *AuthContext) (bool, error) {
			au, ok := ctx.Authentication.(Authenticator)
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
