package services

import "errors"

type HasLock interface {
	IsLocked() bool
}

func LockCheck() AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnBeforeAuth(AuthFunc(func(ctx *AuthContext) error {
			u, ok := ctx.Authentication.(HasLock)
			if !ok {
				return errors.New("user is unsupported for the error lock")
			}
			if u.IsLocked() {
				return ErrUserLocked
			}
			return nil

			// if u.LockedAt.IsZero() || u.Name == "admin" {
			// 	return nil
			// }
			//
			// if u.LockedTimeExpires == 0 {
			// 	return ErrUserLocked
			// }
			// if time.Now().Before(u.LockedAt.Add(u.LockedTimeExpires)) {
			// 	return ErrUserLocked
			// }
			// return nil
		}))
		return nil
	})
}
