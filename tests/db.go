// +build !file

package tests

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
		"models.db.host="+getenv("MOO_MODEL_DB_HOST", "MOO_DB_HOST", "127.0.0.1"),
		"models.db.port="+getenv("MOO_MODEL_DB_PORT", "MOO_DB_PORT", "5432"),
		"models.db.db_name="+getenv("MOO_MODEL_DB_NAME", "moo_test"),
		"models.db.username="+getenv("MOO_MODEL_DB_USER", "moo"),
		"models.db.password="+getenv("MOO_MODEL_DB_PASSWORD", "moo12345678"),
		"data.db.host="+getenv("MOO_DATA_DB_HOST", "MOO_DB_HOST", "127.0.0.1"),
		"data.db.port="+getenv("MOO_DATA_DB_PORT", "MOO_DB_PORT", "5432"),
		"data.db.db_name="+getenv("MOO_DATA_DB_NAME", "moo_data_test"),
		"data.db.username="+getenv("MOO_DATA_DB_USER", "moo"),
		"data.db.password="+getenv("MOO_DATA_DB_PASSWORD", "moo12345678"))

	fmt.Println(a.Args.CommandArgs)
}
