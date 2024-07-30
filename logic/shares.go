package logic

import (
	"fmt"
	"strings"
	"wechatgroupbot/api/gpt"
	"wechatgroupbot/bot"

	"github.com/eatmoreapple/openwechat"
)

var messageCache = msgBuffer{}

// 功能分支逻辑
func WeChatShare() {
	bot := bot.WeChatBot(func(msg *openwechat.Message) {
		if msg.IsText() {
			messageCache.Add(msg.FromUserName, msg.NewMsgId, msg.Content)
		}
		if msg.IsSendBySelf() {
			return
		}
		if msg.IsTickledMe() {
			pailepai(msg)
			return
		}
		switch {
		// 自动同意好友添加
		case msg.IsFriendAdd():
			msg.Agree(fmt.Sprintf("叫我一休哈,有事拍拍我%s", openwechat.Emoji.Grin))
		// 好友消息
		case msg.IsSendByFriend():
			if msg.IsText() {
				analysis(msg.Content, msg)
			}
		// 新人入群
		case msg.IsJoinGroup():
			msg.ReplyText(fmt.Sprintf("欢迎新人入群，请记住我是群里的大哥，有事喊我%s", openwechat.Emoji.Doge))
		// 消息防撤回
		case msg.IsRecalled():
			revokeMsg, err := msg.RevokeMsg()
			if err != nil {
				fmt.Println(err)
				return
			}
			if v, ok := messageCache.Get(msg.FromUserName, revokeMsg.RevokeMsg.MsgId); ok {
				sender, err := msg.Sender()
				if err != nil {
					fmt.Println(err)
					return
				}
				uN := sender.NickName
				if msg.IsSendByGroup() {
					sender, err = msg.SenderInGroup()
					if err != nil {
						fmt.Println(err)
						return
					}
					uN = sender.NickName
				}
				msg.ReplyText(fmt.Sprintf("%s撤回了一条消息\n\"%s\"", uN, v))

			}
		// 群组消息
		case msg.IsSendByGroup():
			if msg.IsRenameGroup() {
				msg.ReplyText("好名字")
				return
			}
			if msg.IsReceiveRedPacket() && !msg.IsSendRedPacket() {
				fs, err := msg.Sender()
				if err != nil {
					fmt.Println(err)
					return
				}
				ZR, err := msg.Owner().Self().Friends()
				if err != nil {
					fmt.Println(err)
					return
				}
				zr := ZR.SearchByRemarkName(1, "主人")
				zrname := ""
				if len(zr) != 0 {
					zrname = zr[0].NickName
				}
				if g, ok := fs.AsGroup(); ok {
					g.SendText(fmt.Sprintf("@%s 红包来了，快抢啊！", zrname))
				}
				return
			}
			if msg.IsText() {
				if msg.IsAt() {
					msg.Content = strings.ReplaceAll(msg.Content, fmt.Sprintf("@%s", msg.Owner().Self().NickName), "")
					msg.Content = strings.TrimSpace(msg.Content)
					analysis(msg.Content, msg)
				} else {
					gpt.Chatbots.CreateUserRecord(msg.FromUserName, msg.Content)
				}
			}

		// 设置为已读
		default:
			msg.AsRead()

		}

	})
	bot.Block()
}

var TaskMap = taskMap{
	"音乐": musicShare,
	"AI": aiToy,
	"天气": SearchWratherByCity,
}

// 关键词解析
func analysis(Content string, msg *openwechat.Message) {
	if Content != "" && Content[0] == '@' && Content[1] !=
		'@' {
		if strings.Contains(Content, ":") {
			valueS := strings.SplitN(Content, ":", 2)
			if fn, ok := TaskMap[valueS[0][1:]]; ok {
				fn(msg, valueS[1])
				return
			}
		}
		aiToy(msg, msg.Content)
		return
	}
	aiToy(msg, msg.Content)
}
