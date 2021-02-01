package menus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
)

var EventName = "menus.changed"

// Layout 菜单布避生成器
type Layout interface {
	Stats() interface{}

	Generate(map[string][]Menu) ([]Menu, error)
}

// sendEvent func(hub.Message),
// sendEvent(hub.CreateDataMessage([]byte(strconv.Itoa(len(menuList)))))

func NewWeaver(logger log.Logger,
	env *moo.Environment,
	sendEvent func(description string),
	layouts map[string]Layout,
) (Weaver, error) {
	weaver := &menuWeaver{Logger: logger,
		env:       env,
		sendEvent: sendEvent,
		layouts:   layouts,
	}
	if err := weaver.Init(); err != nil {
		return nil, err
	}

	return weaver, nil
}

type menuWeaver struct {
	Logger    log.Logger
	env       *moo.Environment
	sendEvent func(descr string)
	layouts   map[string]Layout

	mu               sync.RWMutex
	byApplications   map[string][]Menu
	menuListByLayout map[string][]Menu
}

func (weaver *menuWeaver) Stats() interface{} {
	weaver.mu.RLock()
	defer weaver.mu.RUnlock()
	layouts := map[string]interface{}{}
	for k, v := range weaver.layouts {
		layouts[k] = v.Stats()
	}

	return map[string]interface{}{
		"applications": weaver.byApplications,
		"layout":       layouts,
		"menuList":     weaver.menuListByLayout,
	}
}

// func (weaver *menuWeaver) generate() ([]toolbox.Menu, error) {
// 	return weaver.layout.Generate(weaver.byApplications)
// }

func (weaver *menuWeaver) Init() error {
	byApplications := map[string][]Menu{}
	filename := weaver.env.Fs.FromTMP("app_menus.json")
	in, err := os.Open(filename)
	if err != nil {
		weaver.Logger.Warn("LoadFromDB", log.Error(err))
	} else {
		defer in.Close()

		err = json.NewDecoder(in).Decode(&byApplications)
		if err != nil {
			weaver.Logger.Warn("LoadFromDB", log.Error(err))
		}
	}

	weaver.mu.Lock()
	defer weaver.mu.Unlock()
	weaver.byApplications = byApplications
	weaver.menuListByLayout = nil
	return nil
}

func (weaver *menuWeaver) Update(app string, menuList []Menu) error {
	weaver.mu.RLock()
	oldList := weaver.byApplications[app]
	weaver.mu.RUnlock()

	if len(menuList) == 0 && len(oldList) == 0 {
		return nil
	}
	if IsSameMenuArray(menuList, oldList) {
		return nil
	}

	var err error
	weaver.mu.Lock()
	defer weaver.mu.Unlock()
	if weaver.byApplications == nil {
		weaver.byApplications = map[string][]Menu{}
	}
	weaver.byApplications[app] = menuList
	weaver.menuListByLayout = nil
	weaver.sendEvent(strconv.Itoa(len(menuList)))

	filename := weaver.env.Fs.FromTMP("app_menus.json")
	if err = os.MkdirAll(filepath.Dir(filename), 0777); err != nil {
		weaver.Logger.Warn("update menu list in app "+app+" to file fail", log.Error(err))
		return nil
	}

	out, err := os.Create(filename)
	if err != nil {
		weaver.Logger.Warn("update menu list in app "+app+" to file fail", log.Error(err))
		return nil
	}
	defer out.Close()

	err = json.NewEncoder(out).Encode(weaver.byApplications)
	if err != nil {
		weaver.Logger.Warn("update menu list in app "+app+" to file fail", log.Error(err))
	}
	return nil
}

func (weaver *menuWeaver) Generate(app string) ([]Menu, error) {
	menuListByLayout, err := weaver.GenerateAll()
	if err != nil {
		return nil, err
	}
	if len(menuListByLayout) == 0 {
		return nil, fmt.Errorf("菜单组件 '%s' 没有找到", app)
	}
	results, ok := menuListByLayout[app]
	if !ok {
		return nil, fmt.Errorf("菜单组件 '%s' 没有找到", app)
	}
	return results, nil
}

func (weaver *menuWeaver) GenerateAll() (map[string][]Menu, error) {
	weaver.mu.RLock()
	isRead := true
	defer func() {
		if isRead {
			weaver.mu.RUnlock()
		} else {
			weaver.mu.Unlock()
		}
	}()
	if len(weaver.menuListByLayout) > 0 {
		return weaver.menuListByLayout, nil
	}

	weaver.mu.RUnlock()
	weaver.mu.Lock()
	isRead = false

	if weaver.menuListByLayout == nil {
		weaver.menuListByLayout = map[string][]Menu{}
	}

	for key, cfg := range weaver.layouts {
		menuList, err := cfg.Generate(weaver.byApplications)
		if err != nil {
			weaver.Logger.Error("generate", log.String("layout", key), log.Error(err))
			return nil, err
		}
		weaver.menuListByLayout[key] = menuList
	}
	return weaver.menuListByLayout, nil
}

func isSame(allItems, subset []Menu) bool {
	return IsSameMenuArray(allItems, subset)
}
