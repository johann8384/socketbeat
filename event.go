package main

import (
   "fmt"
   "strconv"
)
type SocketEvent struct {
  Source *string `json:"source,omitempty"`
  Offset int64   `json:"offset,omitempty"`
  Line   uint64  `json:"line,omitempty"`
  Text   *string `json:"text,omitempty"`
  Fields *map[string]string
}

func (event *SocketEvent) Print() {
  writeKV("source", *event.Source)
  writeKV("offest", strconv.FormatInt(event.Offset, 10))
  writeKV("line", *event.Text)
  for k, v := range *event.Fields {
    writeKV(k, v)
  }
}

func writeKV(key string, value string) {
  fmt.Printf("kv: %s %s\n", key, value)
}
