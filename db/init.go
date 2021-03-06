package db

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
)

func initDb(env *moo.Environment, logger log.Logger, db *sql.DB) error {
	args := GetTableNames()
	for k := range args {
		newName := env.Config.StringWithDefault(api.CfgTablenamePrefix+k, "")
		if newName == "" {
			continue
		}
		args[k] = newName
	}

	if env.Config.BoolWithDefault(api.CfgTestCleanDatabase, false) {
		_, err := db.Exec(CleanSQL(env, args))
		if err != nil {
			return errors.New("清理用户相关的表失败: " + err.Error())
		}
		logger.Info("清理用户相关的表成功")
	} else if env.Config.BoolWithDefault(api.CfgTestCleanData, false) {
		_, err := db.Exec(CleanDataSQL(env, args))
		if err != nil {
			return errors.New("清理用户相关的数据失败: " + err.Error())
		}
		logger.Info("清理用户相关的数据成功")
	}

	if env.Config.BoolWithDefault(api.CfgUserInitDatabase, false) {
		_, err := db.Exec(InitSQL(env, args))
		if err != nil {
			return errors.New("初始化用户相关的表失败: " + err.Error())
		}
		logger.Info("初始化用户相关的表成功")
	} else {
		logger.Info("跳过用户相关表的初始化")
	}
	return nil
}

func GetTableNames() map[string]string {
	return map[string]string{
		"moo_operation_logs":       "moo_operation_logs",
		"moo_online_users":         "moo_online_users",
		"moo_users_and_roles":      "moo_users_and_roles",
		"moo_users":                "moo_users",
		"moo_roles":                "moo_roles",
		"moo_usergroups":           "moo_usergroups",
		"moo_users_and_usergroups": "moo_users_and_usergroups",
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
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Invoke(func(env *moo.Environment, logger log.Logger, db InModelDB) error {
			return initDb(env, logger, db.DB)
		})
	})
}

var CleanDataSQL = func(env *moo.Environment, args map[string]string) string {
return ReplaceTableName(`
-- users v1

DELETE FROM moo_operation_logs;
DELETE FROM moo_online_users;
DELETE FROM moo_users_and_roles;
DELETE FROM moo_users_and_usergroups;
DELETE FROM moo_user_profiles;
DELETE FROM moo_users;
DELETE FROM moo_roles;
DELETE FROM moo_usergroups;
`, args)
}

var CleanSQL = func(env *moo.Environment, args map[string]string) string {
return ReplaceTableName(`
-- users v1

DROP TABLE IF EXISTS moo_operation_logs CASCADE;
DROP TABLE IF EXISTS moo_online_users CASCADE;
DROP TABLE IF EXISTS moo_users_and_roles CASCADE;
DROP TABLE IF EXISTS moo_users_and_usergroups CASCADE;
DROP TABLE IF EXISTS moo_user_profiles CASCADE;
DROP TABLE IF EXISTS moo_users CASCADE;
DROP TABLE IF EXISTS moo_roles CASCADE;
DROP TABLE IF EXISTS moo_usergroups CASCADE;
`, args)
}

var InitSQL = func(env *moo.Environment, args map[string]string) string {
	txt := env.Config.StringWithDefault("moo.init_sql_text", "")
	if txt != "" {
		return txt
	}
return ReplaceTableName(`
-- users v1

CREATE TABLE IF NOT EXISTS moo_users (
	id          bigserial   PRIMARY KEY,
	name        character varying(100) NOT NULL UNIQUE,
	nickname    character varying(100) NOT NULL UNIQUE,
	password    character varying(500) ,
	description text,
	signature   text,
	can_login   boolean,
	disabled    boolean,
	is_default  boolean,
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
		id          bigserial PRIMARY KEY,
    name        varchar(100) NOT NULL UNIQUE,
    type        integer,
		is_default  boolean,
		description text,
		created_at  timestamp,
		updated_at  timestamp
);

CREATE TABLE IF NOT EXISTS moo_users_and_roles (
		user_id   bigint REFERENCES moo_users ON DELETE CASCADE,
		role_id   bigint REFERENCES moo_roles ON DELETE CASCADE,
		UNIQUE(user_id,role_id)
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

CREATE TABLE IF NOT EXISTS moo_usergroups
(
		id          bigserial PRIMARY KEY,
    name        varchar(100) NOT NULL UNIQUE,
    description text,
    parent_id   integer REFERENCES moo_usergroups (id) MATCH SIMPLE
								        ON UPDATE NO ACTION
								        ON DELETE CASCADE,
    created_at  timestamp with time zone,
    updated_at  timestamp with time zone,
    disabled    boolean
);

CREATE TABLE IF NOT EXISTS moo_users_and_usergroups
(
    user_id  integer REFERENCES moo_users (id) MATCH SIMPLE
							        ON UPDATE NO ACTION
							        ON DELETE CASCADE,
    group_id integer REFERENCES moo_usergroups (id) MATCH SIMPLE
							        ON UPDATE NO ACTION
							        ON DELETE CASCADE,
    role_id integer REFERENCES moo_roles (id) MATCH SIMPLE
							        ON UPDATE NO ACTION
							        ON DELETE CASCADE,
    UNIQUE (user_id, group_id, role_id)
);

CREATE TABLE IF NOT EXISTS moo_operation_logs (
	id           BIGSERIAL PRIMARY KEY,
	userid       bigint REFERENCES moo_users ON DELETE SET NULL,
	username     varchar(100),
	type         varchar(100),
	successful   boolean,
	content      text,
	attributes   jsonb,
	created_at   timestamp without time zone
);

-- +statementBegin
CREATE OR REPLACE FUNCTION add_admin_user() RETURNS VOID AS $$ 
BEGIN 
	IF NOT EXISTS (SELECT * FROM moo_users WHERE name='` + api.UserAdmin + `') THEN
		INSERT INTO moo_users (name, nickname, password, can_login, created_at, updated_at)
								VALUES('` + api.UserAdmin + `', '` + api.UserAdmin + `', 'Admin', true, now(), now());
	END IF;
END; 
$$ language 'plpgsql'; 
-- +statementEnd
SELECT add_admin_user();
DROP FUNCTION add_admin_user();

-- +statementBegin
CREATE OR REPLACE FUNCTION add_bgopuser_user() RETURNS VOID AS $$ 
BEGIN 
	IF NOT EXISTS (SELECT * FROM moo_users WHERE name='` + api.UserBgOperator + `') THEN
		INSERT INTO moo_users (name, nickname, password, can_login, created_at, updated_at)
								VALUES('` + api.UserBgOperator + `', '` + api.UserBgOperator + `', 'Admin', true, now(), now());
	END IF;
END; 
$$ language 'plpgsql'; 
-- +statementEnd
SELECT add_bgopuser_user();
DROP FUNCTION add_bgopuser_user();
`, args)
}