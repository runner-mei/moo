package auth

import (
	"crypto/md5"
	"crypto/sha1"
	"hash"
	"net/http"
	"strings"

	"github.com/mojocn/base64Captcha"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api/authclient"
	"github.com/runner-mei/moo/authn/services"
)

// Config 服务的配置项
type Config struct {
	Logger log.Logger
	//数字验证码配置
	Captcha         services.CaptchaConfig
	Theme           string
	URLPrefix       string
	ContextPath     string
	PlayPath        string
	ClientTitleText string
	HeaderTitleText string
	FooterTitleText string
	LogoPath        string
	TampletePaths   []string
	ShowForce       bool

	SessionKey       string
	SessionPath      string
	SessionDomain    string
	SessionSecure    bool
	SessionHttpOnly  bool
	SessionHashFunc  string
	SessionSecretKey []byte

	JumpToWelcomeIfNewUser bool
	LoginURL               string
	NewUserURL             string
	DefaultWelcomeURL      string
	RedirectMode           string
	CookiesForLogout       []*http.Cookie
}

func readWelcomeURL(env *moo.Environment) string {
	urlStr := env.Config.StringWithDefault("home_url", "")
	if urlStr == "" {
		return urlStr
	}

	for _, matchStr := range []string{
		"{{DaemonUrlPath}}",
		"/{{DaemonUrlPath}}",
		"{{.DaemonUrlPath}}",
		"/{{.DaemonUrlPath}}",
		"{DaemonUrlPath}",
		"/{DaemonUrlPath}",
	} {
		if strings.HasPrefix(urlStr, matchStr) {
			return urlutil.Join(env.DaemonUrlPath, strings.TrimPrefix(urlStr, matchStr))
		}
	}
	return urlStr
}

func readSessonPath(env *moo.Environment) string {
	sessionPath := env.DaemonUrlPath
	if !strings.HasPrefix(sessionPath, "/") {
		sessionPath = "/" + sessionPath
	}
	// Play 中我们有删除后面的 '/', 所以我们这里也删除一下
	if strings.HasSuffix(sessionPath, "/") {
		sessionPath = strings.TrimSuffix(sessionPath, "/")
	}
	return sessionPath
}

func createSessonHashFunc(method string) func() hash.Hash {
	sessonHashFunc := sha1.New
	switch method {
	case "", "sha1":
	case "md5", "MD5":
		sessonHashFunc = md5.New
	}
	return sessonHashFunc
}

func readConfig(env *moo.Environment) *Config {
	config := &Config{
		Logger: env.Logger.Named("sso.ui"),
		Captcha: services.CaptchaConfig{
			DriverDigit: &base64Captcha.DriverDigit{
				Height:     80,
				Width:      240,
				Length:     5,
				MaxSkew:    0.7,
				DotCount:   80,
			},
		},
		Theme:           "hw",
		URLPrefix:       env.DaemonUrlPath,
		ContextPath:     env.Config.StringWithDefault("sso.context_path", ""),
		PlayPath:        "/web",
		HeaderTitleText: env.LoginHeaderTitleText,
		FooterTitleText: env.LoginFooterTitleText,

		TampletePaths:    []string{env.Fs.FromLib("web/sso"), env.Fs.FromData("resources")},
		RedirectMode:     env.Config.StringWithDefault("users.redirect_mode", "html"),
		ShowForce:        strings.ToLower(strings.TrimSpace(env.Config.StringWithDefault("users.login_conflict", ""))) != "disableforce",
		SessionKey:       authclient.DefaultSessionKey,
		SessionPath:      readSessonPath(env),
		SessionDomain:    "",
		SessionSecure:    false,
		SessionHttpOnly:  false,
		SessionHashFunc:  "sha1",
		SessionSecretKey: nil,

		JumpToWelcomeIfNewUser: env.Config.BoolWithDefault("users.jump_to_welcome_if_new_user", true),
	}

	if secretStr := env.Config.StringWithDefault("app.secret", ""); secretStr != "" {
		config.SessionSecretKey = []byte(secretStr)
	}

	if util.FileExists(env.Fs.FromData("resources/images/logo.png")) {
		config.LogoPath = urlutil.JoinURLPath(env.DaemonUrlPath, "internal/custom_resources/images/logo.png")
	}
	if util.FileExists(env.Fs.FromData("resources/images/login_logo.png")) {
		config.LogoPath = urlutil.JoinURLPath(env.DaemonUrlPath, "internal/custom_resources/images/login_logo.png")
	}

	config.LoginURL = urlutil.JoinURLPath(env.DaemonUrlPath, "sso", "login")
	config.NewUserURL = urlutil.JoinURLPath(env.DaemonUrlPath, "um/welcome/current?is_new=true")
	config.DefaultWelcomeURL = readWelcomeURL(env)
	if config.DefaultWelcomeURL == "" {
		config.DefaultWelcomeURL = urlutil.JoinURLPath(env.DaemonUrlPath, "home")
	}

	return config
}
