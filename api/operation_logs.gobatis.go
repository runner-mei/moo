// Please don't edit this file!
package api

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	gobatis "github.com/runner-mei/GoBatis"
)

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
		{ //// OperationLogDao.Insert
			if _, exists := ctx.Statements["OperationLogDao.Insert"]; !exists {
				sqlStr, err := gobatis.GenerateInsertSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&OperationLog{}),
					[]string{
						"ol",
					},
					[]reflect.Type{
						reflect.TypeOf((*OperationLog)(nil)),
					}, true)
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate OperationLogDao.Insert error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "OperationLogDao.Insert",
					gobatis.StatementTypeInsert,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OperationLogDao.Insert"] = stmt
			}
		}
		{ //// OperationLogDao.DeleteBy
			if _, exists := ctx.Statements["OperationLogDao.DeleteBy"]; !exists {
				sqlStr, err := gobatis.GenerateDeleteSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&OperationLog{}),
					[]string{
						"createdAt",
					},
					[]reflect.Type{
						reflect.TypeOf(&TimeRange{}).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate OperationLogDao.DeleteBy error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "OperationLogDao.DeleteBy",
					gobatis.StatementTypeDelete,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OperationLogDao.DeleteBy"] = stmt
			}
		}
		{ //// OperationLogDao.Count
			if _, exists := ctx.Statements["OperationLogDao.Count"]; !exists {
				sqlStr, err := gobatis.GenerateCountSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&OperationLog{}),
					[]string{
						"userids",
						"successful",
						"typeList",
						"contentLike",
						"createdAt",
					},
					[]reflect.Type{
						reflect.TypeOf([]int64{}),
						reflect.TypeOf(&sql.NullBool{}).Elem(),
						reflect.TypeOf([]string{}),
						reflect.TypeOf(new(string)).Elem(),
						reflect.TypeOf(&TimeRange{}).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate OperationLogDao.Count error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "OperationLogDao.Count",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OperationLogDao.Count"] = stmt
			}
		}
		{ //// OperationLogDao.List
			if _, exists := ctx.Statements["OperationLogDao.List"]; !exists {
				sqlStr, err := gobatis.GenerateSelectSQL(ctx.Dialect, ctx.Mapper,
					reflect.TypeOf(&OperationLog{}),
					[]string{
						"userids",
						"successful",
						"typeList",
						"contentLike",
						"createdAt",
						"offset",
						"limit",
						"sortBy",
					},
					[]reflect.Type{
						reflect.TypeOf([]int64{}),
						reflect.TypeOf(&sql.NullBool{}).Elem(),
						reflect.TypeOf([]string{}),
						reflect.TypeOf(new(string)).Elem(),
						reflect.TypeOf(&TimeRange{}).Elem(),
						reflect.TypeOf(new(int64)).Elem(),
						reflect.TypeOf(new(int64)).Elem(),
						reflect.TypeOf(new(string)).Elem(),
					},
					[]gobatis.Filter{})
				if err != nil {
					return gobatis.ErrForGenerateStmt(err, "generate OperationLogDao.List error")
				}
				stmt, err := gobatis.NewMapppedStatement(ctx, "OperationLogDao.List",
					gobatis.StatementTypeSelect,
					gobatis.ResultStruct,
					sqlStr)
				if err != nil {
					return err
				}
				ctx.Statements["OperationLogDao.List"] = stmt
			}
		}
		return nil
	})
}

func NewOperationLogDao(ref gobatis.SqlSession) OperationLogDao {
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
	return &OperationLogDaoImpl{session: ref}
}

type OperationLogDaoImpl struct {
	session gobatis.SqlSession
}

func (impl *OperationLogDaoImpl) Insert(ctx context.Context, ol *OperationLog) error {
	_, err := impl.session.Insert(ctx, "OperationLogDao.Insert",
		[]string{
			"ol",
		},
		[]interface{}{
			ol,
		},
		true)
	return err
}

func (impl *OperationLogDaoImpl) DeleteBy(ctx context.Context, createdAt TimeRange) error {
	_, err := impl.session.Delete(ctx, "OperationLogDao.DeleteBy",
		[]string{
			"createdAt",
		},
		[]interface{}{
			createdAt,
		})
	return err
}

func (impl *OperationLogDaoImpl) Count(ctx context.Context, userids []int64, successful sql.NullBool, typeList []string, contentLike string, createdAt TimeRange) (int64, error) {
	var instance int64
	var nullable gobatis.Nullable
	nullable.Value = &instance

	err := impl.session.SelectOne(ctx, "OperationLogDao.Count",
		[]string{
			"userids",
			"successful",
			"typeList",
			"contentLike",
			"createdAt",
		},
		[]interface{}{
			userids,
			successful,
			typeList,
			contentLike,
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

func (impl *OperationLogDaoImpl) List(ctx context.Context, userids []int64, successful sql.NullBool, typeList []string, contentLike string, createdAt TimeRange, offset int64, limit int64, sortBy string) ([]OperationLog, error) {
	var instances []OperationLog
	results := impl.session.Select(ctx, "OperationLogDao.List",
		[]string{
			"userids",
			"successful",
			"typeList",
			"contentLike",
			"createdAt",
			"offset",
			"limit",
			"sortBy",
		},
		[]interface{}{
			userids,
			successful,
			typeList,
			contentLike,
			createdAt,
			offset,
			limit,
			sortBy,
		})
	err := results.ScanSlice(&instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}
