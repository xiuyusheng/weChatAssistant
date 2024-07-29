package bot

import (
	"fmt"

	"github.com/eatmoreapple/openwechat"
)

func WeChatBot(task openwechat.MessageHandler) (bot *openwechat.Bot) {
	bot = openwechat.DefaultBot(openwechat.Desktop) // 桌面模式

	// 注册消息处理函数
	// bot.MessageHandler = func(msg *openwechat.Message) {
	// 	if msg.IsText() && msg.Content == "ping" {
	// 		msg.ReplyText("pong")
	// 	}

	// }

	// 注册登陆二维码回调
	// bot.UUIDCallback = func(uuid string) {
	// 	fmt.Println(openwechat.GetQrcodeUrl(uuid))
	// }
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl
	// 创建热存储容器对象
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")

	defer reloadStorage.Close()

	// 登陆
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		fmt.Println(err)
		return
	}

	// 获取登陆的用户
	// self, err := bot.GetCurrentUser()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// 获取所有的好友
	// friends, err := self.Friends()
	// fmt.Println(friends, err)
	// file, _ := os.Open("人间烟火-程响.mp3")
	// defer file.Close()
	// fmt.Println(self.SendFileToFriend(self.FileHelper(), file))
	// 获取所有的群组
	// groups, err := self.Groups()
	// fmt.Println(groups, err)
	// f, _ := os.Open("稻香-周杰伦.flac")
	// groups.SendFile(f)

	bot.MessageHandler = task
	bot.MessageErrorHandler = func(err error) error {
		fmt.Println(err)
		return err
	}
	return
}
