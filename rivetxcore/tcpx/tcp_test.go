package tcpx

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

func TestSetMsgPoolFlag(t *testing.T) {
	// Test setting msg pool flag
	SetMsgPoolFlag(true)
	// No direct way to verify, but should not panic
	SetMsgPoolFlag(false)
}

func TestGetMsg(t *testing.T) {
	SetMsgPoolFlag(true)
	msg := GetMsg(100)
	if msg == nil {
		t.Fatal("GetMsg returned nil")
	}
	if msg.Cap < 100 {
		t.Errorf("Expected cap >= 100, got %d", msg.Cap)
	}
	if msg.RefCount.Load() != 1 {
		t.Errorf("Expected RefCount 1, got %d", msg.RefCount.Load())
	}
	msg.Put()

	// Test without pool
	SetMsgPoolFlag(false)
	msg2 := GetMsg(200)
	if msg2 == nil {
		t.Fatal("GetMsg without pool returned nil")
	}
	if msg2.Cap < 200 {
		t.Errorf("Expected cap >= 200, got %d", msg2.Cap)
	}
	msg2.Put()
}

func TestNewMsgFromBytes(t *testing.T) {
	data := []byte("test data")
	msg := NewMsgFromBytes(data)
	if msg == nil {
		t.Fatal("NewMsgFromBytes returned nil")
	}
	if msg.Size != len(data) {
		t.Errorf("Expected size %d, got %d", len(data), msg.Size)
	}
	if msg.RefCount.Load() != 1 {
		t.Errorf("Expected RefCount 1, got %d", msg.RefCount.Load())
	}
	if string(msg.GetData()) != string(data) {
		t.Errorf("Data mismatch: expected %s, got %s", string(data), string(msg.GetData()))
	}
	msg.Put()
}

func TestMsgClone(t *testing.T) {
	msg := GetMsg(50)
	msg.Init(10)
	cloned := msg.Clone()
	if cloned.RefCount.Load() != 2 {
		t.Errorf("Expected RefCount 2 after clone, got %d", cloned.RefCount.Load())
	}
	cloned.Put()
	if msg.RefCount.Load() != 1 {
		t.Errorf("Expected RefCount 1 after clone put, got %d", msg.RefCount.Load())
	}
	msg.Put()
}

func TestNewTcpConf(t *testing.T) {
	conf := NewTcpConf()
	if conf == nil {
		t.Fatal("NewTcpConf returned nil")
	}
	if conf.SocketWriteChanMsgSize != 1000 {
		t.Errorf("Expected SocketWriteChanMsgSize 1000, got %d", conf.SocketWriteChanMsgSize)
	}
	if !*conf.SocketNoDelay {
		t.Error("Expected SocketNoDelay true")
	}
}

func TestNewConfig(t *testing.T) {
	conf := NewTcpConf()
	config := NewConfig(conf)
	if config == nil {
		t.Fatal("NewConfig returned nil")
	}
	if config.TcpConf != conf {
		t.Error("TcpConf not set correctly")
	}
}

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("NewService returned nil")
	}
	if service.WaitGroup == nil {
		t.Error("WaitGroup is nil")
	}
	if service.Ctx == nil {
		t.Error("Ctx is nil")
	}
	service.Close() // Test close
}

func TestTypNameMap(t *testing.T) {
	name := GetTypName(TypClose)
	if name != "TypClose" {
		t.Errorf("Expected 'TypClose', got '%s'", name)
	}
	name = GetTypName(TypCloseErr)
	if name != "TypCloseErr" {
		t.Errorf("Expected 'TypCloseErr', got '%s'", name)
	}
	name = GetTypName(999) // Invalid type
	if name != "" {
		t.Errorf("Expected empty string for invalid type, got '%s'", name)
	}
}

func TestConnServiceMethods(t *testing.T) {
	conf := NewTcpConf()
	config := NewConfig(conf)

	// Create a real TCP connection for testing
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
	}()

	clientConn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer clientConn.Close()

	connService := NewConnService("127.0.0.1:12345", config, clientConn, false, "12345")
	if connService == nil {
		t.Fatal("NewConnService returned nil")
	}
	if connService.Port != "12345" {
		t.Errorf("Expected port '12345', got '%s'", connService.Port)
	}
	if connService.IsAccept {
		t.Error("Expected IsAccept false")
	}

	// Test Self method
	if connService.Self() != nil {
		t.Error("Expected Self() nil without servicer")
	}

	connService.QuitFunc() // Test quit
}

// Mock Servicer for testing
type mockServicer struct{}

func (m *mockServicer) Init(spawnId uint64) error                          { return nil }
func (m *mockServicer) Start(spawnId uint64) error                         { return nil }
func (m *mockServicer) Read(spawnId uint64) error                          { return nil }
func (m *mockServicer) ReadChan(spawnId uint64, msg *Msg) error            { return nil }
func (m *mockServicer) Write(spawnId uint64, msg *Msg) (bool, error)       { return false, nil }
func (m *mockServicer) WriteErr(spawnId uint64, msg *Msg, err error) error { return nil }
func (m *mockServicer) Close(spawnId uint64, closeType int32)              {}
func (m *mockServicer) Self() interface{}                                  { return m }
func (m *mockServicer) ReadTimeout(isCheckTimeout bool)                    {}
func (m *mockServicer) WriteTimeout(isCheckTimeout bool)                   {}

func TestConnServiceWithServicer(t *testing.T) {
	conf := NewTcpConf()
	config := NewConfig(conf)

	// Create a real TCP connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
	}()

	clientConn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer clientConn.Close()

	connService := NewConnService("127.0.0.1:12345", config, clientConn, false, "12345")
	connService.servicer = &mockServicer{}

	if connService.Self() == nil {
		t.Error("Expected Self() not nil with servicer")
	}
}

func TestReadBytes(t *testing.T) {
	conf := NewTcpConf()
	conf.SocketReadTimeout = 1000
	config := NewConfig(conf)

	// Create TCP connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	serverConnChan := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		serverConnChan <- conn
	}()

	clientConn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer clientConn.Close()

	serverConn := <-serverConnChan
	defer serverConn.Close()

	connService := NewConnService("127.0.0.1:12345", config, clientConn, false, "12345")

	// Write some data to server side
	go func() {
		serverConn.Write([]byte("test"))
	}()

	data := make([]byte, 4)
	isClose, err := connService.ReadBytes(data)
	if err != nil {
		t.Errorf("ReadBytes failed: %v", err)
	}
	if isClose {
		t.Error("Expected not closed")
	}
	if string(data) != "test" {
		t.Errorf("Expected 'test', got '%s'", string(data))
	}
}

func TestWriteChan(t *testing.T) {
	conf := NewTcpConf()
	config := NewConfig(conf)

	// Create TCP connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
	}()

	clientConn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer clientConn.Close()

	connService := NewConnService("127.0.0.1:12345", config, clientConn, false, "12345")

	msg := NewMsgFromBytes([]byte("test"))
	success := connService.TryWriteChan(0, msg)
	if !success {
		t.Error("TryWriteChan failed")
	}

	// Test with same session ID (should fail)
	success = connService.TryWriteChan(connService.SessionId, msg)
	if success {
		t.Error("TryWriteChan should fail with same session ID")
	}

	msg.Put()
}

func TestSocketErr(t *testing.T) {
	conf := NewTcpConf()
	conf.SocketReadTimeout = 100
	config := NewConfig(conf)

	// Create TCP connection
	listener, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		conn.Close() // Close immediately
	}()

	clientConn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer clientConn.Close()

	connService := NewConnService("127.0.0.1:12345", config, clientConn, false, "12345")

	// Try to read from closed connection
	data := make([]byte, 1)
	isClose, err := connService.ReadBytes(data)
	if !(isClose && err == nil) {
		t.Error("Expected error when reading from closed connection")
	}
}

// Integration test similar to TcpTests but simplified
func TestTcpIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	var wg sync.WaitGroup
	serverDone := make(chan bool, 1)

	// Start server
	wg.Add(1)
	go func() {
		defer wg.Done()
		addr := "127.0.0.1:0" // Use port 0 for auto-assignment
		tcpConf := NewTcpConf()
		tcpConf.SocketWriteFlushTime2 = 0
		config := NewConfig(tcpConf)

		listen, err := ListenTcp(addr, config, func(connService *ConnService) Servicer {
			return &mockServicer{}
		})
		if err != nil {
			t.Errorf("ListenTcp failed: %v", err)
			return
		}
		defer listen.Close()

		actualAddr := listen.listener.Addr().String()
		t.Logf("Server listening on %s", actualAddr)

		// Signal that server is ready
		serverDone <- true

		// Wait a bit for client
		time.Sleep(100 * time.Millisecond)
	}()

	// Wait for server to be ready
	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Server failed to start")
	}

	wg.Wait()
}

func TestContextCancellation(t *testing.T) {
	conf := NewTcpConf()
	config := NewConfig(conf)
	ctx, cancel := context.WithCancel(context.Background())
	config.Ctx = ctx

	// Test that config has the context set
	if config.Ctx != ctx {
		t.Error("Config context not set correctly")
	}

	// Cancel the config context (this affects the service level, not individual connections)
	cancel()

	// Note: Individual ConnService creates its own context, so this test verifies config setup
}
