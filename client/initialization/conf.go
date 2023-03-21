package initialization

import (
	"fmt"
	"github.com/Madou-Shinni/file-sync/client/conf"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// 初始化配置
// 将配置文件的信息反序列化到结构体中
func init() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config") // 读取配置文件
	//viper.SetConfigFile("config.yml") // 读取配置文件
	err := viper.ReadInConfig() // 读取配置信息
	if err != nil {
		// 读取配置信息失败
		fmt.Printf("viper.ReadInConfig() faild error:%v\n", err)
		return
	}
	// 把读取到的信息反序列化到Conf变量中
	if err := viper.Unmarshal(conf.Conf); err != nil {
		fmt.Printf("viper.Unmarshal failed,err:%v\n", err)
	}
	viper.WatchConfig()                            // （热加载时读取配置）监控配置文件
	viper.OnConfigChange(func(in fsnotify.Event) { // 配置文件修改时触发回调
		if err := viper.Unmarshal(conf.Conf); err != nil {
			fmt.Printf("viper.Unmarshal failed,err:%v\n", err)
		}
	})
}
