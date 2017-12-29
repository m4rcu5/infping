# Build
Comming soon
[![Build Status](https://travis-ci.org/LanetNetwork/infping.svg?branch=master)](https://travis-ci.org/LanetNetwork/infping)

infping
===========

Description
-----------

Parse fping output, store result in influxdb.

Render graphs from influxdb in "SmokePing Style"
Grafana Dashboard: https://grafana.com/dashboards/3429

Building
--------

### Prerequisites
  * go get github.com/influxdata/influxdb
  * go get github.com/AlekSi/zabbix-sender
  * go get github.com/pelletier/go-toml 


### Compiling

Build into subdir dist
    `go build -o dist/infping  infping/main.go`
    `go build -o dist/infhttp  infhttp/main.go`


Configuration
-------------

See config.toml.examle


Distribution and Contribution
-----------------------------

Distributed under terms and conditions of MIT (only).


Recent Developers:
    
* Anton Baranov &lt;cryol@cryol.kiev.ua&gt;
* Marcus van Dam https://github.com/m4rcu5

Idea and initial release:
* Tor Hveem https://github.com/torhve
