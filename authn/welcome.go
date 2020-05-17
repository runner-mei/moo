package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
)

const WelcomeFieldName = "welcome_url"

type welcomeLocator struct {
	logger                  log.Logger
	welcomeConfigs          []WelcomConfig
	conn                    *sql.DB
	welcomeByUserID         string
	welcomeByUsername       string
	todolistDisabled        bool
	todolistURL             string
	todolistCountByUserID   string
	todolistCountByUsername string
}

func (srv *welcomeLocator) withTodolistURL(userID interface{}, username, defaultURL string) (string, error) {
	if srv.todolistDisabled {
		return defaultURL, nil
	}

	var todolistCount int64
	var err error

	if userID != 0 {
		err = srv.conn.QueryRow(srv.todolistCountByUserID, userID).
			Scan(&todolistCount)
	} else {
		err = srv.conn.QueryRow(srv.todolistCountByUsername, username).
			Scan(&todolistCount)
	}
	if err != nil {
		srv.logger.Warn("read todolist fail, ", log.Any("userid", userID), log.String("username", username), log.Error(err))
	} else if todolistCount > 0 {
		if strings.ContainsRune(srv.todolistURL, '?') {
			return srv.todolistURL + "&returnTo=" + url.QueryEscape(defaultURL), nil
		}
		return srv.todolistURL + "?returnTo=" + url.QueryEscape(defaultURL), nil
	}
	return defaultURL, nil
}

func (srv *welcomeLocator) Locate(userID interface{}, username, defaultURL string) (string, error) {
	var value sql.NullString
	var err error
	if userID != 0 {
		err = srv.conn.QueryRow(srv.welcomeByUserID, userID).Scan(&value)
	} else {
		err = srv.conn.QueryRow(srv.welcomeByUsername, username).Scan(&value)
	}
	if err != nil {
		srv.logger.Warn("read welcome url fail, ", log.Any("userid", userID), log.String("username", username), log.Error(err))
		return srv.withTodolistURL(userID, username, defaultURL)
	}

	if !value.Valid || value.String == "" {
		srv.logger.Warn("welcome url is empty", log.Any("userid", userID), log.String("username", username))
		return srv.withTodolistURL(userID, username, defaultURL)
	}
	ss := strings.SplitN(value.String, ",", 2)
	if len(ss) != 2 {
		if s := strings.ToLower(value.String); strings.HasPrefix(s, "http://") ||
			strings.HasPrefix(s, "https://") {
			return srv.withTodolistURL(userID, username, value.String)
		}

		srv.logger.Warn("welcome_url is invalid value - "+value.String, log.Any("userid", userID), log.String("username", username), log.Error(err))
		return srv.withTodolistURL(userID, username, defaultURL)
	}

	if ss[0] == "url" {
		return srv.withTodolistURL(userID, username, ss[1])
	}

	for idx := range srv.welcomeConfigs {
		if srv.welcomeConfigs[idx].Name == ss[0] {
			var redirectURL string
			if !strings.ContainsRune(srv.welcomeConfigs[idx].RedirectURL, '?') {
				redirectURL = srv.welcomeConfigs[idx].RedirectURL + "?value=" + url.QueryEscape(ss[1])
			} else if strings.HasSuffix(srv.welcomeConfigs[idx].RedirectURL, "?") {
				redirectURL = srv.welcomeConfigs[idx].RedirectURL + "value=" + url.QueryEscape(ss[1])
			} else {
				redirectURL = srv.welcomeConfigs[idx].RedirectURL + "&value=" + url.QueryEscape(ss[1])
			}
			return srv.withTodolistURL(userID, username, redirectURL)
		}
	}

	srv.logger.Warn("application `"+ss[0]+"` isnot found - "+value.String, log.Any("userid", userID), log.String("username", username))
	return srv.withTodolistURL(userID, username, defaultURL)
}

type WelcomConfig struct {
	Name        string `json:"name"`
	ListURL     string `json:"list_url"`
	RedirectURL string `json:"redirect_url"`
}

func ReadWelcomeConfigs(env *moo.Environment) ([]WelcomConfig, error) {
	filename := env.Fs.FromConfig("home.json")
	args := map[string]interface{}{
		"urlRoot": env.DaemonUrlPath,
	}

	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.New("ReadHTTPConfigFromFile: " + err.Error())
	}

	t, err := template.New("default").Funcs(template.FuncMap{
		"join": urlutil.Join,
	}).Parse(string(bs))
	if err != nil {
		return nil, errors.New("parse url template in '" + filename + "' fail: " + err.Error())
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, args); err != nil {
		return nil, errors.New("generate url template in '" + filename + "' fail: " + err.Error())
	}
	if buf.Len() == 0 {
		return nil, errors.New("template result in '" + filename + "' is empty.")
	}

	var config struct {
		Applications []WelcomConfig `json:"applications,omitempty"`
	}

	bs = buf.Bytes()
	err = json.NewDecoder(&buf).Decode(&config)
	if err != nil {
		return nil, errors.New("read '" + filename + "' fail: " + err.Error() + "\r\n" + string(bs))
	}
	return config.Applications, nil
}

func NewWelcomeLocator(env *moo.Environment, logger log.Logger, conn *sql.DB) (WelcomeLocator, error) {
	logger = logger.Named("welcome")

	apps, err := ReadWelcomeConfigs(env)
	if err != nil {
		logger.Warn("NewWelcomeLocator:", log.Error(err))
	}
	locator := &welcomeLocator{
		logger:         logger,
		welcomeConfigs: apps,
		conn:           conn,
		welcomeByUserID: env.Config.StringWithDefault("users.welcome.by_userid",
			"select attributes->>'"+WelcomeFieldName+"' from moo_users where id = $1"),
		welcomeByUsername: env.Config.StringWithDefault("users.welcome.by_username",
			"select attributes->>'"+WelcomeFieldName+"' from moo_users where name = $1"),
		todolistDisabled: env.Config.BoolWithDefault("users.todolist.disabled", false),
		todolistCountByUserID: env.Config.StringWithDefault("users.welcome.todolist_by_userid",
			"select count(*) from tpt_todolists where user_id = $1)"),
		todolistCountByUsername: env.Config.StringWithDefault("users.welcome.todolist_by_username",
			"select count(*) from tpt_todolists as todolists where "+
				"exists(SELECT * FROM moo_users WHERE moo_users.id = todolists.user_id AND moo_users.name = $1)"),
		todolistURL: env.Config.StringWithDefault("users.welcome.todolist_url", ""),
	}
	if !locator.todolistDisabled {
		if locator.todolistURL == "" {
			return nil, errors.New("users.welcome.todolist_url is missing")
		}
	}
	return locator, nil
}
