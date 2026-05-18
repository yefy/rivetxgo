package tcptests2

import (
	"math/rand"
	"net"
	"rivetxgo/rivetxcore/gox"
	"rivetxgo/rivetxcore/log"
	"rivetxgo/rivetxcore/syncx"
	"rivetxgo/rivetxcore/tcpx"
	"rivetxgo/rivetxcore/tcpx/tcptestsbase"
	"sync/atomic"
	"time"

	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
)

const clientSendNum = 5179928

//const clientSendNum = 179928

var clientSendRealNum = 0
var isClose bool
var recvNum atomic.Int32

func TcpTests() error {
	addr := "0.0.0.0:43201"
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

	addr = "127.0.0.1:43201"
	err = tcpx.ConnectTcpSync(false, addr, TcpConfig, func(connService *tcpx.ConnService) tcpx.Servicer {
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
	log4.Info("Client start")
	tg := syncx.NewTaskGroup()
	tg.Add(1)
	gox.Spawn(func(u uint64) error {
		defer tg.Done()
		lastNum := int32(-1)
		for {
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
			//log4.Info("Client recv resp:%v", resp)
			if resp == tcptestsbase.TestDataQuit {
				return nil
			} else {
				if resp != lastNum+1 {
					log4.Error("client recv err resp:%v != lastNum + 1:%v", resp, lastNum+1)
					return nil
				}
				lastNum = resp
				clientSendRealNum += 1
			}
		}
	})
	sendNum := atomic.Int32{}

	gox.Spawn(func(u uint64) error {
		for {
			time.Sleep(time.Second * 1)
			log4.Info("Client sendNum:%v, server recvNum:%v", sendNum.Load(), recvNum.Load())
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
		reqData := tcptestsbase.Int32To4Bytes(req)

		sendNum.Store(req)

		//log4.Info("client send req:%v", req)

		for {
			ok := conn.connService.WriteChan(0, tcpx.NewMsgFromBytes(reqData[:]))
			if ok {
				break
			}
		}
	}
	req := tcptestsbase.TestDataQuit
	reqData := tcptestsbase.Int32To4Bytes(req)
	log4.Info("client send quit req:%v", req)

	for {
		ok := conn.connService.WriteChan(0, tcpx.NewMsgFromBytes(reqData[:]))
		if ok {
			break
		}
	}

	tg.Wait()
	log4.Info("Client end")

	return nil
}

func (conn *Conn) Server() error {
	connService := conn.connService
	isFirst := true
	lastNum := int32(-1)
	for {
		if connService.IsQuit {
			return nil
		}

		if isFirst {
			isFirst = false
			log4.Info("test tcp timeout start")
			for i := 0; i < 6; i++ {
				reqData := [tcptestsbase.TestDataLen]byte{}
				isClose, err := connService.ReadBytes(reqData[:])
				if err != nil {
					return ee.New(err, "conn.ip:%v, ConnAddr:%v",
						conn.ip, conn.connService.ConnAddr)
				}
				if isClose {
					return nil
				}
				req := tcptestsbase.Bytes4ToInt32(reqData)

				log4.Info("Server recv req:%v", req)
				if req != lastNum+1 {
					log4.Error("Server recv err req:%v != lastNum + 1:%v", req, lastNum+1)
					return nil
				}
				lastNum = req
				recvNum.Store(req)

				sendLen := tcptestsbase.TestDataLen / 4
				TotalSendNum := 0
				for {
					if TotalSendNum >= len(reqData) {
						break
					}
					size := min(sendLen, len(reqData)-TotalSendNum)

					data := reqData[TotalSendNum : TotalSendNum+size]
					//log4.Info("Server send data:%v", string(data))
					time.Sleep(time.Millisecond * 2000)

					for {
						ok := conn.connService.WriteChan(0, tcpx.NewMsgFromBytes(data))
						if ok {
							break
						}
					}
					TotalSendNum += len(data)
				}

				randNum := rand.Int31n(3) + 3

				time.Sleep(time.Millisecond * time.Duration(randNum*1000))
			}
			log4.Info("test tcp timeout end")
		}

		reqData := [tcptestsbase.TestDataLen]byte{}
		isClose, err := connService.ReadBytes(reqData[:])
		if err != nil {
			return ee.New(err, "conn.ip:%v, ConnAddr:%v",
				conn.ip, conn.connService.ConnAddr)
		}
		if isClose {
			return nil
		}
		req := tcptestsbase.Bytes4ToInt32(reqData)
		//log4.Info("Server2 recv req:%v", req)

		if req == tcptestsbase.TestDataQuit {

			for {
				ok := conn.connService.WriteChan(0, tcpx.NewMsgFromBytes(reqData[:]))
				if ok {
					break
				}
			}
			break
		} else {
			if req != lastNum+1 {
				log4.Error("Server2 recv err req:%v != lastNum + 1:%v", req, lastNum+1)
				return nil
			}
			lastNum = req
			recvNum.Store(req)

			for {
				ok := conn.connService.WriteChan(0, tcpx.NewMsgFromBytes(reqData[:]))
				if ok {
					break
				}
			}
		}
	}

	return nil
}
