package cas

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/auth"
	"github.com/runner-mei/moo/auth/services"
	"github.com/runner-mei/moo/users/usermodels"
	gocas "gopkg.in/cas.v2"
)

// CASOptions configuration options
type CASOptions struct {
	Env            *moo.Environment
	Logger         log.Logger
	UserPrefix     string
	URL            *url.URL        // URL to the CAS service
	Client         *http.Client    // Custom http client to allow options for http connections
	SendService    bool            // Custom sendService to determine whether you need to send service param
	URLScheme      gocas.URLScheme // Custom url scheme, can be used to modify the request urls for the client
	Renderer       *auth.Renderer
	Sessions       auth.Sessions
	LoginCallback  string
	LogoutCallback string
	IgnoreList     []string

	Fields     map[string]string
	Roles      []string
	UserSyncer UserSyncer
	Users      *usermodels.Users
}

// CASClient implements the main protocol
type CASClient struct {
	logger         log.Logger
	userPrefix     string
	client         *http.Client
	sendService    bool
	urlScheme      gocas.URLScheme
	renderer       *auth.Renderer
	sessions       auth.Sessions
	loginCallback  *url.URL
	logoutCallback *url.URL
	ignoreList     []string

	fields     map[string]string
	roles      []int64
	users      *usermodels.Users
	userSyncer UserSyncer

	stValidator *gocas.ServiceTicketValidator
}

// NewCASClient creates a Client with the provided Options.
func NewCASClient(options *CASOptions) *CASClient {
	var urlScheme gocas.URLScheme
	if options.URLScheme != nil {
		urlScheme = options.URLScheme
	} else {
		urlScheme = gocas.NewDefaultURLScheme(options.URL)
	}

	var client *http.Client
	if options.Client != nil {
		client = options.Client
	} else {
		client = &http.Client{}
	}

	var loginCallback, logoutCallback *url.URL
	var err error
	if options.LoginCallback != "" {
		loginCallback, err = url.Parse(options.LoginCallback)
		if err != nil {
			panic(errors.Wrap(err, "配置 LoginCallback '"+options.LoginCallback+"' 不正确"))
		}
	}

	if options.LogoutCallback != "" {
		logoutCallback, err = url.Parse(options.LogoutCallback)
		if err != nil {
			panic(errors.Wrap(err, "配置 LogoutCallback '"+options.LogoutCallback+"' 不正确"))
		}
	}

	var roles []int64
	for _, roleName := range options.Roles {
		roleName = strings.TrimSpace(roleName)
		if roleName == "" {
			continue
		}

		role, err := options.Users.GetRoleByName(context.Background(), roleName)
		if err != nil {
			panic(errors.Wrap(err, "加载角色 '"+roleName+"' 失败"))
		}
		roles = append(roles, role.ID)
	}

	return &CASClient{
		logger:         options.Logger,
		userPrefix:     options.UserPrefix,
		renderer:       options.Renderer,
		sessions:       options.Sessions,
		client:         client,
		loginCallback:  loginCallback,
		logoutCallback: logoutCallback,
		ignoreList:     options.IgnoreList,
		urlScheme:      urlScheme,
		sendService:    options.SendService,
		stValidator:    gocas.NewServiceTicketValidator(client, options.URL),
		roles:          roles,
		fields:         options.Fields,
		users:          options.Users,
		userSyncer:     options.UserSyncer,
	}
}

// LoginURLForRequest determines the CAS login URL for the http.Request.
func (c *CASClient) LoginURLForRequest(r *http.Request) (string, error) {
	u, err := c.urlScheme.Login()
	if err != nil {
		return "", err
	}

	service, err := requestURL(r)
	if err != nil {
		return "", err
	}

	for _, pa := range c.ignoreList {
		if service.Path == pa {
			service.Path = "/"
			break
		}
	}

	if c.loginCallback != nil {
		o := new(url.URL)
		*o = *c.loginCallback

		o.Host = r.Host
		if host := r.Header.Get("X-Forwarded-Host"); host != "" {
			o.Host = host
		}

		o.Scheme = "http"
		if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
			o.Scheme = scheme
		} else if r.TLS != nil {
			o.Scheme = "https"
		}

		sq := o.Query()
		sq.Set("redirect", service.String())
		o.RawQuery = sq.Encode()
		service = o
	}

	q := u.Query()
	q.Add("service", sanitisedURLString(service))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// LogoutURLForRequest determines the CAS logout URL for the http.Request.
func (c *CASClient) LogoutURLForRequest(r *http.Request) (string, error) {
	u, err := c.urlScheme.Logout()
	if err != nil {
		return "", err
	}

	if c.sendService {
		service, err := requestURL(r)
		if err != nil {
			return "", err
		}
		for _, pa := range c.ignoreList {
			if service.Path == pa {
				service.Path = "/"
				break
			}
		}

		// if c.logoutCallback != nil {
		// 	o := new(url.URL)

		// 	*o = *c.logoutCallback
		// 	o.Host = r.Host
		// 	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		// 		o.Host = host
		// 	}

		// 	o.Scheme = "http"
		// 	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		// 		o.Scheme = scheme
		// 	} else if r.TLS != nil {
		// 		o.Scheme = "https"
		// 	}

		// 	sq := o.Query()
		// 	sq.Set("redirect", service.String())
		// 	o.RawQuery = sq.Encode()
		// 	service = o
		// }

		q := u.Query()
		q.Add("service", sanitisedURLString(service))
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

// ServiceValidateURLForRequest determines the CAS serviceValidate URL for the ticket and http.Request.
func (c *CASClient) ServiceValidateURLForRequest(ticket string, r *http.Request) (string, error) {
	service, err := requestURL(r)
	if err != nil {
		return "", err
	}
	return c.stValidator.ServiceValidateUrl(service, ticket)
}

// ValidateURLForRequest determines the CAS validate URL for the ticket and http.Request.
func (c *CASClient) ValidateURLForRequest(ticket string, r *http.Request) (string, error) {
	service, err := requestURL(r)
	if err != nil {
		return "", err
	}
	return c.stValidator.ValidateUrl(service, ticket)
}

// RedirectToLogout replies to the request with a redirect URL to log out of CAS.
func (c *CASClient) RedirectToLogout(w http.ResponseWriter, r *http.Request) {
	u, err := c.LogoutURLForRequest(r)
	if err != nil {
		http.Error(w, "aaaaaa"+err.Error(), http.StatusInternalServerError)
		return
	}

	c.logger.Info("Logging out, redirecting client to cas server", log.String("redirect", u))

	// c.clearSession(w, r)

	c.renderer.LogoutWithRedirect(r.Context(), w, r, u)
}

// RedirectToLogin replies to the request with a redirect URL to authenticate with CAS.
func (c *CASClient) RedirectToLogin(w http.ResponseWriter, r *http.Request) {
	u, err := c.LoginURLForRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.logger.Info("Logging in, redirecting client to cas server", log.String("redirect", u))

	http.Redirect(w, r, u, http.StatusFound)
}

// validateTicket performs CAS ticket validation with the given ticket and service.
func (c *CASClient) validateTicket(ticket string, service *http.Request) (*gocas.AuthenticationResponse, error) {
	serviceURL, err := requestURL(service)
	if err != nil {
		return nil, err
	}

	return c.stValidator.ValidateTicket(serviceURL, ticket)
}

func (c *CASClient) LoginCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ticket := q.Get("ticket")
	if ticket == "" {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "ticket is missing")
		return
	}

	response, err := c.validateTicket(ticket, r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "validate ticket error!!!\r\n")
		io.WriteString(w, err.Error())
		return
	}

	username := c.userPrefix + response.User
	user, err := c.users.GetUserByName(r.Context(), username)
	if err != nil && !errors.IsNotFound(err) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "从数据库中获取用户信息失败：\r\n")
		io.WriteString(w, err.Error())
		return
	}

	isNewUser := false
	if user == nil {
		isNewUser = true
		user = &usermodels.User{
			Name:        username,
			Nickname:    username,
			Description: "",
			Attributes:  map[string]interface{}{},
			Source:      "cas",
		}

		c.logger.Info("新用户登陆，开始创建新用户",
			log.String("username", username),
			log.Stringer("response", log.StringerFunc(func() string {
				return fmt.Sprintf("%#v", *response)
			})))

		if response.Attributes != nil {
			for key, value := range c.fields {
				values := response.Attributes[key]
				if len(values) == 0 {
					continue
				}

				if len(values) == 1 {
					user.Attributes[value] = values[0]
				} else {
					user.Attributes[value] = values
				}
			}
		}

		if c.userSyncer != nil {
			nickname, fields, err := c.userSyncer.Read(r.Context(), response.User)
			if err == nil {
				if nickname != "" {
					user.Nickname = nickname

					exists, err := c.users.NicknameExists(r.Context(), nickname)
					if err != nil {
						c.logger.Error("新用户登陆，查询用户名是否存在", log.String("username", username), log.Error(err))
					} else if exists {
						user.Nickname = nickname + " - " + response.User
					}
				}
				for k, v := range fields {
					user.Attributes[k] = v
				}
			}
		}

		userid, err := c.users.CreateUser(r.Context(), user, c.roles)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			if e := errors.Unwrap(err); e != nil {
				io.WriteString(w, ":\r\n")
				io.WriteString(w, e.Error())
			}
			return
		}
		user.ID = userid

		c.logger.Info("新用户登陆，创建新用户成功", log.String("username", username))
	}

	address := auth.RealIP(r)

	sessionID, err := c.sessions.Login(r.Context(), user.ID, user.Name, address)
	if err != nil && errors.IsNotFound(err) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "创建在线用户信息失败：\r\n")
		io.WriteString(w, err.Error())
		return
	}

	redirect := q.Get("redirect")

	c.logger.Info("Login successful, redirecting to system.", log.String("redirect", redirect))

	authCtx := &services.AuthContext{
		Logger: c.logger,
		Request: services.LoginRequest{
			UserID:   user.ID,
			Username: username,
			Service:  redirect,
		},
		Response: services.LoginResult{
			IsOK:      true,
			SessionID: sessionID,
			IsNewUser: isNewUser,
		},
	}
	c.renderer.LoginOK(authCtx, w, r)
}

// requestURL determines an absolute URL from the http.Request.
func requestURL(r *http.Request) (*url.URL, error) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		return nil, err
	}

	u.Host = r.Host
	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		u.Host = host
	}

	u.Scheme = "http"
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		u.Scheme = scheme
	} else if r.TLS != nil {
		u.Scheme = "https"
	}

	return u, nil
}

var (
	urlCleanParameters = []string{"gateway", "renew", "service", "ticket"}
)

// sanitisedURL cleans a URL of CAS specific parameters
func sanitisedURL(unclean *url.URL) *url.URL {
	// Shouldn't be any errors parsing an existing *url.URL
	u, _ := url.Parse(unclean.String())
	q := u.Query()

	for _, param := range urlCleanParameters {
		q.Del(param)
	}

	u.RawQuery = q.Encode()
	return u
}

// sanitisedURLString cleans a URL and returns its string value
func sanitisedURLString(unclean *url.URL) string {
	return sanitisedURL(unclean).String()
}
