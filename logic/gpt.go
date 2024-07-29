package logic

import (
	"fmt"
	"wechatgroupbot/api/gpt"
	"wechatgroupbot/api/weather"

	"github.com/eatmoreapple/openwechat"
)

type cmdFn struct {
	keyD gpt.Pattre
	Task func(msg *openwechat.Message, val ...[]string)
}

type cmdTask map[string]*cmdFn

var ct cmdTask

func init() {

	ct = cmdTask{
		"想听音乐的时候，或者想要搜索某首音乐时，或者要求播放音乐时": &cmdFn{
			keyD: gpt.Pattre{Keys: []string{"歌名"}, D: ".*?"},
			Task: func(msg *openwechat.Message, val ...[]string) {
				if len(val) == 0 {
					return
				}
				for _, v := range val {
					musicShare(msg, v[0])
				}
			}},
		"当system记录中没有该城市数据时，当要询问某地（市）天气时，想要去到某地时天气决策;当询问某几个地方的天气对比时,返回多个命令串。一定不要询问查哪一天的广州天气": &cmdFn{
			keyD: gpt.Pattre{Keys: []string{"城市"}, D: ".*?"},
			Task: func(msg *openwechat.Message, val ...[]string) {
				msg.ReplyText("正在查询天气,请稍等...")
				if len(val) == 0 {
					msg.ReplyText(fmt.Sprintf("抱歉，天气查询错误了%s", openwechat.Emoji.Cry))
					return
				}
				for _, v := range val {
					fmt.Println(v)
					if data, err := weather.GetWeather(v[0]); err != nil {
						fmt.Println(err)
						msg.ReplyText(fmt.Sprintf("抱歉，%s天气查询错误了%s", v, openwechat.Emoji.Cry))
						return
					} else {
						gpt.Chatbots.SetBotAttr(msg.FromUserName, fmt.Sprintf("当前聊天对象天气(%s)", v[0]), data)
					}
				}
				gpt.Chatbots.Ask(msg.FromUserName, "根据获取到的数据，重新回答我的上一个的问题", func(ans string, err error) {
					if err != nil {
						msg.ReplyText("裂开,我又出错了!😭")
						return
					}
					msg.ReplyText(ans)
				})
			},
		},
	}
	for k, v := range ct {
		fmt.Println(gpt.Chatbots.KeyCommandAdd(k, v.keyD))

	}
}

// AI玩具
func aiToy(msg *openwechat.Message, val string) {
	gpt.Chatbots.Ask(msg.FromUserName, val, func(ans string, err error) {
		if err != nil {
			fmt.Println(err)
			msg.ReplyText("裂开,我又出错了!😭")
			return
		}
		msg.ReplyText(ans)
	}, func(tag string, s ...[]string) {
		ct[tag].Task(msg, s...)
	})

}

// 拍了拍处理
func pailepai(msg *openwechat.Message) {
	gpt.Chatbots.Ask(msg.FromUserName, "拍了拍你", func(ans string, err error) {
		if err != nil {
			fmt.Println(err)
			msg.ReplyText("裂开,我又出错了!😭")
			return
		}
		msg.ReplyText(fmt.Sprintf("%s\n\n%s", ans, `听音乐: 【@音乐:】+歌名
AI: 【@AI:】+问题
天气: 【@天气:】+城市
其他的命令先等等哦。`))
	})
}
