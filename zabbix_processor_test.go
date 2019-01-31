package main

import (
	"github.com/cbuehlmann/zabbixtools/zabbix"
	"testing"
)

func TestZabbixSenderInMain(t *testing.T) {
	configuration := zabbix.Configuration{}
	configuration.Zabbix.Sender.Host = "192.168.109.51"
	configuration.Zabbix.Sender.Binary = "D:/tools/zabbix_sender.exe"
	sendItemData(configuration, "out.zbx", true)
}
