package authclient

import (
	"crypto/sha1"
	"hash"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Option struct {
	URL         string
	SessionPath string
	SessionKey  string
	SessionHash func() hash.Hash
	SecretKey   []byte
	CurrentURL  func(*http.Request) url.URL
}

func SSO(opt *Option) func(w http.ResponseWriter, req *http.Request, noAuth, next http.Handler) {
	sessionKey := opt.SessionKey
	if sessionKey == "" {
		sessionKey = DefaultSessionKey
	}
	sessionPath := opt.SessionPath
	secretKey := opt.SecretKey

	if sessionPath == "" {
		sessionPath = "/" // 必须指定 Path, 否则会被自动赋成当前请求的 url 中的 path
	} else if !strings.HasPrefix(sessionPath, "/") {
		sessionPath = "/" + sessionPath
	}

	currentURL := opt.CurrentURL
	if currentURL == nil {
		currentURL = func(req *http.Request) url.URL {
			return *req.URL
		}
	}

	h := opt.SessionHash
	if h == nil {
		h = sha1.New
	}

	return func(w http.ResponseWriter, req *http.Request, noAuth, next http.Handler) {
		sess, err := GetValues(req, sessionKey, h, secretKey)
		if err != nil {
			log.Println("read session fail", err)
			noAuth.ServeHTTP(w, req)
			return
		}

		if sess == nil {
			log.Println("session isn't found")
			noAuth.ServeHTTP(w, req)
			return
		}

		if IsInvalid(sess) {
			log.Println("session is invalid")
			noAuth.ServeHTTP(w, req)
			return
		}

		ts := time.Now().Add(30 * time.Minute)
		sess.Set(SESSION_EXPIRE_KEY,
			strconv.FormatInt(ts.Unix(), 10))
		sess.Set("_TS",
			strconv.FormatInt(ts.Unix(), 10))
		sessionData := sess.Encode()

		http.SetCookie(w, &http.Cookie{
			Name:  sessionKey,
			Value: Sign(sessionData, h, secretKey) + "-" + sessionData,
			//Domain:   revel.CookieDomain,
			Path: sessionPath,
			//HttpOnly: true,
			//Secure:   revel.CookieSecure,
			// Expires: ts.UTC(), // 不指定过期时间，那么关闭浏览器后 cookie 会删除
		})

		next.ServeHTTP(w, req)
	}
}
