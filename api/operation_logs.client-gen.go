// Please don't edit this file!
package api

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/runner-mei/resty"
)

// OperationLog is skipped
// ChangeRecord is skipped
// OperationLogRecord is skipped
// OperationLogLocaleConfig is skipped
// TimeRange is skipped
// OperationLogDao is skipped
// OperationLogger is skipped

type OperationQueryerClient struct {
	Proxy *resty.Proxy
}

// Types: annotation is missing

func (client OperationQueryerClient) Count(ctx context.Context, useridList []int64, successful sql.NullBool, typeList []string, contentLike string, beginAt time.Time, endAt time.Time) (int64, error) {
	var result int64

	request := resty.NewRequest(client.Proxy, "/count")
	for idx := range useridList {
		request = request.AddParam("userid_list", strconv.FormatInt(useridList[idx], 10))
	}
	if successful.Valid {
		request = request.SetParam("successful", BoolToString(successful.Bool))
	}
	for idx := range typeList {
		request = request.AddParam("type_list", typeList[idx])
	}
	request = request.SetParam("content_like", contentLike).
		SetParam("begin_at", beginAt.Format(client.Proxy.TimeFormat)).
		SetParam("end_at", endAt.Format(client.Proxy.TimeFormat)).
		Result(&result)

	err := request.GET(ctx)
	resty.ReleaseRequest(client.Proxy, request)
	return result, err
}

func (client OperationQueryerClient) List(ctx context.Context, useridList []int64, successful sql.NullBool, typeList []string, contentLike string, beginAt time.Time, endAt time.Time, offset int64, limit int64, sortBy string) ([]OperationLog, error) {
	var result []OperationLog

	request := resty.NewRequest(client.Proxy, "/")
	for idx := range useridList {
		request = request.AddParam("userid_list", strconv.FormatInt(useridList[idx], 10))
	}
	if successful.Valid {
		request = request.SetParam("successful", BoolToString(successful.Bool))
	}
	for idx := range typeList {
		request = request.AddParam("type_list", typeList[idx])
	}
	request = request.SetParam("content_like", contentLike).
		SetParam("begin_at", beginAt.Format(client.Proxy.TimeFormat)).
		SetParam("end_at", endAt.Format(client.Proxy.TimeFormat)).
		SetParam("offset", strconv.FormatInt(offset, 10)).
		SetParam("limit", strconv.FormatInt(limit, 10)).
		SetParam("sort_by", sortBy).
		Result(&result)

	err := request.GET(ctx)
	resty.ReleaseRequest(client.Proxy, request)
	return result, err
}
