package conf

import (
	cim "cirno-im"
	"cirno-im/logger"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Server struct {
}

// Config Config
type Config struct {
	ServiceID     string
	Listen        string `default:":8005"`
	MonitorPort   int    `default:"8006"`
	PublicAddress string
	PublicPort    int `default:"8005"`
	Tags          []string
	ConsulURL     string
	RedisAddrs    string
	RoyalURL      string
	LogLevel      string `default:"INFO"`
}

func (c Config) String() string {
	bts, _ := json.Marshal(c)
	return string(bts)
}

// Init InitConfig
func Init(file string) (*Config, error) {
	viper.SetConfigFile(file)
	viper.SetConfigName("conf")
	viper.AddConfigPath(".")
	viper.AddConfigPath("F:\\code\\golang\\cirno-im\\services\\server")
	viper.AddConfigPath("/etc/conf")

	var config Config
	if err := viper.ReadInConfig(); err != nil {
		logger.Warn(err)
	} else {
		if err := viper.Unmarshal(&config); err != nil {
			return nil, err
		}
	}
	err := envconfig.Process("cim", &config)
	if err != nil {
		return nil, err
	}
	if config.ServiceID == "" {
		localIP := cim.GetLocalIP()
		config.ServiceID = fmt.Sprintf("server_%s", strings.ReplaceAll(localIP, ".", ""))
	}
	if config.PublicAddress == "" {
		config.PublicAddress = cim.GetLocalIP()
	}
	logger.Info(config)
	return &config, nil
}

func InitRedis(addr string, pass string) (*redis.Client, error) {
	redisdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     pass,
		DialTimeout:  time.Second * 5,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	})

	_, err := redisdb.Ping().Result()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return redisdb, nil
}

// InitFailoverRedis init redis with sentinels
func InitFailoverRedis(masterName string, sentinelAddrs []string, password string, timeout time.Duration) (*redis.Client, error) {
	redisdb := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: sentinelAddrs,
		Password:      password,
		DialTimeout:   time.Second * 5,
		ReadTimeout:   timeout,
		WriteTimeout:  timeout,
	})

	_, err := redisdb.Ping().Result()
	if err != nil {
		logrus.Warn(err)
	}
	return redisdb, nil
}
