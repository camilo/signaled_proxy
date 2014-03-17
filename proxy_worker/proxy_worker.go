package proxy_worker

import . "net"
import (
	"fmt"
	"io"
	"net"
)

type ProxyWorker struct {
	Connections      chan (io.ReadWriteCloser)
	PrimaryBackend   *BackendServer
	SecondaryBackend *BackendServer
	Switch           chan (struct{})
}

type BackendServer struct {
	Address Addr
	active  bool
	socket  io.WriteCloser
	err     error
}

func (b *BackendServer) connect() {
	socket, err := DialTCP("tcp4", nil, (*b).Address.(*TCPAddr))
	b.socket = socket
	b.err = err
}

func (b *BackendServer) close() {
	err := b.socket.Close()
	b.err = err
}

func (b *BackendServer) write(buff []byte) {
	_, err := b.socket.Write(buff)

	if err != nil {
		b.err = err
	}
}

func NewProxyWorker(connections chan io.ReadWriteCloser, primaryAddress net.Addr, secondaryAddres net.Addr) ProxyWorker {
	return ProxyWorker{connections,
		&BackendServer{Address: primaryAddress},
		&BackendServer{Address: secondaryAddres},
		make(chan struct{}, 1)}
}

func (p *ProxyWorker) checkBackendErrors(context string) {
	if p.PrimaryBackend.err != nil || p.SecondaryBackend.err != nil {
		panic("Error" + context + " to one of the backeds")
	}
}

func (p *ProxyWorker) connectToBackends() {
	p.PrimaryBackend.connect()
	p.SecondaryBackend.connect()
	p.checkBackendErrors("connect")
	p.PrimaryBackend.active = true
	p.SecondaryBackend.active = false
}

func (p *ProxyWorker) closeBackends() {
	p.PrimaryBackend.close()
	p.SecondaryBackend.close()
	p.checkBackendErrors("connect")
}

func (p *ProxyWorker) switchIfNeeded() {
	for {
		<-p.Switch
		fmt.Println("SWITCH")
		p.PrimaryBackend.active = !p.PrimaryBackend.active
		p.SecondaryBackend.active = !p.SecondaryBackend.active
	}
}

func (p *ProxyWorker) writeToBackends(buff []byte) {
	if p.PrimaryBackend.active {
		p.PrimaryBackend.write(buff)
	}

	if p.SecondaryBackend.active {
		p.SecondaryBackend.write(buff)
	}

	p.checkBackendErrors("write")
}

func (p *ProxyWorker) ProxyThings() {
	go p.switchIfNeeded()
	p.connectToBackends()

	for {
		c := <-p.Connections
		for {
			// ew gross over alloc
			buff := make([]byte, 1024)
			_, err := c.Read(buff)

			if err != nil {

				if err == io.EOF {
					fmt.Println(err)
					c.Close()
					break
				} else {
					panic(err)
				}
			}

			p.writeToBackends(buff)
		}
	}
}
