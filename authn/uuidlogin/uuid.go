package uuidlogin

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo/authn/services"
	"github.com/runner-mei/moo/users/usermodels"
)

type UuidLogin struct {
	logger      log.Logger
	renderer    *authn.Renderer
	users       *usermodels.Users
	sessions    authn.Sessions
	mu          sync.Mutex
	keys        map[string]string
	redirectURL string
}

// NewUuidLogin creates a Client with the provided Options.
func NewUuidLogin(env *moo.Environment,
	renderer *authn.Renderer,
	sessions authn.Sessions,
	users *usermodels.Users) *UuidLogin {
	redirectURL := env.Config.StringWithDefault(api.CfgUserRedirectTo, "")
	if redirectURL != "" {
		redirectURL = strings.Replace(redirectURL, "\\$\\{appRoot}", env.DaemonUrlPath, -1)
		redirectURL = strings.Replace(redirectURL, "${appRoot}", env.DaemonUrlPath, -1)
	}
	return &UuidLogin{
		logger:      env.Logger.Named("uuidlogin"),
		renderer:    renderer,
		users:       users,
		sessions:    sessions,
		keys:        map[string]string{},
		redirectURL: redirectURL,
	}
}

func (c *UuidLogin) Set(ctx context.Context, uuid, username string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys[uuid] = username
	return nil
}

func (c *UuidLogin) Login(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	uuid := queryParams.Get("uuid")

	c.mu.Lock()
	username, ok := c.keys[uuid]
	if ok {
		delete(c.keys, uuid)
	}
	c.mu.Unlock()

	renderError := func(statusCode int, err string) {
		if c.redirectURL == "" {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, err)
			return
		}

		http.Redirect(w, r, c.redirectURL+"?message="+url.QueryEscape(err), http.StatusTemporaryRedirect)
	}

	if !ok {
		c.logger.Info("UUID 不可识别", log.String("uuid", uuid))
		renderError(http.StatusUnauthorized, "没有找到这个 UUID, 可能已经用过了")
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

	c.renderer.LoginOK(authCtx, w, r)
}
