package health

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"net/http"
	"time"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/components/pubsub"
)

var DefaultComponents = map[string]string{} 

type str struct {
	value atomic.Value
}

func (s *str) get() (string, bool) {
	o := s.value.Load()
	if o == nil {
		return "", false
	}
	v, ok := o.(*string)
	return *v, ok
}

func (s *str) getWithDefault(defaultValue string) string {
	o := s.value.Load()
	if o == nil {
		return defaultValue
	}
	v, ok := o.(*string)
	if !ok {
		return defaultValue
	}
	return *v
}

func (s *str) set(v string) {
	s.value.Store(&v)
}

type Keeplived struct {
	logger  log.Logger
	source  string
	startAt int64
	timeout int64

	components atomic.Value
}

type HealthStatus string

const (
	OK   HealthStatus = "ok"
	FAIL HealthStatus = "fail"
	STARTING HealthStatus = "starting"
)

type ComponentStatus struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	SessionID string       `json:"session_id"`
	Status    HealthStatus `json:"status"`
	Reason    string       `json:"reason,omitempty"`
	FirstAt   time.Time    `json:"first_at"`
	LastAt    time.Time    `json:"last_at"`
}

type component struct {
	title     str
	sessionID str
	firstAt   int64
	lastAt    int64
}


func (com *component) toHealthStatus(startAt, timeout int64) (int64, HealthStatus, string) {
	sec := atomic.LoadInt64(&com.lastAt)

	now := time.Now().Unix()
	if (now - startAt) < (10 * 60) {
		return sec, STARTING, ""
	}

	if (sec + timeout) < time.Now().Unix() {
		return sec, FAIL, "timeout"
	}
	return sec, OK, ""
}

func (com *component) toStatus(startAt, timeout int64) ComponentStatus {
	firstAt := atomic.LoadInt64(&com.firstAt)
	sec, status, reason := com.toHealthStatus(startAt, timeout)
	t := time.Unix(sec, 0)
	return ComponentStatus{
		Title:     com.title.getWithDefault(""),
		SessionID: com.sessionID.getWithDefault(""),
		Status:    status,
		Reason:    reason,
		FirstAt:   time.Unix(firstAt, 0),
		LastAt:    t,
	}
}

func (com *component) toMessage(source, id string, startAt, timeout int64) (bool, moo.Message) {
	sec, status, _ := com.toHealthStatus(startAt, timeout)
	if status == OK || status == STARTING {
		return false, moo.Message{}
	}
	t := time.Unix(sec, 0)
	return true, moo.Message{
		Source:    source,
		ID:        "keeplived." + id,
		Level:     moo.MsgError,
		Content:   com.title.getWithDefault("") + " 不活动了，最后活活时间 - " + t.Format(time.RFC3339),
		CreatedAt: t,
	}
}

func (hs *Keeplived) getComponents() map[string]*component {
	o := hs.components.Load()
	if o == nil {
		return nil
	}
	comps, _ := o.(map[string]*component)
	return comps
}

func (hs *Keeplived) Active(id, sessionID string, unixSec int64) *component {
	com, isNew := hs.addOrGet(id, "")
	if isNew || com.sessionID.getWithDefault("") != sessionID {
		com.sessionID.set(sessionID)
		atomic.StoreInt64(&com.firstAt, unixSec)
	}
	atomic.StoreInt64(&com.lastAt, unixSec)
	return com
}

func (hs *Keeplived) addOrGet(id, title string) (*component, bool) {
	var comps map[string]*component
	o := hs.components.Load()
	if o != nil {
		comps, _ = o.(map[string]*component)
	}

	if comps == nil {
		if title == "" {
			title = id
		}
		value := &component{}
		value.title.set(title)
		value.firstAt = value.lastAt
		hs.components.Store(map[string]*component{
			id: value,
		})
		return value, true
	}

	if value, exists := comps[id]; exists {
		if title != "" {
			value.title.set(title)
		}
		return value, false
	}
	newCopyed := map[string]*component{}
	for key, value := range comps {
		newCopyed[key] = value
	}
	if title == "" {
		title = id
	}
	value := &component{lastAt: time.Now().Unix()}
	value.title.set(title)
	value.firstAt = value.lastAt
	newCopyed[id] = value
	hs.components.Store(newCopyed)
	return value, true
}

func (hs *Keeplived) Remove(id string) *component {
	var comps map[string]*component
	o := hs.components.Load()
	if o != nil {
		comps, _ = o.(map[string]*component)
	}
	if comps == nil {
		return nil
	}

	old, exists := comps[id]
	if !exists {
		return nil
	}
	newCopyed := map[string]*component{}
	for key, value := range comps {
		if key == id {
			continue
		}
		newCopyed[key] = value
	}
	hs.components.Store(newCopyed)
	return old
}

func (hs *Keeplived) Reset() {
	unixSec := time.Now().Unix()
	hs.startAt = unixSec
	for _, comp := range hs.getComponents() {		
		atomic.StoreInt64(&comp.firstAt, unixSec)
		atomic.StoreInt64(&comp.lastAt, unixSec)
	}
}

func (hs *Keeplived) Get() []moo.Message {
	var messages []moo.Message
	for key, comp := range hs.getComponents() {
		ok, msg := comp.toMessage(hs.source, key, hs.startAt,  hs.timeout)
		if ok {
			messages = append(messages, msg)
		}
	}
	return messages
}

func (hs *Keeplived) GetAllStatus() []ComponentStatus {
	comps := hs.getComponents()
	var messages = make([]ComponentStatus, 0, len(comps))
	for key, comp := range comps {
		msg := comp.toStatus(hs.startAt, hs.timeout)
		msg.ID = key
		messages = append(messages, msg)
	}
	return messages
}

func (hs *Keeplived) OnEvent(ctx context.Context, topicName string, value interface{}) {
	evt, ok := value.(*api.SysKeepaliveEvent)
	if !ok {
		hs.logger.Warn("不可识的 action", log.Stringer("value", log.StringerFunc(func() string {
			return fmt.Sprintf("%T", value)
		})))
		return
	}

	switch evt.Action {
	case api.SysKeepaliveEventAdd:
		comp, _ := hs.addOrGet(evt.App, evt.Title)
		comp.sessionID.set(evt.SessionID)
		unixSec := time.Now().Unix()
		atomic.StoreInt64(&comp.firstAt, unixSec)
		atomic.StoreInt64(&comp.lastAt, unixSec)
	case api.SysKeepaliveEventRemove:
		hs.Remove(evt.App)
	case api.SysKeepaliveEventActive, "":
		hs.Active(evt.App, evt.SessionID, time.Now().Unix())
	default:
		hs.logger.Warn("不可识的 action", log.String("action", evt.Action), log.String("app", evt.App), log.String("title", evt.Title))
	}
}

func DrainToBus(ctx context.Context, logger log.Logger, topicName string, bus *moo.Bus, ch <-chan *pubsub.Message) {
	pubsub.DrainToBus(ctx, logger, topicName, bus, ch, func(ctx context.Context, msg *pubsub.Message) (interface{}, error) {
		var evt api.SysKeepaliveEvent
		err := json.Unmarshal(msg.Payload, &evt)
		if evt.ID == "" {
			evt.ID = msg.UUID
		}
		return &evt, err
	})
}

func NewKeeplived(env *moo.Environment, logger log.Logger) *Keeplived {
	keeplived := &Keeplived{
			logger:  logger,
			source:  "health.keeplived.commponents",
			startAt: time.Now().Unix(),
			timeout: env.Config.Int64WithDefault(api.CfgHealthKeepliveTimeout, 60*5),
		}
	for key, value := range DefaultComponents {
	 	keeplived.addOrGet(key, value)
	}
	return keeplived
}

const keepliveTopic = "keeplive"

func Register(publisher pubsub.Publisher, appid, title string) error {
	message := pubsub.NewMessage(appid, &api.SysKeepaliveEvent{
		App:    appid,
		Title:  title,
		Action: api.SysKeepaliveEventAdd,
	})
	return publisher.Publish(keepliveTopic, message)
}

func Active(publisher pubsub.Publisher, appid, title string) error {
	message := pubsub.NewMessage(appid, &api.SysKeepaliveEvent{
		App:    appid,
		Title:  title,
		Action: api.SysKeepaliveEventActive,
	})
	return publisher.Publish(keepliveTopic, message)
}

func StartActive(ctx context.Context, logger log.Logger, pubsubURL, appid, title string, interval time.Duration) error {
	if interval == 0 {
		interval = 2 * time.Minute
	}

	publisher, err := pubsub.NewHTTPPublisher(pubsubURL, logger)
	if err != nil {
		return errors.Wrap(err, "启动心跳 '"+appid+"' 失败")
	}

	err = Register(publisher, appid, title)
	if err != nil {
		return errors.Wrap(err, "注册心跳 '"+appid+"' 失败")
	}

	var activeTimeout func()

	activeTimeout = func() {
		defer time.AfterFunc(interval, activeTimeout)

		err := Active(publisher, appid, title)
		if err != nil {
			logger.Error("active health fail", log.Error(err))
			return
		}
	}

	time.AfterFunc(interval, activeTimeout)
	return nil
}

func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return moo.Invoke(func(env *moo.Environment, logger log.Logger) *Keeplived {
			logger = logger.Named("health.keeplived.commponents")
			keeplived := NewKeeplived(env, logger)
			return keeplived
		})
	})

	moo.On(func(*moo.Environment) moo.Option {
		return moo.Invoke(func(env *moo.Environment, lifecycle moo.Lifecycle, keeplived *Keeplived, bus *moo.Bus, 
			subscriber pubsub.Subscriber, httpSrv *moo.HTTPServer, msgList *moo.MessageList, logger log.Logger) error {
			logger = logger.Named("health.keeplived.commponents")
			bus.RegisterTopics(api.BusSysKeepaliveStatus)

			ctx := context.Background()
			ch, err := subscriber.Subscribe(ctx, keepliveTopic)
			if err != nil {
				return err
			}
			go DrainToBus(ctx, logger, api.BusSysKeepaliveStatus, bus, ch)

			lifecycle.Append(moo.Hook{
				OnStart: func(context.Context) error {
					msgList.SetupProvider(keeplived)
					bus.Register("keeplive_listener", &moo.BusHandler{
						Matcher: api.BusSysKeepaliveStatus,
						Handle:  keeplived.OnEvent,
					})
					return nil
				},
				OnStop: func(context.Context) error {
					bus.Unregister("keeplive_listener")
					return nil
				},
			})

			httpSrv.FastRoute(false, "components", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/reset") {
					keeplived.Reset()
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(keeplived.GetAllStatus())
			}))
			return nil
		})
	})
}
