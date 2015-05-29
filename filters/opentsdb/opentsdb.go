package opentsdb

import (
  "regexp"
  "strings"
  "github.com/elastic/libbeat/common"
  "filters"
)

type TSDBMetricExp struct {
  *regexp.Regexp
}

var metricExp = TSDBMetricExp{regexp.MustCompile(`^put (?P<metric_name>[\w.]+)[\s]+(?P<metric_timestamp>[0-9]+)[\s]+(?P<metric_value>[0-9.]+)[\s]+(?P<metric_tags>.*$)`)}

func (r *TSDBMetricExp) FindStringSubmatchMap(s string) map[string]string {
  captures := make(map[string]string)

  match := r.FindStringSubmatch(s)
  if match == nil {
    return captures
  }

  for i, name := range r.SubexpNames() {
    if i == 0 {
      continue
    }
    captures[name] = match[i]

  }
  return captures
}

type OpenTSDB struct {
  name string
}

func (opentsdb *OpenTSDB) New(name string, config map[string]interface{}) (filters.FilterPlugin, error) {
  return &OpenTSDB{name: name}, nil
}

//TODO: Check for Errors Here
func (opentsdb *OpenTSDB) Filter(event common.MapStr) (common.MapStr, error) {
  text := event["message"]
  metric_data := metricExp.FindStringSubmatchMap(text.(string))
  parsed_tags := strings.Fields(metric_data["metric_tags"])
  tags := make(map[string]string)

  for  _,v := range parsed_tags {
    tag := strings.Split(v, "=")
    tags[tag[0]] = tag[1]
  }

  event["metric_name"]      = metric_data["metric_name"]
  event["metric_value"]     = metric_data["metric_value"]
  event["metric_timestamp"] = metric_data["metric_timestamp"]
  event["metric_tags"]      = metric_data["metric_tags"]
  event["metric_tags_map"]  = tags

  return event, nil
}

func (opentsdb *OpenTSDB) String() string {
  return opentsdb.name
}

func (opentsdb *OpenTSDB) Type() filters.Filter {
  return filters.OpenTSDBFilter
}
