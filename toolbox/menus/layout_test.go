package menus

import (
	"cn/com/hengwei/pkg/goutils/toolbox"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/runner-mei/log/logtest"
)

var (
	menuTestLayout = []toolbox.Menu{
		{
			UID: "1",
			URL: "#",
		},
		{
			UID: "2",
			URL: "#",
		},
		{
			UID: "3",
			URL: "#",
			Children: []toolbox.Menu{
				{
					UID: "3_1",
					URL: "#",
				},
			},
		},
		{
			UID:   "4",
			Title: "m4",
		},
		{ // 特殊案例, 4_1 应该在 4 的 Children 中，
			// 但我在这里指定，确保它也可以被更新
			UID:   "4_1",
			Title: "m4_1",
		},
		{
			UID:   "5",
			Title: "m5",
			Children: []toolbox.Menu{
				{
					UID: "5_1",
				},
			},
		},
		{
			UID:   "6",
			Title: "m6",
			Children: []toolbox.Menu{
				{
					UID: "6_2",
				},
			},
		},
		{ // 用户在创建时被过滤
			UID:   "7",
			Title: "m7",
		},
		{ // 用户在创建时 子节点 被过滤
			UID:   "8",
			Title: "m8",
			Children: []toolbox.Menu{
				{
					UID: "8_2",
				},
			},
		},
		{ // 用户在运行时 子节点 被过滤
			UID:   "9",
			Title: "m9",
		},
	}

	menuTestResults = []toolbox.Menu{
		{
			UID:   "1",
			Title: "m1",
			URL:   "/m1",
		},
		{
			UID:   "2",
			Title: "m2",
			URL:   "/m2",
		},
		{
			UID:   "3",
			Title: "m3",
			URL:   "#",
			Children: []toolbox.Menu{
				{
					UID:   "3_1",
					Title: "m3_1",
					URL:   "/m3_1",
				},
			},
		},
		{
			UID:   "4",
			Title: "m4",
			URL:   "#",
			Children: []toolbox.Menu{
				{
					UID:   "4_1",
					Title: "m4_1",
					URL:   "/m4_1",
				},
				{
					UID:   "4_2",
					Title: "m4_2",
					URL:   "/m4_2",
				},
				{
					UID:   "4_2",
					Title: "m4_2",
					URL:   "/m4_2",
				},
			},
		},
		{ // 特殊案例, 确保 forEach 是正确的
			UID:   "4_1",
			Title: "m4_1",
			URL:   "/m4_1",
		},
		{
			UID:   "5",
			Title: "m5",
			Children: []toolbox.Menu{
				{
					UID:   "5_1",
					Title: "m5_1",
					URL:   "/m5_1",
				},
			},
		},
		{
			UID:   "6",
			Title: "m6",
			Children: []toolbox.Menu{
				{
					UID:   "6_2",
					Title: "m6_2",
					URL:   "/m6_2",
				},
				{
					UID:   "6_1",
					Title: "m6_1",
					URL:   "/m6_1",
				},
			},
		},
		{
			UID:      "8",
			Title:    "m8",
			URL:      "/m8",
			Children: []toolbox.Menu{},
		},
		{
			UID:   "9",
			Title: "m9",
			Children: []toolbox.Menu{
				{
					UID:   "9_2",
					Title: "m9_2",
					URL:   "/m9_2",
				},
			},
		},
	}

	testapp1 = []toolbox.Menu{
		{
			UID:   "1",
			Title: "m1",
			URL:   "/m1",
		},
		{
			UID:   "3",
			Title: "m3",
			URL:   "#",
			Children: []toolbox.Menu{
				{
					UID:   "3_1",
					Title: "m3_1",
					URL:   "/m3_1",
				},
			},
		},
	}
	testapp2 = []toolbox.Menu{
		{
			UID:   "2",
			Title: "m2",
			URL:   "/m2",
		},
		{
			UID:   "4",
			Title: "m4",
			URL:   "#",
			Children: []toolbox.Menu{
				{
					UID:   "4_1",
					Title: "m4_1",
					URL:   "/m4_1",
				},
				{
					UID:   "4_2",
					Title: "m4_2",
					URL:   "/m4_2",
				},
				{
					UID:   "4_2",
					Title: "m4_2",
					URL:   "/m4_2",
				},
			},
		},
	}
	testapp3 = []toolbox.Menu{
		{
			UID:   "5_1",
			Title: "m5_1",
			URL:   "/m5_1",
		},
		{
			UID:   "6",
			Title: "m6",
			Children: []toolbox.Menu{
				{
					UID:   "6_1",
					Title: "m6_1",
					URL:   "/m6_1",
				},
				{
					UID:   "6_2",
					Title: "m6_2",
					URL:   "/m6_2",
				},
			},
		},
	}

	testapp4 = []toolbox.Menu{
		{
			UID:   "7",
			Title: "m7",
			URL:   "/m7",
		},
		{
			UID:   "8",
			Title: "m8",
			URL:   "/m8",
		},
		{
			UID:   "8_2",
			Title: "m8_2",
			URL:   "/m8_2",
		},
		{
			UID:   "9",
			Title: "m9",
			Children: []toolbox.Menu{
				{
					UID:   "9_1",
					Title: "m9_1",
					URL:   "/m9_1",
				},
				{
					UID:   "9_2",
					Title: "m9_2",
					URL:   "/m9_2",
				},
			},
		},
	}
)

func TestLayoutSimple(t *testing.T) {
	layout := NewSimple(
		logtest.NewLogger(t),
		toolbox.CopyMenuArray(menuTestLayout),
		func(menu toolbox.Menu) bool {
			if menu.UID == "7" ||
				menu.UID == "8_2" ||
				menu.UID == "9_1" {
				return false
			}
			return true
		})

	app1 := toolbox.CopyMenuArray(testapp1)
	app2 := toolbox.CopyMenuArray(testapp2)
	app3 := toolbox.CopyMenuArray(testapp3)
	app4 := toolbox.CopyMenuArray(testapp4)

	apps := map[string][]toolbox.Menu{
		"app1": app1,
		"app2": app2,
		"app3": app3,
		"app4": app4,
	}

	results, err := layout.Generate(apps)
	if err != nil {
		t.Error(err)
		return
	}

	if !toolbox.IsSameMenuArray(results, menuTestResults) {
		msg := cmp.Diff(menuTestResults, results)
		t.Error(msg)
	}
	t.Log("修改 m4 下的子菜单，确保它不会影响生成后的结果")
	app2[1].Children[1].URL = "/test"

	if !toolbox.IsSameMenuArray(results, menuTestResults) {
		msg := cmp.Diff(menuTestResults, results)
		t.Error(msg)

		// bs, _ := json.MarshalIndent(results, "", "  ")
		// t.Log(string(bs))
	}
}
