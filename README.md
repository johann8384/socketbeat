# socketbeat
Beat for Collecting events from TCP

Next steps are to integrate libbeat for filter/output stages.

I am probably going to rename thie to MetricsBeat and then start heading down the path described [here](https://discuss.elastic.co/t/metrics-beat/1495/2).

Having [harvesters](https://github.com/elastic/logstash-forwarder/blob/master/harvester.go|harvesters) that execute scripts (like TCollector and it's Collectors) then use the standard libbeat logic to ship the lines of output as events would be valuable. 

For my needs I'll need to add a Kafka output to the outputs in https://github.com/elastic/libbeat.
