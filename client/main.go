package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/dev-shao/iprp/log"
	"gopkg.in/ini.v1"
	"os"
)

var LogFileSize int64 = 1024 * 1024
var LogBackupCount = 5

func main() {
	iniFile := flag.String("c", "", "ini config file path")
	flag.Parse()

	if *iniFile == "" {
		fmt.Println("Missing configuration file")
		os.Exit(1)
	}
	cfg, err := parseConfig(*iniFile)
	if err != nil {
		fmt.Println("Parse config error:%s", err)
		os.Exit(1)
	}
	if cfg.LogFile != "" {
		if file, err := log.NewRotatingFile(cfg.LogFile, LogFileSize, LogBackupCount); err != nil {
			log.Error("Config log file error:%s", err)
		} else {
			defer file.Close()
			log.SetOutput(file)
		}
	}
	client := NewClient(cfg)
	err = client.Run()
	if err != nil {
		log.Error("client over:%s", err)
	}
}

func parseConfig(cfgFile string) (cfg *ClientConfig, err error) {
	iniFile, err := ini.Load(cfgFile)
	if err != nil {
		return
	}
	if !iniFile.HasSection("server") {
		return nil, errors.New("The config file missing [server] section")
	}
	cfg = &ClientConfig{
		ProxyConfigs: map[string]*ProxyConfig{},
	}
	section := iniFile.Section("server")
	err = section.MapTo(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.ServerIp == "" {
		errors.New("Missing server ip")
	}
	if cfg.ServerPort == "" {
		errors.New("Missing server port")
	}
	for _, section := range iniFile.Sections() {
		if section.Name() == "server" || section.Name() == "DEFAULT" {
			continue
		}
		c := &ProxyConfig{
			Name: section.Name(),
		}
		err := section.StrictMapTo(c)
		if err != nil {
			return nil, err
		}
		if c.LocalIp == "" {
			err = fmt.Errorf("Proxy config[%s] missing local ip", c.Name)
		}
		if c.LocalPort == "" {
			err = fmt.Errorf("Proxy config[%s] missing local port", c.Name)
		}
		if c.RemotePort == "" {
			err = fmt.Errorf("Proxy config[%s] missing remote port", c.Name)
		}
		cfg.ProxyConfigs[c.Name] = c
	}
	return
}
