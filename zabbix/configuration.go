package zabbix

import (
	log "github.com/inconshreveable/log15"
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

		Sender struct {
			Host   string
			Port   int
			Binary string
		}
	}

	// Find hosts via template
	Templates []TemplateFilterConfiguration `yaml:"templates"`

	// Find hosts
	Hosts []HostFilterConfiguration `yaml:"hosts"`

	// Filter items on found hosts
	Items []ItemConfiguration `yaml:"items"`
}

type TemplateFilterConfiguration struct {
	Filter map[string][]string
	Search map[string][]string
}

type HostFilterConfiguration struct {
	Filter map[string][]string
	Search map[string][]string
}

type ItemConfiguration struct {
	Filter    map[string][]string
	Search    map[string][]string
	PastWeeks PastWeeksAlgorithmConfiguration
	Postfix   string
}

type PastWeeksAlgorithmConfiguration struct {
	Weeks  int
	Window int64
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
	Log.Debug("parsing yaml from", log.Ctx{"filename": file.Name(), "content": string(data)})

	err = yaml.Unmarshal([]byte(data), &configuration)
	if err != nil {
		return configuration, err
	}

	Log.Debug("parsed yaml", log.Ctx{"structure": configuration})

	return configuration, nil
}
