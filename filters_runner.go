package main

import (
  "fmt"

  "github.com/elastic/libbeat/common"
  "github.com/elastic/libbeat/filters"
  "github.com/elastic/libbeat/logp"
)

// Executes the filters
type FilterRunner struct {
  FiltersQueue chan common.MapStr
  results      chan common.MapStr

  // The order in which the plugins are
  // executed. A filter plugin can be loaded
  // more than once.
  order []filters.FilterPlugin
}

// Goroutine that reads the objects from the FiltersQueue,
// executes all filters on them and writes the modified objects
// int he results channel.
func (runner *FilterRunner) Run() error {
  for event := range runner.FiltersQueue {
    for _, plugin := range runner.order {
      var err error
      event, err = plugin.Filter(event)
      if err != nil {
        logp.Err("Error executing filter %s: %v. Dropping event.", plugin, err)
        break // drop event in case of errors
      }
    }

    runner.results <- event
  }
  return nil
}

// Create a new FilterRunner
func NewFilterRunner(results chan common.MapStr, order []filters.FilterPlugin) *FilterRunner {
  runner := new(FilterRunner)
  runner.results = results
  runner.order = order
  runner.FiltersQueue = make(chan common.MapStr, 1000)
  return runner
}

// LoadConfiguredFilters interprets the [filters] configuration, loads the configured
// plugins and returns the order in which they need to be executed.
func LoadConfiguredFilters(config map[string]interface{}) ([]filters.FilterPlugin, error) {
  var err error
  plugins := []filters.FilterPlugin{}

  filters_list, exists := config["filters"]
  if !exists {
    return plugins, nil
  }
  filters_iface, ok := filters_list.([]interface{})
  if !ok {
    return nil, fmt.Errorf("Expected the filters to be an array of strings")
  }

  for _, filter_iface := range filters_iface {
    filter, ok := filter_iface.(string)
    if !ok {
      return nil, fmt.Errorf("Expected the filters array to only contain strings")
    }
    cfg, exists := config[filter]
    var plugin_type filters.Filter
    var plugin_config map[string]interface{}
    if !exists {
      // Maybe default configuration by name
      plugin_type, err = filters.FilterFromName(filter)
      if err != nil {
        return nil, fmt.Errorf("No such filter type and no corresponding configuration: %s", filter)
      }
    } else {
      logp.Debug("filters", "%v", cfg)
      plugin_config, ok := cfg.(map[interface{}]interface{})
      if !ok {
        return nil, fmt.Errorf("Invalid configuration for: %s", filter)
      }
      type_str, ok := plugin_config["type"].(string)
      if !ok {
        return nil, fmt.Errorf("Couldn't get type for filter: %s", filter)
      }
      plugin_type, err = filters.FilterFromName(type_str)
      if err != nil {
        return nil, fmt.Errorf("No such filter type: %s", type_str)
      }
    }

    filter_plugin := filters.Filters.Get(plugin_type)
    if filter_plugin == nil {
      return nil, fmt.Errorf("No plugin loaded for %s", plugin_type)
    }
    plugin, err := filter_plugin.New(filter, plugin_config)
    if err != nil {
      return nil, fmt.Errorf("Initializing filter plugin %s failed: %v",
        plugin_type, err)
    }
    plugins = append(plugins, plugin)

  }

  return plugins, nil
}
