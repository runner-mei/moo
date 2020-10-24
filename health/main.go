package health

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/pubsub"
)

type Keeplived struct {
	logger  log.Logger
	source  string
	timeout int64

	components atomic.Value
}

type component struct {
	title   string
	firstAt int64
	lastAt  int64
}

func (com *component) toMessage(source, id string, timeout int64) (bool, moo.Message) {
	sec := atomic.LoadInt64(&com.lastAt)
	if (sec + timeout) > time.Now().Unix() {
		return false, moo.Message{}
	}
	t := time.Unix(sec, 0)
	return true, moo.Message{
		Source:    source,
		ID:        "keeplived." + id,
		Level:     moo.MsgError,
		Content:   com.title + " 不活动了，最后活活时间 - " + t.Format(time.RFC3339),
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

func (hs *Keeplived) activeComponent(id string, unixSec int64) *component {
	com := hs.addComponent(id, "")
	atomic.StoreInt64(&com.lastAt, unixSec)
	return com
}

func (hs *Keeplived) addComponent(id, title string) *component {
	var comps map[string]*component
	o := hs.components.Load()
	if o != nil {
		comps, _ = o.(map[string]*component)
	}

	if comps == nil {
		if title == "" {
			title = id
		}
		value := &component{title: title, lastAt: time.Now().Unix()}
		value.firstAt = value.lastAt
		hs.components.Store(map[string]*component{
			id: value,
		})
		return value
	}

	if value, exists := comps[id]; exists {
		if title != "" {
			value.title = title
		}
		return value
	}
	newCopyed := map[string]*component{}
	for key, value := range comps {
		newCopyed[key] = value
	}
	if title == "" {
		title = id
	}
	value := &component{title: title, lastAt: time.Now().Unix()}
	value.firstAt = value.lastAt
	newCopyed[id] = value
	hs.components.Store(newCopyed)
	return value
}

func (hs *Keeplived) removeComponent(id string) *component {
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

func (hs *Keeplived) Get() []moo.Message {
	var messages []moo.Message
	for key, comp := range hs.getComponents() {
		ok, msg := comp.toMessage(hs.source, key, hs.timeout)
		if ok {
			messages = append(messages, msg)
		}
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
		hs.addComponent(evt.App, evt.Title)
	case api.SysKeepaliveEventRemove:
		hs.removeComponent(evt.App)
	case api.SysKeepaliveEventActive, "":
		hs.activeComponent(evt.App, time.Now().Unix())
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
	return &Keeplived{
		logger:  logger,
		source:  "health.keeplived.commponents",
		timeout: env.Config.Int64WithDefault(api.CfgHealthKeepliveTimeout, 60*5),
	}
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
		return moo.Invoke(func(env *moo.Environment, lifecycle moo.Lifecycle, bus *moo.Bus, subscriber pubsub.Subscriber, msgList *moo.MessageList, logger log.Logger) error {
			logger = logger.Named("health.keeplived.commponents")
			bus.RegisterTopics(api.BusSysKeepaliveStatus)

			components := NewKeeplived(env, logger)

			ctx := context.Background()
			ch, err := subscriber.Subscribe(ctx, keepliveTopic)
			if err != nil {
				return err
			}
			go DrainToBus(ctx, logger, api.BusSysKeepaliveStatus, bus, ch)

			lifecycle.Append(moo.Hook{
				OnStart: func(context.Context) error {
					msgList.SetupProvider(components)
					bus.Register("keeplive_listener", &moo.BusHandler{
						Matcher: api.BusSysKeepaliveStatus,
						Handle:  components.OnEvent,
					})
					return nil
				},
				OnStop: func(context.Context) error {
					bus.Unregister("keeplive_listener")
					return nil
				},
			})
			return nil
		})
	})
}
