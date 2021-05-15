//go:generate gogen server -pre_init_object=true -ext=.server-gen.go -config=@loong operation_logs.go
//go:generate gogen client -ext=.client-gen.go operation_logs.go
//go:generate gobatis operation_logs.go

package api

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	gobatis "github.com/runner-mei/GoBatis"
)

type OperationLog struct {
	TableName  struct{}            `json:"-" xorm:"moo_operation_logs"`
	ID         int64               `json:"id,omitempty" xorm:"id pk autoincr"`
	UserID     int64               `json:"userid,omitempty" xorm:"userid null"`
	Username   string              `json:"username,omitempty" xorm:"username null"`
	Successful bool                `json:"successful" xorm:"successful notnull"`
	Type       string              `json:"type" xorm:"type notnull"`
	TypeTitle  string              `json:"type_title" xorm:"-"`
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

type OperationLogLocaleConfig struct {
	Title  string
	Fields map[string]string
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type OperationLogDao interface {
	Insert(ctx context.Context, ol *OperationLog) error
	DeleteBy(ctx context.Context, createdAt TimeRange) error
	Count(ctx context.Context, userids []int64, successful sql.NullBool, typeList []string, contentLike string, createdAt TimeRange) (int64, error)
	List(ctx context.Context, userids []int64, successful sql.NullBool, typeList []string, contentLike string, createdAt TimeRange, offset, limit int64, sortBy string) ([]OperationLog, error)
}

// @gobatis.ignore
type OperationLogger interface {
	Tx(tx *gobatis.Tx) OperationLogger
	WithTx(tx gobatis.DBRunner) OperationLogger
	LogRecord(ctx context.Context, ol *OperationLog) error
}

// @gobatis.ignore
type OperationQueryer interface {
	Types(ctx context.Context) map[string]OperationLogLocaleConfig

	// @http.GET(path="/count")
	Count(ctx context.Context, useridList []int64, successful sql.NullBool, typeList []string, contentLike string, beginAt, endAt time.Time) (int64, error)

	// @http.GET(path="")
	List(ctx context.Context, useridList []int64, successful sql.NullBool, typeList []string, contentLike string, beginAt, endAt time.Time, offset, limit int64, sortBy string) ([]OperationLog, error)
}

func BoolToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func ToBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true"
}

func ToInt64Array(array []string) ([]int64, error) {
	var int64Array []int64
	for _, s := range array {
		ss := strings.Split(s, ",")
		for _, v := range ss {
			if v == "" {
				continue
			}
			i64, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			int64Array = append(int64Array, i64)
		}
	}
	return int64Array, nil
}
