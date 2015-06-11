package main

import (
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "strings"
  "gopkg.in/yaml.v2"
  "filters/opentsdb"
  "filters"
  "github.com/elastic/packetbeat/config"
  "github.com/elastic/libbeat/common"
  "github.com/elastic/libbeat/logp"
  "github.com/elastic/libbeat/publisher"
)

const PORT = 3540
const CommandTimeout = 15
const ReadInterval = 500
const ReadTimeout = 1

var EnabledFilterPlugins map[filters.Filter]filters.FilterPlugin = map[filters.Filter]filters.FilterPlugin{
  filters.OpenTSDBFilter: new(opentsdb.OpenTSDB),
}

func main() {

  // Use our own FlagSet, because some libraries pollute the global one
  var cmdLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

  verbose := cmdLine.Bool("v", false, "Log at INFO level")
  toStderr := cmdLine.Bool("e", false, "Output to stdout instead of syslog")
  tcpPort := cmdLine.Int("p", 4243, "Port to Listen On")
  configfile := cmdLine.String("c", "./packetbeat.yml", "Configuration file")
  publishDisabled := cmdLine.Bool("N", false, "Disable actual publishing for testing")
  debugSelectorsStr := cmdLine.String("d", "", "Enable certain debug selectors")
  cmdLine.Parse(os.Args[1:])

  logLevel := logp.LOG_ERR

  if *verbose {
    logLevel = logp.LOG_INFO
  }

  var err error

  filecontent, err := ioutil.ReadFile(*configfile)
  if err != nil {
    fmt.Printf("Fail to read %s: %s. Exiting.\n", *configfile, err)
    return
  }
  if err = yaml.Unmarshal(filecontent, &config.ConfigSingleton); err != nil {
    fmt.Printf("YAML config parsing failed on %s: %s. Exiting.\n", *configfile, err)
    return
  }

  debugSelectors := []string{}
  if len(*debugSelectorsStr) > 0 {
    debugSelectors = strings.Split(*debugSelectorsStr, ",")
    logLevel = logp.LOG_DEBUG
  }

  if len(debugSelectors) == 0 {
    debugSelectors = config.ConfigSingleton.Logging.Selectors
  }
  logp.LogInit(logp.Priority(logLevel), "", !*toStderr, true, debugSelectors)

  logp.Info("Listening: %d", tcpPort)

  logp.Debug("main", "Configuration %s", config.ConfigSingleton)
  logp.Info("main", "Initializing output plugins")
  if err = publisher.Publisher.Init(*publishDisabled, config.ConfigSingleton.Output,
    config.ConfigSingleton.Shipper); err != nil {

    logp.Critical(err.Error())
    os.Exit(1)
  }

  logp.Info("Initializing filters plugins: %v", EnabledFilterPlugins)
  for filter, plugin := range EnabledFilterPlugins {
    logp.Info("Plugin Registered: %s", filter)
    filters.Filters.Register(filter, plugin)
  }
//  logp.Debug("Filter Config: %v", config.ConfigSingleton.Filter)
  filters_plugins, err :=
    LoadConfiguredFilters(config.ConfigSingleton.Filter)
  if err != nil {
    logp.Err("Error loading filters plugins: %v", err)
    os.Exit(1)
  }
  //logp.Debug("Filters plugins order: %v", filters_plugins)

  var afterInputsQueue chan common.MapStr

  if len(filters_plugins) > 0 {
    runner := NewFilterRunner(publisher.Publisher.Queue, filters_plugins)
    go func() {
      err := runner.Run()
      if err != nil {
        logp.Err("Filters runner failed: %v", err)
        // shutting doen
      }
    }()
    afterInputsQueue = runner.FiltersQueue
  } else {
    logp.Info("Short-circuit the filter runner")
    // short-circuit the runner
    afterInputsQueue = publisher.Publisher.Queue
  }

  if !*toStderr {
    logp.Info("Startup successful, sending output only to syslog from now on")
    logp.SetToStderr(false)
  }

  //executor := Executor{Command: "./fake.sh", Type: "exectutor", CommandTimeout: CommandTimeout * time.Second, ReadInterval: ReadInterval * time.Millisecond, ReadTimeout: ReadTimeout * time.Second}
  //go executor.Execute(publisher.Publisher.Queue)
  listener := Listener{Port: *tcpPort, Type: "tcollector"}
  go listener.Listen(publisher.Publisher.Queue)
  for {
    event := <-afterInputsQueue
    logp.Info("Event: %v", event)
  }
}
