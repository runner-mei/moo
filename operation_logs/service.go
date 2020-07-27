//go:generate gobatis service.go

package operation_logs

import (
	"context"
	"os"
	"time"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/db"
)

type OperationLog = api.OperationLog
type ChangeRecord = api.ChangeRecord
type OperationLogRecord = api.OperationLogRecord
type TimeRange = api.TimeRange
type OperationLogDao = api.OperationLogDao
type OperationLogger = api.OperationLogger
type OperationQueryer = api.OperationQueryer

type OldOperationLog struct {
	TableName  struct{}            `json:"-" xorm:"moo_operation_logs"`
	ID         int64               `json:"id,omitempty" xorm:"id pk autoincr"`
	UserID     int64               `json:"userid,omitempty" xorm:"-"`
	Username   string              `json:"username" xorm:"user_name notnull"`
	Successful bool                `json:"successful" xorm:"successful notnull"`
	Type       string              `json:"type" xorm:"type notnull"`
	Errors     string              `json:"errors" xorm:"errors null"`
	Content    string              `json:"content,omitempty" xorm:"content null"`
	Fields     *OperationLogRecord `json:"attributes,omitempty" xorm:"attributes json null"`
	CreatedAt  time.Time           `json:"created_at,omitempty" xorm:"created_at"`
	UpdatedAt  *time.Time          `json:"updated_at,omitempty" xorm:"updated_at <-"`
}

type OldOperationLogDao interface {
	Insert(ctx context.Context, ol *OldOperationLog) error

	// @default SELECT count(*) FROM <tablename type="OldOperationLog" /> <where>
	// <if test="len(usernames) &gt; 0"> <foreach collection="usernames" open="user_name in (" close=")"  separator="," >#{item}</foreach> </if>
	// <if test="successful"> AND successful = #{successful} </if>
	// <if test="len(typeList) &gt; 0"> AND <foreach collection="typeList" open="type in (" close=")" separator="," >#{item}</foreach> </if>
	// <if test="!createdAt.Start.IsZero()"> AND created_at &gt;= #{createdAt.Start} </if>
	// <if test="!createdAt.End.IsZero()"> AND created_at &lt; #{createdAt.End} </if>
	// </where>
	Count(ctx context.Context, usernames []string, successful bool, typeList []string, createdAt TimeRange) (int64, error)

	// @default SELECT * FROM <tablename type="OldOperationLog" /> <where>
	// <if test="len(usernames) &gt; 0"> <foreach collection="usernames" open="user_name in (" close=")"  separator="," >#{item}</foreach> </if>
	// <if test="successful"> AND successful = #{successful} </if>
	// <if test="len(typeList) &gt; 0"> AND <foreach collection="typeList" open="type in (" close=")"  separator="," >#{item}</foreach> </if>
	// <if test="!createdAt.Start.IsZero()"> AND created_at &gt;= #{createdAt.Start} </if>
	// <if test="!createdAt.End.IsZero()"> AND created_at &lt; #{createdAt.End} </if>
	// </where>
	// <sort_by />
	// <pagination />
	List(ctx context.Context, usernames []string, successful bool, typeList []string, createdAt TimeRange, offset, limit int64, sort string) ([]OldOperationLog, error)
}

var InitOperationQueryer = api.InitOperationQueryer

type operationLogger struct {
	tx  gobatis.DBRunner
	dao OperationLogDao
}

func (logger operationLogger) Tx(tx *gobatis.Tx) OperationLogger {
	if tx == nil {
		return logger
	}
	return operationLogger{dao: api.NewOperationLogDao(tx.SessionReference())}
}

func (logger operationLogger) WithTx(tx gobatis.DBRunner) OperationLogger {
	if tx == nil {
		return logger
	}
	return operationLogger{dao: logger.dao, tx: tx}
}

func (logger operationLogger) LogRecord(ctx context.Context, ol *OperationLog) error {
	if logger.tx != nil {
		if ctx == nil {
			ctx = gobatis.WithDbConnection(context.Background(), logger.tx)
		} else {
			ctx = gobatis.WithDbConnection(ctx, logger.tx)
		}
	}
	return logger.dao.Insert(ctx, ol)
}

type oldOperationLogger struct {
	tx  gobatis.DBRunner
	dao OldOperationLogDao
}

func (logger oldOperationLogger) Tx(tx *gobatis.Tx) OperationLogger {
	if tx == nil {
		return logger
	}
	return oldOperationLogger{dao: NewOldOperationLogDao(tx.SessionReference())}
}

func (logger oldOperationLogger) WithTx(tx gobatis.DBRunner) OperationLogger {
	if tx == nil {
		return logger
	}
	return oldOperationLogger{dao: logger.dao, tx: tx}
}

func (logger oldOperationLogger) LogRecord(ctx context.Context, ol *OperationLog) error {
	if logger.tx != nil {
		if ctx == nil {
			ctx = gobatis.WithDbConnection(context.Background(), logger.tx)
		} else {
			ctx = gobatis.WithDbConnection(ctx, logger.tx)
		}
	}

	username := ol.Username
	if username == "" {
		username = "system"
	}
	return logger.dao.Insert(ctx, &OldOperationLog{
		Username:   username,
		Successful: ol.Successful,
		Type:       ol.Type,
		Content:    ol.Content,
		Fields:     ol.Fields,
	})
}

type operationQueryer struct {
	names map[string]OperationLogLocaleConfig
	dao   OperationLogDao
}

func (queryer operationQueryer) Types(ctx context.Context) map[string]OperationLogLocaleConfig {
	return queryer.names
}

func (queryer operationQueryer) toTypeTilte(ctx context.Context, typeName string) string {
	s, ok := queryer.names[typeName]
	if !ok && s.Title == "" {
		return typeName
	}
	return s.Title
}

func (queryer operationQueryer) Count(ctx context.Context, userid []int64, successful bool, typeList []string, beginAt, endAt time.Time) (int64, error) {
	return queryer.dao.Count(ctx, userid, successful, typeList, TimeRange{Start: beginAt, End: endAt})
}

func (queryer operationQueryer) List(ctx context.Context, userid []int64, successful bool, typeList []string, beginAt, endAt time.Time, offset, limit int64, sortBy string) ([]OperationLog, error) {
	items, err := queryer.dao.List(ctx, userid, successful, typeList, TimeRange{Start: beginAt, End: endAt}, offset, limit, sortBy)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].TypeTitle = queryer.toTypeTilte(ctx, items[idx].Type)
	}
	return items, nil
}

type oldOperationQueryer struct {
	names            map[string]OperationLogLocaleConfig
	dao              OldOperationLogDao
	findUsernameByID func(ctx context.Context, id int64) (string, error)
}

func (queryer oldOperationQueryer) Types(ctx context.Context) map[string]OperationLogLocaleConfig {
	return queryer.names
}

func (queryer oldOperationQueryer) getUsernames(ctx context.Context, userid []int64) ([]string, error) {
	var names []string
	for _, id := range userid {
		name, err := queryer.findUsernameByID(ctx, id)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func (queryer oldOperationQueryer) toTypeTilte(ctx context.Context, typeName string) string {
	s, ok := queryer.names[typeName]
	if !ok && s.Title == "" {
		return typeName
	}
	return s.Title
}

func (queryer oldOperationQueryer) Count(ctx context.Context, userid []int64, successful bool, typeList []string, beginAt, endAt time.Time) (int64, error) {
	usernames, err := queryer.getUsernames(ctx, userid)
	if err != nil {
		return 0, err
	}
	return queryer.dao.Count(ctx, usernames, successful, typeList, TimeRange{Start: beginAt, End: endAt})
}

func (queryer oldOperationQueryer) List(ctx context.Context, userid []int64, successful bool, typeList []string, beginAt, endAt time.Time, offset, limit int64, sortBy string) ([]OperationLog, error) {
	usernames, err := queryer.getUsernames(ctx, userid)
	if err != nil {
		return nil, err
	}
	switch sortBy {
	case "+userid", "userid":
		sortBy = "+user_name"
	case "-userid":
		sortBy = "-user_name"
	}
	logList, err := queryer.dao.List(ctx, usernames, successful, typeList, TimeRange{Start: beginAt, End: endAt}, offset, limit, sortBy)
	if err != nil {
		return nil, err
	}
	var results = make([]OperationLog, len(logList))
	for idx := range logList {
		results[idx].ID = logList[idx].ID
		results[idx].Username = logList[idx].Username
		results[idx].Successful = logList[idx].Successful
		results[idx].Type = logList[idx].Type
		results[idx].TypeTitle = queryer.toTypeTilte(ctx, logList[idx].Type)
		results[idx].Content = logList[idx].Content
		results[idx].Fields = logList[idx].Fields
		results[idx].CreatedAt = logList[idx].CreatedAt
	}
	return results, nil
}

type OperationLogLocaleConfig = api.OperationLogLocaleConfig

func LoadOperationLogLocaleConfig(env *moo.Environment) (map[string]OperationLogLocaleConfig, error) {
	filename := env.Fs.FromConfig("operation_logs.zh.json")
	customFilename := env.Fs.FromDataConfig("operation_logs.zh.json")

	var cfg map[string]OperationLogLocaleConfig
	err := util.FromHjsonFile(filename, &cfg)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var customCfg map[string]OperationLogLocaleConfig
	err = util.FromHjsonFile(customFilename, &customCfg)
	if err != nil {
		if !os.IsNotExist(err) {
			return map[string]OperationLogLocaleConfig{}, nil
		}
	} else if len(cfg) == 0 {
		cfg = customCfg
	} else {
		for key, newValue := range customCfg {
			oldValue, ok := cfg[key]
			if !ok {
				cfg[key] = newValue
				continue
			}
			if newValue.Title != "" {
				oldValue.Title = newValue.Title
			}

			if len(oldValue.Fields) == 0 {
				oldValue.Fields = newValue.Fields
			} else {
				for k, v := range newValue.Fields {
					oldValue.Fields[k] = v
				}
			}
			cfg[key] = newValue
		}
	}
	return cfg, nil
}

func NewOperationQueryer(env *moo.Environment, session gobatis.SqlSession,
	findUsernameByID func(ctx context.Context, id int64) (string, error)) (OperationQueryer, error) {
	names, err := LoadOperationLogLocaleConfig(env)
	if err != nil {
		return nil, err
	}
	if env.Config.IntWithDefault("moo.operation_logger", 0) == 2 {
		return operationQueryer{names: names, dao: api.NewOperationLogDao(session)}, nil
	}

	return oldOperationQueryer{names: names, dao: NewOldOperationLogDao(session), findUsernameByID: findUsernameByID}, nil
}

func NewOperationLogger(env *moo.Environment, session gobatis.SqlSession) OperationLogger {
	if env.Config.IntWithDefault("moo.operation_logger", 0) == 2 {
		return operationLogger{dao: api.NewOperationLogDao(session)}
	}

	return oldOperationLogger{dao: NewOldOperationLogDao(session)}
}

func init() {
	moo.On(func() moo.Option {
		return moo.Provide(func(env *moo.Environment, db db.InModelFactory, logger log.Logger) OperationLogger {
			return NewOperationLogger(env, db.Factory.SessionReference())
		})
	})
}
