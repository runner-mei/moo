package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	_ "github.com/lib/pq"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	ldap "gopkg.in/ldap.v3"
)

type HasSource interface {
	Source() string
}

func isConnectError(err error) bool {
	if ldapErr, ok := err.(*ldap.Error); ok {
		if opErr, ok := ldapErr.Err.(*net.OpError); ok && opErr.Op == "dial" {
			return true
		}
	}
	return false
}

func LdapUserCheck(env *moo.Environment, logger log.Logger) AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		ldapServer := env.Config.StringWithDefault("users.ldap_address", "")
		if ldapServer == "" {
			logger.Warn("ldap 没有配置，跳过它")
			return nil
		}
		ldapTLS := env.Config.BoolWithDefault("users.ldap_tls", false)
		ldapDN := env.Config.StringWithDefault("users.ldap_base_dn", "")
		ldapFilter := env.Config.StringWithDefault("users.ldap_filter", "(&(objectClass=organizationalPerson)(sAMAccountName=%s))")
		ldapUserFormat := env.Config.StringWithDefault("users.ldap_user_format", "")
		if ldapUserFormat == "" {
			if ldapDN != "" {
				ldapUserFormat = "cn=%s," + ldapDN
			} else {
				ldapUserFormat = "%s"
			}
		}
		ldapRoles := env.Config.StringWithDefault("users.ldap_roles", "memberOf")
		exceptedRole := env.Config.StringWithDefault("users.ldap_login_role", "")

		logFields := []log.Field{
			log.String("ldapServer", ldapServer),
			log.Bool("ldapTLS", ldapTLS),
			log.String("ldapDN", ldapDN),
			log.String("ldapFilter", ldapFilter),
			log.String("ldapUserFormat", ldapUserFormat),
		}

		auth.OnAuth(func(ctx *AuthContext) (bool, error) {
			isLdap := false
			isNew := false
			if ctx.Authentication != nil {
				u, ok := ctx.Authentication.(HasSource)
				if !ok {
					return false, nil
				}

				// if o := u.Data["source"]; o != nil {
				// 	method = strings.ToLower(fmt.Sprint(o))
				// }

				var method = u.Source()
				if method != "ldap" {
					return false, nil
				}
				isLdap = true
			} else {
				isNew = true
			}

			l, err := ldap.Dial("tcp", ldapServer)
			if err != nil {
				logger.Info("无法连接到 LDAP 服务器", log.Error(err))
				return isLdap, &ErrExternalServer{Msg: "无法连接到 LDAP 服务器" + err.Error()}
			}
			defer l.Close()

			if ldapTLS {
				// Reconnect with TLS
				err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
				if err != nil {
					logger.Info("无法连接到 LDAP 服务器", log.Error(err))
					return isLdap, &ErrExternalServer{Msg: "无法连接到 LDAP 服务器" + err.Error()}
				}
			}
			// First bind with a read only user
			err = l.Bind(fmt.Sprintf(ldapUserFormat, ctx.Request.Username), ctx.Request.Password)
			if err != nil {
				logger.Info("无法连接到 LDAP 服务器", log.Error(err))
				return isLdap, err
			}

			logger := ctx.Logger.With(logFields...).With(log.String("username", ctx.Request.Username), log.String("password", "********"))
			logger.Info("尝试 ldap 验证, 用户名和密码正确")

			if !isNew {
				if exceptedRole == "" {
					return true, nil
				}
			}

			ldapFilterForUser := fmt.Sprintf(ldapFilter, ctx.Request.Username)

			// dn := "cn=" + username + "," + ldapDN
			userData := map[string]interface{}{"is_new": isNew}
			//获取数据
			searchResult, err := l.Search(ldap.NewSearchRequest(
				ldapDN,
				ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
				ldapFilterForUser, nil, nil,
			))
			if err == nil {
				roles := make([]string, 0, 4)
				for _, ent := range searchResult.Entries {
					//fmt.Println("==============================")
					//fmt.Println(ent.DN)

					for _, attr := range ent.Attributes {
						// fmt.Println(attr.Name, attr.Values)
						if len(attr.Values) > 0 {
							if ldapRoles == attr.Name {
								for _, roleName := range attr.Values {
									dn, err := ldap.ParseDN(roleName)
									if err != nil {
										roles = append(roles, roleName)
										continue
									}

									if len(dn.RDNs) == 0 || len(dn.RDNs[0].Attributes) == 0 {
										continue
									}

									roles = append(roles, dn.RDNs[0].Attributes[0].Value)
								}
								userData["roles"] = roles
								userData["raw_roles"] = attr.Values
							}
							userData[attr.Name] = attr.Values[0]
						}
					}
				}

				if exceptedRole != "" {
					found := false
					for _, role := range roles {
						if role == exceptedRole {
							found = true
							break
						}
					}

					if !found {
						if len(searchResult.Entries) == 0 {
							logger.Warn("user is permission denied - roles is empty", log.String("exceptedRole", exceptedRole))
						} else {
							logger.Warn("user is permission denied", log.String("exceptedRole", exceptedRole), log.StringArray("roles", roles))
						}
						return true, ErrPermissionDenied
					}
				}
			} else {
				logger.Warn("search user and role fail", log.Error(err))

				if exceptedRole != "" {
					return true, ErrPermissionDenied
				}
			}

			if isNew {
				ctx.Request.Username = strings.ToLower(ctx.Request.Username)
				ctx.Response.IsNewUser = true
				return true, nil
			}
			return true, nil
		})
		return nil
	})
}
