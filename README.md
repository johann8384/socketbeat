# socketbeat
Beat for Collecting events from TCP

I am probably going to rename this to MetricsBeat and then start heading down the path described [here](https://discuss.elastic.co/t/metrics-beat/1495/2).

Having [harvesters](https://github.com/elastic/logstash-forwarder/blob/master/harvester.go|harvesters) that execute scripts (like TCollector and it's Collectors) then use the standard libbeat logic to ship the lines of output as events would be valuable. 

For my needs I'll need to add a Kafka output to the [outputs](https://github.com/johann8384/libbeat).

Currently I have very rough code which accepts TCP connections and ships off the lines. It assumes the input is lines intended for OpenTSDB, so the output of a TCollector Collector.

The next steps are to implement something similar to the Prospector/Harvester model like logstash-forwarder users. A prospector will execute commands as configured at various time intervals (like TCollector does). The harvester will contain the running script and will log errors from it's stderr and pass lines from it's stdout to the event channel.

I've made a start on this with the listener.go and executor.go files, they should be the first two harvestors. Listener will listen for TCP connections and will ship each line as an event, the Executor will execute a command and ship each line of stdout as an event. The intention is for the Executor to be able to replace [TCollector](https://github.com/opentsdb/tcollector). I intend for a prospector to be able to read folders name as numbers (0, 15, 30, 60, etc) and execute the executable scripts within those folders on intervals based on their name using Executors. You can see this in the adding_filter_plugins branch. The filter doesn't work there yet, but the Executor does.

From there I would want to move the parsing logic to a filter and make it configurable to read Graphite, OpenTSDB and other formats and send standardized events with "message" being the original line and fields added as appropriate based on what format it is parsing.

Here is one OpenTSDB line parsed with the current code:
```
{
  count: 1,
  line: 2,
  message: "put kafka.producer.ProducerRequestMetrics.ProducerRequestRateAndTimeMs.98percentile 1432876501 0.11 serverid=ps515 server1010.dc.example.com=brokerPort brokerHost=server1010.dc3.example.com domain=dc3 host=server1001 machine_class=server",
  metric_name: "kafka.producer.ProducerRequestMetrics.ProducerRequestRateAndTimeMs.98percentile",
  metric_tags: "serverid=ps515 server1010_dc3_example_com=brokerPort brokerHost=server1010_dc3_example_com domain=dc3 host=server1010 machine_class=server",
  metric_tags_map: {
    server1010_dc3_example_com: "brokerPort",
    brokerHost: "server1010_dc3_example_com",
    domain: "dc3",
    host: "server1010",
    machine_class: "server",
    serverid: "ps515"
  },
  metric_timestamp: "1432876501",
  metric_value: "0.11",
  offset: 213,
  shipper: "C02MR0K3FD58",
  source: "127.0.0.1:59735",
  timestamp: "2015-05-29T05:15:02.427Z",
  type: "tcollector"
}
```
This uses code from Packetbeat and from Logstash-Forwarder. It is no where near even an initial release. It's just really rough PoC code.
