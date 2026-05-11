package tcpx

import (
	"bufio"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"io"
	"net"
	"rivetxgo/rivetxcore/gox"
	"rivetxgo/rivetxcore/session"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var isOpenMsgPool bool

func SetMsgPoolFlag(isOpen bool) {
	isOpenMsgPool = isOpen
}

var msgPoolsBlock int = 64
var msgPools [16384]sync.Pool

func init() {
	for i := range msgPools {
		i := i
		func(i int) {
			msgPools[i].New = func() interface{} {
				msg := NewMsg((i+1)*msgPoolsBlock, &msgPools[i])
				return msg
			}
		}(i)
	}
}

func GetMsg(size int) *Msg {
	// 0 1024-1   0
	// 1024 1024*2-1 1
	index := (size + msgPoolsBlock - 1) / msgPoolsBlock
	if !isOpenMsgPool || index >= len(msgPools) {
		msg := NewMsg((index+1)*msgPoolsBlock, nil)
		msg.Init(size)
		return msg
	} else {
		for {
			msg := msgPools[index].Get().(*Msg)
			if msg.RefCount.Load() > 0 {
				gox.StatNumEndAdd("etcp_msg_err_Get")
				log4.Error("log_tcp_msg GetMsg RefCount > 0, Stack:%s", string(debug.Stack()))
				continue
			}
			msg.Init(size)
			return msg
		}
	}
}

type Msg struct {
	RefCount atomic.Int32
	Size     int
	Cap      int
	Datas    []byte
	Pool     *sync.Pool
}

func NewMsg(cap int, Pool *sync.Pool) *Msg {
	msg := &Msg{
		Datas:    make([]byte, cap),
		Size:     0,
		Cap:      cap,
		Pool:     Pool,
		RefCount: atomic.Int32{},
	}
	//msg.RefCount.Add(1)
	return msg
}

func NewMsgFromBytes(Data []byte) *Msg {
	msg := &Msg{
		Datas:    Data,
		Size:     len(Data),
		Cap:      len(Data),
		Pool:     nil,
		RefCount: atomic.Int32{},
	}
	msg.RefCount.Add(1)
	return msg
}

func (msg *Msg) Init(size int) {
	if size > msg.Cap {
		gox.StatNumEndAdd("etcp_msg_err_Init")
		log4.Error("log_tcp_msg size:%v > msg.Cap:%v, Stack:%s", size, msg.Cap, string(debug.Stack()))
		panic(fmt.Sprintf("log_tcp_msg size:%v > msg.Cap:%v", size, msg.Cap))
	}
	msg.Size = size
	msg.RefCount.Store(1)
	gox.StatNumStartAdd("etcp_msg")
}

func (msg *Msg) GetData() []byte {
	if msg.RefCount.Load() <= 0 {
		gox.StatNumEndAdd("etcp_msg_err_GetData")
		log4.Error("log_tcp_msg GetData RefCount <= 0, msg.Cap:%v, Stack:%s", msg.Cap, string(debug.Stack()))
		//panic(fmt.Sprintf("log_tcp_msg Clone RefCount <= 0"))
	}
	return msg.Datas[0:msg.Size]
}

func (msg *Msg) Clone() *Msg {
	if msg.RefCount.Load() <= 0 {
		gox.StatNumEndAdd("etcp_msg_err_Clone")
		log4.Error("log_tcp_msg Clone RefCount <= 0, msg.Cap:%v, Stack:%s", msg.Cap, string(debug.Stack()))
		//panic(fmt.Sprintf("log_tcp_msg Clone RefCount <= 0"))
	}
	msg.RefCount.Add(1)
	return msg
}

func (msg *Msg) Put() {
	RefCount := msg.RefCount.Add(-1)
	if RefCount == 0 {
		gox.StatNumEndAdd("etcp_msg")
		if msg.Pool != nil {
			msg.Pool.Put(msg)
		}
	} else if RefCount < 0 {
		gox.StatNumEndAdd("etcp_msg_err_put")
		log4.Error("log_tcp_msg RefCount < 0, Stack:%s", string(debug.Stack()))
		//panic(fmt.Sprintf("log_tcp_msg RefCount < 0"))
	}
}

type Servicer interface {
	Init(spawnId uint64) error
	Start(spawnId uint64) error
	Read(spawnId uint64) error
	ReadChan(spawnId uint64, msg *Msg) error
	Write(spawnId uint64, msg *Msg) (bool, error)
	WriteErr(spawnId uint64, msg *Msg, err error) error
	Close(spawnId uint64, closeType int32)
	Self() interface{}
}

func NewTcpConf() *TcpConf {
	return &TcpConf{
		SocketWriteChanMsgSize:  1000,
		SocketWriteFlushTime2:   1,
		SocketConnectTimeout:    10000,
		SocketReadTimeout:       10000,
		SocketWriteTimeout:      10000,
		SocketReadBuffer:        1048576,
		SocketWriteBuffer:       1048576,
		SocketNoDelay:           true,
		SocketPrintCloseErr:     true,
		IsUnidirectionalTimeout: false,
		SocketDelayCloseTimeMs:  2000,
	}
}

type TcpConf struct {
	SocketWriteChanMsgSize  int   `yaml:"socket_write_chan_msg_size"`
	SocketWriteFlushTime2   int64 `yaml:"socket_write_flush_time2" default:"1"`
	SocketConnectTimeout    int64 `yaml:"socket_connect_timeout"`
	SocketReadTimeout       int64 `yaml:"socket_read_timeout"`
	SocketWriteTimeout      int64 `yaml:"socket_write_timeout"`
	SocketReadBuffer        int   `yaml:"socket_read_buffer"`
	SocketWriteBuffer       int   `yaml:"socket_write_buffer"`
	SocketNoDelay           bool  `yaml:"socket_no_delay"`
	SocketPrintCloseErr     bool  `yaml:"socket_print_close_err"`
	IsUnidirectionalTimeout bool  `yaml:"is_unidirectional_timeout"`
	SocketDelayCloseTimeMs  int64 `yaml:"socket_delay_close_time_ms" default:"2000"`
}

type Config struct {
	TcpConf   *TcpConf
	WaitGroup *sync.WaitGroup
	Ctx       context.Context
}

func NewConfig(tcpConf *TcpConf) *Config {
	return &Config{
		TcpConf: tcpConf,
	}
}

type Service struct {
	Port         string
	addr         string
	WaitGroup    *sync.WaitGroup
	IsQuit       bool
	Ctx          context.Context
	QuitFunc     func()
	QuitFuncOnce sync.Once
	listener     net.Listener
}

func NewService() *Service {
	ctx, cancel := context.WithCancel(context.Background())
	service := &Service{
		WaitGroup: &sync.WaitGroup{},
		Ctx:       ctx,
	}

	service.QuitFunc = func() {
		service.QuitFuncOnce.Do(func() {
			if service.IsQuit {
				return
			}
			service.IsQuit = true
			cancel()
		})
	}

	return service
}

func (service *Service) Close() {
	if service.listener != nil {
		log4.Info("close listen addr=%v", service.addr)
		service.QuitFunc()
		service.listener.Close()
		service.WaitGroup.Wait()
	}
}

func ListenTcp(addr string, config *Config, callFunc func(*ConnService) Servicer) (*Service, error) {
	service := NewService()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, ee.New(err, "Error starting TCP server addr:%v", addr)
	}
	addrs := strings.Split(addr, ":")
	if len(addrs) != 2 {
		return nil, ee.New(nil, "len(addrs) != 2")
	}
	service.Port = addrs[1]
	service.addr = addr
	service.listener = listener
	config.Ctx = service.Ctx
	config.WaitGroup = service.WaitGroup

	log4.Info("start listen addr=%v", addr)
	service.WaitGroup.Add(1)
	gox.Spawn(func(spawnId uint64) error {
		defer service.WaitGroup.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				if service.IsQuit {
					return nil
				}
				log4.Error("Error accepting connection:%v", err)
				continue
			}
			log4.Debug("AcceptTcp client connected:%v", conn.RemoteAddr())
			func(conn net.Conn) {
				service.WaitGroup.Add(1)
				gox.Spawn(func(spawnId uint64) error {
					defer service.WaitGroup.Done()
					connService := NewConnService("", config, conn, true, service.Port)
					connService.servicer = callFunc(connService)
					err := connService.Run(spawnId, connService.servicer != nil, false)
					if err != nil {
						return ee.New(err, "connService.Run")
					}
					return nil
				})
			}(conn)
		}
	})
	return service, nil
}

func ConnectTcp(isRunReadChan bool, addr string, config *Config, callFunc func(*ConnService) Servicer) error {
	log4.Trace("ConnectTcp client connected:%v", addr)
	addrs := strings.Split(addr, ":")
	if len(addrs) != 2 {
		return ee.New(nil, "len(addrs) != 2")
	}
	Port := addrs[1]

	conn, err := net.DialTimeout("tcp", addr, time.Millisecond*time.Duration(config.TcpConf.SocketConnectTimeout))
	if err != nil {
		return ee.New(err, "err:connect tcp addr:%v", addr)
	}
	if config.WaitGroup != nil {
		config.WaitGroup.Add(1)
	}

	spawnId := uint64(0)
	if config.WaitGroup != nil {
		defer config.WaitGroup.Done()
	}
	connService := NewConnService(addr, config, conn, false, Port)
	connService.servicer = callFunc(connService)
	err = connService.Run(spawnId, connService.servicer != nil, isRunReadChan)
	if err != nil {
		return ee.New(err, "connService.Run")
	}
	return nil
}

type ConnService struct {
	RemoteIp       string
	Port           string
	PortN          uint16
	ConnAddr       string
	Config         *Config
	SessionId      uint64
	Conn           net.Conn
	Reader         *bufio.Reader
	writer         *bufio.Writer
	writeChan      chan *Msg
	IsQuit         bool
	Ctx            context.Context
	QuitFunc       func()
	QuitFuncOnce   sync.Once
	WaitGroup      sync.WaitGroup
	servicer       Servicer
	IsAccept       bool
	writeDone      <-chan struct{}
	writeMainDone  <-chan struct{}
	Time           time.Time
	TypeClose      int32
	Count          int64
	readChan       chan *Msg
	TotalReadByte  int64
	TotalWriteByte int64
}

func NewConnService(addr string, config *Config, conn net.Conn, isAccept bool, port string) *ConnService {
	currTime := time.Now()
	tcpConn := conn.(*net.TCPConn)
	err := tcpConn.SetNoDelay(config.TcpConf.SocketNoDelay)
	if err != nil {
		log4.Error("err:SetNoDelay:%v, err:%v", config.TcpConf.SocketNoDelay, err)
	}
	if config.TcpConf.SocketReadBuffer > 0 {
		err := tcpConn.SetReadBuffer(config.TcpConf.SocketReadBuffer)
		if err != nil {
			log4.Error("err:SocketReadBuffer:%v, err:%v", config.TcpConf.SocketReadBuffer, err)
		}
	}
	if config.TcpConf.SocketWriteBuffer > 0 {
		err := tcpConn.SetWriteBuffer(config.TcpConf.SocketWriteBuffer)
		if err != nil {
			log4.Error("err:SocketWriteBuffer:%v, err:%v", config.TcpConf.SocketWriteBuffer, err)
		}
	}
	sessionId := session.SessionId()
	reader := bufio.NewReaderSize(conn, 1024)
	writer := bufio.NewWriterSize(conn, 1024)
	writeChan := make(chan *Msg, config.TcpConf.SocketWriteChanMsgSize)
	readChan := make(chan *Msg, config.TcpConf.SocketWriteChanMsgSize)

	portInt, _ := strconv.Atoi(port)
	portN := uint16(portInt)

	ctx, cancel := context.WithCancel(context.Background())
	service := &ConnService{
		Port:      port,
		PortN:     portN,
		ConnAddr:  addr,
		Config:    config,
		SessionId: sessionId,
		Conn:      conn,
		Reader:    reader,
		writer:    writer,
		writeChan: writeChan,
		WaitGroup: sync.WaitGroup{},
		Ctx:       ctx,
		IsAccept:  isAccept,
		Time:      currTime,
		readChan:  readChan,
	}

	writeDone := service.Ctx.Done()
	var writeMainDone <-chan struct{}
	if service.Config.Ctx != nil {
		writeMainDone = service.Config.Ctx.Done()
	}
	service.writeDone = writeDone
	service.writeMainDone = writeMainDone

	service.QuitFunc = func() {
		service.QuitFuncOnce.Do(func() {
			if service.IsQuit {
				return
			}
			service.IsQuit = true

			if service.TypeClose == TypClose {
				service.TypeClose = TypCloseQuit
			}
			log4.Debug("conn service %d start exit", sessionId)
			cancel()

			gox.Spawn(func(u uint64) error {
				sleepTimeMs := service.Config.TcpConf.SocketDelayCloseTimeMs
				if sleepTimeMs > 0 {
					sleepTimeMs = sleepTimeMs / 2
				}
				if sleepTimeMs < 1000 {
					sleepTimeMs = 1000
				}
				time.Sleep(time.Duration(sleepTimeMs) * time.Millisecond)
				close(service.writeChan)
				close(service.readChan)
				return nil
			})
		})
	}
	return service
}

func (service *ConnService) Run(spawnId uint64, isRun bool, isRunReadChan bool) error {
	gox.StatNumStartAdd("etcp")
	defer gox.StatNumEndAdd("etcp")

	addr := service.Conn.RemoteAddr()
	ip, _, _ := net.SplitHostPort(addr.String())
	service.RemoteIp = ip

	err := func() error {
		if !isRun {
			return nil
		}
		defer func() {
			if r := recover(); r != nil {
				log4.Recover(r)
			}
		}()
		if service.servicer != nil {
			err := service.servicer.Init(spawnId)
			if err != nil {
				return ee.New(err, "service.servicer.Init")
			}
		}
		service.WaitGroup.Add(1)
		gox.Spawn(func(spawnId uint64) error {
			gox.StatNumStartAdd("etcp_Write")
			defer gox.StatNumEndAdd("etcp_Write")

			defer service.WaitGroup.Done()
			err := service.Write(spawnId)
			if err != nil {
				return ee.New(err, "RemoteAddr:%v|%v", service.RemoteIp, service.Port)
			}
			return nil
		})
		service.WaitGroup.Add(1)
		gox.Spawn(func(spawnId uint64) error {
			gox.StatNumStartAdd("etcp_Read")
			defer gox.StatNumEndAdd("etcp_Read")

			defer service.WaitGroup.Done()
			err := service.Read(spawnId)
			if err != nil {
				return ee.New(err, "RemoteAddr:%v|%v", service.RemoteIp, service.Port)
			}
			return nil
		})

		if isRunReadChan {
			service.WaitGroup.Add(1)
			gox.Spawn(func(spawnId uint64) error {
				gox.StatNumStartAdd("etcp_ReadChan")
				defer gox.StatNumEndAdd("etcp_ReadChan")

				defer service.WaitGroup.Done()
				err := service.ReadChan(spawnId)
				if err != nil {
					return ee.New(err, "RemoteAddr:%v|%v", service.RemoteIp, service.Port)
				}
				return nil
			})
		}

		if service.servicer != nil {
			err := service.servicer.Start(spawnId)
			if err != nil {
				return ee.New(err, "service.servicer.Init")
			}
		}

		gox.StatNumStartAdd("etcp_wait_done")
		defer gox.StatNumEndAdd("etcp_wait_done")

		done := service.Ctx.Done()
		var mainDone <-chan struct{}
		if service.Config.Ctx != nil {
			mainDone = service.Config.Ctx.Done()
		}
		select {
		case <-done:
		case <-mainDone:
		}
		return nil
	}()
	gox.StatNumStartAdd("etcp_QuitFunc")
	if service.TypeClose == TypClose {
		service.TypeClose = TypCloseQuit
	}
	if service.servicer != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log4.Recover(r)
				}
			}()
			gox.StatNumStartAdd("etcp_servicer_Close")
			service.servicer.Close(spawnId, service.TypeClose)
			gox.StatNumEndAdd("etcp_servicer_Close")
		}()
	}
	service.QuitFunc()
	if service.Config.TcpConf.SocketDelayCloseTimeMs > 0 {
		time.Sleep(time.Duration(service.Config.TcpConf.SocketDelayCloseTimeMs) * time.Millisecond)
	}
	gox.StatNumEndAdd("etcp_QuitFunc")

	gox.StatNumStartAdd("etcp_Close")
	service.Conn.Close()
	gox.StatNumEndAdd("etcp_Close")
	service.WaitGroup.Wait()

	if err != nil {
		return ee.New(err, "RemoteAddr:%v|%v", service.RemoteIp, service.Port)
	}
	return nil
}

func (service *ConnService) Self() interface{} {
	if service.servicer == nil {
		return nil
	}
	return service.servicer.Self()
}

func (service *ConnService) Read(spawnId uint64) error {
	err := func() error {
		defer func() {
			if r := recover(); r != nil {
				log4.Recover(r)
			}
		}()

		if service.servicer != nil {
			return service.servicer.Read(spawnId)
		} else {
			return ee.New(nil, "err:service.servicer nil")
		}
	}()
	service.QuitFunc()
	return err
}

func (service *ConnService) ReadChan(spawnId uint64) error {
	err := func() error {
		defer func() {
			if r := recover(); r != nil {
				log4.Recover(r)
			}
		}()

		done := service.Ctx.Done()
		var mainDone <-chan struct{}
		if service.Config.Ctx != nil {
			mainDone = service.Config.Ctx.Done()
		}

		for {
			if service.IsQuit {
				return nil
			}

			select {
			case msg, ok := <-service.readChan:
				if !ok {
					return nil
				}
				if msg == nil {
					log4.Error("msg == nil")
					continue
				}

				if service.servicer != nil {
					err := func() error {
						defer msg.Put()
						return service.servicer.ReadChan(spawnId, msg)
					}()
					if err != nil {
						return err
					}
				} else {
					log4.Error("service.servicer == nil")
					continue
				}
			doneLoop:
				for {
					select {
					case msg, ok := <-service.readChan:
						if !ok {
							return nil
						}
						if msg == nil {
							log4.Error("msg == nil")
							break doneLoop
						}

						if service.servicer != nil {
							err := func() error {
								defer msg.Put()
								return service.servicer.ReadChan(spawnId, msg)
							}()
							if err != nil {
								return err
							}
						} else {
							log4.Error("service.servicer == nil")
							break doneLoop
						}
					//case <-done:
					//	break doneLoop
					//case <-mainDone:
					//	break doneLoop
					default:
						break doneLoop
					}
				}
			case <-done:
				return nil
			case <-mainDone:
				return nil
			}
		}
	}()
	service.QuitFunc()
	return err
}

func (service *ConnService) TryWriteToReadChan(sessionId uint64, msg *Msg) bool {
	if service.IsQuit {
		return false
	}
	if sessionId == service.SessionId {
		return false
	}
	select {
	case service.readChan <- msg:
		return true
	default:
		return false
	}
}

func (service *ConnService) WriteToReadChan(sessionId uint64, msg *Msg) bool {
	if service.IsQuit {
		return false
	}
	if sessionId == service.SessionId {
		return false
	}
	if service.TryWriteToReadChan(sessionId, msg) {
		return true
	}

	//done := service.Ctx.Done()
	//var mainDone <-chan struct{}
	//if service.Config.Ctx != nil {
	//	mainDone = service.Config.Ctx.Done()
	//}
	socketWriteTimeout := service.Config.TcpConf.SocketWriteTimeout
	timer := time.NewTimer(time.Duration(socketWriteTimeout) * time.Millisecond)
	defer timer.Stop()
	done := service.writeDone
	mainDone := service.writeMainDone
	select {
	case <-done:
		return false
	case <-mainDone:
		return false
	case <-timer.C:
		return false
	case service.readChan <- msg:
	}

	return true
}

func (service *ConnService) TryWriteChanRef(sessionId uint64, msg *Msg) bool {
	writeMsg := msg.Clone()
	if !service.TryWriteChan(sessionId, writeMsg) {
		writeMsg.Put()
		return false
	}
	return true
}

func (service *ConnService) TryWriteChan(sessionId uint64, msg *Msg) bool {
	if service.IsQuit {
		return false
	}
	if sessionId == service.SessionId {
		return false
	}
	select {
	case service.writeChan <- msg:
		return true
	default:
		return false
	}
}

func (service *ConnService) WriteChanRef(sessionId uint64, msg *Msg) bool {
	writeMsg := msg.Clone()
	if !service.WriteChan(sessionId, writeMsg) {
		writeMsg.Put()
		return false
	}
	return true
}

func (service *ConnService) WriteChan(sessionId uint64, msg *Msg) bool {
	if service.IsQuit {
		return false
	}
	if sessionId == service.SessionId {
		return false
	}
	if service.TryWriteChan(sessionId, msg) {
		return true
	}

	//done := service.Ctx.Done()
	//var mainDone <-chan struct{}
	//if service.Config.Ctx != nil {
	//	mainDone = service.Config.Ctx.Done()
	//}
	socketWriteTimeout := service.Config.TcpConf.SocketWriteTimeout
	timer := time.NewTimer(time.Duration(socketWriteTimeout) * time.Millisecond)
	defer timer.Stop()
	done := service.writeDone
	mainDone := service.writeMainDone
	select {
	case <-done:
		return false
	case <-mainDone:
		return false
	case <-timer.C:
		return false
	case service.writeChan <- msg:
	}

	return true
}

func (service *ConnService) Write(spawnId uint64) error {
	err := func() error {
		defer func() {
			if r := recover(); r != nil {
				log4.Recover(r)
			}
		}()
		done := service.Ctx.Done()
		var mainDone <-chan struct{}
		if service.Config.Ctx != nil {
			mainDone = service.Config.Ctx.Done()
		}
		defer service.writer.Flush()
		socketWriteFlushTime := service.Config.TcpConf.SocketWriteFlushTime2
		socketWriteTimeout := service.Config.TcpConf.SocketWriteTimeout

		flushTimer := time.NewTimer(time.Duration(socketWriteFlushTime) * time.Millisecond)
		if !flushTimer.Stop() {
			select {
			case <-flushTimer.C:
			default:
			}
		}
		defer flushTimer.Stop()

		var timeoutChan <-chan time.Time

		writeFunc := func(msg *Msg) (bool, error) {
			defer msg.Put()

			if service.servicer != nil {
				isDiscard, err := service.servicer.Write(spawnId, msg)
				if err != nil {
					if service.TypeClose == TypClose {
						service.TypeClose = TypCloseErr
					}
					return true, err
				}
				if isDiscard {
					return false, nil
				}
			}

			data := msg.GetData()
			totalWritten := 0
			for totalWritten < len(data) {
				//if service.IsQuit {
				//	return true, nil
				//}
				if socketWriteTimeout > 0 {
					service.Conn.SetWriteDeadline(time.Now().Add(time.Duration(socketWriteTimeout) * time.Millisecond))
				}
				Count := service.Count
				n, err := service.writer.Write(data[totalWritten:])
				if err != nil {
					isClose, err := SocketErr(service, err, false)
					if service.TypeClose == TypCloseWriteTimeout || service.TypeClose == TypCloseReadTimeout {
						if Count != service.Count && !service.Config.TcpConf.IsUnidirectionalTimeout {
							service.TypeClose = TypClose
							continue
						}
					}

					if err != nil {
						if service.servicer != nil {
							err2 := service.servicer.WriteErr(spawnId, msg, err)
							if err2 != nil {
								return isClose, ee.New(err2, "service.writer.Write")
							} else {
								return isClose, ee.New(err, "service.writer.Write")
							}
						} else {
							return isClose, ee.New(err, "service.writer.Write")
						}
					}
					return isClose, nil
				}
				service.Count += 1
				service.TotalWriteByte += int64(n)
				totalWritten += n
			}
			return false, nil
		}

		for {
			//if service.IsQuit {
			//	return nil
			//}

			select {
			case msg, ok := <-service.writeChan:
				if !ok {
					return nil
				}
				if timeoutChan != nil {
					if !flushTimer.Stop() {
						select {
						case <-flushTimer.C:
						default:
						}
					}
					timeoutChan = nil
				}
				isClose, err := writeFunc(msg)
				if err != nil {
					return ee.New(err, "writeFunc")
				}
				if isClose {
					return nil
				}
			doneLoop:
				for {
					select {
					case msg, ok := <-service.writeChan:
						if !ok {
							return nil
						}
						isClose, err := writeFunc(msg)
						if err != nil {
							return ee.New(err, "writeFunc")
						}
						if isClose {
							return nil
						}
					//case <-done:
					//	break doneLoop
					//case <-mainDone:
					//	break doneLoop
					default:
						break doneLoop
					}
				}

				if service.writer.Size() > 0 {
					if socketWriteFlushTime > 0 && !service.IsQuit {
						flushTimer.Reset(time.Duration(socketWriteFlushTime) * time.Millisecond)
						timeoutChan = flushTimer.C
					} else {
						service.writer.Flush()
					}
				}
			case <-timeoutChan:
				timeoutChan = nil
				if service.writer.Size() > 0 {
					service.writer.Flush()
				}
			case <-done:
				return nil
			case <-mainDone:
				return nil
			}
		}
	}()
	service.QuitFunc()
	return err
}

func (service *ConnService) ReadBytes(data []byte, socketReadTimeout int64) (bool, error) {
	totalReadSize := 0
	for {
		if service.IsQuit {
			return true, nil
		}
		if socketReadTimeout > 0 {
			service.Conn.SetReadDeadline(time.Now().Add(time.Duration(socketReadTimeout) * time.Millisecond))
		} else {
			service.Conn.SetReadDeadline(time.Time{})
		}
		n, err := service.Reader.Read(data[totalReadSize:])
		if err != nil {
			isClose, err := SocketErr(service, err, true)
			if err != nil {
				return isClose, ee.New(err, "connService.Reader.Read")
			}
			return isClose, nil
		}
		totalReadSize += n
		if totalReadSize >= len(data) {
			return false, nil
		}
	}
}

const (
	TypClose int32 = iota
	TypCloseQuit
	TypCloseErr
	TypCloseReadReset
	TypCloseReadTimeout
	TypCloseReadErr
	TypCloseWriteReset
	TypCloseWriteTimeout
	TypCloseWriteErr
)

var TypNameMap map[int32]string

func init() {
	TypNameMap = map[int32]string{
		TypClose:             "TypClose",
		TypCloseQuit:         "TypCloseQuit",
		TypCloseErr:          "TypCloseErr",
		TypCloseReadReset:    "TypCloseReadReset",
		TypCloseReadTimeout:  "TypCloseReadTimeout",
		TypCloseReadErr:      "TypCloseReadErr",
		TypCloseWriteReset:   "TypCloseWriteReset",
		TypCloseWriteTimeout: "TypCloseWriteTimeout",
		TypCloseWriteErr:     "TypCloseWriteErr",
	}
}

func GetTypName(typ int32) string {
	return TypNameMap[typ]
}

func SocketErr(service *ConnService, err error, isRead bool) (bool, error) {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		// handle timeout error
		if service.TypeClose == TypClose {
			if isRead {
				service.TypeClose = TypCloseReadTimeout
			} else {
				service.TypeClose = TypCloseWriteTimeout
			}
		}
		return true, ee.New(err, "Timeout error isRead:%v", isRead)
	} else if errors.Is(err, syscall.EPIPE) {
		// handle EPIPE error, indicates write to a closed connection
		//return ee.New(err, "Broken pipe error")
		if service.TypeClose == TypClose {
			if isRead {
				service.TypeClose = TypCloseReadReset
			} else {
				service.TypeClose = TypCloseWriteReset
			}
		}
		if service.Config.TcpConf.SocketPrintCloseErr {
			return true, ee.New(err, "closed syscall.EPIPE isRead:%v", isRead)
		}
		return true, nil
	} else if errors.Is(err, syscall.ECONNRESET) {
		// handle connection reset error
		//return ee.New(err, "Connection reset error")
		if service.TypeClose == TypClose {
			if isRead {
				service.TypeClose = TypCloseReadReset
			} else {
				service.TypeClose = TypCloseWriteReset
			}
		}
		if service.Config.TcpConf.SocketPrintCloseErr {
			return true, ee.New(err, "closed syscall.ECONNRESET isRead:%v", isRead)
		}
		return true, nil
	} else if err == io.EOF {
		// EOF means the peer closed the connection (usually during read)
		//return ee.New(err, "Connection closed by peer with EOF")
		if service.TypeClose == TypClose {
			if isRead {
				service.TypeClose = TypCloseReadReset
			} else {
				service.TypeClose = TypCloseWriteReset
			}
		}
		if service.Config.TcpConf.SocketPrintCloseErr {
			return true, ee.New(err, "closed io.EOF isRead:%v", isRead)
		}
		return true, nil
	} else {
		if strings.Contains(err.Error(), "use of closed network connection") {
			if service.TypeClose == TypClose {
				if isRead {
					service.TypeClose = TypCloseReadReset
				} else {
					service.TypeClose = TypCloseWriteReset
				}
			}

			if service.Config.TcpConf.SocketPrintCloseErr {
				return true, ee.New(err, "closed network connection isRead:%v", isRead)
			}
			return true, nil
		}
		// handle other error types
		if service.TypeClose == TypClose {
			if isRead {
				service.TypeClose = TypCloseReadErr
			} else {
				service.TypeClose = TypCloseWriteErr
			}
		}
		return true, ee.New(err, "SocketErr")
	}
}
