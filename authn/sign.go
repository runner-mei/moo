package authn

import (
	"context"
	"errors"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/runner-mei/moo/bcrypto"
)

// ErrSignatureInvalid Specific instances for HS256 and company
var ErrSignatureInvalid = jwt.ErrSignatureInvalid
var ErrSigningMethodMissing = errors.New("signing method is missing")

// SigningMethod Implement SigningMethod to add new methods for signing or verifying tokens.
type SigningMethod = jwt.SigningMethod

// RegisterSigningMethod Register the "alg" name and a factory function for signing method.
// This is typically done during init() in the method's implementation
func RegisterSigningMethod(alg string, f func() SigningMethod) {
	jwt.RegisterSigningMethod(alg, func() jwt.SigningMethod {
		return f()
	})
}

// GetSigningMethod Get a signing method from an "alg" string
func GetSigningMethod(alg string) SigningMethod {
	return jwt.GetSigningMethod(alg)
}

type signingMethodDefault struct{}

var methodDefault = &signingMethodDefault{}

func init() {
	RegisterSigningMethod(methodDefault.Alg(), func() SigningMethod {
		return methodDefault
	})
}

func (m *signingMethodDefault) Alg() string {
	return "default"
}

// Only allow 'none' alg type if UnsafeAllowNoneSignatureType is specified as the key
func (m *signingMethodDefault) Verify(signingString, signature string, key interface{}) (err error) {
	if len(signature) > 3 {
		if signature[0] == '[' {
			if signature[2] == ']' {
				c := signature[1] - '0'
				if c >= 0 && c <= 9 {
					if bcrypto.Hashers[c].Comparer != nil {
						return bcrypto.Hashers[c].Comparer(signingString, signature[3:], key)
					}
				}
				return ErrSigningMethodMissing
			}

			idx := strings.Index(signature, "]")
			if idx > 0 {
				alg := signature[1:idx]
				signingMethod := GetSigningMethod(alg)
				if signingMethod == nil {
					return ErrSigningMethodMissing
				}
				return signingMethod.Verify(signingString, signature[idx+1:], key)
			}
		}
	}
	// If signing method is none, signature must be an empty string
	if signature != signingString {
		return jwt.ErrSignatureInvalid
	}

	// Accept 'none' signing method.
	return nil
}

// Only allow 'none' signing if UnsafeAllowNoneSignatureType is specified as the key
func (m *signingMethodDefault) Sign(signingString string, key interface{}) (string, error) {
	if len(signingString) > 3 {
		if signingString[0] == '[' {
			if signingString[2] == ']' {
				return signingString, nil
			}
			idx := strings.Index(signingString, "]")
			if idx > 0 {
				return signingString, nil
			}
		}
	}

	return bcrypto.DefaultHasher(context.Background(), signingString)
}

func CreateVerify(method string, secretKey []byte) (func(password, excepted string) error, error) {
	signingMethod := GetSigningMethod(method)
	if signingMethod == nil {
		return nil, errors.New("算法 '" + method + "' 不支持")
	}

	return func(password, excepted string) error {
		return signingMethod.Verify(password, excepted, secretKey)
	}, nil
}
