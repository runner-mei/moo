package db

import (
	"database/sql"
	"fmt"
	"net"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/cfg"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"go.uber.org/fx"
)

type InModelDB struct {
	fx.In
	DrvName string  `name:"drv_models"`
	ConnURL string  `name:"conn_url_models"`
	DB      *sql.DB `name:"models"`

	InitSQL func(env *moo.Environment, logger log.Logger, db *sql.DB) error `name:"initSQL"`
}

type InDataDB struct {
	fx.In
	DrvName string  `name:"drv_data"`
	ConnURL string  `name:"conn_url_data"`
	DB      *sql.DB `name:"data"`
}

type InModelFactory struct {
	fx.In
	Factory *gobatis.SessionFactory `name:"modelFactory"`

	InitSQL func(env *moo.Environment, logger log.Logger, db *sql.DB) error `name:"initSQL"`
}

type InDataFactory struct {
	fx.In
	Factory *gobatis.SessionFactory `name:"dataFactory"`
}

type InConstants struct {
	fx.In
	Constants map[string]interface{} `name:"db_constants"`
}

type DbModelResult struct {
	fx.Out
	Constants            map[string]interface{}                                          `name:"db_constants"`
	DrvModels            string                                                          `name:"drv_models"`
	ConnURL              string                                                          `name:"conn_url_models"`
	Models               *sql.DB                                                         `name:"models"`
	ModelsSessionFactory *gobatis.SessionFactory                                         `name:"modelFactory"`
	InitSQL              func(env *moo.Environment, logger log.Logger, db *sql.DB) error `name:"initSQL"`
}

type DbDataResult struct {
	fx.Out
	DrvData            string                  `name:"drv_data"`
	ConnURL            string                  `name:"conn_url_data"`
	Data               *sql.DB                 `name:"data"`
	DataSessionFactory *gobatis.SessionFactory `name:"dataFactory"`
}

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return fx.Provide(func(env *moo.Environment, logger log.Logger) (DbModelResult, error) {
			dbPrefix := env.Config.StringWithDefault(env.Namespace+api.CfgDbPrefix, env.Namespace+".")
			dbConfig := readDbConfig(dbPrefix, env.Config)

			drvModels, urlModels := dbConfig.Url()

			logger.Debug("准备连接数据库", log.String("drvName", drvModels), log.String("URL", urlModels))

			dbModels, err := sql.Open(drvModels, urlModels)
			if nil != err {
				return DbModelResult{}, errors.Wrap(err, "connect to models database failed")
			}
			err = dbModels.Ping()
			if nil != err {
				dbModels.Close()
				return DbModelResult{}, errors.Wrap(err, "connect to models database failed")
			}

			err = initDb(env, logger, dbModels)
			if nil != err {
				dbModels.Close()
				return DbModelResult{}, errors.Wrap(err, "connect to models database failed")
			}

			constants := map[string]interface{}{
				// "discriminator_request": itsm_models.DiscriminatorRequest,
				// "discriminator_task":    itsm_models.DiscriminatorTask,
			}

			tracer := log.NewSQLTracer(logger.Named("gobatis.models"))
			modelFactory, err := gobatis.New(&gobatis.Config{
				Tracer:     tracer,
				TagPrefix:  "xorm",
				TagMapper:  gobatis.TagSplitForXORM,
				Constants:  constants,
				DriverName: drvModels,
				DB:         dbModels,
				XMLPaths: []string{
					"gobatis",
				},
			})
			if err != nil {
				dbModels.Close()
				return DbModelResult{}, errors.Wrap(err, "connect to models factory failed")
			}

			logger.Debug("models 数据库连接成功", log.String("drvName", drvModels), log.String("URL", urlModels))

			return DbModelResult{
				Constants:            constants,
				DrvModels:            drvModels,
				ConnURL:              urlModels,
				Models:               dbModels,
				ModelsSessionFactory: modelFactory,
				InitSQL:              initDb,
			}, nil
		})
	})

	moo.On(func(*moo.Environment) moo.Option {
		return fx.Provide(func(env *moo.Environment, logger log.Logger, constants InConstants) (DbDataResult, error) {
			dbPrefix := env.Config.StringWithDefault(env.Namespace+api.CfgDbDataPrefix, env.Namespace+".data.")
			dbConfig := readDbConfig(dbPrefix, env.Config)

			drvData, urlData := dbConfig.Url()

			logger.Debug("准备连接数据库", log.String("drvName", drvData), log.String("URL", urlData))

			dbData, err := sql.Open(drvData, urlData)
			if nil != err {
				return DbDataResult{}, errors.Wrap(err, "connect to models database failed")
			}
			err = dbData.Ping()
			if nil != err {
				dbData.Close()
				return DbDataResult{}, errors.Wrap(err, "connect to models database failed")
			}

			tracer := log.NewSQLTracer(logger.Named("gobatis.data"))
			dataFactory, err := gobatis.New(&gobatis.Config{Tracer: tracer,
				TagPrefix:  "xorm",
				TagMapper:  gobatis.TagSplitForXORM,
				Constants:  constants.Constants,
				DriverName: drvData,
				DB:         dbData,
				XMLPaths: []string{
					"gobatis",
				},
			})
			if err != nil {
				dbData.Close()
				return DbDataResult{}, errors.Wrap(err, "connect to data factory failed")
			}

			logger.Debug("data 数据库连接成功", log.String("drvName", drvData), log.String("URL", urlData))

			return DbDataResult{
				DrvData:            drvData,
				ConnURL:            urlData,
				Data:               dbData,
				DataSessionFactory: dataFactory,
			}, nil
		})
	})
}

// DbConfig 数据库配置
type DbConfig struct {
	// NOTE: 为测试增加的
	drvName string
	connURL string

	DbType   string
	Address  string
	Port     string
	DbName   string
	Username string
	Password string
}

// func (db *DbConfig) Host() string {
// 	if "" != db.Port && "0" != db.Port {
// 		return net.JoinHostPort(db.Address, db.Port)
// 	}
// 	switch db.DbType {
// 	case "postgresql":
// 		return net.JoinHostPort(db.Address, "5432")
// 	case "mysql":
// 		return net.JoinHostPort(db.Address, "3306")
// 	default:
// 		panic(errors.New("unknown db type - " + db.DbType))
// 	}
// }

func (db *DbConfig) getURL() (string, string, error) {
	if db.connURL != "" {
		// NOTE: 为测试增加的
		return db.drvName, db.connURL, nil
	}
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
	dbDrv, dbUrl, err := db.getURL()
	if err != nil {
		panic(errors.New("unknown db type - " + db.DbType))
	}
	return dbDrv, dbUrl
}

func readDbConfig(prefix string, props *cfg.Config) DbConfig {
	dbConfig := DbConfig{
		DbType:   props.StringWithDefault(prefix+"db.type", "postgresql"),
		Address:  props.StringWithDefault(prefix+"db.address", "127.0.0.1"),
		Port:     props.StringWithDefault(prefix+"db.port", ""),
		DbName:   props.StringWithDefault(prefix+"db.dbname", ""),
		Username: props.StringWithDefault(prefix+"db.username", ""),
		Password: props.StringWithDefault(prefix+"db.password", ""),
	}

	if props.StringWithDefault("moo.runMode", "") == "dev" {
		// NOTE: 为测试增加的
		switch prefix {
		case "models.":
			dbConfig.drvName = props.StringWithDefault("moo.test.models.db_drv", "")
			dbConfig.connURL = props.StringWithDefault("moo.test.models.db_url", "")
		case "data.":
			dbConfig.drvName = props.StringWithDefault("moo.test.data.db_drv", "")
			dbConfig.connURL = props.StringWithDefault("moo.test.data.db_url", "")
		}
	}
	return dbConfig
}
