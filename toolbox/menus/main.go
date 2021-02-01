package menus

import (
	"bytes"
	"io"
	"strings"
)

// 菜单的分类
const (
	MenuDivider = "divider"
	MenuNull    = "null"
)

// Menu 表示一个菜单
type Menu struct {
	TableName  string `json:"-" xorm:"tpt_menus"`
	UID        string `json:"uid,omitempty" xorm:"uid notnull"`
	Title      string `json:"title,omitempty" xorm:"title notnull"`
	Permission string `json:"permission,omitempty" xorm:"permission"`
	License    string `json:"license,omitempty" xorm:"license"`
	URL        string `json:"url,omitempty" xorm:"url"`
	Icon       string `json:"icon,omitempty" xorm:"icon"`
	Classes    string `json:"classes,omitempty" xorm:"classes"`

	Children []Menu `json:"children,omitempty" xorm:"-"`
}

func (menu Menu) IsNewWindow() bool {
	return strings.Contains(menu.Classes, "new-window")
}

// IsActiveWith 判断这个菜单是否是展开的
func (menu Menu) IsActiveWith(ctx map[string]interface{}) bool {
	o := ctx["active"]
	if o == nil {
		o = ctx["controller"]
		if o == nil {
			return false
		}
	}

	name, ok := o.(string)
	if !ok {
		return false
	}
	return menu.IsActive(name)
}

// IsActive 判断这个菜单是否是展开的
func (menu Menu) IsActive(name string) bool {
	if name == menu.UID || strings.HasPrefix(menu.UID, name) {
		return true
	}

	for _, child := range menu.Children {
		if child.IsActive(name) {
			return true
		}
	}
	return false
}

// Fail 产生一个 panic
func (menu Menu) Fail() interface{} {
	panic("菜单的级数太多了，最多只支持 3 级 - " + menu.Title + "/" + menu.UID)
}

func FormatMenus(out io.Writer, isIgnore func(name string) bool, menuList []Menu, layer int, indent bool) {
	if isIgnore == nil {
		isIgnore = func(string) bool {
			return false
		}
	}
	if layer > 0 && indent {
		out.Write(bytes.Repeat([]byte("  "), layer))
	}
	out.Write([]byte("[\r\n"))
	layer++
	for idx, menu := range menuList {
		if layer > 0 {
			out.Write(bytes.Repeat([]byte("  "), layer))
		}
		out.Write([]byte("{"))

		needComma := false
		if menu.UID != "" && !isIgnore("uid") {
			io.WriteString(out, `"uid":"`)
			io.WriteString(out, menu.UID)
			io.WriteString(out, "\"")
			needComma = true
		}

		if menu.Title != "" && !isIgnore("title") {
			if needComma {
				io.WriteString(out, `,`)
			}
			io.WriteString(out, `"title":"`)
			io.WriteString(out, menu.Title)
			io.WriteString(out, "\"")
			needComma = true
		}

		if menu.Permission != "" && !isIgnore("permission") {
			if needComma {
				io.WriteString(out, `,`)
			}
			io.WriteString(out, `"permission":"`)
			io.WriteString(out, menu.Permission)
			io.WriteString(out, "\"")
			needComma = true
		}

		if menu.License != "" && !isIgnore("license") {
			if needComma {
				io.WriteString(out, `,`)
			}
			io.WriteString(out, `"license":"`)
			io.WriteString(out, menu.License)
			io.WriteString(out, "\"")
			needComma = true
		}
		if menu.Icon != "" && !isIgnore("icon") {
			if needComma {
				io.WriteString(out, `,`)
			}
			io.WriteString(out, `"icon":"`)
			io.WriteString(out, menu.Icon)
			io.WriteString(out, "\"")
		}

		if menu.Classes != "" && !isIgnore("classes") {
			if needComma {
				io.WriteString(out, `,`)
			}
			io.WriteString(out, `"classes":"`)
			io.WriteString(out, menu.Classes)
			io.WriteString(out, "\"")
			needComma = true
		}

		if menu.URL != "" && !isIgnore("url") {
			if needComma {
				io.WriteString(out, `,`)
			}
			io.WriteString(out, `"url":"`)
			io.WriteString(out, menu.URL)
			io.WriteString(out, "\"")
			needComma = true
		}

		if len(menu.Children) > 0 && !isIgnore("children") {
			if needComma {
				io.WriteString(out, `,`)
			}

			out.Write([]byte("\r\n"))
			if layer > 0 {
				out.Write(bytes.Repeat([]byte("  "), layer+1))
			}

			io.WriteString(out, `"children":`)
			FormatMenus(out, isIgnore, menu.Children, layer+1, false)
		}

		out.Write([]byte("}"))

		if idx != len(menuList)-1 {
			out.Write([]byte(",\r\n"))
		} else {
			out.Write([]byte("\r\n"))
		}
	}

	if (layer - 1) > 0 {
		out.Write(bytes.Repeat([]byte("  "), layer))
	}
	out.Write([]byte("]"))
}

func FilterBy(menuList []Menu, isCopy bool, prud func(*Menu) bool) []Menu {
	if menuList == nil {
		return nil
	}
	if len(menuList) == 0 {
		return menuList
	}

	if isCopy {
		results := make([]Menu, 0, len(menuList))
		for idx := range menuList {
			if !prud(&menuList[idx]) {
				continue
			}
			results = append(results, menuList[idx])
			results[len(results)-1].Children = FilterBy(results[len(results)-1].Children, isCopy, prud)
		}
		return results
	}

	offset := 0
	for idx := range menuList {
		if !prud(&menuList[idx]) {
			continue
		}
		menuList[idx].Children = FilterBy(menuList[idx].Children, isCopy, prud)
		if offset != idx {
			menuList[offset] = menuList[idx]
		}
		offset++
	}
	return menuList[:offset]
}

func ForEach(list []Menu, cb func(*Menu) error) error {
	if len(list) == 0 {
		return nil
	}

	for idx := range list {
		if e := cb(&list[idx]); e != nil {
			return e
		}

		if e := ForEach(list[idx].Children, cb); e != nil {
			return e
		}
	}
	return nil
}

func Copy(list []Menu) []Menu {
	if list == nil {
		return nil
	}
	results := make([]Menu, len(list))
	copy(results, list)
	for idx := range list {
		results[idx].Children = Copy(list[idx].Children)
	}
	return results
}

// IsSameMenuArray 判断两个菜单列表是否相等
func IsSameMenuArray(newList, oldList []Menu) bool {
	if len(newList) != len(oldList) {
		return false
	}

	for idx, newMenu := range newList {
		if !IsSameMenu(newMenu, oldList[idx]) {
			return false
		}
	}
	return true
}

// IsSameMenu 判断两个菜单是否相等
func IsSameMenu(newMenu, oldMenu Menu) bool {
	if newMenu.UID != oldMenu.UID {
		return false
	}
	if newMenu.Title != oldMenu.Title {
		return false
	}
	if newMenu.Classes != oldMenu.Classes {
		return false
	}
	if newMenu.Permission != oldMenu.Permission {
		return false
	}
	if newMenu.License != oldMenu.License {
		return false
	}
	if newMenu.URL != oldMenu.URL {
		return false
	}
	if newMenu.Icon != oldMenu.Icon {
		return false
	}
	return IsSameMenuArray(newMenu.Children, oldMenu.Children)
}

// RemoveMenuInTree 从列表中删除指定的菜单
func RemoveMenuInTree(menuList []Menu, name string) []Menu {
	return FilterBy(menuList, false, func(menu *Menu) bool {
		return menu.UID != name
	})
}

func RemoveDividerInTree(list []Menu) []Menu {
	if len(list) == 0 {
		return nil
	}

	offset := 0
	prev := true
	for idx := range list {
		list[idx].Children = RemoveDividerInTree(list[idx].Children)
		if list[idx].UID == MenuDivider || list[idx].Title == MenuDivider {
			if prev {
				continue
			}
			prev = true
		} else {
			prev = false
		}

		if idx != offset {
			list[offset] = list[idx]
		}
		offset++
	}

	if prev {
		offset--
	}
	if offset <= 0 {
		return nil
	}
	return list[:offset]
}
