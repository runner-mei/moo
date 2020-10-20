// +build !file

package moo_tests

import (
	"fmt"
	"os"

	_ "github.com/runner-mei/moo/operation_logs"
	_ "github.com/runner-mei/moo/users"
)

func getenv(name string, args ...string) string {
	s := os.Getenv(name)
	if s == "" {
		if len(args) == 0 {
			return ""
		}
		if len(args) == 1 {
			return args[0]
		}

		return getenv(args[0], args[1:]...)
	}
	return s
}

func (a *TestApp) init() {
	a.Args.CommandArgs = append(a.Args.CommandArgs,
		"moo.log.level=debug",
		"test.clean_data=true",
		"test.clean_database=true",
		"users.init_database=true",

		//  NOTE: 下面的测试数据库设置要注意，它可能会被下列变量覆盖
		//  moo.test.models.db_drv, moo.test.models.db_url, moo.test.data.db_drv,moo.test.data.db_url
		//
		//  或者也会因为 ns 改名导致无效

		"moo.db.host="+getenv("MOO_MODEL_DB_HOST", "MOO_DB_HOST", "127.0.0.1"),
		"moo.db.port="+getenv("MOO_MODEL_DB_PORT", "MOO_DB_PORT", "5432"),
		"moo.db.dbname="+getenv("MOO_MODEL_DB_NAME", "moo_test"),
		"moo.db.username="+getenv("MOO_MODEL_DB_USER", "moo"),
		"moo.db.password="+getenv("MOO_MODEL_DB_PASSWORD", "moo12345678"),
		"moo.data.db.host="+getenv("MOO_DATA_DB_HOST", "MOO_DB_HOST", "127.0.0.1"),
		"moo.data.db.port="+getenv("MOO_DATA_DB_PORT", "MOO_DB_PORT", "5432"),
		"moo.data.db.dbname="+getenv("MOO_DATA_DB_NAME", "moo_data_test"),
		"moo.data.db.username="+getenv("MOO_DATA_DB_USER", "moo"),
		"moo.data.db.password="+getenv("MOO_DATA_DB_PASSWORD", "moo12345678"))

	fmt.Println(a.Args.CommandArgs)
}
