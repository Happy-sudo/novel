package util

import (
	"io"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {

	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)

	//日志文件
	rl, _ := rotatelogs.New("log/api.%Y%m%d.log")
	mw := io.MultiWriter(os.Stdout, rl)
	log.SetOutput(mw)

	//配置文件解析
	viper.AutomaticEnv()
	viper.AddConfigPath("conf") // 配置文件路径	相对路径
	viper.SetEnvPrefix("novel") //配置文件前缀名称
	viper.SetConfigType("yml")  //配置文件后缀

	//配置文件设置
	viper.SetConfigName("novel.dev") // SetConfigName设置配置文件的名称。
	err := viper.ReadInConfig()      // 查找配置文件
	if err != nil {                  // 未找到配置文件
		log.Error("Fatal error config file", err)
	}

	// watch
	viper.WatchConfig()

	// 配置文件读取
	var profile = viper.GetString("profile")
	log.Info("novel profile = ", profile)
	if profile != "" {
		viper.SetConfigName("novel." + profile) //SetConfigName设置配置文件的名称。
		err = viper.MergeInConfig()             //将新配置与现有配置合并。
		if err != nil {                         // 错误配置文件
			log.Error("配置文件错误", err)
		}

		// 监视配置文件，重新读取配置数据
		viper.WatchConfig()
	}

	//重新读取配置数据
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed:", e.Name)
		log.Info("novel name ", viper.GetString("name"))
		log.Info("snowflake.machineId = ", viper.GetInt("snowflake.machineId"))
	})

	// 默认值
	viper.SetDefault("db.maxIdleConn", 10)
	viper.SetDefault("db.maxOpenConn", 100)
}
