package cas

import (
	"context"
	"database/sql"

	"github.com/runner-mei/goutils/util"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/moo"
)

type UserSyncer interface {
	Read(ctx context.Context, id string) (string, map[string]string, error)
}

func CreateUserSyncer(env *moo.Environment, conn *sql.DB) (UserSyncer, error) {
	find := env.Config.StringWithDefault("users.sync.db.find", "")
	return &userSyncer{conn: conn, find: find}, nil
}

type userSyncer struct {
	conn *sql.DB
	find string
}

func (syncer *userSyncer) Read(ctx context.Context, id string) (string, map[string]string, error) {
	rows, err := syncer.conn.QueryContext(ctx, syncer.find, id)
	if err != nil {
		return "", nil, err
	}
	defer util.CloseWith(rows)

	if !rows.Next() {
		return "", nil, sql.ErrNoRows
	}

	columns, err := rows.Columns()
	if err != nil {
		return "", nil, err
	}

	var values = make([]string, len(columns))
	var args = make([]interface{}, len(columns))
	for idx := range values {
		args[idx] = &values[idx]
	}

	err = rows.Scan(args...)
	if err != nil {
		return "", nil, err
	}

	if len(values) < 3 {
		return "", nil, errors.New("缺少必要的字段")
	}

	var results = map[string]string{}
	for idx := range columns[1:] {
		results[columns[1+idx]] = values[1+idx]
	}
	return values[0], results, nil
}
