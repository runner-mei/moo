package moo

import (
	"bytes"
	"io/ioutil"
	"strings"

	"github.com/runner-mei/log"
	"github.com/runner-mei/goutils/cfg"
)

type Environment struct {
	Logger               log.Logger
	HeaderTitleText      string
	FooterTitleText      string
	LoginHeaderTitleText string
	LoginFooterTitleText string

	Name   string
	Config *cfg.Config
	Fs     FileSystem

	DaemonUrlPath string
}

func ReadFileWithDefault(files []string, defaultValue string) string {
	for _, s := range files {
		content, e := ioutil.ReadFile(s)
		if nil == e {
			if content = bytes.TrimSpace(content); len(content) > 0 {
				return string(content)
			}
		}
	}
	return defaultValue
}

func NewEnvironment(cfg *cfg.Config, fs FileSystem, logger log.Logger) *Environment {
	env := &Environment{
		Logger:        logger,
		Name:          cfg.StringWithDefault("product.name", "moo"),
		Config:        cfg,
		Fs:            fs,
		DaemonUrlPath: cfg.StringWithDefault("daemon.urlpath", "moo"),
	}
	if !strings.HasPrefix(env.DaemonUrlPath, "/") {
		env.DaemonUrlPath = "/" + env.DaemonUrlPath
	}
	if !strings.HasSuffix(env.DaemonUrlPath, "/") {
		env.DaemonUrlPath = env.DaemonUrlPath + "/"
	}
	env.HeaderTitleText = cfg.StringWithDefault("product.header_title",
		ReadFileWithDefault([]string{
			fs.FromDataConfig("resources/profiles/header.txt"),
			fs.FromData("resources/profiles/header.txt")},
			"IT综合运维管理平台"))

	env.FooterTitleText = cfg.StringWithDefault("product.footer_title",
		ReadFileWithDefault([]string{
			fs.FromDataConfig("resources/profiles/footer.txt"),
			fs.FromData("resources/profiles/footer.txt")},
			"© 2020 恒维信息技术(上海)有限公司, 保留所有版权。"))

	env.LoginHeaderTitleText = cfg.StringWithDefault("product.login_header_title",
		ReadFileWithDefault([]string{
			fs.FromDataConfig("resources/profiles/login-title.txt"),
			fs.FromData("resources/profiles/login-title.txt")},
			env.HeaderTitleText))

	env.LoginFooterTitleText = cfg.StringWithDefault("product.login_footer_title",
		ReadFileWithDefault([]string{
			fs.FromDataConfig("resources/profiles/login-footer.txt"),
			fs.FromData("resources/profiles/login-footer.txt")},
			env.FooterTitleText))

	return env
}