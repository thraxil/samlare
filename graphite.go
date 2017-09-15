package main

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

type Submitable interface {
	Submit([]metric) error
}

type graphiteServer struct {
	Host string
	Port int
}

func newGraphiteServer(host string, port int) *graphiteServer {
	return &graphiteServer{
		Host: host,
		Port: port,
	}
}

func (g graphiteServer) Submit(metrics []metric) error {
	clientGraphite, err := net.Dial("tcp", fmt.Sprintf("%s:%d", g.Host, g.Port))
	if clientGraphite != nil {
		defer clientGraphite.Close()
	}
	if err != nil {
		return err
	}

	now := int32(time.Now().Unix())
	buffer := bytes.NewBufferString("")

	for _, m := range metrics {
		fmt.Fprintf(buffer, "%s %f %d\n", m.Name, m.Value, now)
	}
	if clientGraphite != nil {
		clientGraphite.Write(buffer.Bytes())
	}
	return nil
}
