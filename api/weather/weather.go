package weather

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type tags struct {
	Tag     string
	Meaning string
}

var weather_type = []tags{
	{Tag: "observe", Meaning: "观测站数据"},
	// {Tag: "forecast_1h", Meaning: "小时天气预报"},
	// {Tag: "forecast_24h", Meaning: "按天天气预报"},
	// {Tag: "index", Meaning: "指数"},
	{Tag: "alarm", Meaning: "警告"},
	{Tag: "limit", Meaning: "限制"},
	{Tag: "tips", Meaning: "小提示"},
	// {Tag: "rise", Meaning: "日出日落，月出月落"},
	{Tag: "air", Meaning: "空气质量"},
}

type weatherParam struct {
	Source       string
	Province     string
	City         string
	Country      string
	Weather_Type string
}

func (p weatherParam) GetRequest() []byte {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://wis.qq.com/weather/common?source=%s&province=%s&city=%s&country=中国&weather_type=%s", p.Source, p.Province, p.City, p.Weather_Type), nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Host", "wis.qq.com")
	clien := &http.Client{}
	resp, err := clien.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return data
}

func (p *weatherParam) FillCity(city string) error {
	resp, err := http.Get(fmt.Sprintf("https://wis.qq.com/city/like?source=pc&city=%s", city))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var temMap map[string]interface{}
	err = json.Unmarshal(data, &temMap)
	if err != nil {
		return err
	}
	if temMap["status"].(float64) != 200 || len(temMap["data"].(map[string]interface{})) == 0 {
		return fmt.Errorf("city not found")
	}
	for _, v := range temMap["data"].(map[string]interface{}) {
		d := strings.Split(v.(string), ",")
		if len(d) == 2 {
			p.City = strings.Trim(d[1], " ")
			p.Province = d[0]
		}
	}
	return nil
}

type weatherCache struct {
	WeatherParam weatherParam
	Data         map[string]interface{}
	Time         time.Time
}

type reqCache map[string]weatherCache

var rch = reqCache{}

var ErrNotFoundCity = errors.New("city not found")

func GetWeather(city_ string) (map[string]interface{}, error) {
	if v, ok := rch[city_]; ok && time.Since(v.Time) < 5*time.Minute {
		return v.Data, nil
	}
	var reqP weatherParam
	reqP.Source = "pc"
	reqP.Country = "中国"
	if err := reqP.FillCity(city_); err != nil {
		fmt.Println(err)
		return nil, ErrNotFoundCity
	}
	var temMap = map[string]interface{}{}
	var wg sync.WaitGroup
	for _, v := range weather_type {
		wg.Add(1)
		go func(v tags) {
			defer wg.Done()
			reqP.Weather_Type = v.Tag
			d := reqP.GetRequest()
			if d == nil {
				return
			}
			var temMap_ interface{}
			if err := json.Unmarshal(d, &temMap_); err != nil {
				fmt.Println(err)
				return
			}
			temMap[v.Meaning] = temMap_.(map[string]interface{})["data"]
		}(v)

	}
	wg.Wait()
	return temMap, nil
}
