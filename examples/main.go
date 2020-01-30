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
			"models.db.db_name=moo_test",
			"models.db.username=moo",
			"models.db.password=moo12345678",
			"data.db.db_name=moo_data_test",
			"data.db.username=moo",
			"data.db.password=moo12345678",
		},
	})
	if err != nil {
		fmt.Println(err)
	}
}
