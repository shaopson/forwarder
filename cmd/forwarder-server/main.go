package main

import (
	"errors"
	"fmt"
	"github.com/shaopson/forwarder/server"
	log "github.com/shaopson/grlog"
	"github.com/spf13/pflag"
	"gopkg.in/ini.v1"
	"os"
)

const (
	defaultPort        = "8848"
	defaultAuthToken   = "forwarder"
	logFileSize        = 1 << 24 //16384 kb
	logBackupFileCount = 9
)

var cfgFile = ""
var addr = ""
var port = ""
var token = ""
var logFile = ""
var debug = false

func main() {
	flagSet := pflag.NewFlagSet("forwarder-server", pflag.ExitOnError)
	flagSet.StringVarP(&cfgFile, "config", "c", "", "config file path")
	flagSet.StringVarP(&addr, "addr", "a", "0.0.0.0", "bind addr")
	flagSet.StringVarP(&port, "port", "p", defaultPort, "bind port")
	flagSet.StringVarP(&token, "token", "t", "", "auth token")
	flagSet.StringVarP(&logFile, "log_file", "", "", "log file path")
	flagSet.BoolVarP(&debug, "debug", "", false, "")
	flagSet.SortFlags = false
	flagSet.MarkHidden("debug")
	flagSet.Parse(os.Args[1:])

	var config *server.Config
	var err error
	if cfgFile != "" {
		fmt.Println("use config file:", cfgFile)
		config, err = parseConfig(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	} else {
		if token == "" {
			token = defaultAuthToken
		}
		config = &server.Config{
			Ip:        addr,
			Port:      port,
			AuthToken: token,
			LogFile:   logFile,
		}
	}
	if debug {
		log.SetFlags(log.FlagStd | log.FlagSFile)
		log.SetLevel(log.LevelDebug)
	}
	if config.LogFile != "" {
		file, err := log.NewRotateFile(config.LogFile, logBackupFileCount, logFileSize, true)
		defer file.Close()
		if err != nil {
			fmt.Println("open log file error:", err)
			os.Exit(1)
		}
		log.SetOutput(file)
	}
	srv := server.NewServer(config)
	if err = srv.Run(); err != nil {
		log.Error("%s", err)
	}
}

func parseConfig(path string) (*server.Config, error) {
	file, err := ini.Load(path)
	if err != nil {
		return nil, err
	}
	config := &server.Config{}
	section := file.Section(ini.DefaultSection)
	if err = section.MapTo(config); err != nil {
		return nil, err
	}
	if len(config.Port) == 0 {
		return nil, errors.New("config file missing `bind_port`")
	}
	if config.AuthToken == "" {
		config.AuthToken = defaultAuthToken
	}
	return config, nil
}
