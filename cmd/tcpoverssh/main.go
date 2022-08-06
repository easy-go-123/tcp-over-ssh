package main

import (
	"context"
	"io/ioutil"
	"log"
	"sync"

	"github.com/easy-go-123/tcp-over-ssh/tcpoverssh"
	"gopkg.in/yaml.v3"
)

type SSHProfile struct {
	Host     string `yaml:"Host"`
	Port     int    `yaml:"Port"`
	User     string `yaml:"User"`
	Key      string `yaml:"Key"`
	Password string `yaml:"Password"`
}

type Item struct {
	Listen     string      `yaml:"Listen"`
	Intranet   string      `yaml:"Intranet"`
	SSHProfile *SSHProfile `yaml:"SSHProfile"`
}

type Config struct {
	SSHProfile *SSHProfile `yaml:"SSHProfile"`
	Items      []Item      `yaml:"Items"`
}

func main() {
	d, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalln("read config file failed:", err)
	}

	var cfg Config
	err = yaml.Unmarshal(d, &cfg)

	if err != nil {
		log.Fatalln("parse config file to yaml failed:", err)
	}

	fnMapSSHProfile := func(profile *SSHProfile) *tcpoverssh.SSHClientConfig {
		if profile == nil {
			return nil
		}

		profileRet := &tcpoverssh.SSHClientConfig{
			User:            profile.User,
			Host:            profile.Host,
			Port:            profile.Port,
			HostKeyCallback: nil,
		}

		if len(profile.Password) > 0 {
			profileRet.Passwords = []string{profile.Password}
		}

		if len(profile.Key) > 0 {
			profileRet.Keys = []string{profile.Key}
		}

		return profileRet
	}

	defSSHProfile := fnMapSSHProfile(cfg.SSHProfile)

	wg := sync.WaitGroup{}

	for _, item := range cfg.Items {
		if item.Listen == "" || item.Intranet == "" {
			log.Println("invalid config item: ", item)
		}

		wg.Add(1)

		go func(item Item) {
			defer wg.Done()

			profile := defSSHProfile
			if item.SSHProfile != nil {
				profile = fnMapSSHProfile(item.SSHProfile)
			}

			proxy := tcpoverssh.NewTCPFixProxyOverSSH(context.Background(),
				item.Listen, item.Intranet, *profile)
			if proxy == nil {
				log.Println("createProxy failed")

				return
			}

			proxy.Wait()
		}(item)
	}

	wg.Wait()
}
