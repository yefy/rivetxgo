package tcptests1

import (
	"net"
	"rivetxgo/rivetxcore/gox"
	"rivetxgo/rivetxcore/log"
	"rivetxgo/rivetxcore/tcpx"
	"rivetxgo/rivetxcore/tcpx/tcptestsbase"
	"sync/atomic"
	"time"

	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
)

const clientSendNum = 15000

var clientSendRealNum = 0
var isClose bool

func TcpTests() error {
	addr := "0.0.0.0:43200"
	tcpConf := tcpx.NewTcpConf()
	tcpConf.SocketWriteFlushTime2 = 0
	TcpConfig := tcpx.NewConfig(tcpConf)
	listen, err := tcpx.ListenTcp(addr, TcpConfig, func(connService *tcpx.ConnService) tcpx.Servicer {
		log.LogTcp().Info("tcp_log accept ok addr:%v", connService.Conn.RemoteAddr())
		return NewConn(connService)
	})
	if err != nil {
		return ee.New(err, "Error starting TCP server addr:%v", addr)
	}
	defer listen.Close()

	addr = "127.0.0.1:43200"
	err = tcpx.ConnectTcp(false, addr, TcpConfig, func(connService *tcpx.ConnService) tcpx.Servicer {
		service := NewConn(connService)
		return service
	})
	if err != nil {
		return ee.New(err, "Error starting TCP conn addr:%v", addr)
	}
	isClose = true

	if clientSendRealNum != clientSendNum {
		return ee.New(nil, "clientSendRealNum:%v != clientSendNum:%v", clientSendRealNum, clientSendNum)
	}

	return nil
}

type Conn struct {
	connService *tcpx.ConnService
	ip          string
}

func NewConn(connService *tcpx.ConnService) *Conn {
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

func (conn *Conn) ReadChan(spawnId uint64, msg *tcpx.Msg) error {
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

func (conn *Conn) Write(spawnId uint64, msg *tcpx.Msg) (bool, error) {
	return false, nil
}

func (conn *Conn) WriteErr(spawnId uint64, msg *tcpx.Msg, err error) error {
	return ee.New(err, "conn.ip:%v, ConnAddr:%v",
		conn.ip, conn.connService.ConnAddr)
}

func (conn *Conn) Self() interface{} {
	return conn
}

func (conn *Conn) Close(spawnId uint64, closeType int32) {
	log4.Info("ip:%v, closeType:%v", conn.ip, closeType)
}

func (conn *Conn) ReadTimeout(isCheckTimeout bool) {
	if !isCheckTimeout {
		panic("ReadTimeout")
	}
	log4.Info("ReadTimeout isCheckTimeout:%v", isCheckTimeout)
}

func (conn *Conn) WriteTimeout(isCheckTimeout bool) {
	if !isCheckTimeout {
		panic("WriteTimeout")
	}
	log4.Info("WriteTimeout isCheckTimeout:%v", isCheckTimeout)
}

func (conn *Conn) Client() error {
	connService := conn.connService
	sendNum := atomic.Int32{}
	log4.Info("Client start")

	gox.Spawn(func(u uint64) error {
		for {
			time.Sleep(time.Second * 1)
			log4.Info("Client index:%v", sendNum.Load())
			if isClose {
				break
			}
		}
		return nil
	})

	for i := 0; i < clientSendNum; i++ {
		if connService.IsQuit {
			return nil
		}
		req := int32(i)
		sendNum.Store(req)
		reqData := tcptestsbase.Int32To4Bytes(req)
		//log4.Info("client send req:%v", req)
		for {
			ok := conn.connService.WriteChan(0, tcpx.NewMsgFromBytes(reqData[:]))
			if ok {
				break
			}
		}

		respData := [tcptestsbase.TestDataLen]byte{}
		isClose, err := connService.ReadBytes(respData[:])
		if err != nil {
			return ee.New(err, "conn.ip:%v, ConnAddr:%v",
				conn.ip, conn.connService.ConnAddr)
		}
		if isClose {
			return nil
		}
		resp := tcptestsbase.Bytes4ToInt32(respData)
		if req != resp {
			log4.Error("client recv err req:%v != resp:%v", req, resp)
			break
		}
		clientSendRealNum += 1
	}
	log4.Info("Client end")

	req := tcptestsbase.TestDataQuit
	reqData := tcptestsbase.Int32To4Bytes(req)
	log4.Info("client send req:%v", req)
	for {
		ok := conn.connService.WriteChan(0, tcpx.NewMsgFromBytes(reqData[:]))
		if ok {
			break
		}
	}

	return nil
}

func (conn *Conn) Server() error {
	connService := conn.connService
	for {
		if connService.IsQuit {
			return nil
		}
		repData := [tcptestsbase.TestDataLen]byte{}
		isClose, err := connService.ReadBytes(repData[:])
		if err != nil {
			return ee.New(err, "conn.ip:%v, ConnAddr:%v",
				conn.ip, conn.connService.ConnAddr)
		}
		if isClose {
			return nil
		}
		req := tcptestsbase.Bytes4ToInt32(repData)

		//log4.Info("server recv req:%v", req)
		if req == tcptestsbase.TestDataQuit {
			break
		} else {
			for {
				ok := conn.connService.WriteChan(0, tcpx.NewMsgFromBytes(repData[:]))
				if ok {
					break
				}
			}
		}
	}

	return nil
}
