// +build !file

package tests

import (
	_ "github.com/runner-mei/moo/auth/users/db"
)

func (a *AppTest) init() {
	a.Args.CommandArgs = append(a.Args.CommandArgs,
		"test.clean_data=true",
		"users.init_database=true",
		"models.db.db_name=moo_test",
		"models.db.username=moo",
		"models.db.password=moo12345678",
		"data.db.db_name=moo_data_test",
		"data.db.username=moo",
		"data.db.password=moo12345678")
}
