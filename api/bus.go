package api

const (
	BusSysKeepaliveStatus = "sys.keepalive"
)

type SysKeepaliveEvent struct {
	ID  string `json:"id,omitempty"`
	App string `json:"app,omitempty"`
}

// func SendKeepalive(ctx context.Context, id, app string) error {

// }
