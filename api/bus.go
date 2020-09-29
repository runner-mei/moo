package api

const (
	BusSysKeepaliveStatus = "sys.keepalive"

	SysKeepaliveEventAdd    = "add"
	SysKeepaliveEventRemove = "remove"
	SysKeepaliveEventActive = "active"
)

type SysKeepaliveEvent struct {
	ID     string `json:"id,omitempty"`
	App    string `json:"app,omitempty"`
	Title  string `json:"title,omitempty"`
	Action string `json:"action,omitempty"`
}

// func SendKeepalive(ctx context.Context, id, app string) error {

// }
