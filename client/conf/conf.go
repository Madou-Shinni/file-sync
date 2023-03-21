package conf

var Conf = new(App)

// 配置
type App struct {
	UrlPrefix string `mapstructure:"url-prefix"` // 请求前缀
}
