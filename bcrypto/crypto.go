package bcrypto

import (
	"context"
	"encoding/base64"

	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn/services"
	"golang.org/x/crypto/bcrypt"
)

var GoHasher = api.UserPasswordHasherFunc(func(ctx context.Context, s string) (string, error) {
	sum, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return "[9]" + base64.StdEncoding.EncodeToString(sum), nil
})

var GoComparer = func(signingString, signature string, key interface{}) error {
	sum, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}
	err = bcrypt.CompareHashAndPassword(sum, []byte(signingString))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return services.ErrPasswordNotMatch
		}
		return err
	}
	return nil
}

var DefaultHasher = GoHasher
var DefaultComparer = GoComparer

var Hashers [10]struct {
	Hasher   api.UserPasswordHasherFunc
	Comparer func(signingString, signature string, key interface{}) error
}

func init() {
	Hashers[9].Hasher = GoHasher
	Hashers[9].Comparer = GoComparer
}
