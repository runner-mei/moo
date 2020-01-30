//go:generate gogen server -pre_init_object=true -ext=.server-gen.go -config=@loong operation_logs.go
//go:generate gogen client -ext=.client-gen.go operation_logs.go
//go:generate gobatis operation_logs.go

package operation_logs

import (
	"context"
	"time"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/db"
	"go.uber.org/fx"
)

type OperationLog struct {
	TableName  struct{}            `json:"-" xorm:"moo_operation_logs"`
	ID         int64               `json:"id,omitempty" xorm:"id pk autoincr"`
	UserID     int64               `json:"userid,omitempty" xorm:"userid null"`
	Username   string              `json:"username,omitempty" xorm:"username null"`
	Successful bool                `json:"successful" xorm:"successful notnull"`
	Type       string              `json:"type" xorm:"type notnull"`
	Content    string              `json:"content,omitempty" xorm:"content null"`
	Fields     *OperationLogRecord `json:"attributes,omitempty" xorm:"attributes json null"`
	CreatedAt  time.Time           `json:"created_at,omitempty" xorm:"created_at"`
}

type ChangeRecord struct {
	Name     string      `json:"name"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

type OperationLogRecord struct {
	ObjectType string         `json:"object_type,omitempty"`
	ObjectID   int64          `json:"object_id,omitempty"`
	Records    []ChangeRecord `json:"records,omitempty"`
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type OperationLogDao interface {
	Insert(ctx context.Context, ol *OperationLog) error
	DeleteBy(ctx context.Context, createdAt TimeRange) error
	Count(ctx context.Context, userid int64, successful bool, typeList []string, createdAt TimeRange) (int64, error)
	List(ctx context.Context, userid int64, successful bool, typeList []string, createdAt TimeRange, offset, limit int64, sortBy string) ([]OperationLog, error)
}

// @gobatis.ignore
type OperationLogger interface {
	WithTx(tx gobatis.DBRunner) OperationLogger
	LogRecord(ctx context.Context, ol *OperationLog) error
}

type operationLogger struct {
	tx  gobatis.DBRunner
	dao OperationLogDao
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

type OldOperationLog struct {
	TableName  struct{}            `json:"-" xorm:"moo_operation_logs"`
	ID         int64               `json:"id,omitempty" xorm:"id pk autoincr"`
	UserID     int64               `json:"userid,omitempty" xorm:"-"`
	Username   string              `json:"username" xorm:"user_name notnull"`
	Successful bool                `json:"successful" xorm:"successful notnull"`
	Type       string              `json:"type" xorm:"type notnull"`
	Content    string              `json:"content,omitempty" xorm:"content null"`
	Fields     *OperationLogRecord `json:"attributes,omitempty" xorm:"attributes json null"`
	CreatedAt  time.Time           `json:"created_at,omitempty" xorm:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at,omitempty" xorm:"updated_at"`
}

type OldOperationLogDao interface {
	Insert(ctx context.Context, ol *OldOperationLog) error
}

type oldOperationLogger struct {
	tx  gobatis.DBRunner
	dao OldOperationLogDao
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
func NewOperationLogger(env *moo.Environment, dbFactory *gobatis.SessionFactory) OperationLogger {
	if env.Config.IntWithDefault("operation_logger", 0) == 2 {
		return operationLogger{dao: NewOperationLogDao(dbFactory.SessionReference())}
	}

	return oldOperationLogger{dao: NewOldOperationLogDao(dbFactory.SessionReference())}
}

func init() {
	moo.On(func() moo.Option {
		return fx.Provide(func(env *moo.Environment, db db.ArgModelFactory, logger log.Logger) OperationLogger {
			return NewOperationLogger(env, db.Factory)
		})
	})
}
