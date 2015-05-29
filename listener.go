package main

import (
  "net"
  "bufio"
  "bytes"
  "strconv"
  "io"
  "time"
  "github.com/elastic/libbeat/common"
  "github.com/elastic/libbeat/logp"
)

type Listener struct {
  Port       int /* the port to listen on */
  Type       string /* the type to add to events */
}

func (l *Listener) Listen(output chan common.MapStr) {
  server, err := net.Listen("tcp", ":" + strconv.Itoa(l.Port))
  if server == nil {
      panic("couldn't start listening: " + err.Error())
  }
  conns := clientConns(server)
  for {
    go l.handleConn(<-conns, output)
  }
}

func clientConns(listener net.Listener) chan net.Conn {
    ch := make(chan net.Conn)
    i := 0
    go func() {
        for {
            client, err := listener.Accept()
            if client == nil {
                logp.Info("couldn't accept: " + err.Error())
                continue
            }
            i++
            logp.Info("%d: %v <-> %v\n", i, client.LocalAddr(), client.RemoteAddr())
            ch <- client
        }
    }()
    return ch
}

func (l *Listener) handleConn(client net.Conn, output chan common.MapStr) {
    reader := bufio.NewReader(client)
    buffer := new(bytes.Buffer)

    var source string = client.RemoteAddr().String()
    var offset int64 = 0
    var line uint64 = 0
    var read_timeout = 10 * time.Second


    now := func() time.Time {
      t := time.Now()
      return t
      //return t.Format(time.RFC3339)
    }

    for {
        text, bytesread, err := l.readline(reader, buffer, read_timeout)

        if err != nil {
          logp.Info("Unexpected state reading from %v; error: %s\n", client.RemoteAddr().String, err)
          return
        }

        line++

        event := common.MapStr{}
        event["source"] = &source
        event["offset"] = offset
        event["line"] = line
        event["message"] = text
        event["type"] = l.Type

        event.EnsureTimestampField(now)
        event.EnsureCountField()

        offset += int64(bytesread)


        output <- event // ship the new event downstream
        client.Write([]byte("OK"))
    }
}

func (l *Listener) readline(reader *bufio.Reader, buffer *bytes.Buffer, eof_timeout time.Duration) (*string, int, error) {
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
