package internal

import (
	"time"
)

type Config struct {
	Kafka KafkaConfig
	Redis RedisConfig
	MySQL MySQLConfig
	Server ServerConfig
}

type KafkaConfig struct {
	Brokers         []string
	Topic           string
	GroupID         string
	BatchSize       int
	FlushInterval   time.Duration
	MaxAttempts     int
	BatchTimeout    time.Duration
	MinBytes        int64
	MaxBytes        int64
	MaxWait         time.Duration
	ReadBackoffMin  time.Duration
	ReadBackoffMax  time.Duration
	CommitInterval  time.Duration
}

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	PoolTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type MySQLConfig struct {
	DSN          string
	SlowThreshold time.Duration
	LogLevel      int
}

type ServerConfig struct {
	Port          string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
}

var GlobalConfig = &Config{
	Kafka: KafkaConfig{
		Brokers:       []string{"localhost:9092"},
		Topic:         "orderMsg",
		GroupID:       "1",
		BatchSize:     300,
		FlushInterval: time.Second / 10,
		MaxAttempts:   3,
		BatchTimeout:  time.Second,
		MinBytes:      10e3,
		MaxBytes:      10e6,
		MaxWait:       time.Millisecond * 100,
		ReadBackoffMin: time.Millisecond * 50,
		ReadBackoffMax: time.Second * 1,
		CommitInterval: time.Second * 1,
	},
	Redis: RedisConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     1000,
		MinIdleConns: 100,
		PoolTimeout:  4 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	},
	// user name & password & db name should be related to the docker-compose.yml
	MySQL: MySQLConfig{
		DSN:           "root:root@tcp(localhost:3306)/seckill?charset=utf8mb4&parseTime=True&loc=Local",
		SlowThreshold: time.Second,
		LogLevel:      1,
	},
	Server: ServerConfig{
		Port:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	},
} 