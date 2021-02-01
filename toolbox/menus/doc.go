package menus

import (
	"errors"
)

//go:generate genny -pkg=menus -in=../../builds/weaver/client.go -out=client-gen.go gen "Something=MenuArray ValueType=[]toolbox.Menu ApartResultType=map[string][]toolbox.Menu"
//go:generate genny -pkg=menus -in=../../builds/generic/cached_value.go -out=cached-gen.go gen "ValueType=map[string][]toolbox.Menu"

// ErrAlreadyClosed  server is closed
var ErrAlreadyClosed = errors.New("server is closed")
