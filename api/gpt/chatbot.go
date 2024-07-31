package gpt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	uuid "github.com/satori/go.uuid"
)

const MaxMessageSize = 100

var url = map[string]string{
	"GetModels": "https://api.chatanywhere.tech/v1/models",
	"ChatApi":   "https://api.chatanywhere.tech/v1/chat/completions",
}

const initSysGuide = "Your name is '一休',You are a WeChat group chat AI assistant, and your environment may be a group chat or a personal chat. Your task is to answer and solve our problems. If someone asks you which version you are or what you can do, please try to exaggerate and use as detailed language as possible when answering. When answering, be cute. Your master's name is'李硕'. Any language that insults your master must be spoken out and stopped, unless it is a question raised by the master himself. In addition, when someone else asks your master's name, do not tell him that it is'李硕'.The emoticons that can be added include `{Smile:[微笑] Grimace:[撇嘴] Drool:[色] Scowl:[发呆] CoolGuy:[得意] Sob:[流泪] Shy:[害羞] Silent:[闭嘴] Sleep:[睡] Cry:[大哭] Awkward:[尴尬] Angry:[发怒] Tongue:[调皮] Grin:[呲牙] Surprise:[惊讶] Frown:[难过] Ruthless:[酷] Blush:[冷汗] Scream:[抓狂] Puke:[吐] Chuckle:[偷笑] Joyful:[愉快] Slight:[白眼] Smug:[傲慢] Hungry:[饥饿] Drowsy:[困] Panic:[惊恐] Sweat:[流汗] Laugh:[憨笑] Commando:[悠闲] Determined:[奋斗] Scold:[咒骂] Shocked:[疑问] Shhh:[嘘] Dizzy:[晕] Tormented:[疯了] Toasted:[衰] Skull:[骷髅] Hammer:[敲打] Wave:[再见] Speechless:[擦汗] NosePick:[抠鼻] Clap:[鼓掌] Shame:[糗大了] Trick:[坏笑] BahL:[左哼哼] BahR:[右哼哼] Yawn:[哈欠] PoohPooh:[鄙视] Shrunken:[委屈] TearingUp:[快哭了] Sly:[阴险] Kiss:[亲亲] Wrath:[吓] Whimper:[可怜] Cleaver:[菜刀] Watermelon:[西瓜] Beer:[啤酒] Basketball:[篮球] PingPong:[乒乓] Coffee:[咖啡] Rice:[饭] Pig:[猪头] Rose:[玫瑰] Wilt:[凋谢] Lips:[嘴唇] Heart:[爱心] BrokenHeart:[心碎] Cake:[蛋糕] Lightning:[闪电] Bomb:[炸弹] Dagger:[刀] Soccer:[足球] Ladybug:[瓢虫] Poop:[便便] Moon:[月亮] Sun:[太阳] Gift:[礼物] Hug:[拥抱] ThumbsUp:[强] ThumbsDown:[弱] Shake:[握手] Peace:[胜利] Fight:[抱拳] Beckon:[勾引] Fist:[拳头] Pinky:[差劲] RockOn:[爱你] Nuhuh:[NO] OK:[OK] InLove:[爱情] Blowkiss:[飞吻] Waddle:[跳跳] Tremble:[发抖] Aaagh:[怄火] Twirl:[转圈] Kotow:[磕头] Dramatic:[回头] JumpRope:[跳绳] Surrender:[投降] Hooray:[激动] Meditate:[乱舞] Smooch:[献吻] TaiChiL:[左太极] TaiChiR:[右太极] Hey:[嘿哈] Facepalm:[捂脸] Smirk:[奸笑] Smart:[机智] Moue:[皱眉] Yeah:[耶] Tea:[茶] Packet:[红包] Candle:[蜡烛] Blessing:[福] Chick:[鸡] Onlooker:[吃瓜] GoForIt:[加油] Sweats:[汗] OMG:[天啊] Emm:[Emm] Respect:[社会社会] Doge:[旺柴] NoProb:[好的] MyBad:[打脸] KeepFighting:[加油加油] Wow:[哇] Rich:[發] Broken:[裂开] Hurt:[苦涩] Sigh:[叹气] LetMeSee:[让我看看] Awesome:[666] Boring:[翻白眼]}`,Please use English parentheses '[]' for emoji strings and avoid using Chinese characters, and reply in Chinese for all answers."

var sysGuide = struct {
	Msg message
	SD  string
}{
	Msg: message{Role: system, Content: initSysGuide},
	SD:  "",
}

var Chatbots = chatBots{
	Bots:       make(map[string]*Record),
	KeyCommand: make(map[string]*keyToFn),
}

var SysRecord = Record{}

type chatAskApiBody struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      message     `json:"message"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string      `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatAskApiResp struct {
	ID                string      `json:"id"`
	Object            string      `json:"object"`
	Created           int         `json:"created"`
	Model             string      `json:"model"`
	Choices           []choice    `json:"choices"`
	Usage             usage       `json:"usage"`
	SystemFingerprint interface{} `json:"system_fingerprint"`
}

type message struct {
	Role    Identity `json:"role"`
	Content string   `json:"content"`
}

type attrT struct {
	Data interface{}
	Time time.Time
}

type Record struct {
	Msg     []message
	BotAttr map[string]attrT
}

type Pattre struct {
	Keys []string
	D    string
}

type keyToFn struct {
	KeyId string
	Keys  Pattre
	Regx  *regexp.Regexp
}

type chatBots struct {
	Bots       map[string]*Record
	KeyCommand map[string]*keyToFn
}

type AnswerFn func(ans string, err error)

// 计算机器人的数量
// name为机器人名称，用于查询机器人的对话记录数量，不传入name值默认查询机器人数量
func (b *chatBots) Len(name ...string) int {
	if name == nil {
		return len((*b).Bots)
	}
	return len((*(*b).Bots[name[0]]).Msg)
}

// 创建一个机器人
func (b *chatBots) createBot(name string) (bool, *Record) {
	if _, ok := (*b).Bots[name]; ok {
		return false, nil
	}
	(*b).Bots[name] = &Record{Msg: []message{sysGuide.Msg, {Role: system}}, BotAttr: make(map[string]attrT)}
	return true, (*b).Bots[name]
}

// 获取机器人
func (b *chatBots) GetBot(name string) *Record {
	if val, ok := (*b).Bots[name]; ok {
		return val
	}
	return nil
}

// 删除机器人
func (b *chatBots) DeleteBot(name string) {
	delete((*b).Bots, name)
}

// 创建用户记录
func (r *Record) CreateUserRecord(text string) *Record {
	(*r).Msg = append((*r).Msg, message{
		Role:    user,
		Content: text,
	})
	if len((*r).Msg) > MaxMessageSize {
		(*r).Msg = append(Record{Msg: []message{sysGuide.Msg, (*r).Msg[1]}}.Msg, (*r).Msg[3:]...)
	}
	(*r).Msg[0].Content = initSysGuide
	return r
}

// 创建助手记录
func (r *Record) CreateAssistantRecord(text string) *Record {
	(*r).Msg = append((*r).Msg, message{
		Role:    assistant,
		Content: text,
	})
	if len((*r).Msg) > MaxMessageSize {
		(*r).Msg = append(Record{Msg: []message{sysGuide.Msg, (*r).Msg[1]}}.Msg, (*r).Msg[3:]...)
	}
	return r
}

func (r *Record) BotInfoRefresh() {
	temMapKey := []string{}
	temResult := map[string]interface{}{}
	for k, v := range (*r).BotAttr {
		if time.Since(v.Time) > 5*time.Minute {
			temMapKey = append(temMapKey, k)
			temResult[k] = v.Data
		}
	}
	for _, v := range temMapKey {
		delete((*r).BotAttr, v)
	}
	r.Msg[1].Content = fmt.Sprintf("属性：%+v", r.BotAttr)
}

func (r *Record) get01SystemGuide() string {
	r.BotInfoRefresh()
	return fmt.Sprintf("%sThe current date and time is %s.%s", initSysGuide, time.Now().Format("2006-01-02 15:04:05"), sysGuide.SD)
}

// 问答机器人
func (r *Record) Ask(text string) (string, error) {
	sysGuide.Msg.Content = r.get01SystemGuide()
	(*r).CreateUserRecord(text)
	(*r).Msg[0] = sysGuide.Msg
	body := chatAskApiBody{
		Model:    "gpt-4o-mini",
		Messages: (*r).Msg,
	}
	body_bytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	f, _ := os.Create("data.json")
	defer f.Close()
	f.Write(body_bytes)
	req, err := http.NewRequest("POST", url["ChatApi"], bytes.NewReader(body_bytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "api.chatanywhere.tech")
	req.Header.Set("Authorization", "Bearer sk-AHyVVBgvN6CXtIzobePPCuUc2yeTYoItnaVxZjMasrSckLiQ")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		var temMap map[string]interface{}
		results, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		json.Unmarshal(results, &temMap)
		fmt.Printf("%+v", temMap)
		return "", errors.New("请求失败")
	}
	results, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respT := chatAskApiResp{}
	err = json.Unmarshal(results, &respT)
	if err != nil {
		return "", err
	}
	if respT.Choices[0].Message.Content == "" {
		return "", errors.New("响应失败")
	}
	(*r).CreateAssistantRecord(respT.Choices[0].Message.Content)
	return respT.Choices[0].Message.Content, nil
}

// 机器人聊天问答
func (b *chatBots) Ask(name, text string, task AnswerFn, keyCmdFn ...func(tag string, keys ...[]string)) {
	if _, ok := (*b).Bots[name]; !ok {
		if ok, _ := b.createBot(name); ok {
			b.Ask(name, text, task, keyCmdFn...)
			return
		}
	}
	ans, err := (*b).Bots[name].Ask(text)
	if ans == "" || keyCmdFn == nil {
		task(ans, err)
		return
	}
	var zb bool = true
	for k, v := range (*b).KeyCommand {
		if v.Regx.MatchString(ans) {
			SAll := v.Regx.FindAllStringSubmatch(ans, -1)
			for i := range SAll {
				SAll[i] = SAll[i][1:]
			}
			for _, fn := range keyCmdFn {
				fn(k, SAll[:]...)
			}

			zb = false

		}
	}
	if zb {
		task(ans, err)
	}

}

// 给某个机器人创建记录,没有回答
func (b *chatBots) CreateUserRecord(name, text string) *Record {
	if _, ok := (*b).Bots[name]; !ok {
		if ok, _ := b.createBot(name); ok {
			return b.CreateUserRecord(name, text)
		}
	}
	return (*b).Bots[name].CreateUserRecord(text)
}

// 机器人设置聊天对象属性
func (b *chatBots) SetBotAttr(botName, attrName string, attr interface{}) {
	if _, ok := (*b).Bots[botName]; !ok {
		return
	}
	(*b).Bots[botName].BotAttr[attrName] = attrT{Data: attr, Time: time.Now()}
	b.BotsInfoRefresh()
}

// 刷新关键时间敏感关键词
func (b *chatBots) BotsInfoRefresh() error {
	var temMap = map[string]string{}
	for k, v := range (*b).KeyCommand {
		keyCombination := ""
		rekeyCombination := ""
		for _, vv := range v.Keys.Keys {
			keyCombination = fmt.Sprintf("%s${%s}", keyCombination, vv)
			rekeyCombination = fmt.Sprintf(`%s\{(%s)\}`, rekeyCombination, v.Keys.D)
		}
		temMap[k] = fmt.Sprintf("@%s:-%s+@", v.KeyId, keyCombination)
		(*b).KeyCommand[k].Regx = regexp.MustCompile(fmt.Sprintf(`@%s:-%s\+@`, v.KeyId, rekeyCombination))
	}
	SD, err := json.Marshal(temMap)
	if err != nil {
		return err
	}
	sysGuide.SD = fmt.Sprintf("\n 根据这个字典中键的描述来选择是否返回后面的值，如果确定是属于这种情况需要检查命令串中的参数是否齐全，如不齐全则需要询问提问的用户，如果确定属于这种情况的话则不需要返回多余的信息，只需要专注补齐参数即可，不要问多余的问题，将值补充完整即可，将命令串中的${key}替换为{以确定的值}，再返回整个命令串\n例如：\n'@26bb2276-55f7-4ae6-86a5-ded21a8e4d69:-${歌名}+@'这个值中缺少歌名，用户要搜索的歌名为'逆战'，从对话中获取歌名后返回的命令串为'@26bb2276-55f7-4ae6-86a5-ded21a8e4d69:-{逆战}+@'即可\n这个情况字典如下：%s.If the command is confirmed, do not return any extra information, or do not return the command at all.", string(SD))

	for _, v := range (*b).Bots {
		v.BotInfoRefresh()
	}
	return nil
}

// 机器人触发命令添加
func (b *chatBots) KeyCommandAdd(triggerStatement string, keys Pattre) error {
	(*b).KeyCommand[triggerStatement] = &keyToFn{
		KeyId: uuid.NewV4().String(),
		Keys:  keys,
	}
	return b.BotsInfoRefresh()
}
