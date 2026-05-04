package rivetxsql

import (
	"database/sql"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"time"

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
	obj.Pool.Close()
}

func CreateRivetxSql(config *Config) (*RivetxSql, error) {
	log4.Info("CreateMysql url:%v", config.Url)
	dsn := config.Url
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, ee.New(err, "Error opening database")
	}

	// 设置连接池参数
	pool.SetMaxOpenConns(config.MaxOpenConns)                                         // 最大打开连接数
	pool.SetMaxIdleConns(config.MaxIdleConns)                                         // 最大空闲连接数
	pool.SetConnMaxLifetime(time.Duration(config.ConnMaxLifetime) * time.Millisecond) // 连接最大生命周期
	pool.SetConnMaxIdleTime(time.Duration(config.ConnMaxIdleTime) * time.Millisecond) // 最大空闲时间

	// 测试连接
	if err := pool.Ping(); err != nil {
		pool.Close()
		return nil, ee.New(err, "Error pinging database")
	}
	log4.Info("Connected to MySQL database successfully!")
	return &RivetxSql{Pool: pool}, nil
}
