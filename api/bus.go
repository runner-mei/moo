package api

import "context"

const (
	BusSysKeepaliveStatus = "sys.keepalive"

	SysKeepaliveEventAdd    = "add"
	SysKeepaliveEventRemove = "remove"
	SysKeepaliveEventActive = "active"

	BusMessageEventCreated = "moo.messages.created"
	BusMessageEventUpdated = "moo.messages.updated"
	BusMessageEventDeleted = "moo.messages.deleted"

	EventAlerts = "event.alerts"
)

type SysKeepaliveEvent struct {
	ID     string `json:"id,omitempty"`
	App    string `json:"app,omitempty"`
	Title  string `json:"title,omitempty"`
	Action string `json:"action,omitempty"`
}

type Sender interface {
	Send(ctx context.Context, toppic, source string, payload interface{}) error
}
