package main

import (
	"errors"
	"fmt"
	"github.com/shaopson/forwarder/client"
	"github.com/shaopson/forwarder/client/proxy"
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
var debug = false

func main() {
	flagSet := pflag.NewFlagSet("forwarder-client", pflag.ExitOnError)
	flagSet.StringVarP(&cfgFile, "config", "c", "", "config file path")
	flagSet.BoolVarP(&debug, "debug", "", false, "")
	flagSet.SortFlags = false
	flagSet.MarkHidden("debug")
	flagSet.Parse(os.Args[1:])

	if cfgFile == "" {
		fmt.Println("missing config file")
		os.Exit(1)
	}
	config, err := parseConfig(cfgFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	if debug {
		log.SetFlags(log.Fstd | log.Fshortfile)
		log.SetLevel(log.LevelDebug)
	}
	if config.LogFile != "" {
		file, err := log.NewRotatingFile(config.LogFile, logBackupFileCount, logFileSize, false)
		if err != nil {
			fmt.Println("set log file error:%s", err)
			os.Exit(1)
		}
		log.SetOutput(file)
	}
	cli := client.NewClient(config)
	if err := cli.Run(); err != nil {
		log.Error("%s", err)
	}
}

func parseConfig(path string) (*client.Config, error) {
	file, err := ini.Load(path)
	if err != nil {
		return nil, err
	}
	if !file.HasSection("server") {
		return nil, errors.New("config file messing '[server]' section")
	}
	section := file.Section("server")
	config := &client.Config{
		Proxies: make(map[string]*proxy.Config),
	}
	if err = section.MapTo(config); err != nil {
		return nil, err
	}
	if config.ServerIp == "" {
		return nil, errors.New("[server] config missing 'server_ip'")
	}
	if config.ServerPort == "" {
		config.ServerPort = defaultPort
		//return nil, errors.New("[server] config missing 'server_port'")
	}
	if config.AuthToken == "" {
		config.AuthToken = defaultAuthToken
	}
	for _, section = range file.Sections() {
		if section.Name() == "server" || section.Name() == ini.DefaultSection {
			continue
		}
		c := &proxy.Config{
			Name: section.Name(),
		}
		if err = section.StrictMapTo(c); err != nil {
			return nil, err
		}
		config.Proxies[c.Name] = c
	}
	return config, nil
}
