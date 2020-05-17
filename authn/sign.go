package authn

import (
	"errors"

	jwt "github.com/dgrijalva/jwt-go"
)

// ErrSignatureInvalid Specific instances for HS256 and company
var ErrSignatureInvalid = jwt.ErrSignatureInvalid

// SigningMethod Implement SigningMethod to add new methods for signing or verifying tokens.
type SigningMethod interface {
	Verify(signingString, signature string, key interface{}) error // Returns nil if signature is valid
	Sign(signingString string, key interface{}) (string, error)    // Returns encoded signature or error
	Alg() string                                                   // returns the alg identifier for this method (example: 'HS256')
}

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
	// If signing method is none, signature must be an empty string
	if signature != signingString {
		return jwt.ErrSignatureInvalid
	}

	// Accept 'none' signing method.
	return nil
}

// Only allow 'none' signing if UnsafeAllowNoneSignatureType is specified as the key
func (m *signingMethodDefault) Sign(signingString string, key interface{}) (string, error) {
	return signingString, nil
}

func CreateVerify(method string, secretKey []byte) (func(password, excepted string) error, error) {
	var signingMethod SigningMethod = methodDefault

	signingMethod = GetSigningMethod(method)
	if signingMethod == nil {
		return nil, errors.New("算法 '" + method + "' 不支持")
	}

	return func(password, excepted string) error {
		return signingMethod.Verify(password, excepted, secretKey)
	}, nil
}
