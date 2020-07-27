// Please don't edit this file!
package operation_logs

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/moo/api"
)

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
		{ //// OldOperationLogDao.Insert
			if _, exists := ctx.Statements["OldOperationLogDao.Insert"]; !exists {
				sqlStr, err := gobatis.GenerateInsertSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&OldOperationLog{}),
					[]string{
						"ol",
					},
					[]reflect.Type{
						reflect.TypeOf((*OldOperationLog)(nil)),
					}, true)
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate OldOperationLogDao.Insert error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "OldOperationLogDao.Insert",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OldOperationLogDao.Insert"] = stmt
			}
		}
		{ //// OldOperationLogDao.Count
			if _, exists := ctx.Statements["OldOperationLogDao.Count"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT count(*) FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&OldOperationLog{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" <where>\r\n <if test=\"len(usernames) &gt; 0\"> <foreach collection=\"usernames\" open=\"user_name in (\" close=\")\"  separator=\",\" >#{item}</foreach> </if>\r\n <if test=\"successful\"> AND successful = #{successful} </if>\r\n <if test=\"len(typeList) &gt; 0\"> AND <foreach collection=\"typeList\" open=\"type in (\" close=\")\" separator=\",\" >#{item}</foreach> </if>\r\n <if test=\"!createdAt.Start.IsZero()\"> AND created_at &gt;= #{createdAt.Start} </if>\r\n <if test=\"!createdAt.End.IsZero()\"> AND created_at &lt; #{createdAt.End} </if>\r\n </where>")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "OldOperationLogDao.Count",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OldOperationLogDao.Count"] = stmt
			}
		}
		{ //// OldOperationLogDao.List
			if _, exists := ctx.Statements["OldOperationLogDao.List"]; !exists {
				var sb strings.Builder
				sb.WriteString("SELECT * FROM ")
				if tablename, err := gobatis.ReadTableName(ctx.Mapper, reflect.TypeOf(&OldOperationLog{})); err != nil {
					return err
				} else {
					sb.WriteString(tablename)
				}
				sb.WriteString(" <where>\r\n <if test=\"len(usernames) &gt; 0\"> <foreach collection=\"usernames\" open=\"user_name in (\" close=\")\"  separator=\",\" >#{item}</foreach> </if>\r\n <if test=\"successful\"> AND successful = #{successful} </if>\r\n <if test=\"len(typeList) &gt; 0\"> AND <foreach collection=\"typeList\" open=\"type in (\" close=\")\"  separator=\",\" >#{item}</foreach> </if>\r\n <if test=\"!createdAt.Start.IsZero()\"> AND created_at &gt;= #{createdAt.Start} </if>\r\n <if test=\"!createdAt.End.IsZero()\"> AND created_at &lt; #{createdAt.End} </if>\r\n </where>\r\n <sort_by />\r\n <pagination />")
				sqlStr := sb.String()

				stmt, err := gobatis.NewMapppedStatement(ctx, "OldOperationLogDao.List",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OldOperationLogDao.List"] = stmt
			}
		}
		return nil
	})
}

func NewOldOperationLogDao(ref gobatis.SqlSession) OldOperationLogDao {
	if ref == nil {
		panic(errors.New("param 'ref' is nil"))
	}
	if reference, ok := ref.(*gobatis.Reference); ok {
		if reference.SqlSession == nil {
			panic(errors.New("param 'ref.SqlSession' is nil"))
		}
	} else if valueReference, ok := ref.(gobatis.Reference); ok {
		if valueReference.SqlSession == nil {
			panic(errors.New("param 'ref.SqlSession' is nil"))
		}
	}
	return &OldOperationLogDaoImpl{session: ref}
}

type OldOperationLogDaoImpl struct {
	session gobatis.SqlSession
}

func (impl *OldOperationLogDaoImpl) Insert(ctx context.Context, ol *OldOperationLog) error {
	_, err := impl.session.Insert(ctx, "OldOperationLogDao.Insert",
		[]string{
			"ol",
		},
		[]interface{}{
			ol,
		},
		true)
	return err
}

func (impl *OldOperationLogDaoImpl) Count(ctx context.Context, usernames []string, successful bool, typeList []string, createdAt api.TimeRange) (int64, error) {
	var instance int64
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "OldOperationLogDao.Count",
		[]string{
			"usernames",
			"successful",
			"typeList",
			"createdAt",
		},
		[]interface{}{
			usernames,
			successful,
			typeList,
			createdAt,
		}).Scan(&nullable)
	if err != nil {
		return 0, err
	}
	if !nullable.Valid {
		return 0, sql.ErrNoRows
	}

	return instance, nil
}

func (impl *OldOperationLogDaoImpl) List(ctx context.Context, usernames []string, successful bool, typeList []string, createdAt api.TimeRange, offset int64, limit int64, sort string) ([]OldOperationLog, error) {
	var instances []OldOperationLog
	results := impl.session.Select(ctx, "OldOperationLogDao.List",
		[]string{
			"usernames",
			"successful",
			"typeList",
			"createdAt",
			"offset",
			"limit",
			"sort",
		},
		[]interface{}{
			usernames,
			successful,
			typeList,
			createdAt,
			offset,
			limit,
			sort,
		})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}
