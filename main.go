package moo

import (
	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/log"
	"github.com/runner-mei/loong"
	"go.uber.org/fx"
)

func init() {
	var _ *gobatis.SessionFactory = &gobatis.SessionFactory{}
	var _ fx.Printer = nil
	var _ log.Logger = nil
	var _ = loong.New()
}
