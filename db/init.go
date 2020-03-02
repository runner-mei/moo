package db

import (
	"database/sql"
	"strings"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
)

func initDb(env *moo.Environment, logger log.Logger, db *sql.DB) error {
	args := GetTableNames()
	for k := range args {
		newName := env.Config.StringWithDefault("moo.tablename." + k, "")
		if newName == "" {
			continue
		}
		args[k] = newName
	}

	if env.Config.BoolWithDefault("test.clean_database", false) {
		_, err := db.Exec(ReplaceTableName(CleanSQL, args))
		if err != nil {
			return err
		}
		logger.Info("清理用户相关的表成功")
	} else if env.Config.BoolWithDefault("test.clean_data", false) {
		_, err := db.Exec(ReplaceTableName(CleanDataSQL, args))
		if err != nil {
			return err
		}
		logger.Info("清理用户相关的数据成功")
	}

	if env.Config.BoolWithDefault("users.init_database", false) {
		_, err := db.Exec(ReplaceTableName(InitSQL, args))
		if err != nil {
			return err
		}
		logger.Info("初始化用户相关的表成功")
	} else {
		logger.Info("跳过用户相关表的初始化")
	}
	return nil
}

func GetTableNames() map[string]string {
	return map[string]string {
		"moo_operation_logs" : "moo_operation_logs",
		"moo_online_users": "moo_online_users",
		"moo_permission_and_roles":"moo_permission_and_roles", 
		"moo_users_and_roles": "moo_users_and_roles",
		"moo_users": "moo_users",
		"moo_roles":"moo_roles",
	}
}

func ReplaceTableName(sqlStr string, args map[string]string) string {
	for k, v := range args {
		if k == v {
			continue
		}
		if k == "" || v == "" {
			continue
		}

		sqlStr = strings.ReplaceAll(sqlStr, k, v)
	}
	return sqlStr
}


func init() {
	moo.On(func() moo.Option {
		return moo.Invoke(func(env *moo.Environment, logger log.Logger, db InModelDB) error {
			return initDb(env, logger, db.DB)
		})
	})
}

var CleanDataSQL = `
-- users v1

DELETE FROM moo_operation_logs;
DELETE FROM moo_online_users;
DELETE FROM moo_permission_and_roles;
DELETE FROM moo_users_and_roles;
DELETE FROM moo_users;
DELETE FROM moo_roles;
`

var CleanSQL = `
-- users v1

DROP TABLE IF EXISTS moo_operation_logs CASCADE;
DROP TABLE IF EXISTS moo_online_users CASCADE;
DROP TABLE IF EXISTS moo_permission_and_roles CASCADE;
DROP TABLE IF EXISTS moo_users_and_roles CASCADE;
DROP TABLE IF EXISTS moo_users CASCADE;
DROP TABLE IF EXISTS moo_roles CASCADE;
`

var InitSQL = `
-- users v1

CREATE TABLE IF NOT EXISTS moo_users (
	id          bigserial   PRIMARY KEY,
	name        character varying(100) NOT NULL UNIQUE,
	nickname    character varying(100) NOT NULL UNIQUE,
	password    character varying(500) ,
	description text,
	signature   text,
	disabled    bool,
	attributes  jsonb,
	source      character varying(50),
	locked_at   timestamp WITH TIME ZONE,
	created_at  timestamp WITH TIME ZONE,
	updated_at  timestamp WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS moo_user_profiles (
		id          bigserial PRIMARY KEY,
		user_id     bigint REFERENCES moo_users ON DELETE CASCADE,
		name        varchar(100) NOT NULL,
		value       text,
		created_at  timestamp,
		updated_at  timestamp,

		UNIQUE(user_id,name)
);

CREATE TABLE IF NOT EXISTS moo_roles (
		id      bigserial PRIMARY KEY,
		name    character varying(100) NOT NULL UNIQUE,
		description text,
		created_at  timestamp,
		updated_at  timestamp
);

CREATE TABLE IF NOT EXISTS moo_users_and_roles (
		user_id   bigint REFERENCES moo_users ON DELETE CASCADE,
		role_id   bigint REFERENCES moo_roles ON DELETE CASCADE,
		UNIQUE(user_id,role_id)
);

CREATE TABLE IF NOT EXISTS moo_permission_and_roles (
		role_id             bigint REFERENCES moo_roles ON DELETE CASCADE,
		permission          varchar(100),
		UNIQUE(role_id,permission)
);

CREATE TABLE IF NOT EXISTS moo_online_users (
		user_id     bigint REFERENCES moo_users ON DELETE CASCADE,
		address     inet,
		uuid        varchar(50),
		created_at  timestamp,
		updated_at  timestamp,

		PRIMARY KEY(user_id, address),
		UNIQUE(uuid)
);

-- +statementBegin
CREATE OR REPLACE FUNCTION add_admin_user() RETURNS VOID AS $$ 
BEGIN 
	IF NOT EXISTS (SELECT * FROM moo_users WHERE name='`+ api.UserAdmin +`') THEN
		INSERT INTO moo_users (name, nickname, password, created_at, updated_at)
								VALUES('`+ api.UserAdmin +`', '`+ api.UserAdmin +`', 'Admin', now(), now());
	END IF;
END; 
$$ language 'plpgsql'; 
-- +statementEnd
SELECT add_admin_user();
DROP FUNCTION add_admin_user();

-- +statementBegin
CREATE OR REPLACE FUNCTION add_bgopuser_user() RETURNS VOID AS $$ 
BEGIN 
	IF NOT EXISTS (SELECT * FROM moo_users WHERE name='`+ api.UserBgOperator +`') THEN
		INSERT INTO moo_users (name, nickname, password, disabled, created_at, updated_at)
								VALUES('`+ api.UserBgOperator +`', '`+ api.UserBgOperator +`', 'Admin', true, now(), now());
	END IF;
END; 
$$ language 'plpgsql'; 
-- +statementEnd
SELECT add_bgopuser_user();
DROP FUNCTION add_bgopuser_user();

CREATE TABLE IF NOT EXISTS moo_operation_logs (
	id           BIGSERIAL PRIMARY KEY,
	userid       bigint REFERENCES moo_users ON DELETE CASCADE,
	username     varchar(100),
	type         varchar(100),
	successful   boolean,
	content      text,
	attributes   jsonb,
	created_at   timestamp without time zone
);
`
