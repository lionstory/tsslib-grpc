package mobile

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func GenRandConf(configPath string, RemoteMode bool, ServerAddr string) {
	fd, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic("Unable to open config file for writing.")
	}

	type RemoteRandConfig struct {
		Remote bool   `yaml: "remote"`
		Server string `yaml: "server"`
	}

	var config = RemoteRandConfig{}
	config.Server = ServerAddr
	config.Remote = RemoteMode
	// if ServerAddr == "true" {
	// 	config.Remote = true
	// }

	bz, err := yaml.Marshal(&config)
	if err != nil {
		panic("Unable to marshal RemoteRandConfig data.")
	}

	fmt.Println("Config data:", bz)
	_, err = fd.Write(bz)
	if err != nil {
		panic(err)
	}
	fmt.Println("Saved RemoteRandConfig data in file ", configPath)
}
