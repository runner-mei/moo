package menus

import (
	"os"

	"github.com/runner-mei/log"
)

var hasOrphan = os.Getenv("is_devel_env") == "true"

// Simple 简单布局器
var _ Layout = &simpleLayout{}

func NewSimple(logger log.Logger, layout []Menu,
	filter func(Menu) bool) Layout {
	layout = FilterBy(layout, true, func(menu *Menu) bool {
		return filter(*menu)
	})
	return &simpleLayout{
		logger: logger,
		layout: layout,
		filter: filter,
	}
}

// Layout 菜单布避生成器
type simpleLayout struct {
	logger log.Logger
	layout []Menu
	filter func(Menu) bool
}

func (layout *simpleLayout) Stats() interface{} {
	return "simple"
}

func makeMergeFunc(logger log.Logger, results []Menu, filter func(Menu) bool) func(source string, menu Menu) bool {
	return func(source string, menu Menu) bool {
		toList := searchMenuListInTree(results, menu.UID, nil)
		if len(toList) == 0 {
			return false
		}

		for _, to := range toList {
			if to.Title == "" {
				to.Title = menu.Title
			}
			if to.Classes == "" {
				to.Classes = menu.Classes
			}
			if to.Permission == "" {
				to.Permission = menu.Permission
			}
			if to.License == "" {
				to.License = menu.License
			}
			if to.Icon == "" {
				to.Icon = menu.Icon
			}

			if len(menu.Children) > 0 {
				if !isEmptyURL(to.URL) {
					logger.Warn("在合并菜单中发现原菜单已经有 URL, 但仍然有 app 想添加子菜单",
						log.String("source", source),
						log.String("uid", menu.UID),
						log.String("title", menu.Title),
						log.String("old_url", to.URL))
				} else if to.URL == "" {
					to.URL = menu.URL
				}

				merge := makeMergeFunc(logger, to.Children, filter)
				forEach(menu.Children, func(submenu Menu) {
					if !filter(submenu) {
						return
					}

					if !merge(source, submenu) {
						to.Children = append(to.Children, submenu)
					}
				})
			} else if !isEmptyURL(menu.URL) {
				if isEmptyURL(to.URL) {
					to.URL = menu.URL
				} else if to.URL != menu.URL {
					logger.Warn("在合并菜单中发现原菜单已经有 URL",
						log.String("source", source),
						log.String("uid", menu.UID),
						log.String("title", menu.Title),
						log.String("old_url", to.URL),
						log.String("new_url", menu.URL))
				}
			} else if to.URL == "" {
				to.URL = menu.URL
			}
		}
		return true
	}
}

func (layout *simpleLayout) Generate(menuList map[string][]Menu) ([]Menu, error) {
	if len(menuList) == 0 {
		return nil, nil
	}
	results := Copy(layout.layout)
	merge := makeMergeFunc(layout.logger, results, layout.filter)
	for key, list := range menuList {
		forEach(list, func(menu Menu) {
			if layout.filter(menu) {
				merge(key, menu)
			}
		})
	}
	return results, nil
}

func isEmptyURL(u string) bool {
	return u == "" || u == "#"
}

func forEach(list []Menu, cb func(Menu)) {
	for idx := range list {
		cb(list[idx])
		forEach(list[idx].Children, cb)
	}
}

func searchMenuListInTree(allList []Menu, uid string, results []*Menu) []*Menu {
	for idx := range allList {
		if allList[idx].UID == uid {
			results = append(results, &allList[idx])
		}

		results = searchMenuListInTree(allList[idx].Children, uid, results)
	}

	return results
}
