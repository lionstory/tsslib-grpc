package config

import (
	"gopkg.in/yaml.v2"
	"os"
)

type TssConfig struct {
	Router      []string `yaml:"router"`
	SavePath    string   `yaml:"savepath"`
	PartyNum    int      `yaml:"partyNum"`
	Threshold   int      `yaml:"threshold"`
	KeyRevision int      `yaml:"keyrevision"`
}

func LoadConfig(path string) (*TssConfig, error) {
	tss_config := TssConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &tss_config)
	if err != nil {
		return nil, err
	}
	return &tss_config, nil
}
