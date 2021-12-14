package syslog

import (
	"github.com/weitrue/log/utils"
	"net"
	"time"
)

type dialFunc func(network, address string, timeout time.Duration) (net.Conn, error)

type sysConn struct {
	conn       net.Conn
	createTime int64
	lifeTime   int64
	timeOut    time.Duration
}

func (s *sysConn) setTimeout() {
	s.conn.SetDeadline(time.Now().Add(s.timeOut * time.Millisecond))
}

func (s *sysConn) isOld() bool {
	return time.Now().Unix()-s.createTime > s.lifeTime
}

type connPool struct {
	connQueue     *queue
	dialTimeoutFn dialFunc
	timeout       time.Duration //发送超时时间
	raddr         string        //连接地址
	lifeTime      int64         //连接最大生存时间,默认是100毫秒
}

func (cp *connPool) createConn() error {
	conn, err := cp.dialTimeoutFn("tcp", cp.raddr, time.Millisecond*cp.timeout)
	if err != nil {
		_, _ = utils.ErrorOutput(err.Error())
		cp.connQueue.Put("")
		return err
	}
	//将创建的连接放到连接池中
	cp.connQueue.Put(&sysConn{conn: conn, createTime: time.Now().Unix(), lifeTime: cp.lifeTime, timeOut: cp.timeout})
	return nil
}

func (cp *connPool) get() *sysConn {
	var c *sysConn
	length := cp.connQueue.Size()
	if length == 0 {
		cp.createConn()
		length = 1
	}
	for i := 0; i < length; i++ {
		conn, ok := cp.connQueue.Get()
		if !ok { // have no connect to use
			continue
		}
		connect, ok := conn.(*sysConn)
		if !ok { //conn is not sysconn
			continue
		}
		if connect.isOld() {
			connect.conn.Close()
			continue
		}
		c = connect
		break
	}
	return c
}

func (cp *connPool) put(conn *sysConn) {
	cp.connQueue.Put(conn)
}

func (cp *connPool) Close() {
	for !cp.connQueue.Empty() {
		conn, ok := cp.connQueue.Get()
		if !ok { // have no connect to use
			continue
		}
		connect, ok := conn.(*sysConn)
		if !ok { //conn is not sysconn
			continue
		}
		connect.conn.Close()
	}
	cp.connQueue.Close()
}
