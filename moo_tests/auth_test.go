package moo_tests

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	_ "net/http/pprof"
	"strings"
	"testing"

	"github.com/runner-mei/goutils/httputil"
	"github.com/runner-mei/goutils/urlutil"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/authn"
	"go.uber.org/fx"
	"golang.org/x/net/publicsuffix"
)

type loginServer struct {
	*AppTest

	responseText string
	client       *http.Client
	sessions auth.SessionsForTest
	userManager auth.UserManager
}

func (srv *loginServer) newHTTPClient(t *testing.T, hasJar bool) *http.Client {
	client := &http.Client{}
	if hasJar {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			t.Fatal(err)
		}
		client.Jar = jar
	}
	return client
}

func (srv *loginServer) resetCookies(t *testing.T) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		t.Fatal(err)
	}
	if srv.client == nil {
		srv.client = &http.Client{}
	}
	srv.client.Jar = jar
}

func (srv *loginServer) assertOnlineCount(t *testing.T, username string, address string, exceptCount int) {
	t.Helper()

	count, err := srv.sessions.Count(context.Background(), username, address)

	// var count int
	// var err error
	// if address == "" {
	// 	if username == "" {
	// 		err = srv.db.QueryRow("select count(*) from moo_online_users where exists(select * from moo_users as users where moo_online_users.user_id = users.id and users.name = $1)", username).Scan(&count)
	// 	} else {
	// 		err = srv.db.QueryRow("select count(*) from moo_online_users where exists(select * from moo_users as users where moo_online_users.user_id = users.id)").Scan(&count)
	// 	}
	// } else {
	// 	if username == "" {
	// 		err = srv.db.QueryRow("select count(*) from moo_online_users where exists(select * from moo_users as users where moo_online_users.user_id = users.id and moo_online_users.address = $1)", address).Scan(&count)
	// 	} else {
	// 		err = srv.db.QueryRow("select count(*) from moo_online_users where exists(select * from moo_users as users where moo_online_users.user_id = users.id and users.name = $1 and moo_online_users.address = $2)", username, address).Scan(&count)
	// 	}
	// }
	if err != nil {
		t.Error(err)
		return
	}

	if count != exceptCount {
		t.Error("except is", exceptCount, ", actual is", count)
	}
}

func (a *loginServer) assertResult(t *testing.T, url string, responseText string, headers ...map[string]string) {
	t.Helper()

	client := a.client
	if client == nil {
		client = httputil.InsecureHttpClent
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Error(err)
		return
	}

	for _, header := range headers {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}
	res, err := client.Do(req)

	if err != nil {
		t.Error(err)
		return
	}
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}

	resposeAll := string(bs)
	if res.StatusCode != http.StatusOK {
		t.Error("响应码不正确",  res.Status)
		t.Log(resposeAll)
		return
	}

	if !strings.Contains(resposeAll, responseText) {
		t.Error("excepted is", responseText)
		t.Error("actual   is", resposeAll)
	}
}

func startLoginServer(t *testing.T, opts map[string]interface{}) *loginServer {
	srv := &loginServer{
		AppTest: NewAppTest(t),
		responseText: "WelcomeOK",
	}
	for key, value := range opts {
		srv.Args.CommandArgs = append(srv.Args.CommandArgs, fmt.Sprintf("%s=%v", key, value))
	}

	moo.On(func() moo.Option {
		return fx.Populate(&srv.userManager)
	})
	moo.On(func() moo.Option {
		return fx.Populate(&srv.sessions)
	})
	moo.On(func() moo.Option {
		return fx.Invoke(func(env *moo.Environment, httpSrv *moo.HTTPServer) {
			httpSrv.FastRoute(false, "home", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(srv.responseText))
			}))
		})
	})

	srv.Start(t)

	_, err := srv.userManager.Create(context.Background(), "adm", "adm", "", "123",
		map[string]interface{}{
			"welcome_url":        "url,`+hsrv.URL+`",
			"white_address_list": []string{"192.168.1.2", "192.168.1.9"},
		}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func TestLoginError(t *testing.T) {
	hwsrv := startLoginServer(t, map[string]interface{}{
		"users.captcha.disabled": true,
	})
	defer hwsrv.Close()

	for i := 0; i < 3; i++ {
		t.Log(i)
		hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
			"sso/login?_method=POST&username=adm&password=sss&service="), "用户名或密码不正确")
	}

	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/login?_method=POST&username=adm&password=123&service=/"), "错误次数大多")
}

func TestLoginCaptcha(t *testing.T) {
	hwsrv := startLoginServer(t, nil)
	defer hwsrv.Close()

	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/login?_method=POST&username=adm&password=sss&service="), "用户名或密码不正确")
	for i := 0; i < 10; i++ {
		hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
			"sso/login?_method=POST&username=adm&password=sss&service="), "验证码错误")
	}
}

func TestLoginOnline(t *testing.T) {
	hwsrv := startLoginServer(t, map[string]interface{}{
		"users.login_conflict": "",
	})
	defer hwsrv.Close()

	client1 := hwsrv.newHTTPClient(t, true)
	hwsrv.client = client1

	fmt.Println("第一个用户登录成功")
	t.Log("第一个用户登录成功")

	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/login?_method=POST&username=adm&password=123&service="+
			strings.TrimSuffix(hwsrv.Env.DaemonUrlPath, "/")+"/"), hwsrv.responseText)

	hwsrv.assertOnlineCount(t, "adm", "127.0.0.1", 1)
	hwsrv.assertOnlineCount(t, "adm", "", 1)

	// fmt.Println("=============================")
	// fmt.Println("=============================")
	// fmt.Println("=============================")
	// fmt.Println("=============================")
	// u, _ := url.Parse(urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath))
	// fmt.Println(hwsrv.client.Jar.Cookies(u))

	client2 := hwsrv.newHTTPClient(t, true)
	hwsrv.client = client2

	fmt.Println("第二个用户登录失败， 因为已在线")
	t.Log("第二个用户登录失败， 因为已在线")
	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/login?_method=POST&username=adm&password=123&service="+
			strings.TrimSuffix(hwsrv.Env.DaemonUrlPath, "/")+"/"), "上登录，最后一次活动时间为",
		map[string]string{
			auth.HeaderXForwardedFor: "192.168.1.2",
		})

	// fmt.Println("=============================")
	// fmt.Println("=============================")
	// fmt.Println("=============================")
	// fmt.Println("=============================")
	// u, _ = url.Parse(urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath))
	// fmt.Println(hwsrv.client.Jar.Cookies(u))

	hwsrv.assertOnlineCount(t, "adm", "127.0.0.1", 1)
	hwsrv.assertOnlineCount(t, "adm", "", 1)

	fmt.Println("第一个用户登出")
	t.Log("第一个用户登出")
	hwsrv.client = client1
	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/logout"), "请输入用户名")
	hwsrv.assertOnlineCount(t, "adm", "127.0.0.1", 0)

	fmt.Println("第一个用户登出成功后, 第二个用户登录成功， 因为前一个已退出")
	t.Log("第一个用户登出成功后, 第二个用户登录成功， 因为前一个已退出")
	hwsrv.client = client2
	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/login?_method=POST&username=adm&password=123&service="+
			strings.TrimSuffix(hwsrv.Env.DaemonUrlPath, "/")+"/"), hwsrv.responseText,
		map[string]string{
			auth.HeaderXForwardedFor: "192.168.1.2",
		})

	hwsrv.assertOnlineCount(t, "adm", "127.0.0.1", 0)
	hwsrv.assertOnlineCount(t, "adm", "192.168.1.2", 1)

	fmt.Println("第二个用户再次登录成功，因为同一个用户同一地点是没有问题的")
	t.Log("第二个用户再次登录成功， 因为同一个用户同一地点是没有问题的")
	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/login?_method=POST&username=adm&password=123&service="+
			strings.TrimSuffix(hwsrv.Env.DaemonUrlPath, "/")+"/"), hwsrv.responseText,
		map[string]string{
			auth.HeaderXForwardedFor: "192.168.1.2",
		})

	hwsrv.assertOnlineCount(t, "adm", "127.0.0.1", 0)
	hwsrv.assertOnlineCount(t, "adm", "192.168.1.2", 1)

	fmt.Println("第三个用户再次登录失败")
	t.Log("第三个用户再次登录失败")

	client3 := hwsrv.newHTTPClient(t, true)
	hwsrv.client = client3
	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/login?_method=POST&username=adm&password=123&service="+
			strings.TrimSuffix(hwsrv.Env.DaemonUrlPath, "/")+"/"),
		"上登录，最后一次活动时间为",
		map[string]string{
			auth.HeaderXForwardedFor: "192.168.1.9",
		})

	hwsrv.assertOnlineCount(t, "adm", "127.0.0.1", 0)
	hwsrv.assertOnlineCount(t, "adm", "192.168.1.2", 1)
	hwsrv.assertOnlineCount(t, "adm", "", 1)
}

func TestLoginBlockIP(t *testing.T) {
	hwsrv := startLoginServer(t, nil)
	defer hwsrv.Close()

	hwsrv.assertResult(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
		"sso/login?_method=POST&username=adm&password=123&service="), "用户不能在该地址访问",
		map[string]string{
			auth.HeaderXForwardedFor: "192.168.100.9",
		})
}

func TestWelcome(t *testing.T) {
	hwsrv := startLoginServer(t, nil)
	defer hwsrv.Close()

	assert := func(t *testing.T, s string) {
		hwsrv.assertResult(t, s, hwsrv.responseText)
	}

	t.Run("test 1", func(t *testing.T) {
		assert(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
			"sso/login?_method=POST&username=adm&password=123&service="))
	})

	t.Run("test 2", func(t *testing.T) {
		assert(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
			"sso/login?_method=POST&username=adm&password=123&service=/"))
	})

	t.Run("test 3", func(t *testing.T) {
		assert(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
			"sso/login?_method=POST&username=adm&password=123&service="+
				strings.TrimPrefix(hwsrv.Env.DaemonUrlPath, "/")))
	})

	t.Run("test 4", func(t *testing.T) {
		assert(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
			"sso/login?_method=POST&username=adm&password=123&service="+
				strings.TrimPrefix(hwsrv.Env.DaemonUrlPath, "/")+"/"))
	})

	t.Run("test 5", func(t *testing.T) {
		assert(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
			"sso/login?_method=POST&username=adm&password=123&service="+
				strings.TrimSuffix(hwsrv.Env.DaemonUrlPath, "/")+"/"))
	})

	t.Run("test 6", func(t *testing.T) {
		assert(t, urlutil.Join(hwsrv.URL, hwsrv.Env.DaemonUrlPath,
			"sso/login?_method=POST&username=adm&password=123&service="+
				strings.TrimSuffix(hwsrv.Env.DaemonUrlPath, "/")+"//"))
	})
}
