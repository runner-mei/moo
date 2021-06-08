package authn

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"hash"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/mojocn/base64Captcha"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
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
	DisableUserList []string
	ShowForce       bool
	DisableCaptcha  bool
	AutoLoad        string

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

	sessionHashFunc func() hash.Hash
}

func (cfg *Config) GetSessionHashFunc() func() hash.Hash {
	if cfg.sessionHashFunc == nil {
		cfg.sessionHashFunc = createSessonHashFunc(cfg.SessionHashFunc)
	}
	return cfg.sessionHashFunc
}

func readWelcomeURL(env *moo.Environment) string {
	urlStr := env.Config.StringWithDefault(api.CfgHomeURL, "")
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
	sessionHashFunc := sha1.New
	switch method {
	case "", "sha1":
	case "md5", "MD5":
		sessionHashFunc = md5.New
	}
	return sessionHashFunc
}

func ReadConfig(env *moo.Environment) *Config {
	config := &Config{
		Logger: env.Logger.Named("sso.ui"),
		Captcha: services.CaptchaConfig{
			DriverDigit: &base64Captcha.DriverDigit{
				Height:   80,
				Width:    240,
				Length:   5,
				MaxSkew:  0.7,
				DotCount: 80,
			},
		},
		Theme:           "hw",
		URLPrefix:       env.DaemonUrlPath,
		ContextPath:     env.Config.StringWithDefault(api.CfgSSOContextPath, ""),
		PlayPath:        "/web",
		HeaderTitleText: env.LoginHeaderTitleText,
		FooterTitleText: env.LoginFooterTitleText,

		TampletePaths:    []string{env.Fs.FromLib("web/sso"), env.Fs.FromData("resources")},
		RedirectMode:     env.Config.StringWithDefault(api.CfgUserRedirectMode, "html"),
		DisableUserList:  split.Split(env.Config.StringWithDefault(api.CfgSysDisableUsers, ""), ",", true, true),
		DisableCaptcha:   env.Config.BoolWithDefault(api.CfgUserCaptchaDisabled, false),
		ShowForce:        strings.ToLower(strings.TrimSpace(env.Config.StringWithDefault(api.CfgUserLoginConflict, ""))) != "disableforce",
		SessionKey:       authclient.DefaultSessionKey,
		SessionPath:      readSessonPath(env),
		SessionDomain:    "",
		SessionSecure:    false,
		SessionHttpOnly:  false,
		SessionHashFunc:  "sha1",
		SessionSecretKey: nil,

		JumpToWelcomeIfNewUser: env.Config.BoolWithDefault(api.CfgUserJumpToWelcomeIfNewUser, true),
	}

	if secretStr := env.Config.StringWithDefault(api.CfgUserAppSecret, ""); secretStr != "" {
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

	filename := env.Fs.FromConfig("autoload.html")
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		filename = env.Fs.FromDataConfig("autoload.html")
		bs, err = ioutil.ReadFile(filename)
		if err != nil && !os.IsNotExist(err) {
			panic(err)
		}
	}
	if bs := bytes.TrimSpace(bs); len(bs) > 0 {
		config.AutoLoad = string(bs)
	}
	return config
}
