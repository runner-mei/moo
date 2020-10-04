package authn

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	"github.com/runner-mei/loong"
	"github.com/runner-mei/moo"
)

func readJWTAuth(env *moo.Environment) (*loong.JWTAuth, error) {
	alg := env.Config.StringWithDefault("api_auth.jwt.alg", "HS256")
	if strings.HasPrefix(alg, "HS") {
		signStr := "tpt-sign-" + time.Now().Format(time.RFC3339)
		signStr = env.Config.StringWithDefault("api_auth.jwt.signKey", signStr)
		verifyStr := env.Config.StringWithDefault("api_auth.jwt.verifyKey", "")

		var signKey, verifyKey interface{}
		if signStr != "" {
			signKey = []byte(signStr)
		}
		if verifyStr != "" {
			verifyKey = []byte(verifyStr)
		}

		return loong.NewJWTAuth(alg, signKey, verifyKey), nil
	}
	if strings.HasPrefix(alg, "RS") {
		privateKeyString := env.Config.StringWithDefault("api_auth.jwt.privateKey", "")
		privateKeyBlock, _ := pem.Decode([]byte(privateKeyString))
		if privateKeyBlock == nil {
			return nil, errors.New("decode privateKey fail")
		}
		privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
		if err != nil {
			return nil, err
		}

		publicKeyString := env.Config.StringWithDefault("api_auth.jwt.publicKey", "")
		publicKeyBlock, _ := pem.Decode([]byte(publicKeyString))
		if publicKeyBlock == nil {
			return nil, errors.New("decode publicKey fail")
		}
		publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
		if err != nil {
			return nil, err
		}
		return loong.NewJWTAuth(alg, privateKey, publicKey), nil
	}
	return nil, errors.New("SigningMethod is unsupported")
}
