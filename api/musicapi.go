package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	urlE "net/url"
	"regexp"
	"strconv"
	"strings"

	uuid "github.com/satori/go.uuid"
)

type Song struct {
	Sn         string
	An         string
	Rid        string
	CoverUrl   string
	Album      string
	SampleRate string
	Size       string
	Bitrate    int
}

func getKwRid(name string) Song {
	url := fmt.Sprintf("http://search.kuwo.cn/r.s?&correct=1&stype=comprehensive&encoding=utf8&rformat=json&mobi=1&show_copyright_off=1&searchapi=6&all=%s", name)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error: ", err)
		return Song{}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error: ", err)
		return Song{}
	}
	var data map[string]interface{}
	// f, _ := os.Create("data.json")
	// defer f.Close()
	// f.Write(body)
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error: ", err)
		return Song{}
	}
	musicpage := map[string]interface{}{}
	if data["content"].([]interface{}) == nil || len(data["content"].([]interface{})) == 0 || data["content"].([]interface{}) == nil || len(data["content"].([]interface{})) < 1 || len(data["content"].([]interface{})) < 1 {
		return Song{}
	}
	for _, v := range data["content"].([]interface{}) {
		v, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if vv, okk := v["musicpage"]; okk {
			if vvv, ok := vv.(map[string]interface{}); ok {
				musicpage = vvv
				break
			}
		}
	}
	if len(musicpage) == 0 {
		return Song{}
	}
	rid := musicpage["abslist"].([]interface{})[0].(map[string]interface{})["DC_TARGETID"].(string)
	name = musicpage["abslist"].([]interface{})[0].(map[string]interface{})["SONGNAME"].(string)
	artist := musicpage["abslist"].([]interface{})[0].(map[string]interface{})["ARTIST"].(string)
	coverurl := "https://img1.kuwo.cn/star/albumcover/" + musicpage["abslist"].([]interface{})[0].(map[string]interface{})["web_albumpic_short"].(string)
	album := musicpage["abslist"].([]interface{})[0].(map[string]interface{})["ALBUM"].(string)
	minfo := musicpage["abslist"].([]interface{})[0].(map[string]interface{})["MINFO"].(string)
	minfoR := regexp.MustCompile(`level:ff,bitrate:(\d+),format:flac,size:(.*?);`).FindAllStringSubmatch(minfo, -1)
	song := Song{
		Sn:       name,
		An:       artist,
		Rid:      rid,
		CoverUrl: coverurl,
		Album:    album,
	}
	if len(minfoR) != 0 && len(minfoR[0]) == 3 {
		bitrate, err := strconv.ParseInt(minfoR[0][1], 10, 64)
		if err == nil {
			song.Bitrate = int(bitrate)
			song.Size = minfoR[0][2]
		}
	}
	return song
}

func getKwFlacUrl(rid string) string {
	url := fmt.Sprintf("http://mobi.kuwo.cn/mobi.s?f=web&source=kwplayerhd_ar_4.3.0.8_tianbao_T1A_qirui.apk&type=convert_url_with_sign&rid=%s&br=2000kflac", rid)
	body := downloadKw(url)
	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error: ", err)
		return ""
	}
	url = data["data"].(map[string]interface{})["url"].(string)
	return url
}

func downloadKw(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return body
}

type lrc struct {
	LineLyric string `json:"lineLyric"`
	Time      string `json:"time"`
}

type Lrclist []lrc

func getLyric(rid string) (Lrclist, string, error) {
	uuid := uuid.NewV4().String()
	url := fmt.Sprintf("https://kuwo.cn/openapi/v1/www/lyric/getlyric?musicId=%s&httpsStatus=1&reqId=%s&plat=web_www&from=", rid, uuid)
	resp, err := http.Get(url)
	if err != nil {
		return Lrclist{}, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Lrclist{}, "", err
	}
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return Lrclist{}, "", err
	}
	if data["code"].(float64) != 200 {
		return Lrclist{}, "", fmt.Errorf("get lyric error")
	}
	lrclist := data["data"].(map[string]interface{})["lrclist"].([]interface{})
	var Rlrclist Lrclist
	for _, lrc := range lrclist {
		Rlrclist = append(Rlrclist, struct {
			LineLyric string `json:"lineLyric"`
			Time      string `json:"time"`
		}{
			LineLyric: lrc.(map[string]interface{})["lineLyric"].(string),
			Time:      lrc.(map[string]interface{})["time"].(string),
		})
	}
	return Rlrclist, url, nil
}

type SongInfo struct {
	Song []byte
	// Sn       string
	// An       string
	// CoverUrl string
	Info Song
	Url  string
	Lrc  struct {
		Url     string
		Lrclist Lrclist
	}
}

func SearchSong(name string, fn func(songinfo SongInfo), lyricFn ...func(songinfo SongInfo, err error)) {
	rid := getKwRid(name)
	if rid.Rid == "" {
		fn(SongInfo{})
		return
	}
	url := getKwFlacUrl(rid.Rid)

	if url == "" {
		fn(SongInfo{})
		return
	}
	urlF := strings.SplitN(url, "?", 2)
	urlPrefix := urlF[0] + "?"
	urlSuffix := urlF[1]
	songinfo := SongInfo{
		Song: downloadKw(url),
		Url:  urlPrefix + urlE.QueryEscape(urlSuffix),
		Info: rid,
	}
	fn(songinfo)
	for _, l := range lyricFn {
		lyricT, url, err := getLyric(rid.Rid)
		if err != nil {
			l(songinfo, err)
			continue
		}
		songinfo.Lrc.Lrclist = lyricT
		songinfo.Lrc.Url = url
		l(songinfo, nil)
	}
}
