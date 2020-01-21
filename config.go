package moo

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/as"
	"github.com/runner-mei/moo/cfg"
)

func readConfigs(fs FileSystem, prefix string, args *Arguments) (*cfg.Config, error) {
	var allProps = map[string]interface{}{}

	read := func(isCustom bool, files []string) error {
		for i := range files {
			var filename = files[i]
			if fs != nil {
				if isCustom {
					filename = fs.FromDataConfig(filename)
				} else {
					filename = fs.FromConfig(filename)
				}
			}
			props, err := cfg.ReadProperties(filename)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return err
			}

			for k, v := range props {
				allProps[k] = v
			}
		}
		return nil
	}

	err := read(false, args.Defaults)
	if err != nil {
		return nil, err
	}

	for name := range allProps {
		value := os.Getenv(prefix + name)
		if value != "" {
			allProps[name] = value
		}
	}

	err = read(true, args.Customs)
	if err != nil {
		return nil, err
	}

	httpPort, admPort, err := readTSDBConfig(fs)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		allProps["tsdb.http_port"] = httpPort
		allProps["tsdb.admin_port"] = admPort
	}

	if minioConfig, err := readMinioConfig(fs); err == nil && minioConfig != nil {
		allProps["minio_config"] = minioConfig
	} else if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return cfg.NewConfig(allProps), nil
}

func readCommandLineArgs(args []string) (map[string]string, error) {
	props := map[string]string{}

	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("Invalid command line argument. argument: " + arg)
		}

		props[parts[0]] = parts[1]
	}
	return props, nil
}

func fileExists(filename string) bool {
	st, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return st != nil && !st.IsDir()
}

func readMinioConfig(fs FileSystem) (map[string]interface{}, error) {
	configFile := fs.FromData("minio", ".minio.sys", "config", "config.json")
	if !fileExists(configFile) {

		configFile2 := fs.FromData(".minio", "config.json")
		if !fileExists(configFile2) {
			return nil, errors.Wrapf(os.ErrNotExist, "file '%s' is not exist", configFile)
		}
		configFile = configFile2
	}

	r, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var config map[string]interface{}
	if err := json.NewDecoder(r).Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}

func readTSDBConfig(fs FileSystem) (httpPort, admPort string, err error) {
	var tsdbConfigFile string

	if runtime.GOOS == "windows" {
		tsdbConfigFile = fs.FromConfig("tsdb_config.win.conf")
	} else {
		tsdbConfigFile = fs.FromConfig("tsdb_config.conf")
	}
	if filename := fs.FromDataConfig("tsdb_config.conf"); fileExists(filename) {
		tsdbConfigFile = filename
	}

	var tsdbConfig map[string]interface{}
	_, err = toml.DecodeFile(tsdbConfigFile, &tsdbConfig)
	if err != nil {
		return
	}
	if tsdbConfig == nil {
		return
	}

	tsdbHTTP, _ := as.Object(tsdbConfig["http"])
	if tsdbHTTP != nil {
		if _, port, err := net.SplitHostPort(fmt.Sprint(tsdbHTTP["bind-address"])); err == nil {
			httpPort = port
		}
	}

	tsdbAdmin, _ := as.Object(tsdbConfig["admin"])
	if tsdbAdmin != nil {
		if _, port, err := net.SplitHostPort(fmt.Sprint(tsdbAdmin["bind-address"])); err == nil {
			admPort = port
		}
	}
	return
}
