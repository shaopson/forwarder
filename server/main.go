package main

import (
	"flag"
	"fmt"
	"github.com/dev-shao/iprp/log"
	"gopkg.in/ini.v1"
	"os"
)

// var DefaultConfig = &ServerConfig{}
var LogFileSize int64 = 1024 * 1024 //1m
var LogBackupCount = 5

func main() {
	iniFile := flag.String("c", "", "ini config file path")
	//debug := flag.Bool("d",false, "debug mode")
	flag.Parse()

	if *iniFile == "" {
		fmt.Println("Missing configuration file")
		os.Exit(1)
	}
	cfg, err := parseConfig(*iniFile)
	if err != nil {
		fmt.Println("Parse config error:", err)
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
	server := NewServer(cfg)
	err = server.Run()
	if err != nil {
		log.Error("server over:%s", err)
	}
}

func parseConfig(cfgFile string) (cfg *ServerConfig, err error) {
	iniFile, err := ini.Load(cfgFile)
	if err != nil {
		return
	}
	cfg = &ServerConfig{}
	section := iniFile.Section("")
	err = section.MapTo(cfg)
	if err != nil {
		return nil, err
	}
	if len(cfg.Port) == 0 {
		err = fmt.Errorf("The config file '%s' missing 'bind_port'")
	}
	return
}
