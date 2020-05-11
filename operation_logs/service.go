package operation_logs

import (
	"context"
	"strings"
	"time"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/moo"
)

func BoolToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func toBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true"
}

type operationQueryer struct {
	dao OperationLogDao
}

func (queryer operationQueryer) Count(ctx context.Context, userid int64, successful bool, typeList []string, beginAt, endAt time.Time) (int64, error) {
	return queryer.dao.Count(ctx, userid, successful, typeList, TimeRange{Start: beginAt, End: endAt})
}

func (queryer operationQueryer) List(ctx context.Context, userid int64, successful bool, typeList []string, beginAt, endAt time.Time, offset, limit int64, sortBy string) ([]OperationLog, error) {
	return queryer.dao.List(ctx, userid, successful, typeList, TimeRange{Start: beginAt, End: endAt}, offset, limit, sortBy)
}

type oldOperationQueryer struct {
	dao OldOperationLogDao
}

func (queryer oldOperationQueryer) Count(ctx context.Context, userid int64, successful bool, typeList []string, beginAt, endAt time.Time) (int64, error) {
	return queryer.dao.Count(ctx, userid, successful, typeList, TimeRange{Start: beginAt, End: endAt})
}

func (queryer oldOperationQueryer) List(ctx context.Context, userid int64, successful bool, typeList []string, beginAt, endAt time.Time, offset, limit int64, sortBy string) ([]OperationLog, error) {
	logList, err := queryer.dao.List(ctx, userid, successful, typeList, TimeRange{Start: beginAt, End: endAt}, offset, limit, sortBy)
	if err != nil {
		return nil, err
	}
	var results = make([]OperationLog, len(logList))
	for idx := range logList {
		results[idx].ID = logList[idx].ID
		results[idx].Username = logList[idx].Username
		results[idx].Successful = logList[idx].Successful
		results[idx].Type = logList[idx].Type
		results[idx].Content = logList[idx].Content
		results[idx].Fields = logList[idx].Fields
		results[idx].CreatedAt = logList[idx].CreatedAt
	}
	return results, nil
}

func NewOperationQueryer(env *moo.Environment, session gobatis.SqlSession) OperationQueryer {
	if env.Config.IntWithDefault("moo.operation_logger", 0) == 2 {
		panic("")
		return operationQueryer{dao: NewOperationLogDao(session)}
	}

	return oldOperationQueryer{dao: NewOldOperationLogDao(session)}
}
