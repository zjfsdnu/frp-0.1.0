package conn

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"

	"frp/utils/log"
)

type Listener struct {
	addr      net.Addr
	l         *net.TCPListener
	conn      chan *Conn
	closeFlag bool
}

func Listen(bindAddr string, bindPort int64) (l *Listener, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return l, err
	}

	l = &Listener{
		addr:      listener.Addr(),
		l:         listener,
		conn:      make(chan *Conn),
		closeFlag: false,
	}

	go func() {
		for {
			conn, err := l.l.AcceptTCP()
			if err != nil {
				if l.closeFlag {
					return
				}
				continue
			}

			c := &Conn{
				TcpConn:   conn,
				closeFlag: false,
			}
			c.Reader = bufio.NewReader(c.TcpConn)
			l.conn <- c
		}
	}()
	return l, err
}

// wait util get one new connection or listener is closed
// if listener is closed, err returned
func (l *Listener) GetConn() (*Conn, error) {
	if conn, ok := <-l.conn; !ok {
		return nil, fmt.Errorf("channel close")
	} else {
		return conn, nil
	}
}

func (l *Listener) Close() {
	if l.l != nil && l.closeFlag == false {
		l.closeFlag = true
		_ = l.l.Close()
		close(l.conn)
	}
}

// wrap for TCPConn
type Conn struct {
	TcpConn   *net.TCPConn
	Reader    *bufio.Reader
	closeFlag bool
}

func ConnectServer(host string, port int64) (c *Conn, err error) {
	c = &Conn{}
	serverAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return
	}
	conn, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		return
	}
	c.TcpConn = conn
	c.Reader = bufio.NewReader(c.TcpConn)
	c.closeFlag = false
	return c, nil
}

func (c *Conn) GetRemoteAddr() (addr string) {
	return c.TcpConn.RemoteAddr().String()
}

func (c *Conn) GetLocalAddr() (addr string) {
	return c.TcpConn.LocalAddr().String()
}

func (c *Conn) ReadLine() (buff string, err error) {
	buff, err = c.Reader.ReadString('\n')
	if err == io.EOF {
		c.closeFlag = true
	}
	return buff, err
}

func (c *Conn) Write(content string) (err error) {
	_, err = c.TcpConn.Write([]byte(content))
	return err
}

func (c *Conn) Close() {
	if c.TcpConn != nil && c.closeFlag == false {
		c.closeFlag = true
		_ = c.TcpConn.Close()
	}
}

func (c *Conn) IsClosed() bool {
	return c.closeFlag
}

// will block until connection close
func Join(c1 *Conn, c2 *Conn) {
	var wait sync.WaitGroup
	pipe := func(to *Conn, from *Conn) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()

		var err error
		_, err = io.Copy(to.TcpConn, from.TcpConn)
		if err != nil {
			log.Warn("join conn error, %v", err)
		}
	}

	wait.Add(2)
	go pipe(c1, c2)
	go pipe(c2, c1)
	wait.Wait()
	return
}
