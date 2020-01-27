package moo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/cfg"
	"go.uber.org/fx"
)

type Environment struct {
	Logger               log.Logger
	HeaderTitleText      string
	FooterTitleText      string
	LoginHeaderTitleText string
	LoginFooterTitleText string

	Name   string
	Config *cfg.Config
	Fs     FileSystem

	Db struct {
		Models DbConfig
		Data   DbConfig
	}

	DaemonUrlPath string
}

// DbConfig 数据库配置
type DbConfig struct {
	DbType   string
	Address  string
	Port     string
	DbName   string
	Username string
	Password string
}

func (db *DbConfig) Host() string {
	if "" != db.Port && "0" != db.Port {
		return net.JoinHostPort(db.Address, db.Port)
	}
	switch db.DbType {
	case "postgresql":
		return net.JoinHostPort(db.Address, "5432")
	case "mysql":
		return net.JoinHostPort(db.Address, "3306")
	default:
		panic(errors.New("unknown db type - " + db.DbType))
	}
}

func (db *DbConfig) dbUrl() (string, string, error) {
	switch db.DbType {
	case "postgresql":
		if db.Port == "" {
			db.Port = "5432"
		}
		return "postgres", fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
			db.Address, db.Port, db.DbName, db.Username, db.Password), nil
	case "mysql":
		if db.Port == "" {
			db.Port = "3306"
		}
		return "mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?autocommit=true&parseTime=true",
			db.Username, db.Password, net.JoinHostPort(db.Address, db.Port), db.DbName), nil
	case "odbc_with_mssql":
		return "odbc_with_mssql", fmt.Sprintf("dsn=%s;uid=%s;pwd=%s",
			db.DbName, db.Username, db.Password), nil
	default:
		return "", "", errors.New("unknown db type - " + db.DbType)
	}
}

func (db *DbConfig) Url() (string, string) {
	dbDrv, dbUrl, err := db.dbUrl()
	if err != nil {
		panic(errors.New("unknown db type - " + db.DbType))
	}
	return dbDrv, dbUrl
}

func ReadFileWithDefault(files []string, defaultValue string) string {
	for _, s := range files {
		content, e := ioutil.ReadFile(s)
		if nil == e {
			if content = bytes.TrimSpace(content); len(content) > 0 {
				return string(content)
			}
		}
	}
	return defaultValue
}

func readDbConfig(prefix string, props *cfg.Config) DbConfig {
	return DbConfig{
		DbType:   props.StringWithDefault(prefix+"db.type", "postgresql"),
		Address:  props.StringWithDefault(prefix+"db.address", "127.0.0.1"),
		Port:     props.StringWithDefault(prefix+"db.port", ""),
		DbName:   props.StringWithDefault(prefix+"db.db_name", ""),
		Username: props.StringWithDefault(prefix+"db.username", ""),
		Password: props.StringWithDefault(prefix+"db.password", ""),
	}
}

func init() {
	On(func() Option {
		return fx.Provide(func(cfg *cfg.Config, fs FileSystem, logger log.Logger) *Environment {
			env := &Environment{
				Logger:        logger,
				Name:          cfg.StringWithDefault("product.name", "moo"),
				Config:        cfg,
				Fs:            fs,
				DaemonUrlPath: cfg.StringWithDefault("daemon.urlpath", "moo"),
			}
			env.Db.Models = readDbConfig("models.", cfg)
			env.Db.Data = readDbConfig("data.", cfg)

			if !strings.HasPrefix(env.DaemonUrlPath, "/") {
				env.DaemonUrlPath = "/" + env.DaemonUrlPath
			}
			if !strings.HasSuffix(env.DaemonUrlPath, "/") {
				env.DaemonUrlPath = env.DaemonUrlPath + "/"
			}
			env.HeaderTitleText = cfg.StringWithDefault("product.header_title",
				ReadFileWithDefault([]string{
					fs.FromDataConfig("resources/profiles/header.txt"),
					fs.FromData("resources/profiles/header.txt")},
					"IT综合运维管理平台"))

			env.FooterTitleText = cfg.StringWithDefault("product.footer_title",
				ReadFileWithDefault([]string{
					fs.FromDataConfig("resources/profiles/footer.txt"),
					fs.FromData("resources/profiles/footer.txt")},
					"© 2020 恒维信息技术(上海)有限公司, 保留所有版权。"))

			env.LoginHeaderTitleText = cfg.StringWithDefault("product.login_header_title",
				ReadFileWithDefault([]string{
					fs.FromDataConfig("resources/profiles/login-title.txt"),
					fs.FromData("resources/profiles/login-title.txt")},
					env.HeaderTitleText))

			env.LoginFooterTitleText = cfg.StringWithDefault("product.login_footer_title",
				ReadFileWithDefault([]string{
					fs.FromDataConfig("resources/profiles/login-footer.txt"),
					fs.FromData("resources/profiles/login-footer.txt")},
					env.FooterTitleText))

			return env
		})
	})
}
