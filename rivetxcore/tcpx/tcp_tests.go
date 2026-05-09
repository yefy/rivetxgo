package tcpx

import (
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"net"
	"rivetxgo/rivetxcore/gox"
	"rivetxgo/rivetxcore/log"
)

const clientSend = "abcd"
const serverSend = "dcba"
const serverSendErr = "errs"
const connQuit = "quit"
const clientSendNum = 15000

var clientSendRealNum = 0

func TcpTests() error {
	addr := "0.0.0.0:43200"
	tcpConf := NewTcpConf()
	tcpConf.SocketWriteFlushTime2 = 0
	TcpConfig := NewConfig(tcpConf)
	listen, err := ListenTcp(addr, TcpConfig, func(connService *ConnService) Servicer {
		log.LogTcp().Info("tcp_log accept ok addr:%v", connService.Conn.RemoteAddr())
		return NewConn(connService)
	})
	if err != nil {
		return ee.New(err, "Error starting TCP server addr:%v", addr)
	}
	defer listen.Close()

	addr = "127.0.0.1:43200"
	err = ConnectTcp(false, addr, TcpConfig, func(connService *ConnService) Servicer {
		service := NewConn(connService)
		return service
	})
	if err != nil {
		return ee.New(err, "Error starting TCP conn addr:%v", addr)
	}

	if clientSendRealNum != clientSendNum {
		return ee.New(nil, "clientSendRealNum:%v != clientSendNum:%v", clientSendRealNum, clientSendNum)
	}

	return nil
}

type Conn struct {
	connService *ConnService
	ip          string
}

func NewConn(connService *ConnService) *Conn {
	conn := &Conn{connService: connService}
	return conn
}

func (conn *Conn) Init(spawnId uint64) error {
	addr := conn.connService.Conn.RemoteAddr()
	ip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return ee.New(err, "net.SplitHostPort addr: %v", addr)
	}
	conn.ip = ip
	log4.Trace("RemoteAddr:%v", addr)

	return nil
}

func (conn *Conn) Start(spawnId uint64) error {
	return nil
}

func (conn *Conn) ReadChan(spawnId uint64, msg *Msg) error {
	log4.Error("Conn.ReadChan, ip:%v", conn.ip)
	return nil
}

func (conn *Conn) Read(spawnId uint64) error {
	if conn.connService.IsAccept {
		return conn.Server()
	} else {
		return conn.Client()
	}
}

func (conn *Conn) Write(spawnId uint64, msg *Msg) (bool, error) {
	return false, nil
}

func (conn *Conn) WriteErr(spawnId uint64, msg *Msg, err error) error {
	return ee.New(err, "conn.ip:%v, ConnAddr:%v",
		conn.ip, conn.connService.ConnAddr)
}

func (conn *Conn) Self() interface{} {
	return conn
}

func (conn *Conn) Close(spawnId uint64, closeType int32) {
	log4.Info("ip:%v, closeType:%v", conn.ip, closeType)
}

func (conn *Conn) Client() error {
	gox.StatNumStartAdd("tcp_test_client")
	defer gox.StatNumEndAdd("tcp_test_client")

	connService := conn.connService
	socketReadTimeout := int64(1000 * 60)
	for i := 0; i < clientSendNum; i++ {
		if connService.IsQuit {
			return nil
		}
		cmd := []byte(clientSend)
		//log4.Info("client send cmd:%v", string(cmd))
		conn.connService.WriteChan(0, NewMsgFromBytes(cmd))

		isClose, err := connService.ReadBytes(cmd, socketReadTimeout)
		if err != nil {
			return ee.New(err, "conn.ip:%v, ConnAddr:%v",
				conn.ip, conn.connService.ConnAddr)
		}
		if isClose {
			return nil
		}
		if string(cmd) != serverSend {
			log4.Error("client recv err cmd:%v != serverSend:%v", string(cmd), serverSend)
			break
		}
		clientSendRealNum += 1
		//log4.Info("client recv cmd:%v", string(cmd))
	}

	cmd := []byte(connQuit)
	log4.Info("client send cmd:%v", string(cmd))
	conn.connService.WriteChan(0, NewMsgFromBytes(cmd))

	return nil
}

func (conn *Conn) Server() error {
	gox.StatNumStartAdd("tcp_test_server")
	defer gox.StatNumEndAdd("tcp_test_server")

	connService := conn.connService
	socketReadTimeout := int64(1000 * 60)
	for {
		if connService.IsQuit {
			return nil
		}
		cmd := make([]byte, 4, 4)
		isClose, err := connService.ReadBytes(cmd, socketReadTimeout)
		if err != nil {
			return ee.New(err, "conn.ip:%v, ConnAddr:%v",
				conn.ip, conn.connService.ConnAddr)
		}
		if isClose {
			return nil
		}

		//log4.Info("server recv cmd:%v", string(cmd))
		if string(cmd) == connQuit {
			conn.connService.WriteChan(0, NewMsgFromBytes([]byte(serverSend)))
			break
		} else if string(cmd) == clientSend {
			conn.connService.WriteChan(0, NewMsgFromBytes([]byte(serverSend)))
		} else {
			conn.connService.WriteChan(0, NewMsgFromBytes([]byte(serverSendErr)))
		}
	}

	return nil
}
