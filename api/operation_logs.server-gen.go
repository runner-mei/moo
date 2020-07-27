// Please don't edit this file!
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/runner-mei/loong"
)

// OperationLog is skipped
// ChangeRecord is skipped
// OperationLogRecord is skipped
// OperationLogLocaleConfig is skipped
// TimeRange is skipped
// OperationLogDao is skipped
// OperationLogger is skipped

func InitOperationQueryer(mux loong.Party, svc OperationQueryer) {
	// Types: annotation is missing
	mux.GET("/count", func(ctx *loong.Context) error {
		var useridList []int64
		if ss := ctx.QueryParamArray("userid_list"); len(ss) != 0 {
			useridListValue, err := ToInt64Array(ss)
			if err != nil {
				return ctx.ReturnError(loong.ErrBadArgument("userid_list", ss, err), http.StatusBadRequest)
			}
			useridList = useridListValue
		}
		var successful bool
		if s := ctx.QueryParam("successful"); s != "" {
			successful = toBool(s)
		}
		var typeList = ctx.QueryParamArray("type_list")
		var beginAt time.Time
		if s := ctx.QueryParam("begin_at"); s != "" {
			beginAtValue, err := loong.ToDatetime(s)
			if err != nil {
				return ctx.ReturnError(loong.ErrBadArgument("begin_at", s, err), http.StatusBadRequest)
			}
			beginAt = beginAtValue
		}
		var endAt time.Time
		if s := ctx.QueryParam("end_at"); s != "" {
			endAtValue, err := loong.ToDatetime(s)
			if err != nil {
				return ctx.ReturnError(loong.ErrBadArgument("end_at", s, err), http.StatusBadRequest)
			}
			endAt = endAtValue
		}

		result, err := svc.Count(ctx.StdContext, useridList, successful, typeList, beginAt, endAt)
		if err != nil {
			return ctx.ReturnError(err)
		}
		return ctx.ReturnQueryResult(result)
	})
	mux.GET("", func(ctx *loong.Context) error {
		var useridList []int64
		if ss := ctx.QueryParamArray("userid_list"); len(ss) != 0 {
			useridListValue, err := ToInt64Array(ss)
			if err != nil {
				return ctx.ReturnError(loong.ErrBadArgument("userid_list", ss, err), http.StatusBadRequest)
			}
			useridList = useridListValue
		}
		var successful bool
		if s := ctx.QueryParam("successful"); s != "" {
			successful = toBool(s)
		}
		var typeList = ctx.QueryParamArray("type_list")
		var beginAt time.Time
		if s := ctx.QueryParam("begin_at"); s != "" {
			beginAtValue, err := loong.ToDatetime(s)
			if err != nil {
				return ctx.ReturnError(loong.ErrBadArgument("begin_at", s, err), http.StatusBadRequest)
			}
			beginAt = beginAtValue
		}
		var endAt time.Time
		if s := ctx.QueryParam("end_at"); s != "" {
			endAtValue, err := loong.ToDatetime(s)
			if err != nil {
				return ctx.ReturnError(loong.ErrBadArgument("end_at", s, err), http.StatusBadRequest)
			}
			endAt = endAtValue
		}
		var offset int64
		if s := ctx.QueryParam("offset"); s != "" {
			offsetValue, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return ctx.ReturnError(loong.ErrBadArgument("offset", s, err), http.StatusBadRequest)
			}
			offset = offsetValue
		}
		var limit int64
		if s := ctx.QueryParam("limit"); s != "" {
			limitValue, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return ctx.ReturnError(loong.ErrBadArgument("limit", s, err), http.StatusBadRequest)
			}
			limit = limitValue
		}
		var sortBy = ctx.QueryParam("sort_by")

		result, err := svc.List(ctx.StdContext, useridList, successful, typeList, beginAt, endAt, offset, limit, sortBy)
		if err != nil {
			return ctx.ReturnError(err)
		}
		return ctx.ReturnQueryResult(result)
	})
}
