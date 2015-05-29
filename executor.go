package main

import (
  "os"
  "os/exec"
  "io"
  "fmt"
  "bytes"
  "bufio"
  "time"
  "github.com/elastic/libbeat/common"
  "github.com/elastic/libbeat/logp"
)

const CommandTimeout = 30 * time.Second
const ReadInterval = 1 * time.Second
const ReadTimeout = 1 * time.Second

type Executor struct {
  Command   string /* the command to be run */
  Type      string /* the type to add to events */
}

func (e *Executor) Execute(output chan common.MapStr) {
  cmd := exec.Command(e.Command)
  stdoutbuffer := &bytes.Buffer{}
  cmd.Stdout = stdoutbuffer

  // Start command asynchronously
  err := cmd.Start()
  logp.Err("Error starting command: %s", err)

  // Create a ticker that outputs elapsed time
  ticker := time.NewTicker(ReadInterval)
  go func(ticker *time.Ticker, command string, event_type string, stdoutbuffer *bytes.Buffer, output chan common.MapStr) {
    for _ = range ticker.C {
      reader := bufio.NewReader(stdoutbuffer)
      buffer := new(bytes.Buffer)

      var offset int64 = 0
      var line uint64 = 0

      for {
        text, bytesread, err := readline(reader, buffer, 1 * time.Second)
        if err != nil {
          if err == io.ErrUnexpectedEOF {
            break
          } else if err == io.EOF {
            break
          } else {
            logp.Err("Unexpected state reading from stdout; error: %s\n", err)
            return
          }
        }

        event_timestamp := func() time.Time {
          t := time.Now()
          return t
        }

        line++

        event := common.MapStr{}
        event["source"] = command
        event["offset"] = offset
        event["length"] = bytesread
        event["line"] = line
        event["message"] = text
        event["type"] = event_type

        event.EnsureTimestampField(event_timestamp)
        event.EnsureCountField()

        offset += int64(bytesread)

        output <- event // ship the new event downstream

        logp.Info("\nOutput:\nBytes Read: %v\n%s", bytesread, *text)
      }
    }
  }(ticker, e.Command, e.Type, stdoutbuffer, output)

  // Create a timer that will kill the process
  timer := time.NewTimer(CommandTimeout)
  go func(timer *time.Timer, ticker *time.Ticker, cmd *exec.Cmd) {
    for _ = range timer.C {
      err := cmd.Process.Signal(os.Kill)
      logp.Err("Kill Process Error: %s", err)
      ticker.Stop()
    }
  }(timer, ticker, cmd)

  // Only proceed once the process has finished
  cmd.Wait()
  logp.Info(fmt.Sprintf("%d bytes generated!", len(stdoutbuffer.Bytes())))
}

func readline(reader *bufio.Reader, buffer *bytes.Buffer, eof_timeout time.Duration) (*string, int, error) {
  var is_partial bool = true
  var newline_length int = 1
  start_time := time.Now()

  for {
    segment, err := reader.ReadBytes('\n')

    if segment != nil && len(segment) > 0 {
      if segment[len(segment)-1] == '\n' {
        // Found a complete line
        is_partial = false

        // Check if also a CR present
        if len(segment) > 1 && segment[len(segment)-2] == '\r' {
          newline_length++
        }
      }

      // TODO(sissel): if buffer exceeds a certain length, maybe report an error condition? chop it?
      buffer.Write(segment)
    }

    if err != nil {
      if err == io.EOF && is_partial {
        time.Sleep(1 * time.Second) // TODO(sissel): Implement backoff

        // Give up waiting for data after a certain amount of time.
        // If we time out, return the error (eof)
        if time.Since(start_time) > eof_timeout {
          return nil, 0, err
        }
        continue
      } else {
        //emit("error: Harvester.readLine: %s", err.Error())
        return nil, 0, err // TODO(sissel): don't do this?
      }
    }

    // If we got a full line, return the whole line without the EOL chars (CRLF or LF)
    if !is_partial {
      // Get the str length with the EOL chars (LF or CRLF)
      bufferSize := buffer.Len()
      str := new(string)
      *str = buffer.String()[:bufferSize-newline_length]
      // Reset the buffer for the next line
      buffer.Reset()
      return str, bufferSize, nil
    }
  } /* forever read chunks */

  return nil, 0, nil
}
