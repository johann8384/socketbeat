package stdout

import (
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "time"

  "github.com/elastic/libbeat/common"
  "github.com/elastic/libbeat/logp"
  "github.com/elastic/libbeat/outputs"
)


func (out *FileOutput) Init(config outputs.MothershipConfig, topology_expire int) error {
  // not supported by this output type
  return nil
}

func (out *FileOutput) PublishIPs(name string, localAddrs []string) error {
  // not supported by this output type
  return nil
}

func (out *FileOutput) GetNameByIP(ip string) string {
  // not supported by this output type
  return ""
}

func (out *FileOutput) PublishEvent(ts time.Time, event common.MapStr) error {

  json_event, err := json.Marshal(event)
  if err != nil {
    logp.Err("Fail to convert the event to JSON: %s", err)
    return err
  }

  err = fmt.printf("%s", json_event)
  if err != nil {
    return err
  }

  return nil
}
