package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type Configuration struct {
	Zabbix struct {
		Api struct {
			URL      string `yaml:"URL"`
			Username string
			Password string
		}

		Trapper struct {
			Host string
			Port int
		}
	}

	ItemFilter []struct {
		ItemFilter struct {
			TemplateName    string `yaml:"template-name"`
			ItemFilter      string `yaml:"item-name"`
			ItemFilterRegex string `yaml:"item-key-regex"`
		}
	} `yaml:"filter"`

	Algorithm interface{} `yaml:"algo"`
}

func ReadConfigurationFromFile(filename string) (Configuration, error) {

	file, err := os.Open(filename)
	if err != nil {
		return Configuration{}, err
	}

	info, err := os.Stat(file.Name())

	data := make([]byte, info.Size())
	_, err = file.Read(data)
	if err != nil {
		return Configuration{}, err
	}

	configuration := Configuration{}
	fmt.Printf("parsing yaml from %s %s\n", file.Name(), string(data))

	err = yaml.Unmarshal([]byte(data), &configuration)
	if err != nil {
		return configuration, err
	}

	return configuration, nil
}
