package menus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	mooapi "github.com/runner-mei/moo/api"
)

// Weaver 菜单的组织工具
type Weaver interface {
	Update(group string, value []Menu) error
	Generate(app string) ([]Menu, error)
	GenerateAll() (map[string][]Menu, error)
	Stats() interface{}
}

// Server 菜单的服备
type Server struct {
	env    *moo.Environment
	weaver Weaver
	logger log.Logger
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer io.Copy(ioutil.Discard, r.Body)
	}

	switch r.Method {
	case "GET":
		if strings.HasSuffix(r.URL.Path, "/stats") ||
			strings.HasSuffix(r.URL.Path, "/stats/") {
			srv.stats(w, r)
			return
		}
		srv.read(w, r)
	case "PUT", "POST":
		srv.write(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (srv *Server) Get(filter func(user mooapi.User, menuItem *Menu) bool) func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		user, err := mooapi.ReadUserFromContext(ctx)
		if err != nil {
			srv.logger.Error("request is unauthorized", log.Error(err))
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		app := r.URL.Query().Get("app")
		if app == "" {
			srv.logger.Error("app is missing")
			http.Error(w, "app is missing", http.StatusBadRequest)
			return
		}
		menuList, err := srv.weaver.Generate(app)
		if err != nil {
			srv.logger.Error("generate menu fail", log.String("app", app))
			http.Error(w, "generate menu fail", http.StatusInternalServerError)
			return
		}

		// 必须拷贝一下, 不然下面的修改会导致 weaver 中的数据被修改
		menuList = FilterBy(menuList, true, func(menuItem *Menu) bool {
			return filter(user, menuItem)
		})
		menuList = RemoveDividerInTree(menuList)

		if err != nil {
			srv.logger.Error("stats fail", log.String("app", app), log.Error(err))
		} else {
			srv.logger.Info("query is ok", log.String("app", app))
		}
	}
}

func isConsumeHTML(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return true
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

func renderTEXT(w http.ResponseWriter, code int, txt string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_, err := fmt.Fprintln(w, txt)
	return err
}

func renderJSON(w http.ResponseWriter, code int, value interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(value)
}

func (srv *Server) stats(w http.ResponseWriter, r *http.Request) {
	err := renderJSON(w, http.StatusOK, srv.weaver.Stats())
	if err != nil {
		srv.logger.Error("stats fail", log.Error(err))
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
}

func (srv *Server) read(w http.ResponseWriter, r *http.Request) {
	app := r.URL.Query().Get("app")
	switch app {
	case "stats":
		err := renderJSON(w, http.StatusOK, srv.weaver.Stats())
		if err != nil {
			srv.logger.Error("stats fail", log.String("app", app), log.Error(err))
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
	case "", "all":
		results, err := srv.weaver.GenerateAll()
		if err != nil {
			srv.logger.Error("query all fail", log.String("app", app), log.Error(err))
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		//if srv.renderHTML != nil && isConsumeHTML(r) {
		//	err = srv.renderHTML(w, r, results)
		//} else {
		err = renderJSON(w, http.StatusOK, results)
		//}
		if err != nil {
			srv.logger.Error("query all fail", log.String("app", app), log.Error(err))
		} else {
			srv.logger.Info("query all is ok", log.String("app", app))
		}
	default:
		results, err := srv.weaver.Generate(app)
		if err != nil {
			srv.logger.Error("stats fail", log.String("app", app), log.Error(err))
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		// if srv.renderHTML != nil && isConsumeHTML(r) {
		// 	err = srv.renderHTML(w, r, results)
		// } else {
		err = renderJSON(w, http.StatusOK, results)
		//}
		if err != nil {
			srv.logger.Error("stats fail", log.String("app", app), log.Error(err))
		} else {
			srv.logger.Info("query is ok", log.String("app", app))
		}
	}
}

func (srv *Server) write(w http.ResponseWriter, r *http.Request) {
	app := r.URL.Query().Get("app")
	if app == "" {
		srv.logger.Error("app is missing")
		http.Error(w, "app is missing", http.StatusBadRequest)
		return
	}

	var data []Menu
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		srv.logger.Error("update fail", log.String("app", app), log.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = srv.weaver.Update(app, data)
	if err != nil {
		srv.logger.Error("update fail", log.String("app", app), log.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	renderTEXT(w, http.StatusOK, "OK")
	srv.logger.Info("update is successful", log.String("app", app))
}

// NewServer 创建一个菜单服备
func NewServer(env *moo.Environment, weaver Weaver, logger log.Logger) (*Server, error) {
	return &Server{
		env:    env,
		weaver: weaver,
		logger: logger,
	}, nil
}
