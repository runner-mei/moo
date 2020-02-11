package auth

import (
	"context"
	"fmt"
	"hash"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/mojocn/base64Captcha"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/api/authclient"
	"github.com/runner-mei/moo/auth/services"
)

var (
	isDebug = os.Getenv("IsSSODebug") == "true"
)

type WelcomeLocator interface {
	Locate(userID interface{}, username, defaultURL string) (string, error)
}

type WelcomeFunc func(userID interface{}, username, defaultURL string) (string, error)

func (f WelcomeFunc) Locate(userID interface{}, username, defaultURL string) (string, error) {
	return f(userID, username, defaultURL)
}

// Renderer SSO 服务器
type Renderer struct {
	config         Config
	sessonHashFunc func() hash.Hash
	data           map[string]interface{}
	templates      templates
	redirect       func(c context.Context, w http.ResponseWriter, r *http.Request, url string) error
	AssetsHandler  http.Handler
	captchaStore   base64Captcha.Store
	welcomeLocator WelcomeLocator

	homePaths []string
}

func (srv *Renderer) readContextPath(r *http.Request) string {
	pa := strings.TrimPrefix(r.URL.Path, srv.config.URLPrefix)
	if !strings.HasPrefix(srv.config.URLPrefix, "/") {
		pa = strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/"), srv.config.URLPrefix)
	}

	pa = strings.TrimPrefix(pa, "/")
	idx := strings.Index(pa, "/")
	if idx <= 0 {
		return srv.config.ContextPath
	}
	return pa[:idx]
}

func (srv *Renderer) ReturnError(authCtx *services.AuthContext, w http.ResponseWriter, r *http.Request, rawerr error) error {
	err := rawerr
	if e := errors.Unwrap(err); e != nil {
		err = e
	}
	message := "用户名或密码不正确!"

	if err == services.ErrCaptchaKey || rawerr == services.ErrCaptchaKey {
		message = "请输入验证码"
	} else if err == services.ErrCaptchaMissing || rawerr == services.ErrCaptchaMissing {
		message = "验证码错误"
	} else if err == services.ErrUserIPBlocked || rawerr == services.ErrUserIPBlocked {
		message = "用户不能在该地址访问"
	} else if err == services.ErrUserLocked || rawerr == services.ErrUserLocked ||
		err == services.ErrUserErrorCountExceedLimit || rawerr == services.ErrUserErrorCountExceedLimit {
		message = "错误次数大多，帐号被锁定！"
	} else if err == services.ErrPermissionDenied || rawerr == services.ErrPermissionDenied {
		message = "用户没有访问权限"
	} else if err == services.ErrMutiUsers || rawerr == services.ErrMutiUsers {
		message = "同名的用户有多个"
	} else if services.IsErrExternalServer(err) {
		message = err.Error()
	} else if _, ok := IsOnlinedError(err); ok {
		message = err.Error()
		err = services.ErrUserAlreadyOnline
	}
	if message == "" {
		message = "用户名或密码不正确!"
	}

	data := map[string]interface{}{"global": srv.data,
		"service":          authCtx.Request.Service,
		"login_fail_count": authCtx.ErrorCount,
		"username":         authCtx.Request.Username,
		"errorMessage":     message,
		"context_path":     srv.readContextPath(r),
	}

	if authCtx.ErrorCount > 0 {
		captchaKey, captchaCode, err := services.GenerateCaptcha(srv.captchaStore, srv.config.Captcha)
		if err != nil {
			authCtx.Logger.Warn("登录失败", log.String("username", authCtx.Request.Username),
				log.String("address", authCtx.Request.Address),
				log.Error(err))
		} else {
			data["captcha_data"] = captchaCode
			data["captcha_key"] = captchaKey
		}
	}

	if err == services.ErrUserAlreadyOnline {
		data["showForce"] = srv.config.ShowForce
	}

	if e, ok := err.(*services.ErrExternalServer); ok {
		authCtx.Logger.Warn("登录失败", log.String("username", authCtx.Request.Username),
			log.String("address", authCtx.Request.Address),
			log.String("err_title", e.Msg),
			log.NamedError("details", e.Err))
	} else if e, ok := rawerr.(*services.ErrExternalServer); ok {
		authCtx.Logger.Warn("登录失败", log.String("username", authCtx.Request.Username),
			log.String("address", authCtx.Request.Address),
			log.String("err_title", e.Msg),
			log.NamedError("details", e.Err))
	} else {
		authCtx.Logger.Warn("登录失败", log.String("username", authCtx.Request.Username),
			log.String("address", authCtx.Request.Address), log.Error(rawerr))
	}
	return srv.templates.Render(w, r, "login.html", data)
}

func (srv *Renderer) Relogin(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	for _, cookie := range r.Cookies() {
		if cookie.Name == srv.config.SessionKey {
			a := *cookie
			a.Expires = time.Now().Add(-1 * time.Second)
			a.MaxAge = -1
			http.SetCookie(w, &a)
		}
	}

	queryParams := r.URL.Query()
	return srv.templates.Render(w, r, "login.html", map[string]interface{}{
		"global":           srv.data,
		"context_path":     srv.readContextPath(r),
		"service":          queryParams.Get("service"),
		"login_fail_count": 0,
		"username":         "",
		"errorMessage":     "",
	})
}

func (srv *Renderer) isRootPath(pa string) bool {
	u, _ := url.Parse(pa)
	if u != nil {
		pa = u.Path
	}

	if strings.HasSuffix(pa, "//") {
		pa = strings.TrimSuffix(pa, "/")
	}
	for _, s := range srv.homePaths {
		if pa == s {
			return true
		}
	}
	return false
}

func (srv *Renderer) LoginOK(authCtx *services.AuthContext, w http.ResponseWriter, r *http.Request) error {
	for _, cookie := range r.Cookies() {
		if cookie.Name == srv.config.SessionKey &&
			cookie.Path != srv.config.SessionPath {
			a := *cookie
			a.Expires = time.Now().Add(-1 * time.Second)
			a.MaxAge = -1
			http.SetCookie(w, &a)
		}
	}

	var values = url.Values{}
	for k, v := range authCtx.Response.Data {
		found := false
		for _, s := range []string{
			"uuid",
			"username",
			"password",
			"name",
			"expired_at",
			"issued_at",
			"admin"} {
			if s == k {
				found = true
				break
			}
		}
		if found {
			continue
		}
		values.Set(k, fmt.Sprint(v))
	}

	values.Set("issued_at", time.Now().Format(time.RFC3339))
	values.Set(authclient.SESSION_ID_KEY, authCtx.Response.SessionID)
	values.Set(authclient.SESSION_EXPIRE_KEY, "session")
	values.Set(authclient.SESSION_VALID_KEY, "true")
	values.Set(authclient.SESSION_USER_KEY, authCtx.Request.Username)

	http.SetCookie(w, &http.Cookie{
		Name:     srv.config.SessionKey,
		Value:    authclient.Encode(values, srv.sessonHashFunc, srv.config.SessionSecretKey),
		Domain:   srv.config.SessionDomain,
		Path:     srv.config.SessionPath,
		Secure:   srv.config.SessionSecure,
		HttpOnly: srv.config.SessionHttpOnly,
	})

	// return c.JSON(http.StatusOK, map[string]interface{}{
	// 	"userid":     authCtx.Request.UserID,
	// 	"username":   authCtx.Request.Username,
	// 	"session_id": authCtx.Respnse.SessionID,
	// 	"is_new":     authCtx.Respnse.IsNew,
	// 	"roles":      authCtx.Respnse.Roles(),
	// })

	var urlStr string
	if srv.welcomeLocator != nil {
		var err error
		urlStr, err = srv.welcomeLocator.Locate(authCtx.Request.UserID, authCtx.Request.Username, "")
		if err != nil {
			authCtx.Logger.Warn("获取 welcome 页地址失败", 
				log.Any("userid", authCtx.Request.UserID),
				log.String("username", authCtx.Request.Username),
				log.String("address", authCtx.Request.Address))
		}
	}
	if srv.config.JumpToWelcomeIfNewUser && authCtx.Response.IsNewUser {
		urlStr = srv.config.NewUserURL
		u, _ := url.Parse(srv.config.NewUserURL)
		if u != nil {
			q := u.Query()
			q.Set("is_new", "true")
			q.Set("username", authCtx.Request.Username)
			q.Set("sessionID", authCtx.Response.SessionID)
			q.Set("usersource", authCtx.Response.UserSource)
			q.Set("service", authCtx.Request.Service)
			u.RawQuery = q.Encode()
		}
		urlStr = u.String()
	} else if !srv.isRootPath(authCtx.Request.Service) {
		urlStr = authCtx.Request.Service
	} else if urlStr == "" {
		urlStr = srv.config.DefaultWelcomeURL
	}

	authCtx.Logger.Info("登录成功", log.String("username", authCtx.Request.Username),
		log.String("address", authCtx.Request.Address), log.String("redirect", urlStr))

	return srv.redirect(authCtx.Ctx, w, r, urlStr)
}

func (srv *Renderer) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	queryParams := r.URL.Query()
	returnURL := queryParams.Get("service")
	if returnURL == "" {
		if returnURL = queryParams.Get("redirect"); returnURL == "" {
			if context := srv.readContextPath(r); context == "" || context == "sso" {
				returnURL = srv.config.LoginURL
			} else {
				returnURL = urlutil.Join(srv.config.URLPrefix, context)
			}
		}
	}

	u, err := url.Parse(returnURL)
	if err == nil {
		queryParams := u.Query()
		queryParams.Del("ticket")
		queryParams.Del("token")
		u.RawQuery = queryParams.Encode()
		returnURL = u.String()
	}
	return srv.LogoutWithRedirect(ctx, w, r, returnURL)
}

func (srv *Renderer) LogoutWithRedirect(ctx context.Context, w http.ResponseWriter, r *http.Request, redirectURL string) error {
	var values = url.Values{}
	values.Set(authclient.SESSION_EXPIRE_KEY, strconv.FormatInt(time.Now().Unix()-30*24*40, 10))
	values.Set(authclient.SESSION_VALID_KEY, "false")

	http.SetCookie(w, &http.Cookie{
		Name:     srv.config.SessionKey,
		Value:    authclient.Encode(values, srv.sessonHashFunc, srv.config.SessionSecretKey),
		Domain:   srv.config.SessionDomain,
		Path:     srv.config.SessionPath,
		Secure:   srv.config.SessionSecure,
		HttpOnly: srv.config.SessionHttpOnly,
		Expires:  time.Now().Add(-1 * time.Second),
		MaxAge:   -1,
	})
	for _, cookie := range srv.config.CookiesForLogout {
		a := &http.Cookie{}
		*a = *cookie
		a.Expires = time.Now().Add(-1 * time.Second)
		a.MaxAge = -1
		http.SetCookie(w, a)
	}

	log.LoggerOrEmptyFromContext(ctx).Info("登出成功", log.String("redirect", redirectURL))
	return srv.redirect(ctx, w, r, redirectURL)
}

type templates struct {
	srv           *Renderer
	templatesLock sync.Mutex
	templates     map[string]*template.Template
	templateRoots []string
	templateBox   *rice.Box
}

func (r *templates) Render(w http.ResponseWriter, req *http.Request, name string, data interface{}) error {
	var t *template.Template
	var err error
	if name == "login.html" {

		queryParams := req.URL.Query()
		theme := queryParams.Get("theme")
		if theme == "" {
			theme = r.srv.config.Theme
		}

		if theme != "" {
			t, err = r.loadTemplate("login_" + theme + ".html")
			if err != nil {
				r.srv.config.Logger.Warn("load login_"+theme+".html", log.Error(err))
			}
		}
	}
	if t == nil {
		t, err = r.loadTemplate(name)
		if err != nil {
			return err
		}
	}
	return t.Execute(w, data)
}

var funcs = template.FuncMap{
	"query": url.QueryEscape,
	"htmlattr": func(s string) template.HTMLAttr {
		return template.HTMLAttr(s)
	},
	"html": func(s string) template.HTML {
		return template.HTML(s)
	},
	"js": func(s string) template.JS {
		return template.JS(s)
	},
	"set_src": func(s string) template.Srcset {
		return template.Srcset(s)
	},
	"jsstr": func(s string) template.JSStr {
		return template.JSStr(s)
	},
	"urljoin": urlutil.Join,
}

func (r *templates) loadTemplate(name string) (*template.Template, error) {
	r.templatesLock.Lock()
	t := r.templates[name]
	r.templatesLock.Unlock()
	if t != nil {
		return t, nil
	}

	for _, pa := range r.templateRoots {
		filename := filepath.Join(pa, name)
		bs, err := ioutil.ReadFile(filename)
		if err == nil {
			t, err = template.New(name).Funcs(funcs).Parse(string(bs))
			if err != nil {
				r.srv.config.Logger.Error("failed to load template("+name+") from "+filename, log.Error(err))
				return nil, err
			}
			r.srv.config.Logger.Info("load template(" + name + ") from " + filename)
			break
		}

		if !os.IsNotExist(err) {
			r.srv.config.Logger.Error("failed to load template("+name+") from "+filename, log.Error(err))
			return nil, err
		}
	}

	if t == nil {
		bs, err := r.templateBox.Bytes(name)
		if err != nil {
			r.srv.config.Logger.Error("failed to load template("+name+") from rice box", log.Error(err))
			return nil, err
		}
		if len(bs) == 0 {
			r.srv.config.Logger.Error("failed to load template(" + name + ") from rice box, file is empty.")
			return nil, err
		}

		t, err = template.New(name).Funcs(funcs).Parse(string(bs))
		if err != nil {
			r.srv.config.Logger.Error("failed to load template("+name+") from rice box, ", log.Error(err))
			return nil, err
		}
	}

	if !isDebug {
		r.templatesLock.Lock()
		r.templates[name] = t
		r.templatesLock.Unlock()
	}
	return t, nil
}

// CreateRenderer 创建一个 sso 服务
func CreateRenderer(config *Config, locator WelcomeLocator) (*Renderer, error) {
	if strings.HasSuffix(config.URLPrefix, "/") {
		config.URLPrefix = strings.TrimSuffix(config.URLPrefix, "/")
	}
	if config.SessionPath == "" {
		config.SessionPath = "/"
	} else if !strings.HasPrefix(config.SessionPath, "/") {
		config.SessionPath = "/" + config.SessionPath
	}

	if config.ContextPath == "" {
		config.ContextPath = "sessions"
	}

	templateBox, err := rice.FindBox("static")
	if err != nil {
		return nil, errors.New("load static directory fail, " + err.Error())
	}

	variables := map[string]interface{}{}
	variables["url_prefix"] = config.URLPrefix
	variables["play_path"] = config.PlayPath
	variables["application_context"] = config.URLPrefix

	variables["client_title_text"] = config.ClientTitleText
	variables["header_title_text"] = config.HeaderTitleText
	variables["footer_title_text"] = config.FooterTitleText
	variables["logo_png"] = config.LogoPath
	variables["new_user_url"] = config.NewUserURL

	srv := &Renderer{
		config: *config,
		redirect: func(c context.Context, w http.ResponseWriter, r *http.Request, toURL string) error {
			http.Redirect(w, r, toURL, http.StatusTemporaryRedirect)
			return nil
		},
		data: variables,
		homePaths: []string{
			"",
			"/",
			config.URLPrefix,
		},
		welcomeLocator: locator,
	}

	if srv.captchaStore == nil {
		srv.captchaStore = base64Captcha.DefaultMemStore
	}

	if strings.HasPrefix(config.URLPrefix, "/") {
		if strings.HasSuffix(config.URLPrefix, "/") {
			srv.homePaths = append(srv.homePaths, strings.TrimSuffix(config.URLPrefix, "/"))
			srv.homePaths = append(srv.homePaths, strings.TrimPrefix(config.URLPrefix, "/"))
			srv.homePaths = append(srv.homePaths, strings.TrimSuffix(strings.TrimPrefix(config.URLPrefix, "/"), "/"))
		} else {
			srv.homePaths = append(srv.homePaths, strings.TrimSuffix(config.URLPrefix, "/")+"/")
			srv.homePaths = append(srv.homePaths, strings.TrimPrefix(config.URLPrefix, "/"))
			srv.homePaths = append(srv.homePaths, strings.TrimPrefix(config.URLPrefix, "/")+"/")
		}
	} else {
		if strings.HasSuffix(config.URLPrefix, "/") {
			srv.homePaths = append(srv.homePaths, "/"+config.URLPrefix)
			srv.homePaths = append(srv.homePaths, "/"+strings.TrimSuffix(config.URLPrefix, "/"))
			srv.homePaths = append(srv.homePaths, strings.TrimSuffix(config.URLPrefix, "/"))
		} else {
			srv.homePaths = append(srv.homePaths, "/"+config.URLPrefix)
			srv.homePaths = append(srv.homePaths, "/"+config.URLPrefix+"/")
			srv.homePaths = append(srv.homePaths, config.URLPrefix+"/")
		}
	}

	if config.RedirectMode == "html" {
		srv.redirect = func(c context.Context, w http.ResponseWriter, r *http.Request, toURL string) error {
			data := map[string]interface{}{
				"global":    srv.data,
				"returnURL": toURL,
			}
			return srv.templates.Render(w, r, "success.html", data)
		}
	}

	if len(config.TampletePaths) == 0 {
		config.TampletePaths = append(config.TampletePaths, filepath.Join("lib/web/sso"))
	}
	srv.templates = templates{
		srv:           srv,
		templates:     map[string]*template.Template{},
		templateRoots: config.TampletePaths,
		templateBox:   templateBox,
	}

	fs := http.FileServer(templateBox.HTTPBox())
	srv.AssetsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if strings.HasPrefix(upath, "/") {
			upath = strings.TrimPrefix(upath, "/")
		}
		for _, root := range config.TampletePaths {
			filename := filepath.Join(root, "static", upath)
			if _, err := os.Stat(filename); err == nil {
				http.ServeFile(w, r, filename)
				return
			}
		}

		fs.ServeHTTP(w, r)
	})

	srv.sessonHashFunc = createSessonHashFunc(srv.config.SessionHashFunc)
	return srv, nil
}