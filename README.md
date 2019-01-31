# ZABBIX history processing example

Calculates the difference between the last value of an item and the average value for the same element over the past n weeks at the same weekday and hour.

See --help documentation.


## Installation

go get github.com/cbuehlmann/zabbixtools

zabbixtools -config conf/example.yaml -output zabbix_sender_file_name