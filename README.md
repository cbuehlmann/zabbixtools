# ZABBIX history processing example

[![Build Status](https://travis-ci.org/cbuehlmann/redirect.svg?branch=master)](https://travis-ci.org/cbuehlmann/redirect)

Calculates the difference between the last value of an item and the average value for the same element over the past n weeks at the same weekday and hour.

## Installation

go get github.com/cbuehlmann/zabbixtools

zabbixtools -config conf/example.yaml -output zabbix_sender_file_name

## Usage

See [conf/example.yaml](conf/example.yaml) and zabbixtools --help.