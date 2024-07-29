package logic

import (
	"fmt"
	"os"
	"strconv"
	"wechatgroupbot/api"
	"wechatgroupbot/api/gpt"

	"github.com/eatmoreapple/openwechat"
)

// 音乐分享
func musicShare(msg *openwechat.Message, val string) {
	api.SearchSong(val, func(songinfo api.SongInfo) {
		if len(songinfo.Song) == 0 {
			msg.ReplyText(fmt.Sprintf("找不到音乐:%s", val))
			return
		}
		if len(songinfo.Song) > 26093458 {
			msg.ReplyText(fmt.Sprintf("歌曲文件太大了,试试在网页上听吧\n歌名：%s\n歌手：%s\n歌曲封面：%s\n文件大小：%s\n\n链接🔗：%s", songinfo.Info.Sn, songinfo.Info.An, songinfo.Info.CoverUrl, songinfo.Info.Size, songinfo.Url))
			gpt.Chatbots.GetBot(msg.FromUserName).CreateAssistantRecord(fmt.Sprintf("已发送歌曲文件：%s", fmt.Sprintf("%s-%s 链接", songinfo.Info.Sn, songinfo.Info.An)))
			return
		}

		f, err := os.Create(fmt.Sprintf("%s-%s.flac", songinfo.Info.Sn, songinfo.Info.An))
		if err != nil {
			msg.ReplyText("系统繁忙,请稍后重试!")
			fmt.Println(err)
			return
		}
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()
		f.Write(songinfo.Song)
		msg.ReplyText(fmt.Sprintf("%s,开始下载", f.Name()))
		f.Seek(0, 0)
		_, err = msg.ReplyFile(f)
		if err != nil {
			msg.ReplyText("裂开,我又出错了!😭")
			fmt.Println(err)
		}
		gpt.Chatbots.GetBot(msg.FromUserName).CreateAssistantRecord(fmt.Sprintf("已发送歌曲文件：%s", fmt.Sprintf("%s-%s.flac", songinfo.Info.Sn, songinfo.Info.An)))
	}, func(songinfo api.SongInfo, err error) {
		if err != nil {
			if songinfo.Lrc.Url != "" {
				msg.ReplyText(fmt.Sprintf(`系统繁忙,歌词自己下载吧!😭
			链接🔗：%s`, songinfo.Lrc.Url))
				return
			}
			msg.ReplyText(`系统繁忙,想看歌词一会儿再试吧！`)
			return
		}
		var lrc []byte
		for _, r := range songinfo.Lrc.Lrclist {
			st, err := strconv.ParseFloat(r.Time, 64)
			if err != nil && st < 0 {
				msg.ReplyText(fmt.Sprintf("歌词解析出现了点问题，自己动手下载吧\n链接🔗：%s", songinfo.Lrc.Url))
				return
			}
			m := int(st / 60)
			s := st - float64(m*60)
			lrc = append(lrc, []byte(fmt.Sprintf("[%02d:%4.2f]%s\n", m, s, r.LineLyric))...)
		}
		f, err := os.Create(fmt.Sprintf("%s-%s-歌词.lrc", songinfo.Info.Sn, songinfo.Info.An))
		if err != nil {
			msg.ReplyText(fmt.Sprintf(`系统繁忙,歌词自己下载吧!😭
			链接🔗：%s`, songinfo.Lrc.Url))
			fmt.Println(err)
			return
		}
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()
		f.Write(lrc)
		f.Seek(0, 0)
		_, err = msg.ReplyFile(f)
		if err != nil {
			msg.ReplyText(fmt.Sprintf(`系统繁忙,歌词自己下载吧!😭
			链接🔗：%s`, songinfo.Lrc.Url))
			fmt.Println(err)
		}

	})

}
