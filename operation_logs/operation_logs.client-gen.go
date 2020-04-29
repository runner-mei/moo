// Please don't edit this file!
package operation_logs

import (
	"context"
	"strconv"
	"time"

	"github.com/runner-mei/resty"
)

// OperationLog is skipped
// ChangeRecord is skipped
// OperationLogRecord is skipped
// TimeRange is skipped
// OperationLogDao is skipped
// OperationLogger is skipped
// operationLogger is skipped
// OldOperationLog is skipped
// OldOperationLogDao is skipped
// oldOperationLogger is skipped

type OperationQueryerClient struct {
	Proxy *resty.Proxy
}

func (client OperationQueryerClient) Count(ctx context.Context, userid int64, successful bool, typeList []string, beginAt time.Time, endAt time.Time) (int64, error) {
	var result int64

	request := resty.NewRequest(client.Proxy, "/count").
		SetParam("userid", strconv.FormatInt(userid, 10)).
		SetParam("successful", BoolToString(successful))
	for idx := range typeList {
		request = request.AddParam("type_list", typeList[idx])
	}
	request = request.SetParam("begin_at", beginAt.Format(client.Proxy.TimeFormat)).
		SetParam("end_at", endAt.Format(client.Proxy.TimeFormat)).
		Result(&result)

	err := request.GET(ctx)
	resty.ReleaseRequest(client.Proxy, request)
	return result, err
}

func (client OperationQueryerClient) List(ctx context.Context, userid int64, successful bool, typeList []string, beginAt time.Time, endAt time.Time, offset int64, limit int64, sortBy string) ([]OperationLog, error) {
	var result []OperationLog

	request := resty.NewRequest(client.Proxy, "/").
		SetParam("userid", strconv.FormatInt(userid, 10)).
		SetParam("successful", BoolToString(successful))
	for idx := range typeList {
		request = request.AddParam("type_list", typeList[idx])
	}
	request = request.SetParam("begin_at", beginAt.Format(client.Proxy.TimeFormat)).
		SetParam("end_at", endAt.Format(client.Proxy.TimeFormat)).
		SetParam("offset", strconv.FormatInt(offset, 10)).
		SetParam("limit", strconv.FormatInt(limit, 10)).
		SetParam("sort_by", sortBy).
		Result(&result)

	err := request.GET(ctx)
	resty.ReleaseRequest(client.Proxy, request)
	return result, err
}
