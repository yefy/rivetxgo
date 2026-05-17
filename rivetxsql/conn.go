package rivetxsql

import (
	"database/sql"
	"time"

	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Url             string `yaml:"url"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int64  `yaml:"conn_max_life_time"`
	ConnMaxIdleTime int64  `yaml:"conn_max_idle_time"`
}

type RivetxSql struct {
	Pool *sql.DB
}

func (obj *RivetxSql) Close() {
	if obj.Pool == nil {
		return
	}
	obj.Pool.Close()
	obj.Pool = nil
}

func CreateRivetxSql(config *Config) (*RivetxSql, error) {
	if config == nil {
		return nil, ee.New(nil, "config is nil")
	}
	if config.Url == "" {
		return nil, ee.New(nil, "config.Url is empty")
	}
	if config.MaxOpenConns <= 0 {
		config.MaxOpenConns = 10
	}
	if config.MaxIdleConns <= 0 {
		config.MaxIdleConns = 5
	}
	if config.ConnMaxLifetime <= 0 {
		config.ConnMaxLifetime = 100000
	}
	if config.ConnMaxIdleTime <= 0 {
		config.ConnMaxIdleTime = 100000
	}
	log4.Info("CreateMysql url:%v", config.Url)
	dsn := config.Url
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, ee.New(err, "Error opening database")
	}

	// set connection pool parameters
	pool.SetMaxOpenConns(config.MaxOpenConns)                                         // max open connections
	pool.SetMaxIdleConns(config.MaxIdleConns)                                         // max idle connections
	pool.SetConnMaxLifetime(time.Duration(config.ConnMaxLifetime) * time.Millisecond) // max connection lifetime
	pool.SetConnMaxIdleTime(time.Duration(config.ConnMaxIdleTime) * time.Millisecond) // max idle time

	// test connection
	if err := pool.Ping(); err != nil {
		pool.Close()
		return nil, ee.New(err, "Error pinging database")
	}
	log4.Info("Connected to MySQL database successfully!")
	return &RivetxSql{Pool: pool}, nil
}
