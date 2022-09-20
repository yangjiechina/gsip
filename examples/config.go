package examples

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Client ClientConfig `json:"client"`
	Server ServerConfig `json:"server"`
}

type ClientConfig struct {
	ListenIP            string `json:"listenIP"`
	ListenPort          int    `json:"listenPort"`
	Transport           string `json:"transport"`
	PortSharedMode      bool   `json:"portSharedMode"` //共用一个端口
	ParentIP            string `json:"parentIP"`
	ParentPort          int    `json:"parentPort"`
	ParentId            string `json:"parentId"`
	Password            string `json:"password"`
	Count               int    `json:"count"`
	Heartbeat           int    `json:"heartbeat"`
	RegisterExpires     int    `json:"registerExpires"`
	ActiveAlarmInterval int    `json:"activeAlarmInterval"`
	Sleep               int    `json:"sleep"`
}

type ServerConfig struct {
	ListenIP   string `json:"listenIP"`
	ListenPort int    `json:"listenPort"`
	SipId      string `json:"sipId"`
	Password   string `json:"password"`
}

func ReadConfig(path string) (*Config, error) {
	config := &Config{
		Client: ClientConfig{
			ListenIP:        "0.0.0.0",
			Transport:       "UDP",
			PortSharedMode:  true,
			Password:        "12345678",
			Count:           1,
			Heartbeat:       60,
			RegisterExpires: 3600,
		},
		Server: ServerConfig{
			ListenIP:   "0.0.0.0",
			ListenPort: 5060,
			SipId:      "34020000002000000001",
			Password:   "12345678",
		},
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(file, config); err != nil {
		return nil, err
	}

	return config, nil
}
