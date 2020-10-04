package authn

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/api/authclient"
	"github.com/runner-mei/moo/authn/services"
	"go.uber.org/fx"
)

type ArgWelcomeLocator struct {
	fx.In

	Locator WelcomeLocator `optional:"true"`
}

type AuthOut struct {
	fx.Out

	LoginManager *LoginManager
	Renderer     *Renderer
	JWT          loong.AuthValidateFunc `group:"authValidate"`
	Session      loong.AuthValidateFunc `group:"authValidate"`
}

type InAuthFunc struct {
	moo.In

	Funcs []loong.AuthValidateFunc `group:"authValidate"`
}


func init() {
	moo.On(func() moo.Option {
		return moo.Provide(func(env *moo.Environment, userManager UserManager, online Sessions, locator ArgWelcomeLocator, authopts services.InAuthOpts) (AuthOut, error) {
			loginManager, err := NewLoginManager(env, userManager, online, locator.Locator, authopts.Opts)
			if err != nil {
				return AuthOut{}, err
			}

			authValidates := loginManager.AuthValidates()
			return AuthOut{
				LoginManager: loginManager,
				Renderer:     loginManager.Renderer,
				JWT:          authValidates[0],
				Session:      authValidates[1],
			}, nil
		})
	})
}

type LoginManager struct {
	env         *moo.Environment
	logger      log.Logger
	cfg         *Config
	sessionHash func() hash.Hash
	Renderer    *Renderer
	userManager UserManager
	online      Sessions
	authSrv     *services.AuthService
	jwtConfig   *loong.JWTAuth
	expiresIn   time.Duration
}

func (mgr *LoginManager) Close() error {
	if closer, ok := mgr.online.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (mgr *LoginManager) StaticDir(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	mgr.Renderer.StaticDir(ctx, w, r)
}

func (mgr *LoginManager) LoginGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	values, err := mgr.GetSession(r)
	if err == nil {
		authCtx := &services.AuthContext{
			Logger: mgr.logger,
			Ctx:    ctx,
			// Request:  &services.LoginRequest{},
			// Response: &services.LoginResult{},
		}

		authCtx.Request.Username = queryParams.Get("username")
		authCtx.Request.Service = queryParams.Get("service")
		authCtx.Response.IsOK = true
		authCtx.Response.SessionID = values.Get(authclient.SESSION_ID_KEY)
		if oldUsername := values.Get(authclient.SESSION_USER_KEY); authCtx.Request.Username != oldUsername {
			mgr.logger.Warn("已经登录了， 先登出吧", log.String("sessionID", authCtx.Response.SessionID),
				log.String("old_username", oldUsername),
				log.String("new_username", authCtx.Request.Username))

			mgr.Logout(ctx, w, r)
			return
		}

		err = mgr.Renderer.LoginOK(authCtx, w, r)
		if err != nil {
			mgr.logger.Warn("生成登录页面出错", log.Error(err))
		}
		return
	}

	method := queryParams.Get("_method")
	if method == "POST" {
		mgr.LoginPost(ctx, w, r)
		return
	}

	err = mgr.Renderer.Relogin(ctx, w, r)
	if err != nil {
		mgr.logger.Warn("生成登录页面出错", log.Error(err))
	}
}

type LoginType int

const (
	tokenNone LoginType = iota
	tokenJWT
)

func (t LoginType) String() string {
	switch t {
	case tokenNone:
		return "none"
	case tokenJWT:
		return "jwt"
	default:
		return "LoginType-" + strconv.Itoa(int(t))
	}
}

func (mgr *LoginManager) LoginJWT(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	mgr.LoginWith(ctx, w, r, tokenJWT)
}

func (mgr *LoginManager) LoginPost(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	mgr.LoginWith(ctx, w, r, tokenNone)
}

func (mgr *LoginManager) LoginWith(ctx context.Context, w http.ResponseWriter, r *http.Request, loginType LoginType) {
	authCtx := &services.AuthContext{
		Logger: mgr.logger,
		Ctx:    ctx,
		// Request:  &services.LoginRequest{},
		// Response: &services.LoginResult{},
	}

	returnError := func(authCtx *services.AuthContext, w http.ResponseWriter, r *http.Request, err error) {
		authCtx.Logger.Warn("登录失败", log.String("username", authCtx.Request.Username),
			log.String("address", authCtx.Request.Address), log.Error(err))

		statusCode := errors.HTTPCode(err)
		ReturnError(w, r, err.Error(), statusCode)
	}
	returnOK := func(authCtx *services.AuthContext, w http.ResponseWriter, r *http.Request, value interface{}) {
		authCtx.Logger.Info("登录成功", log.String("username", authCtx.Request.Username),
			log.String("address", authCtx.Request.Address))

		ReturnJSON(w, r, value, http.StatusOK)
	}

	if loginType == tokenNone {
		returnError = func(authCtx *services.AuthContext, w http.ResponseWriter, r *http.Request, err error) {
			e := mgr.Renderer.ReturnError(authCtx, w, r, err)
			if e != nil {
				mgr.logger.Warn("生成登录页面出错", log.Error(e))
			}
		}
		returnOK = func(authCtx *services.AuthContext, w http.ResponseWriter, r *http.Request, value interface{}) {
			err := mgr.Renderer.LoginOK(authCtx, w, r)
			if err != nil {
				mgr.logger.Warn("生成登录页面出错", log.Error(err))
			}
		}
	}

	if r.Method == "GET" {
		queryParams := r.URL.Query()
		method := queryParams.Get("_method")
		if method != "POST" {
			returnError(authCtx, w, r, errors.NewError(http.StatusMethodNotAllowed, "Error in request"))
			return
		}

		authCtx.Request.Username = queryParams.Get("username")
		authCtx.Request.Password = queryParams.Get("password")
		authCtx.Request.Service = queryParams.Get("service")
		authCtx.Request.ForceLogin = queryParams.Get("force")
		authCtx.Request.CaptchaKey = queryParams.Get("captcha_key")
		authCtx.Request.CaptchaValue = queryParams.Get("captcha_value")
	} else {
		ctype := r.Header.Get(HeaderContentType)
		switch {
		case strings.HasPrefix(ctype, MIMEApplicationJSON):
			if err := json.NewDecoder(r.Body).Decode(authCtx.Request); err != nil {
				if ute, ok := err.(*json.UnmarshalTypeError); ok {
					msg := fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)
					returnError(authCtx, w, r, errors.NewError(http.StatusBadRequest, msg))
					return
				} else if se, ok := err.(*json.SyntaxError); ok {
					msg := fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())
					returnError(authCtx, w, r, errors.NewError(http.StatusBadRequest, msg))
					return
				}

				returnError(authCtx, w, r, errors.WithHTTPCode(err, http.StatusBadRequest))
				return
			}
		case strings.HasPrefix(ctype, MIMEApplicationXML), strings.HasPrefix(ctype, MIMETextXML):
			if err := xml.NewDecoder(r.Body).Decode(authCtx.Request); err != nil {
				if ute, ok := err.(*xml.UnsupportedTypeError); ok {
					msg := fmt.Sprintf("Unsupported type error: type=%v, error=%v", ute.Type, ute.Error())
					returnError(authCtx, w, r, errors.NewError(http.StatusBadRequest, msg))
					return
				} else if se, ok := err.(*xml.SyntaxError); ok {
					msg := fmt.Sprintf("Syntax error: line=%v, error=%v", se.Line, se.Error())
					returnError(authCtx, w, r, errors.NewError(http.StatusBadRequest, msg))
					return
				}
				returnError(authCtx, w, r, errors.WithHTTPCode(err, http.StatusBadRequest))
				return
			}
		case strings.HasPrefix(ctype, MIMEApplicationForm), strings.HasPrefix(ctype, MIMEMultipartForm):
			if strings.HasPrefix(ctype, MIMEMultipartForm) {
				if err := r.ParseMultipartForm(defaultMemory); err != nil {
					returnError(authCtx, w, r, errors.WithHTTPCode(err, http.StatusBadRequest))
					return
				}
			} else {
				if err := r.ParseForm(); err != nil {
					returnError(authCtx, w, r, errors.WithHTTPCode(err, http.StatusBadRequest))
					return
				}
			}
			params := r.Form

			authCtx.Request.Username = params.Get("username")
			authCtx.Request.Password = params.Get("password")
			authCtx.Request.Service = params.Get("service")
			authCtx.Request.ForceLogin = params.Get("force")
			authCtx.Request.CaptchaKey = params.Get("captcha_key")
			authCtx.Request.CaptchaValue = params.Get("captcha_value")
		default:
			returnError(authCtx, w, r, errors.NewError(http.StatusUnsupportedMediaType, "Unsupported media type"))
			return
		}
	}

	if authCtx.Request.Username == "" {
		returnError(authCtx, w, r, errors.NewError(http.StatusForbidden, "username is missing"))
		return
	}
	if authCtx.Request.Password == "" {
		returnError(authCtx, w, r, errors.NewError(http.StatusForbidden, "password is missing"))
		return
	}
	authCtx.Request.Address = RealIP(r)

	if loginType != tokenNone {
		authCtx.SkipCaptcha = true
	}

	err := mgr.authSrv.Auth(authCtx)
	if err != nil {
		returnError(authCtx, w, r, errors.WithHTTPCode(err, http.StatusForbidden))
		return
	}
	if !authCtx.Response.IsOK {
		returnError(authCtx, w, r, errors.WithHTTPCode(services.ErrPasswordNotMatch, http.StatusForbidden))
		return
	}

	if authCtx.Response.UserSource != "api" {
		if authCtx.Response.IsNewUser && authCtx.Request.UserID == nil {
			var roles []string
			u, ok := authCtx.Authentication.(services.User)
			if ok {
				roles = u.Roles()
			}
			userid, err := mgr.userManager.Create(ctx,
				authCtx.Request.Username,
				authCtx.Request.Username,
				"ldap",
				"",
				map[string]interface{}{},
				roles)
			if err != nil {
				returnError(authCtx, w, r, &services.ErrExternalServer{
					Msg: "内部错误",
					Err: err,
				})
				return
			}
			authCtx.Request.UserID = userid
		}

		authCtx.Response.SessionID, err = mgr.online.Login(ctx, authCtx.Request.UserID, authCtx.Request.Username, authCtx.Request.Address)
		if err != nil {
			returnError(authCtx, w, r, &services.ErrExternalServer{
				Msg: "内部错误",
				Err: errors.Wrap(err, "registr user to online table fail"),
			})
			return
		}
	}

	switch loginType {
	case tokenNone:
		returnOK(authCtx, w, r, nil)
	case tokenJWT:
		tokenString, err := mgr.generateJWT(authCtx.Ctx, w, r, authCtx.Response.SessionID, authCtx.Request.UserID, authCtx.Request.Username)
		if err != nil {
			returnError(authCtx, w, r, errors.Wrap(err, "Error while signing the token"))
			return
		}

		returnOK(authCtx, w, r, map[string]interface{}{
			"token":      tokenString,
			"expires_in": int(mgr.expiresIn.Seconds()),
		})
		return
	default:
		returnError(authCtx, w, r, errors.New("login is ok, but token type is unsupport - "+loginType.String()))
		return
	}
}

func (mgr *LoginManager) generateJWT(ctx context.Context, w http.ResponseWriter, r *http.Request, sessionID string, userID interface{}, username string) (string, error) {
	claims := &jwt.StandardClaims{
		Id:        sessionID,
		ExpiresAt: time.Now().Add(mgr.expiresIn).Unix(),
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Unix(),
	}

	if userID != nil {
		claims.Audience = fmt.Sprint(userID) + " " + username
		claims.Issuer = "hengwei"
	} else {
		claims.Audience = "0 " + username
		claims.Issuer = "hengwei-internal"
	}

	_, tokenString, err := mgr.jwtConfig.Encode(claims)
	return tokenString, err
}

func (mgr *LoginManager) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	returnError := func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		log.LoggerOrEmptyFromContext(ctx).Warn("登出失败", log.Error(err))

		statusCode := errors.HTTPCode(err)
		ReturnError(w, r, err.Error(), statusCode)
	}
	returnOK := func(ctx context.Context, w http.ResponseWriter, r *http.Request, value interface{}) {
		log.LoggerOrEmptyFromContext(ctx).Warn("登出成功")

		ReturnJSON(w, r, value, http.StatusOK)
	}

	if !IsConsumeJSON(r) {
		returnError = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
			log.LoggerOrEmptyFromContext(ctx).Warn("登出失败", log.Error(err))

			e := mgr.Renderer.Logout(ctx, w, r)
			if e != nil {
				mgr.logger.Warn("生成登出页面出错", log.Error(e))
			}
		}
		returnOK = func(ctx context.Context, w http.ResponseWriter, r *http.Request, value interface{}) {
			err := mgr.Renderer.Logout(ctx, w, r)
			if err != nil {
				mgr.logger.Warn("生成登出页面出错", log.Error(err))
			}
		}
	}

	var sessionID string
	session := loong.SessionFromContext(ctx)
	if session == nil {
		values, err := mgr.GetSession(r)
		if err != nil {
			logger := mgr.logger
			if cookie, _ := r.Cookie(authclient.DefaultSessionKey); cookie != nil {
				logger = logger.With(log.Stringer("cookie", cookie))
			}
			ctx = log.ContextWithLogger(ctx, logger)

			returnError(ctx, w, r, errors.Wrap(err, "读会话失败"))
			return
		}
		sessionID = values.Get(authclient.SESSION_ID_KEY)

		ctx = log.ContextWithLogger(ctx, mgr.logger.With(log.String("session", sessionID),
			log.String("username", values.Get(authclient.SESSION_USER_KEY)),
			log.String("address", RealIP(r))))
	} else {
		sessionInfo, ok := session.(*SessionInfo)
		if !ok {
			ctx = log.ContextWithLogger(ctx, mgr.logger)

			returnError(ctx, w, r, errors.NewError(http.StatusBadRequest, "sess is missing"))
			return
		}
		sessionID = sessionInfo.UUID

		ctx = log.ContextWithLogger(ctx, mgr.logger.With(log.String("session", sessionID),
			log.String("username", sessionInfo.Username),
			log.String("address", sessionInfo.Address)))
	}

	if err := mgr.online.Logout(ctx, sessionID); err != nil {
		returnError(ctx, w, r, errors.Wrap(err, "unregistr user from online table fail"))
		return
	}

	returnOK(ctx, w, r, map[string]interface{}{
		"message": "OK",
	})
}

func (mgr *LoginManager) GetCurrentToken(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	values, err := mgr.GetSession(r)
	if err != nil {
		ReturnError(w, r, "sess is missing", http.StatusUnauthorized)
		return
	}
	sessionInfo, err := mgr.online.Get(ctx, values.Get(authclient.SESSION_ID_KEY))
	if err != nil {
		ReturnError(w, r, "sess is notfound", http.StatusUnauthorized)
		return
	}

	tokenString, err := mgr.generateJWT(ctx, w, r, sessionInfo.UUID, sessionInfo.UserID, sessionInfo.Username)
	if err != nil {
		ReturnError(w, r, errors.Wrap(err, "Error while signing the token").Error(), http.StatusUnauthorized)
		return
	}

	ReturnJSON(w, r, map[string]interface{}{
		"token":      tokenString,
		"expires_in": int(mgr.expiresIn.Seconds()),
	}, http.StatusOK)
}

func (mgr *LoginManager) Get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	user, err := api.ReadUserFromContext(ctx)
	if err != nil || user == nil {
		values, err := mgr.GetSession(r)
		if err == nil {
			sessionInfo, err := mgr.online.Get(ctx, values.Get(authclient.SESSION_ID_KEY))
			if err != nil {
				ReturnError(w, r, "sess is notfound", http.StatusUnauthorized)
				return
			}

			ReturnJSON(w, r, map[string]interface{}{
				"id": sessionInfo.UserID,
				// roles: data.roles,
				// name: data.name,
				// avatar: data.avatar,
				"roles": []string{"admin"},
				"name":  values.Get(authclient.SESSION_USER_KEY),
			}, http.StatusOK)
			return
		}

		ReturnError(w, r, "token is missing", http.StatusBadRequest)
		return
	}

	ReturnJSON(w, r, map[string]interface{}{
		"id": user.ID(),
		// roles: data.roles,
		// name: data.name,
		// avatar: data.avatar,
		"roles": []string{"admin"},
		"name":  user.Nickname(),
	}, http.StatusOK)
}

func (mgr *LoginManager) List(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	list, err := mgr.online.All(ctx)
	if err != nil {
		ReturnError(w, r, "list online users fail - "+err.Error(), http.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, list, http.StatusOK)
}

func (mgr *LoginManager) Signature(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	queryParams.Del("token")
	queryParams.Set(authclient.SESSION_EXPIRE_KEY, "session")
	queryParams.Set(authclient.SESSION_VALID_KEY, "true")
	if user := queryParams.Get("username"); user != "" {
		queryParams.Set(authclient.SESSION_USER_KEY, user)
	}

	value := authclient.Encode(queryParams, mgr.sessionHash, mgr.cfg.SessionSecretKey)

	http.SetCookie(w, &http.Cookie{Name: mgr.cfg.SessionKey,
		Value: value,
		Path:  mgr.cfg.SessionPath})

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(map[string]interface{}{"name": mgr.cfg.SessionKey,
		"value": value,
		"path":  mgr.cfg.SessionPath})
	if err != nil {
		mgr.logger.Warn("generate login ticket fail", log.Error(err))
	}
}

// GetSession 从当前请求是获取该请求的会话
func (mgr *LoginManager) GetSession(r *http.Request) (url.Values, error) {
	return authclient.GetValues(r, mgr.cfg.SessionKey, mgr.sessionHash, mgr.cfg.SessionSecretKey)
}

// CurrentUser 从当前请求是获取该请求的用户名
func (mgr *LoginManager) CurrentUsername(r *http.Request) string {
	values, err := mgr.GetSession(r)
	if err != nil {
		return ""
	}
	return values.Get(authclient.SESSION_USER_KEY)
}

func (mgr *LoginManager) AuthValidates() []loong.AuthValidateFunc {
	return []loong.AuthValidateFunc{
		loong.TokenVerify(
			[]loong.TokenFindFunc{
				loong.TokenFromQuery,
				loong.TokenFromHeader,
			},
			[]loong.TokenCheckFunc{
				tokenToUser(mgr.userManager, loong.JWTCheck(mgr.jwtConfig)),
			}),

		loong.AuthValidateFunc(func(ctx context.Context, req *http.Request) (context.Context, error) {
			values, err := mgr.GetSession(req)
			if err != nil {
				if err == authclient.ErrCookieNotFound || err == authclient.ErrCookieEmpty {
					return nil, loong.ErrTokenNotFound
				}
				return nil, err
			}

			if mgr.userManager == nil {
				return ctx, nil
			}

			return api.ContextWithUser(ctx, api.ReadCurrentUserFunc(func(ctx context.Context) (api.User, error) {
				username := values.Get(authclient.SESSION_USER_KEY)
				return mgr.userManager.UserByName(ctx, username)
			})), nil
		}),
	}
}

func NewLoginManager(env *moo.Environment, userManager UserManager, online Sessions, locator WelcomeLocator, authOpts []services.AuthOption) (*LoginManager, error) {
	logger := env.Logger.Named("sessions")

	counter := services.CreateFailCounter()
	opts := []services.AuthOption{
		services.Whitelist(),
		services.ErrorCountCheck(userManager, counter, env.Config.IntWithDefault(api.CfgUserMaxLoginFailCount, 3)),
		services.LockCheck(),
		services.OnlineCheck(online, env.Config.StringWithDefault(api.CfgUserLoginConflict, "")),
		//services.TptInternalUserCheck(env),
		services.DefaultUserCheck(),
		services.LdapUserCheck(env, logger),
	}
	if len(authOpts) > 0 {
		opts = append(opts, authOpts...)
	}
	if !env.Config.BoolWithDefault(api.CfgUserCaptchaDisabled, false) {
		opts = append(opts, services.CaptchaCheck(nil, counter))
	}

	authSrv, err := services.NewAuthService(userManager, opts...)
	if err != nil {
		return nil, err
	}

	jwtToken, err := readJWTAuth(env)
	if err != nil {
		return nil, err
	}

	cfg := readConfig(env)
	ui, err := CreateRenderer(cfg, locator)
	if err != nil {
		return nil, err
	}

	mgr := &LoginManager{
		env:         env,
		logger:      logger,
		cfg:         cfg,
		sessionHash: ui.sessonHashFunc,
		Renderer:    ui,
		userManager: userManager,
		online:      online,
		authSrv:     authSrv,
		expiresIn:   1 * time.Hour,
		jwtConfig:   jwtToken,
	}
	return mgr, nil
}

func ReturnError(w http.ResponseWriter, r *http.Request, errText string, statusCode int) {
	var err = map[string]interface{}{
		"code":    statusCode,
		"error":   errText,
		"message": errText,
	}

	w.Header().Set("Content-Type", "application/json")
	if statusCode >= 100000 {
		statusCode = statusCode / 1000
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(err)
}

func ReturnJSON(w http.ResponseWriter, r *http.Request, value interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	if statusCode >= 100000 {
		statusCode = statusCode / 1000
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(value)
}
