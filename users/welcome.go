package users

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"reflect"
	"strings"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/authn"
	moodb "github.com/runner-mei/moo/db"
	"github.com/runner-mei/moo/users/usermodels"
	"github.com/runner-mei/moo/users/welcome"
)

func init() {
	moo.On(func() moo.Option {
		return moo.Provide(func(env *moo.Environment, db moodb.InModelFactory, logger log.Logger) (authn.WelcomeLocator, error) {
			return NewWelcomeLocator(env, logger, db.Factory)
		})
	})
}

type welcomeLocator struct {
	logger                  log.Logger
	welcomeConfigs          []welcome.Config
	conn                    gobatis.DBRunner
	welcomeByUserID         string
	welcomeByUsername       string
	todolistDisabled        bool
	todolistURL             string
	todolistCountByUserID   string
	todolistCountByUsername string
}

func (srv *welcomeLocator) withTodolistURL(ctx context.Context, userID interface{}, username, defaultURL string) (string, error) {
	if srv.todolistDisabled {
		return defaultURL, nil
	}

	var todolistCount int64
	var err error

	if userID != 0 {
		err = srv.conn.QueryRowContext(ctx, srv.todolistCountByUserID, userID).
			Scan(&todolistCount)
	} else {
		err = srv.conn.QueryRowContext(ctx, srv.todolistCountByUsername, username).
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

func (srv *welcomeLocator) Locate(ctx context.Context, userID interface{}, username, defaultURL string) (string, error) {
	var value sql.NullString
	var err error
	if userID != 0 {
		err = srv.conn.QueryRowContext(ctx, srv.welcomeByUserID, userID).Scan(&value)
	} else {
		err = srv.conn.QueryRowContext(ctx, srv.welcomeByUsername, username).Scan(&value)
	}
	if err != nil {
		srv.logger.Warn("read welcome url fail, ", log.Any("userid", userID), log.String("username", username), log.Error(err))
		return srv.withTodolistURL(ctx, userID, username, defaultURL)
	}

	if !value.Valid || value.String == "" {
		srv.logger.Warn("welcome url is empty", log.Any("userid", userID), log.String("username", username))
		return srv.withTodolistURL(ctx, userID, username, defaultURL)
	}
	ss := strings.SplitN(value.String, ",", 2)
	if len(ss) != 2 {
		if s := strings.ToLower(value.String); strings.HasPrefix(s, "http://") ||
			strings.HasPrefix(s, "https://") {
			return srv.withTodolistURL(ctx, userID, username, value.String)
		}

		srv.logger.Warn("welcome_url is invalid value - "+value.String, log.Any("userid", userID), log.String("username", username), log.Error(err))
		return srv.withTodolistURL(ctx, userID, username, defaultURL)
	}

	if ss[0] == "url" {
		return srv.withTodolistURL(ctx, userID, username, ss[1])
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
			return srv.withTodolistURL(ctx, userID, username, redirectURL)
		}
	}

	srv.logger.Warn("application `"+ss[0]+"` isnot found - "+value.String, log.Any("userid", userID), log.String("username", username))
	return srv.withTodolistURL(ctx, userID, username, defaultURL)
}

func NewWelcomeLocator(env *moo.Environment, logger log.Logger, factory *gobatis.SessionFactory) (authn.WelcomeLocator, error) {
	logger = logger.Named("welcome")

	tablename, err := gobatis.ReadTableName(factory.Mapper(), reflect.TypeOf(&usermodels.User{}))
	if err != nil {
		return nil, errors.New("读用户的表名失败")
	}
	todoListTable := env.Config.StringWithDefault(api.CfgUserTodoListTableName, "moo_todolists")

	apps, err := welcome.ReadConfigs(env)
	if err != nil {
		logger.Warn("NewWelcomeLocator:", log.Error(err))
	}
	locator := &welcomeLocator{
		logger:         logger,
		welcomeConfigs: apps,
		conn:           factory.DB(),
		welcomeByUserID: env.Config.StringWithDefault(api.CfgUserWelcomeByUserID,
			"select attributes->>'"+welcome.FieldName+"' from "+tablename+" where id = $1"),
		welcomeByUsername: env.Config.StringWithDefault(api.CfgUserWelcomeByUsername,
			"select attributes->>'"+welcome.FieldName+"' from "+tablename+" where name = $1"),
		todolistDisabled: env.Config.BoolWithDefault(api.CfgUserTodoListDisabled, true),
		todolistCountByUserID: env.Config.StringWithDefault(api.CfgUserTodoListByUserID,
			"select count(*) from "+todoListTable+" where user_id = $1)"),
		todolistCountByUsername: env.Config.StringWithDefault(api.CfgUserTodoListByUsername,
			"select count(*) from "+todoListTable+" as todolists where "+
				"exists(SELECT * FROM "+tablename+" WHERE "+tablename+".id = todolists.user_id AND "+tablename+".name = $1)"),
		todolistURL: env.Config.StringWithDefault(api.CfgUserTodoListURL, ""),
	}
	if !locator.todolistDisabled {
		if locator.todolistURL == "" {
			return nil, errors.New("users.welcome.todolist_url is missing")
		}
	}
	return locator, nil
}
