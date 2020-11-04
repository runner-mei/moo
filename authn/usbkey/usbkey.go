package usbkey

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/tid"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/authn/services"
	"github.com/runner-mei/moo/authn/uuidlogin"
	"github.com/runner-mei/moo/users/usermodels"
)

type UsbKey struct {
	logger      log.Logger
	renderer    *authn.Renderer
	usbCheckURL string
	client      *http.Client
	sessions    authn.Sessions
	users       *usermodels.Users
	uuidLogin   *uuidlogin.UuidLogin
}

// NewUSBKey creates a Client with the provided Options.
func NewUSBKey(env *moo.Environment,
	usbCheckURL string,
	uuidLogin *uuidlogin.UuidLogin,
	renderer *authn.Renderer,
	sessions authn.Sessions,
	users *usermodels.Users) *UsbKey {
	return &UsbKey{
		logger:      env.Logger.Named("usbkey"),
		usbCheckURL: usbCheckURL,
		uuidLogin:   uuidLogin,
		renderer:    renderer,
		sessions:    sessions,
		users:       users,
	}
}

func (c *UsbKey) validate(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, error) {
	urlStr := c.usbCheckURL
	if len(r.URL.RawQuery) > 0 {
		if strings.Contains(urlStr, "?") {
			urlStr = urlStr + "&" + r.URL.RawQuery
		} else {
			urlStr = urlStr + "?" + r.URL.RawQuery
		}
	}

	request, err := http.NewRequest(r.Method, urlStr, r.Body)
	if err != nil {
		c.logger.Error("创建请求失败", log.Error(err))
		return "", err
	}
	for _, key := range []string{
		"Accept",
		"Accept-Encoding",
		"Content-Type",
		"X-HTTP-Method-Override",
		"X-Forwarded-For",
		"X-Real-IP",
	} {
		value := r.Header.Get(key)
		if value != "" {
			request.Header.Set(key, value)
		}
	}
	if c.client == nil {
		c.client = authn.InsecureHttpClent
	}
	response, err := c.client.Do(request)
	if err != nil {
		c.logger.Error("发送请求失败", log.Error(err))
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		var message string
		bs, err := ioutil.ReadAll(r.Body)
		if err != nil {
			message = response.Status
		} else {
			var o struct {
				Message string `json:"message"`
			}
			err = json.Unmarshal(bs, &o)
			if err != nil || o.Message == "" {
				message = string(bs)
			} else {
				message = o.Message
			}
		}

		err = errors.New(message)
		c.logger.Error("读响应失败", log.Error(err))
		return "", err
	}

	bs, err := ioutil.ReadAll(response.Body)
	if err != nil {
		c.logger.Error("读响应失败", log.Error(err))
		return "", err
	}

	var o struct {
		DomainUser string `json:"domainUser"`
	}
	err = json.Unmarshal(bs, &o)
	if err != nil {
		c.logger.Error("解析响应失败", log.Error(err))
		return "", err
	}

	return o.DomainUser, nil
}

func (c *UsbKey) Login(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	isLogin := queryParams.Get("login") == "true"
	isUUID := queryParams.Get("uuid") == "true"

	renderError := func(statusCode int, err string) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		io.WriteString(w, "{\"success\": false, \"message\": \"")
		io.WriteString(w, strings.Replace(err, "\"", "'", -1))
		io.WriteString(w, "\"}")
	}
	renderOK := func(statusCode int, to, username string) {
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		io.WriteString(w, "{\"success\": true, \"message\": \"登录成功\", \"redirect\": \"")
		io.WriteString(w, strings.Replace(to, "\"", "'", -1))
		if isUUID {
			io.WriteString(w, "\", \"uuid\": \"")
			uuid := tid.GenerateID()
			c.uuidLogin.Set(ctx, uuid, username)
			io.WriteString(w, uuid)
		}

		io.WriteString(w, "\"}")
	}

	if isLogin {
		renderError = func(statusCode int, err string) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, err)
		}
	}

	username, err := c.validate(ctx, w, r)
	if err != nil {
		c.logger.Info("验证失败", log.Error(err))

		renderError(http.StatusUnauthorized, "验证失败: "+err.Error())
		return
	}

	user, err := c.users.GetUserByName(ctx, username)
	if err != nil {
		c.logger.Info("读用户信息失败", log.Error(err))
		if errors.IsNotFound(err) {
			renderError(http.StatusUnauthorized, "用户信息没找到，请添加这个用户： "+err.Error())
			return
		}
		renderError(http.StatusUnauthorized, "读用户信息失败： "+err.Error())
		return
	}

	address := authn.RealIP(r)

	sessionID, err := c.sessions.Login(ctx, user.ID, user.Name, address)
	if err != nil && errors.IsNotFound(err) {
		c.logger.Info("创建在线用户信息失败", log.Error(err))

		renderError(http.StatusUnauthorized, "创建在线用户信息失败： "+err.Error())
		return
	}

	q := r.URL.Query()
	redirect := q.Get("redirect")
	if redirect == "" {
		redirect = q.Get("service")
		if redirect == "" {
			redirect = q.Get("returnTo")
		}
	}
	authCtx := &services.AuthContext{
		Ctx:    ctx,
		Logger: c.logger,
		Request: services.LoginRequest{
			UserID:   user.ID,
			Username: username,
			Service:  redirect,
		},
		Response: services.LoginResult{
			IsOK:      true,
			SessionID: sessionID,
			IsNewUser: false,
		},
	}

	if !isLogin {
		authCtx.Ctx = authn.ContextWithRedirectFunc(ctx, func(c context.Context, isLogin bool, w http.ResponseWriter, r *http.Request, to string) error {
			renderOK(http.StatusOK, to, username)
			return nil
		})
	}
	c.renderer.LoginOK(authCtx, w, r)
}
