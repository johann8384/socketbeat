package listener

import (
    "net"
    "bufio"
    "bytes"
    "strconv"
    "io"
    "fmt"
    "time"
)

type Listener struct {
  Port       int /* the port to listen on */
}

func (l *Listener) Listen(output chan *SocketEvent) {
  var line uint64 = 0

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
                fmt.Printf("couldn't accept: " + err.Error())
                continue
            }
            i++
            fmt.Printf("%d: %v <-> %v\n", i, client.LocalAddr(), client.RemoteAddr())
            ch <- client
        }
    }()
    return ch
}

func (l *Listener) handleConn(client net.Conn, output chan *SocketEvent) {
    reader := bufio.NewReader(client)
    buffer := new(bytes.Buffer)
    var offset int64 = 0
    var line uint64 = 0
    var fields = "{}"
    var read_timeout = 10 * time.Second
    last_read_time := time.Now()
    for {
        text, bytesread, err := l.readline(reader, buffer, read_timeout)

        if err != nil {
          //emit("Unexpected state reading from %s; error: %s\n", client.RemoteAddr(), err)
          return
        }

//        log.Debug("debug %s", text)
        last_read_time = time.Now()

        line++
        event := &SocketEvent{
          Source:   client.RemoteAddr(),
          offset:   offset,
          Line:     line,
          Text:     text,
          Fields:   fields,
        }
        offset += int64(bytesread)

        output <- event // ship the new event downstream
        client.Write(event)
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
