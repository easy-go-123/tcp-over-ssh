package tcpoverssh

import (
	"context"
	"net"
	"sync"
)

type FNRemoteDialer func(rAddr string) (net.Conn, error)

func sshDialer(sshConfig SSHClientConfig) FNRemoteDialer {
	return func(rAddr string) (net.Conn, error) {
		sshClient, err := NewSSHClientEx(sshConfig)
		if err != nil {
			return nil, err
		}

		return sshClient.Dial("tcp", rAddr)
	}
}

type TCPFixProxy interface {
	Wait()
	CloseAndWait()
}

func NewTCPFixProxyOverSSH(ctx context.Context, lAddr, rAddr string, sshConfig SSHClientConfig) TCPFixProxy {
	return NewTCPFixProxy(ctx, lAddr, rAddr, sshDialer(sshConfig))
}

func NewTCPFixProxy(ctx context.Context, lAddr, rAddr string, dialer FNRemoteDialer) TCPFixProxy {
	if lAddr == "" || rAddr == "" || dialer == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)

	impl := &tcpFixProxyImpl{
		ctx:         ctx,
		ctxCancel:   cancel,
		lAddr:       lAddr,
		rAddr:       rAddr,
		dialer:      dialer,
		connNewCh:   make(chan net.Conn, 100),
		connCloseCh: make(chan net.Conn, 100),
	}

	if err := impl.init(); err != nil {
		return nil
	}

	return impl
}

type tcpFixProxyImpl struct {
	wg         sync.WaitGroup
	wgListener sync.WaitGroup

	ctx       context.Context
	ctxCancel context.CancelFunc

	lAddr, rAddr string
	dialer       FNRemoteDialer

	listener    net.Listener
	connNewCh   chan net.Conn
	connCloseCh chan net.Conn
}

func (impl *tcpFixProxyImpl) init() (err error) {
	impl.listener, err = net.Listen("tcp", impl.lAddr)
	if err != nil {
		return
	}

	impl.wg.Add(1)

	go impl.mainRoutine()

	impl.wgListener.Add(1)

	go impl.acceptRoutine()

	return
}

func (impl *tcpFixProxyImpl) mainRoutine() {
	defer func() {
		impl.wg.Done()
	}()

	connMap := make(map[net.Conn]interface{})

	loop := true
	for loop {
		select {
		case <-impl.ctx.Done():
			_ = impl.listener.Close()

			impl.wgListener.Wait()

			loop = false

			for conn := range connMap {
				_ = conn.Close()
			}

			continue
		case conn := <-impl.connNewCh:
			connMap[conn] = true
		case conn := <-impl.connCloseCh:
			delete(connMap, conn)
		}
	}
}

func (impl *tcpFixProxyImpl) acceptRoutine() {
	defer impl.wgListener.Done()

	loop := true
	for loop {
		conn, err := impl.listener.Accept()
		if err != nil {
			select {
			case <-impl.ctx.Done():
				loop = false

				continue
			default:
			}
		}

		impl.wg.Add(1)

		go impl.startRequest(conn)
	}
}

func (impl *tcpFixProxyImpl) startRequest(conn net.Conn) {
	defer impl.wg.Done()

	rConn, err := impl.dialer(impl.rAddr)
	if err != nil {
		return
	}

	impl.connNewCh <- conn
	impl.connNewCh <- rConn

	impl.wg.Add(1)

	go impl.pipeRoutine(conn, rConn)

	impl.wg.Add(1)

	go impl.pipeRoutine(rConn, conn)
}

func (impl *tcpFixProxyImpl) pipeRoutine(connFrom, connTo net.Conn) {
	defer func() {
		impl.wg.Done()

		impl.connCloseCh <- connFrom
	}()

	buff := make([]byte, 0xffff)

	for {
		n, err := connFrom.Read(buff)
		if err != nil {
			break
		}

		b := buff[:n]

		_, err = connTo.Write(b)
		if err != nil {
			break
		}
	}
}

func (impl *tcpFixProxyImpl) Wait() {
	impl.wg.Wait()
}

func (impl *tcpFixProxyImpl) CloseAndWait() {
	_ = impl.listener.Close()
	impl.wg.Wait()
}
