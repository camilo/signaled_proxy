package main

import (
	. "github.com/camilo/signaled_proxy/proxy_worker"
	"io"
	. "net"
	"os"
	"os/signal"
	"syscall"
)

var Signals = make(chan os.Signal, 100)

var WorkersPool [10]ProxyWorker
var Connections = make(chan io.ReadWriteCloser, 1024)
var PrimaryAdress, _ = ResolveTCPAddr("tcp4", "localhost:2000")
var SecondaryAdress, _ = ResolveTCPAddr("tcp4", "localhost:2001")

func Start() {
	for i, _ := range WorkersPool {
		WorkersPool[i] = NewProxyWorker(Connections, PrimaryAdress, SecondaryAdress)
		go WorkersPool[i].ProxyThings()
	}
}

func SwitchBackends() {
	for {
		<-Signals
		for i, _ := range WorkersPool {
			WorkersPool[i].Switch <- (struct{}{})
		}
	}
}

func ProxyListen() {

	go SwitchBackends()

	ln, err := Listen("tcp", ":3307")

	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		Connections <- (io.ReadWriteCloser)(conn)
	}
}

func main() {
	signal.Notify(Signals, syscall.SIGQUIT)
	Start()
	ProxyListen()
}
