module github.com/runner-mei/moo

require (
	emperror.dev/emperror v0.33.0 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ThreeDotsLabs/watermill v1.1.1
	github.com/ThreeDotsLabs/watermill-http v1.1.2
	github.com/astaxie/beego v1.12.1
	github.com/certifi/gocertifi v0.0.0-20200104152315-a6d78f326758 // indirect
	github.com/cockroachdb/errors v1.2.4 // indirect
	github.com/cockroachdb/logtags v0.0.0-20190617123548-eb05cc24525f // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/digitalcrab/browscap_go v0.0.0-20160912072603-465055751e36
	github.com/getsentry/raven-go v0.2.0 // indirect
	github.com/go-playground/validator/v10 v10.4.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/google/go-cmp v0.4.0
	github.com/inconshreveable/log15 v0.0.0-20200109203555-b30bc20e4fd1 // indirect
	github.com/jaypipes/ghw v0.5.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/labstack/echo/v4 v4.1.17
	github.com/lib/pq v1.8.0
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mojocn/base64Captcha v1.3.0
	github.com/nats-io/nats.go v1.10.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/revel/config v0.21.0 // indirect
	github.com/revel/log15 v2.11.20+incompatible // indirect
	github.com/revel/pathtree v0.0.0-20140121041023-41257a1839e9 // indirect
	github.com/revel/revel v0.21.0
	github.com/runner-mei/GoBatis v1.1.15
	github.com/runner-mei/errors v0.0.0-20201207000105-8ab5f961a154
	github.com/runner-mei/goutils v0.0.0-20210131034305-05dc6a928a3e
	github.com/runner-mei/log v1.0.6
	github.com/runner-mei/loong v1.1.6
	github.com/runner-mei/resty v0.0.0-20201027090334-0e59ca0b4638
	github.com/runner-mei/validation v0.0.0-20200908120153-bc4aa6175f56
	github.com/stretchr/testify v1.6.1
	github.com/twinj/uuid v1.0.0 // indirect
	github.com/xeonx/timeago v1.0.0-rc4 // indirect
	go.uber.org/atomic v1.7.0
	go.uber.org/fx v1.13.0
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201203163018-be400aefbc4c
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	golang.org/x/sys v0.0.0-20201204225414-ed752295db88 // indirect
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/tools v0.0.0-20200804011535-6c149bb5ef0d // indirect
	gopkg.in/cas.v2 v2.2.0
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/ldap.v3 v3.1.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/stack.v0 v0.0.0-20141108040640-9b43fcefddd0 // indirect
	honnef.co/go/tools v0.0.1-2020.1.3 // indirect
)

exclude github.com/labstack/echo v3.3.10+incompatible

go 1.13

replace github.com/ThreeDotsLabs/watermill-http v1.1.2 => github.com/runner-mei/watermill-http v1.1.3-0.20200928103208-f1b3bd8e5246
