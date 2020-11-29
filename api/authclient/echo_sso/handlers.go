package echo_sso

import (
	"hash"
	"log"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/runner-mei/moo/api/authclient"
)

func SSO(sessionKey, sessionPath string, h func() hash.Hash, secretKey []byte, skipVerifyIfEmpty ...bool) echo.MiddlewareFunc {
	if sessionPath == "" {
		sessionPath = "/" // 必须指定 Path, 否则会被自动赋成当前请求的 url 中的 path
	} else if !strings.HasPrefix(sessionPath, "/") {
		sessionPath = "/" + sessionPath
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			var sess url.Values
			var err error

			if len(skipVerifyIfEmpty) > 0 && skipVerifyIfEmpty[0] {
				sess, err = authclient.GetValuesWithSkipVerifyIfEmpty(c.Request(), sessionKey, h, secretKey)
			} else {
				sess, err = authclient.GetValues(c.Request(), sessionKey, h, secretKey)
			}

			if err != nil {
				log.Println("fetch session fail,", err)
				return echo.ErrUnauthorized
			}

			if sess == nil {
				log.Println("session isn't found")
				return echo.ErrUnauthorized
			}

			if authclient.IsInvalid(sess) {
				log.Println("session is invalid")
				return echo.ErrUnauthorized
			}

			return next(c)
		}
	}
}
