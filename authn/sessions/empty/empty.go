package empty

import (
	"context"

	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo"
	"go.uber.org/fx"
)

func init() {
	moo.On(func() moo.Option {
		return fx.Provide(func() authn.Sessions {
			return &EmptySessions{}
		})
	})
}

type EmptySessions struct{}

func (sess EmptySessions) Count(ctx context.Context, username string, address string) (int, error) {
	return 0, nil	
}
func (sess EmptySessions) Login(ctx context.Context, userid interface{}, username, address string) (string, error) {
	return "", nil
}
func (sess EmptySessions) Logout(ctx context.Context, key string) error {
	return nil
}
func (sess EmptySessions) Get(ctx context.Context, id string) (*authn.SessionInfo, error) {
	return nil, nil
}
func (sess EmptySessions) Query(ctx context.Context, username string) ([]authn.SessionInfo, error) {
	return nil, nil
}
func (sess EmptySessions) All(ctx context.Context) ([]authn.SessionInfo, error) {
	return nil, nil
}
func (sess EmptySessions) UpdateNow(ctx context.Context, key string) error {
	return nil
}
func (sess EmptySessions) DeleteExpired(ctx context.Context) error {
	return nil
}
func (sess EmptySessions) IsOnlineExists(ctx context.Context, userid interface{}, username, loginAddress string) error {
	return nil
}
