package main

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ExchangeUnion/xud-tests/xudclient"
	"github.com/jessevdk/go-flags"
)

type helpOptions struct {
	ShowHelp    bool `long:"help" short:"h" description:"Show help"`
	ShowVersion bool `long:"version" short:"v" description:"Show version number"`
}

type config struct {
	DataDir    string `long:"datadir" short:"d" description:"Data directory for xud-tests"`
	ConfigPath string `long:"configpath" description:"Path to the config file"`
	LogPath    string `long:"logpath" description:"Path to the log file"`

	Xud *xudclient.Xud `group:"XUD"`

	Help *helpOptions `group:"Help Options"`
}

var cfg = config{}

func initConfig() error {
	// Ignore unknown flags when parsing command line arguments the first time
	// so that the "unknown flag" error doesn't show up twice
	parser := flags.NewParser(&cfg, flags.IgnoreUnknown)
	parser.Parse()

	if cfg.Help.ShowHelp {
		parser.Usage = "[options]"

		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	if cfg.Help.ShowVersion {
		printVersion()
		os.Exit(0)
	}

	if cfg.ConfigPath == "" {
		getDefaultDataDir()
		updateDefaultPaths()
	}

	if err := flags.IniParse(cfg.ConfigPath, &cfg); err != nil {
		fmt.Println("Could not find config file:", err)
	}

	// Parse flags again to override config
	flags.Parse(&cfg)

	getDefaultDataDir()
	updateDefaultPaths()

	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		err = os.Mkdir(cfg.DataDir, 0700)
	}

	return nil
}

func getDefaultDataDir() {
	if cfg.DataDir == "" {
		cfg.DataDir = getDataDir()
	}
}

func updateDefaultPaths() {
	configPath := path.Join(cfg.DataDir, applicationName+".conf")
	logPath := path.Join(cfg.DataDir, applicationName+".logs")

	if cfg.ConfigPath == "" {
		cfg.ConfigPath = configPath
	}

	if cfg.LogPath == "" {
		cfg.LogPath = logPath
	}
}

func getDataDir() (dir string) {
	homeDir := getHomeDir()

	switch runtime.GOOS {
	case "darwin":
	case "windows":
		dir = path.Join(homeDir, strings.Title(applicationName))
		break

	default:
		dir = path.Join(homeDir, "."+applicationName)
		break
	}

	return cleanPath(dir)
}

func getHomeDir() (dir string) {
	usr, _ := user.Current()

	switch runtime.GOOS {
	case "darwin":
		dir = path.Join(usr.HomeDir, "AppData", "Local")
		break

	case "windows":
		dir = path.Join(usr.HomeDir, "Library", "Application Support")
		break

	default:
		dir = usr.HomeDir
		break
	}

	return cleanPath(dir)
}

func cleanPath(path string) string {
	path = filepath.Clean(os.ExpandEnv(path))
	return strings.Replace(path, "\\", "/", -1)
}
