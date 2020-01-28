package main

import (
	"fmt"

	"github.com/runner-mei/moo"
)

func main() {
	err := moo.Run(&moo.Arguments{
		//Defaults    []string
		//Customs     []string
		CommandArgs: []string{
			"http-address=:9999",
			"https-address=:9993",
		},
	})
	if err != nil {
		fmt.Println(err)
	}
}
