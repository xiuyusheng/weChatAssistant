package logic

import "github.com/eatmoreapple/openwechat"

type taskMap map[string]func(msg *openwechat.Message, val string)

type msg map[int64]string

func (m *msg) Add(msgid int64, text string) {
	(*m)[msgid] = text
}

func (m *msg) Get(msgid int64) (string, bool) {
	if v, ok := (*m)[msgid]; ok {
		return v, true
	}
	return "", false
}

type msgBuffer map[string]*msg

func (m *msgBuffer) Add(name string, msgid int64, text string) {
	if _, ok := (*m)[name]; !ok {
		(*m)[name] = &msg{}
	}
	(*m)[name].Add(msgid, text)
}

func (m *msgBuffer) Get(name string, msgid int64) (string, bool) {
	if _, ok := (*m)[name]; !ok {
		return "", false
	}
	if v, ok := (*m)[name].Get(msgid); ok {
		return v, true
	}
	return "", false
}
