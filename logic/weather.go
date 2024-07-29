package logic

import (
	"fmt"
	"wechatgroupbot/api/weather"

	"github.com/eatmoreapple/openwechat"
)

func SearchWratherByCity(msg *openwechat.Message, val string) {
	weather_, err := weather.GetWeather(val)
	if err != nil {
		if err == weather.ErrNotFoundCity {
			msg.ReplyText("没有找到该城市，请检查输入是否正确")
			return
		}
		msg.ReplyText("天气查询出了点问题，晚点再试试")
		return
	}
	gczsj := weather_["观测站数据"].(map[string]interface{})["observe"].(map[string]interface{})
	kqzl := weather_["空气质量"].(map[string]interface{})["air"].(map[string]interface{})
	msg.ReplyText(fmt.Sprintf(`%s天气查询成功：
	气温：%s℃
	湿度：%s%%
	天气：%s
	风向：%s
	风力:%s
	空气质量%s（AQI:%f）`, val,
		gczsj["degree"].(string),
		gczsj["humidity"].(string),
		gczsj["weather"].(string),
		gczsj["wind_direction_name"].(string),
		gczsj["wind_power"].(string),
		kqzl["aqi_name"].(string),
		kqzl["aqi"].(float64),
	))
}
