// +build !file

package moo_tests

import (
	"fmt"
	"os"

	_ "github.com/runner-mei/moo/auth/users/db"
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

func (a *AppTest) init() {
	a.Args.CommandArgs = append(a.Args.CommandArgs,
		"test.clean_data=true",
		"users.init_database=true",
		"moo.db.host="+getenv("MOO_MODEL_DB_HOST", "MOO_DB_HOST", "127.0.0.1"),
		"moo.db.port="+getenv("MOO_MODEL_DB_PORT", "MOO_DB_PORT", "5432"),
		"moo.db.db_name="+getenv("MOO_MODEL_DB_NAME", "moo_test"),
		"moo.db.username="+getenv("MOO_MODEL_DB_USER", "moo"),
		"moo.db.password="+getenv("MOO_MODEL_DB_PASSWORD", "moo12345678"),
		"moo.data.db.host="+getenv("MOO_DATA_DB_HOST", "MOO_DB_HOST", "127.0.0.1"),
		"moo.data.db.port="+getenv("MOO_DATA_DB_PORT", "MOO_DB_PORT", "5432"),
		"moo.data.db.db_name="+getenv("MOO_DATA_DB_NAME", "moo_data_test"),
		"moo.data.db.username="+getenv("MOO_DATA_DB_USER", "moo"),
		"moo.data.db.password="+getenv("MOO_DATA_DB_PASSWORD", "moo12345678"))

	fmt.Println(a.Args.CommandArgs)
}
