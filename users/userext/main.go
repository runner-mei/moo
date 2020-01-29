package userext

import (
	"context"
	"fmt"
	"net/http"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/loong"
	"github.com/runner-mei/moo/users"
)

func ContextWithUser(ctx context.Context, u users.ReadCurrentUserFunc) context.Context {
	return loong.ContextWithUser(ctx, u)
}

func UserFromContext(ctx context.Context) users.ReadCurrentUserFunc {
	o := loong.UserFromContext(ctx)
	if o == nil {
		return nil
	}
	f, _ := o.(users.ReadCurrentUserFunc)
	return f
}

func ReadUserFromContext(ctx context.Context) (users.User, error) {
	o := loong.UserFromContext(ctx)
	if o == nil {
		return nil, errors.NewError(http.StatusUnauthorized, "user isnot exists because session is unauthorized")
	}
	f, ok := o.(users.ReadCurrentUserFunc)
	if ok {
		return f(ctx)
	}
	u, ok := o.(users.User)
	if ok {
		return u, nil
	}
	return nil, errors.NewError(http.StatusInternalServerError, fmt.Sprintf("user is unknown type - %T", o))
}
