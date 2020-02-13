package main

import (
	"fmt"

	"github.com/runner-mei/moo"
	_ "github.com/runner-mei/moo/auth/sessions/inmem"
)

func main() {
	err := moo.Run(&moo.Arguments{
		//Defaults    []string
		//Customs     []string
		CommandArgs: []string{
			"http-address=:9999",
			"https-address=:9993",
			"users.init_database=true",
			"moo.db.db_name=moo_test",
			"moo.db.username=moo",
			"moo.db.password=moo12345678",
			"moo.data.db.db_name=moo_data_test",
			"moo.data.db.username=moo",
			"moo.data.db.password=moo12345678",
		},
	})
	if err != nil {
		fmt.Println(err)
	}
}
