package tcpx

import (
	"context"
	"fmt"
	"github.com/yefy/rivetxgo/rivetxcore/bufiox"
	"github.com/yefy/rivetxgo/rivetxcore/gox"
	"github.com/yefy/rivetxgo/rivetxcore/session"
	"io"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
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
	ReadTimeout(isCheckTimeout bool)
	WriteTimeout(isCheckTimeout bool)
}

func NewTcpConf() *TcpConf {
	SocketNoDelay := true
	return &TcpConf{
		SocketWriteChanMsgSize:  1000,
		SocketWriteFlushTime2:   1,
		SocketConnectTimeout:    10000,
		SocketReadTimeout:       10000,
		SocketWriteTimeout:      10000,
		SocketReadBuffer:        1048576,
		SocketWriteBuffer:       1048576,
		SocketNoDelay:           &SocketNoDelay,
		SocketPrintCloseErr:     false,
		IsUnidirectionalTimeout: false,
		SocketDelayCloseTimeMs:  2000,
		ReadBufferCache:         4096,
		WriteBufferCache:        4096,
		CheckTimeoutInterval:    500,
	}
}

type TcpConf struct {
	SocketWriteChanMsgSize  int   `yaml:"socket_write_chan_msg_size" default:"1000"`
	SocketWriteFlushTime2   int64 `yaml:"socket_write_flush_time2" default:"1"`
	SocketConnectTimeout    int64 `yaml:"socket_connect_timeout" default:"10000"`
	SocketReadTimeout       int64 `yaml:"socket_read_timeout" default:"10000"`
	SocketWriteTimeout      int64 `yaml:"socket_write_timeout" default:"10000"`
	SocketReadBuffer        int   `yaml:"socket_read_buffer" default:"1048576"`
	SocketWriteBuffer       int   `yaml:"socket_write_buffer" default:"1048576"`
	SocketNoDelay           *bool `yaml:"socket_no_delay" default:"true"`
	SocketPrintCloseErr     bool  `yaml:"socket_print_close_err"`
	IsUnidirectionalTimeout bool  `yaml:"is_unidirectional_timeout"`
	SocketDelayCloseTimeMs  int64 `yaml:"socket_delay_close_time_ms" default:"2000"`
	ReadBufferCache         int   `yaml:"read_buffer_cache" default:"4096"`
	WriteBufferCache        int   `yaml:"write_buffer_cache" default:"4096"`
	CheckTimeoutInterval    int64 `yaml:"check_timeout_interval" default:"500"`
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

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		listener.Close()
		return nil, ee.New(err, "Error splitting host port addr:%v", addr)
	}

	service.Port = port
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


func ConnectTcpSync(isRunReadChan bool, addr string, config *Config, callFunc func(*ConnService) Servicer) error {
	return DoConnectTcp(false, isRunReadChan, addr, config, callFunc)
}

func ConnectTcp(isRunReadChan bool, addr string, config *Config, callFunc func(*ConnService) Servicer) error {
	return DoConnectTcp(true, isRunReadChan, addr, config, callFunc)
}

func DoConnectTcp(isSpawn bool, isRunReadChan bool, addr string, config *Config, callFunc func(*ConnService) Servicer) error {
	log4.Trace("ConnectTcp client connected:%v", addr)
	_, Port, err := net.SplitHostPort(addr)
	if err != nil {
		return ee.New(err, "Error splitting host port addr:%v", addr)
	}

	conn, err := net.DialTimeout("tcp", addr, time.Millisecond*time.Duration(config.TcpConf.SocketConnectTimeout))
	if err != nil {
		return ee.New(err, "err:connect tcp addr:%v", addr)
	}
	if config.WaitGroup != nil {
		config.WaitGroup.Add(1)
	}

	if !isSpawn {
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
	} else {
		gox.Spawn(func(spawnId uint64) error {
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
		})
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
	Reader         *bufiox.Reader
	writer         *bufiox.Writer
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
	readChan       chan *Msg
	TotalReadByte  int64
	TotalWriteByte int64
}

func NewConnService(addr string, config *Config, conn net.Conn, isAccept bool, port string) *ConnService {
	currTime := time.Now()
	tcpConn := conn.(*net.TCPConn)
	if config.TcpConf.SocketNoDelay != nil {
		if *config.TcpConf.SocketNoDelay {
			err := tcpConn.SetNoDelay(true)
			if err != nil {
				log4.Error("err:SetNoDelay:%v, err:%v", config.TcpConf.SocketNoDelay, err)
			}
		}
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
	reader := bufiox.NewReaderSize(conn, config.TcpConf.ReadBufferCache)
	writer := bufiox.NewWriterSize(conn, config.TcpConf.WriteBufferCache)
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

func (service *ConnService) IsTimeout() bool {
	if service.TypeClose == TypCloseReadTimeout || service.TypeClose == TypCloseWriteTimeout {
		return true
	}
	return false
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

		service.WaitGroup.Add(1)
		gox.Spawn(func(u uint64) error {
			gox.StatNumStartAdd("etcp_checkTimeout")
			defer gox.StatNumEndAdd("etcp_checkTimeout")

			defer service.WaitGroup.Done()
			err := service.checkTimeout()
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
				return ee.New(err, "service.servicer.Start")
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

	sleepTimeMs := service.Config.TcpConf.SocketDelayCloseTimeMs
	if sleepTimeMs < 1000 {
		sleepTimeMs = 1000
	}
	time.Sleep(time.Duration(sleepTimeMs) * time.Millisecond)

	gox.StatNumEndAdd("etcp_QuitFunc")

	gox.StatNumStartAdd("etcp_Close")
	service.Conn.Close()
	gox.StatNumEndAdd("etcp_Close")
	service.WaitGroup.Wait()

	for msg := range service.writeChan {
		if msg != nil {
			msg.Put()
		}
	}

	for msg := range service.readChan {
		if msg != nil {
			msg.Put()
		}
	}

	if err != nil {
		return ee.New(err, "RemoteAddr:%v|%v", service.RemoteIp, service.Port)
	}
	return nil
}

func (service *ConnService) checkTimeout() error {
	tcpConf := service.Config.TcpConf
	totalReadByte := service.TotalReadByte
	isReadTimeout := false
	isReadTimeoutTime := time.Now()
	totalReadByteTime := int64(0)

	totalWriteByte := service.TotalWriteByte
	isWriteTimeout := false
	isWriteTimeoutTime := time.Now()
	totalWriteByteTime := int64(0)
	if tcpConf.CheckTimeoutInterval < 500 {
		tcpConf.CheckTimeoutInterval = 500
	}
	for {
		if service.IsQuit {
			return nil
		}
		time.Sleep(time.Millisecond * time.Duration(tcpConf.CheckTimeoutInterval))
		totalReadByteTime += tcpConf.CheckTimeoutInterval
		totalWriteByteTime += tcpConf.CheckTimeoutInterval

		if totalReadByte != service.TotalReadByte {
			totalReadByte = service.TotalReadByte
			isReadTimeout = false
			totalReadByteTime = 0
		} else {
			if tcpConf.SocketReadTimeout > 0 {
				if totalReadByteTime >= tcpConf.SocketReadTimeout {
					isReadTimeout = true
					isReadTimeoutTime = time.Now()
					if service.servicer != nil {
						service.servicer.ReadTimeout(true)
					}
				}
			}
		}

		if totalWriteByte != service.TotalWriteByte {
			totalWriteByte = service.TotalWriteByte
			isWriteTimeout = false
			totalWriteByteTime = 0
		} else {
			if tcpConf.SocketWriteTimeout > 0 {
				if totalWriteByteTime >= tcpConf.SocketWriteTimeout {
					isWriteTimeout = true
					isWriteTimeoutTime = time.Now()
					if service.servicer != nil {
						service.servicer.WriteTimeout(true)
					}
				}
			}
		}

		if tcpConf.IsUnidirectionalTimeout {
			if isReadTimeout {
				if service.TypeClose == TypClose {
					service.TypeClose = TypCloseReadTimeout
				}
				if service.servicer != nil {
					service.servicer.ReadTimeout(true)
				}
				service.QuitFunc()
				return nil
			} else if isWriteTimeout {
				if service.TypeClose == TypClose {
					service.TypeClose = TypCloseWriteTimeout
				}
				if service.servicer != nil {
					service.servicer.WriteTimeout(true)
				}
				service.QuitFunc()
				return nil
			}
		} else {
			if isReadTimeout && isWriteTimeout {
				if isReadTimeoutTime.UnixMilli() >= isWriteTimeoutTime.UnixMilli() {
					if service.TypeClose == TypClose {
						service.TypeClose = TypCloseReadTimeout
					}
				} else {
					if service.TypeClose == TypClose {
						service.TypeClose = TypCloseWriteTimeout
					}
				}
				if service.servicer != nil {
					service.servicer.ReadTimeout(true)
				}
				if service.servicer != nil {
					service.servicer.WriteTimeout(true)
				}
				service.QuitFunc()
				return nil
			}
		}
	}
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
	return ee.NewErr(err)
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
						return ee.NewErr(err)
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
								return ee.NewErr(err)
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
	return ee.NewErr(err)
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
		//socketWriteTimeout := service.Config.TcpConf.SocketWriteTimeout

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
					return true, ee.NewErr(err)
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

				n, err := service.writer.Write(data[totalWritten:])
				if n >= 0 {
					service.TotalWriteByte += int64(n)
					totalWritten += n
					if totalWritten >= len(data) {
						break
					}
				}

				if err != nil {
					service.writer.ResetErr()
					isClose, err := SocketErr(service, err, false)
					if service.TypeClose == TypClose {
						service.TypeClose = TypCloseWriteTimeout
						if service.servicer != nil {
							service.servicer.WriteTimeout(false)
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

				if service.writer.Buffered() > 0 {
					if socketWriteFlushTime > 0 && !service.IsQuit {
						flushTimer.Reset(time.Duration(socketWriteFlushTime) * time.Millisecond)
						timeoutChan = flushTimer.C
					} else {
						service.writer.Flush()
					}
				}
			case <-timeoutChan:
				timeoutChan = nil
				if service.writer.Buffered() > 0 {
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
	return ee.NewErr(err)
}

func (service *ConnService) ReadBytes(data []byte) (bool, error) {
	if len(data) == 0 {
		return false, ee.New(nil, "len(data) == 0")
	}

	totalReadSize := 0
	for {
		if service.IsQuit {
			return true, nil
		}
		n, err := service.Reader.Read(data[totalReadSize:])
		if n >= 0 {
			service.TotalReadByte += int64(n)
			totalReadSize += n
			if totalReadSize >= len(data) {
				return false, nil
			}
		}

		if err != nil {
			service.Reader.ResetErr()
			isClose, err := SocketErr(service, err, true)
			if service.TypeClose == TypClose {
				service.TypeClose = TypCloseReadTimeout
				if service.servicer != nil {
					service.servicer.ReadTimeout(false)
				}
			}

			if err != nil {
				return isClose, ee.New(err, "connService.Reader.Read")
			}
			return isClose, nil
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
		//if service.TypeClose == TypClose {
		//	if isRead {
		//		service.TypeClose = TypCloseReadTimeout
		//	} else {
		//		service.TypeClose = TypCloseWriteTimeout
		//	}
		//}
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
	} else if errors.Is(err, io.EOF) {
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
		return true, ee.New(err, "SocketErr GetTypName:%v", GetTypName(service.TypeClose))
	}
}
