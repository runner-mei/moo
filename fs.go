package moo

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/kardianos/osext"
)

// FileSystem 运行环境中文件系统的抽象
type FileSystem interface {
	FromRun(s ...string) string
	FromInstallRoot(s ...string) string
	FromWebConfig(s ...string) string
	//FromBin(s ...string) string
	FromLib(s ...string) string
	FromRuntimeEnv(s ...string) string
	FromData(s ...string) string
	FromTMP(s ...string) string
	FromConfig(s ...string) string
	FromLogDir(s ...string) string
	FromDataConfig(s ...string) string
	SearchConfig(s ...string) []string
}

type linuxFs struct {
	installDir string
	binDir     string
	logDir     string
	dataDir    string
	confDir    string
	tmpDir     string
	runDir     string
}

func (fs *linuxFs) FromInstallRoot(s ...string) string {
	return filepath.Join(fs.installDir, filepath.Join(s...))
}

func (fs *linuxFs) FromRun(s ...string) string {
	return filepath.Join(fs.runDir, filepath.Join(s...))
}

func (fs *linuxFs) FromWebConfig(s ...string) string {
	return filepath.Join(fs.confDir, "web", filepath.Join(s...))
}

func (fs *linuxFs) FromBin(s ...string) string {
	return filepath.Join(fs.binDir, filepath.Join(s...))
}

func (fs *linuxFs) FromLib(s ...string) string {
	return filepath.Join(fs.installDir, "lib", filepath.Join(s...))
}

func (fs *linuxFs) FromRuntimeEnv(s ...string) string {
	return filepath.Join(fs.installDir, "runtime_env", filepath.Join(s...))
}

func (fs *linuxFs) FromData(s ...string) string {
	return filepath.Join(fs.dataDir, filepath.Join(s...))
}

func (fs *linuxFs) FromTMP(s ...string) string {
	return filepath.Join(fs.tmpDir, filepath.Join(s...))
}

func (fs *linuxFs) FromConfig(s ...string) string {
	return filepath.Join(fs.installDir, "conf", filepath.Join(s...))
}

func (fs *linuxFs) FromDataConfig(s ...string) string {
	return filepath.Join(fs.confDir, filepath.Join(s...))
}

func (fs *linuxFs) FromLogDir(s ...string) string {
	return filepath.Join(fs.logDir, filepath.Join(s...))
}

func (fs *linuxFs) SearchConfig(s ...string) []string {
	var files []string
	for _, nm := range []string{fs.FromConfig(filepath.Join(s...)),
		fs.FromDataConfig(filepath.Join(s...))} {
		if st, err := os.Stat(nm); err == nil && !st.IsDir() {
			files = append(files, nm)
		} else if err != nil && os.IsPermission(err) {
			panic(err)
		}
	}
	return files
}

func NewFileSystem(namespace string, params map[string]string) (FileSystem, error) {
	var fs *linuxFs
	if runtime.GOOS == "windows" {
		var rootDir = os.Getenv(namespace + "_root_dir")
		if params != nil {
			if s := params[namespace+"_root_dir"]; s != "" {
				rootDir = s
			}
		}
		if rootDir == "<default>" || rootDir == "." {
			// "<default>" 作为一个特殊的字符，自动使用当前目录
			if cwd, e := os.Getwd(); nil == e {
				rootDir = cwd
			} else {
				rootDir = "."
			}
		}

		if rootDir == "" {
			exeDir, _ := osext.ExecutableFolder()

			for _, filename := range []string{
				filepath.Join("conf", "app.properties"),
				filepath.Join("..", "conf", "app.properties"),

				filepath.Join(exeDir, "conf", "app.properties"),
				filepath.Join(exeDir, "..", "conf", "app.properties"),
			} {
				if abs, err := filepath.Abs(filename); err == nil {
					filename = abs
				}

				if st, err := os.Stat(filename); err == nil && !st.IsDir() {
					rootDir = filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
					break
				} else if os.IsPermission(err) {
					return nil, err
				}
			}
		}
		if rootDir == "" {
			for _, s := range filepath.SplitList(os.Getenv("GOPATH")) {
				abs, _ := filepath.Abs(filepath.Join(s, "src/cn/com/hengwei"))
				abs = filepath.Clean(abs)
				if st, err := os.Stat(abs); err == nil && st.IsDir() {
					rootDir = abs
					break
				} else if err != nil && os.IsPermission(err) {
					panic(err)
				}
			}
		}
		if rootDir == "" {
			if cwd, e := os.Getwd(); nil == e {
				rootDir = cwd
			} else {
				rootDir = "."
			}
		}

		fs = &linuxFs{
			installDir: rootDir,
			binDir:     filepath.Join(rootDir, "bin"),
			logDir:     filepath.Join(rootDir, "logs"),
			dataDir:    filepath.Join(rootDir, "data"),
			confDir:    filepath.Join(rootDir, "data", "conf"),
			tmpDir:     filepath.Join(rootDir, "data", "tmp"),
			runDir:     rootDir,
		}
	} else {
		fs = &linuxFs{
			installDir: "/usr/local/" + namespace,
			binDir:     "/usr/local/" + namespace + "/bin",
			logDir:     "/var/log/" + namespace,
			dataDir:    "/var/lib/" + namespace,
			confDir:    "/etc/" + namespace,
			tmpDir:     "/tmp/" + namespace,
			runDir:     "/var/run/" + namespace,
		}
	}

	if confDir := os.Getenv(namespace + "_conf_dir"); confDir != "" {
		fs.confDir = confDir
	}

	if dataDir := os.Getenv(namespace + "_data_dir"); dataDir != "" {
		fs.dataDir = dataDir
	}

	if params != nil {
		if s := params[namespace+"_conf_dir"]; s != "" {
			fs.confDir = s
		}

		if s := params[namespace+"_data_dir"]; s != "" {
			fs.dataDir = s
		}
	}

	return fs, nil
}
