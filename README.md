check_redis_go
----
Nagios plugin for Redis created by golang


## Usage

```sh
$ go build check_redis.go
$ ./check_redis -host localhost -port 6379 -timeout 1000 -metrics used_memory:1073741824:2147483648:gt,mem_fragmentation_ratio:1.5::gt,mem_fragmentation_ratio::1:lt
[OK]Connection and Authorization Ok
[OK]used_memory:543296.0 gt (warn1073741824.0 crit2147483648.0)
[WARN]mem_fragmentation_ratio:4.7 gt (warn1.5)
[OK]mem_fragmentation_ratio:4.7 lt (crit1.0)
$ echo $?
1 # warning
```
## Options

Option|Description
:--|:--
-h\|-host|	redis server host (default "127.0.0.1")
-p\|-port|	port to connect redis (default 6379)
-password|	redis server password
-timeout|	connection timeout(msec) (default 1000)
-metrics|	metrics and threshold to be monitored "metricname:warnstring:criticalstring,..." (e.g. used_memory:1073741824:2147483648:gt,metric2:10:20:lt..)


## Setting Examples of Nagios

```
command[check_redis]=/usr/lib64/nagios/plugins/check_redis -host $ARG1$ -port $ARG2$ -timeout $ARG3$ -metrics $ARG4$
define service {
       use                 　prod-service
       hostgroups       　   redishosts
       service_description　Redis
       check_command        check_nrpe!check_redis!localhost 6379 1000 used_memory:1073741824:2147483648:gt,mem_fragmentation_ratio:1.5::gt,mem_fragmentation_ratio::1:lt
}
```
