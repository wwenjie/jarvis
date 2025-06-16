package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"server/framework/logger"
	"server/framework/redis"
)

const (
	// 天气 API 配置
	weatherAPIKey     = "SENIVERSE_WEATHER_API_KEY" // 从环境变量获取
	weatherAPIBaseURL = "https://api.seniverse.com/v3"

	// 缓存配置
	weatherCachePrefix = "weather:"
	weatherCacheTTL    = 30 * time.Minute // 天气数据缓存30分钟
	hourlyCachePrefix  = "hourly:"
	hourlyCacheTTL     = 1 * time.Hour // 24小时预报缓存1小时
	dailyCachePrefix   = "daily:"
	dailyCacheTTL      = 3 * time.Hour // 15天预报缓存3小时
	lifeCachePrefix    = "life:"
	lifeCacheTTL       = 1 * time.Hour // 生活指数缓存1小时
)

// WeatherData 实时天气数据结构
type WeatherData struct {
	Location    string    `json:"location"`
	Weather     string    `json:"weather"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	WindSpeed   float64   `json:"wind_speed"`
	WindDir     string    `json:"wind_direction"`
	UpdateTime  time.Time `json:"update_time"`
}

// HourlyWeatherData 24小时天气预报数据结构
type HourlyWeatherData struct {
	Location string `json:"location"`
	Hourly   []struct {
		Time        string  `json:"time"`
		Weather     string  `json:"weather"`
		Temperature float64 `json:"temperature"`
		Humidity    float64 `json:"humidity"`
		WindSpeed   float64 `json:"wind_speed"`
		WindDir     string  `json:"wind_direction"`
	} `json:"hourly"`
}

// DailyWeatherData 15天天气预报数据结构
type DailyWeatherData struct {
	Location string `json:"location"`
	Daily    []struct {
		Date      string  `json:"date"`
		TextDay   string  `json:"text_day"`
		TextNight string  `json:"text_night"`
		HighTemp  float64 `json:"high_temp"`
		LowTemp   float64 `json:"low_temp"`
		Rainfall  float64 `json:"rainfall"`
		Precip    float64 `json:"precip"`
		WindDir   string  `json:"wind_direction"`
		WindSpeed float64 `json:"wind_speed"`
		WindScale string  `json:"wind_scale"`
		Humidity  float64 `json:"humidity"`
	} `json:"daily"`
}

// LifeIndexData 生活指数数据结构
type LifeIndexData struct {
	Location   string `json:"location"`
	Suggestion []struct {
		Name    string `json:"name"`
		Brief   string `json:"brief"`
		Details string `json:"details"`
	} `json:"suggestion"`
}

// SeniverseWeatherResponse 心知天气 API 响应结构
type SeniverseWeatherResponse struct {
	Results []struct {
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
		Now struct {
			Text        string `json:"text"`           // 天气状况
			Temperature string `json:"temperature"`    // 温度
			Humidity    string `json:"humidity"`       // 相对湿度
			WindSpeed   string `json:"wind_speed"`     // 风速
			WindDir     string `json:"wind_direction"` // 风向
		} `json:"now"`
		LastUpdate string `json:"last_update"` // 数据更新时间
	} `json:"results"`
}

// LocationInfo 地理位置信息
type LocationInfo struct {
	Province string `json:"province"`
	City     string `json:"city"`
	District string `json:"district"`
	Address  string `json:"address"`
}

// ParseLocation 解析地理位置
func ParseLocation(ctx context.Context, location string) (*LocationInfo, error) {
	// 尝试从缓存获取
	cacheKey := "location:" + location
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var locInfo LocationInfo
		if err := json.Unmarshal([]byte(cached), &locInfo); err == nil {
			return &locInfo, nil
		}
	}

	// 从环境变量获取配置
	apiKey := os.Getenv("LOCATION_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 LOCATION_API_KEY 未设置")
	}
	baseURL := os.Getenv("LOCATION_API_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("环境变量 LOCATION_API_BASE_URL 未设置")
	}

	// 构建请求
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/geocode?address=%s", baseURL, location), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var apiResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
			Address  string `json:"address"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应状态
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API请求失败: %s", apiResp.Message)
	}

	// 构建位置信息
	locInfo := &LocationInfo{
		Province: apiResp.Data.Province,
		City:     apiResp.Data.City,
		District: apiResp.Data.District,
		Address:  apiResp.Data.Address,
	}

	// 缓存位置信息
	if locJSON, err := json.Marshal(locInfo); err == nil {
		redis.Set(ctx, cacheKey, string(locJSON), 24*time.Hour) // 缓存24小时
	}

	return locInfo, nil
}

// GetWeather 获取实时天气
func GetWeather(ctx context.Context, location string) (*WeatherData, error) {
	// 尝试从缓存获取
	cacheKey := weatherCachePrefix + location
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var weather WeatherData
		if err := json.Unmarshal([]byte(cached), &weather); err == nil {
			return &weather, nil
		}
	}

	// 从环境变量获取配置
	apiKey := os.Getenv(weatherAPIKey)
	if apiKey == "" {
		logger.Errorf("环境变量 %s 未设置", weatherAPIKey)
		return nil, fmt.Errorf("环境变量 %s 未设置", weatherAPIKey)
	}

	// 构建请求
	url := fmt.Sprintf("%s/weather/now.json?key=%s&location=%s&language=zh-Hans&unit=c",
		weatherAPIBaseURL, apiKey, location)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Errorf("创建请求失败: %v", err)
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("发送请求失败: %v", err)
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取响应失败: %v", err)
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 打印原始响应
	logger.Infof("天气API原始响应: %s", string(body))

	// 解析响应
	var apiResp struct {
		Results []struct {
			Location struct {
				Name string `json:"name"`
			} `json:"location"`
			Now struct {
				Text        string `json:"text"`           // 天气状况
				Temperature string `json:"temperature"`    // 温度
				Humidity    string `json:"humidity"`       // 相对湿度
				WindSpeed   string `json:"wind_speed"`     // 风速
				WindDir     string `json:"wind_direction"` // 风向
			} `json:"now"`
			LastUpdate string `json:"last_update"` // 数据更新时间
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		logger.Errorf("解析响应失败: %v", err)
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应
	if len(apiResp.Results) == 0 {
		logger.Errorf("未获取到天气数据")
		return nil, fmt.Errorf("未获取到天气数据")
	}

	result := apiResp.Results[0]
	logger.Infof("解析后的天气数据: %+v", result)

	// 解析温度
	temperature, err := strconv.ParseFloat(result.Now.Temperature, 64)
	if err != nil {
		logger.Errorf("解析温度失败: %v, 原始值: %s", err, result.Now.Temperature)
		return nil, fmt.Errorf("解析温度失败: %v", err)
	}

	// 解析湿度
	var humidity float64
	if result.Now.Humidity != "" {
		humidity, err = strconv.ParseFloat(result.Now.Humidity, 64)
		if err != nil {
			logger.Errorf("解析湿度失败: %v, 原始值: %s", err, result.Now.Humidity)
			return nil, fmt.Errorf("解析湿度失败: %v", err)
		}
	}

	// 解析风速
	var windSpeed float64
	if result.Now.WindSpeed != "" {
		windSpeed, err = strconv.ParseFloat(result.Now.WindSpeed, 64)
		if err != nil {
			logger.Errorf("解析风速失败: %v, 原始值: %s", err, result.Now.WindSpeed)
			return nil, fmt.Errorf("解析风速失败: %v", err)
		}
	}

	// 解析更新时间
	updateTime, err := time.Parse("2006-01-02T15:04:05+08:00", result.LastUpdate)
	if err != nil {
		updateTime = time.Now()
	}

	// 构建天气数据
	weather := &WeatherData{
		Location:    result.Location.Name,
		Weather:     result.Now.Text,
		Temperature: temperature,
		Humidity:    humidity,
		WindSpeed:   windSpeed,
		WindDir:     result.Now.WindDir,
		UpdateTime:  updateTime,
	}

	// 缓存天气数据
	if weatherJSON, err := json.Marshal(weather); err == nil {
		redis.Set(ctx, cacheKey, string(weatherJSON), weatherCacheTTL)
	}

	return weather, nil
}

// GetHourlyWeather 获取24小时天气预报
func GetHourlyWeather(ctx context.Context, location string) (*HourlyWeatherData, error) {
	// 尝试从缓存获取
	cacheKey := hourlyCachePrefix + location
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var weather HourlyWeatherData
		if err := json.Unmarshal([]byte(cached), &weather); err == nil {
			return &weather, nil
		}
	}

	// 从环境变量获取配置
	apiKey := os.Getenv(weatherAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 %s 未设置", weatherAPIKey)
	}

	// 构建请求
	url := fmt.Sprintf("%s/weather/hourly.json?key=%s&location=%s&language=zh-Hans&unit=c&start=0&hours=24",
		weatherAPIBaseURL, apiKey, location)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var apiResp struct {
		Results []struct {
			Location struct {
				Name string `json:"name"`
			} `json:"location"`
			Hourly []struct {
				Time        string `json:"time"`
				Text        string `json:"text"`
				Temperature string `json:"temperature"`
				Humidity    string `json:"humidity"`
				WindSpeed   string `json:"wind_speed"`
				WindDir     string `json:"wind_direction"`
			} `json:"hourly"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应
	if len(apiResp.Results) == 0 {
		return nil, fmt.Errorf("未获取到天气预报数据")
	}

	result := apiResp.Results[0]
	hourlyWeather := &HourlyWeatherData{
		Location: result.Location.Name,
		Hourly: make([]struct {
			Time        string  `json:"time"`
			Weather     string  `json:"weather"`
			Temperature float64 `json:"temperature"`
			Humidity    float64 `json:"humidity"`
			WindSpeed   float64 `json:"wind_speed"`
			WindDir     string  `json:"wind_direction"`
		}, len(result.Hourly)),
	}

	// 解析每小时天气数据
	for i, hour := range result.Hourly {
		temperature, _ := strconv.ParseFloat(hour.Temperature, 64)
		humidity, _ := strconv.ParseFloat(hour.Humidity, 64)
		windSpeed, _ := strconv.ParseFloat(hour.WindSpeed, 64)

		hourlyWeather.Hourly[i] = struct {
			Time        string  `json:"time"`
			Weather     string  `json:"weather"`
			Temperature float64 `json:"temperature"`
			Humidity    float64 `json:"humidity"`
			WindSpeed   float64 `json:"wind_speed"`
			WindDir     string  `json:"wind_direction"`
		}{
			Time:        hour.Time,
			Weather:     hour.Text,
			Temperature: temperature,
			Humidity:    humidity,
			WindSpeed:   windSpeed,
			WindDir:     hour.WindDir,
		}
	}

	// 缓存数据
	if weatherJSON, err := json.Marshal(hourlyWeather); err == nil {
		redis.Set(ctx, cacheKey, string(weatherJSON), hourlyCacheTTL)
	}

	return hourlyWeather, nil
}

// GetDailyWeather 获取15天天气预报
func GetDailyWeather(ctx context.Context, location string) (*DailyWeatherData, error) {
	// 尝试从缓存获取
	cacheKey := dailyCachePrefix + location
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var weather DailyWeatherData
		if err := json.Unmarshal([]byte(cached), &weather); err == nil {
			return &weather, nil
		}
	}

	// 从环境变量获取配置
	apiKey := os.Getenv(weatherAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 %s 未设置", weatherAPIKey)
	}

	// 构建请求
	url := fmt.Sprintf("%s/weather/daily.json?key=%s&location=%s&language=zh-Hans&unit=c&start=0&days=15",
		weatherAPIBaseURL, apiKey, location)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var apiResp struct {
		Results []struct {
			Location struct {
				Name string `json:"name"`
			} `json:"location"`
			Daily []struct {
				Date      string `json:"date"`
				TextDay   string `json:"text_day"`
				TextNight string `json:"text_night"`
				HighTemp  string `json:"high"`
				LowTemp   string `json:"low"`
				Rainfall  string `json:"rainfall"`
				Precip    string `json:"precip"`
				WindDir   string `json:"wind_direction"`
				WindSpeed string `json:"wind_speed"`
				WindScale string `json:"wind_scale"`
				Humidity  string `json:"humidity"`
			} `json:"daily"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应
	if len(apiResp.Results) == 0 {
		return nil, fmt.Errorf("未获取到天气预报数据")
	}

	result := apiResp.Results[0]
	dailyWeather := &DailyWeatherData{
		Location: result.Location.Name,
		Daily: make([]struct {
			Date      string  `json:"date"`
			TextDay   string  `json:"text_day"`
			TextNight string  `json:"text_night"`
			HighTemp  float64 `json:"high_temp"`
			LowTemp   float64 `json:"low_temp"`
			Rainfall  float64 `json:"rainfall"`
			Precip    float64 `json:"precip"`
			WindDir   string  `json:"wind_direction"`
			WindSpeed float64 `json:"wind_speed"`
			WindScale string  `json:"wind_scale"`
			Humidity  float64 `json:"humidity"`
		}, len(result.Daily)),
	}

	// 解析每日天气数据
	for i, day := range result.Daily {
		highTemp, _ := strconv.ParseFloat(day.HighTemp, 64)
		lowTemp, _ := strconv.ParseFloat(day.LowTemp, 64)
		rainfall, _ := strconv.ParseFloat(day.Rainfall, 64)
		precip, _ := strconv.ParseFloat(day.Precip, 64)
		windSpeed, _ := strconv.ParseFloat(day.WindSpeed, 64)
		humidity, _ := strconv.ParseFloat(day.Humidity, 64)

		dailyWeather.Daily[i] = struct {
			Date      string  `json:"date"`
			TextDay   string  `json:"text_day"`
			TextNight string  `json:"text_night"`
			HighTemp  float64 `json:"high_temp"`
			LowTemp   float64 `json:"low_temp"`
			Rainfall  float64 `json:"rainfall"`
			Precip    float64 `json:"precip"`
			WindDir   string  `json:"wind_direction"`
			WindSpeed float64 `json:"wind_speed"`
			WindScale string  `json:"wind_scale"`
			Humidity  float64 `json:"humidity"`
		}{
			Date:      day.Date,
			TextDay:   day.TextDay,
			TextNight: day.TextNight,
			HighTemp:  highTemp,
			LowTemp:   lowTemp,
			Rainfall:  rainfall,
			Precip:    precip,
			WindDir:   day.WindDir,
			WindSpeed: windSpeed,
			WindScale: day.WindScale,
			Humidity:  humidity,
		}
	}

	// 缓存数据
	if weatherJSON, err := json.Marshal(dailyWeather); err == nil {
		redis.Set(ctx, cacheKey, string(weatherJSON), dailyCacheTTL)
	}

	return dailyWeather, nil
}

// GetLifeIndex 获取生活指数
func GetLifeIndex(ctx context.Context, location string) (*LifeIndexData, error) {
	// 尝试从缓存获取
	cacheKey := lifeCachePrefix + location
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var lifeIndex LifeIndexData
		if err := json.Unmarshal([]byte(cached), &lifeIndex); err == nil {
			return &lifeIndex, nil
		}
	}

	// 从环境变量获取配置
	apiKey := os.Getenv(weatherAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 %s 未设置", weatherAPIKey)
	}

	// 构建请求
	url := fmt.Sprintf("%s/life/suggestion.json?key=%s&location=%s&language=zh-Hans",
		weatherAPIBaseURL, apiKey, location)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var apiResp struct {
		Results []struct {
			Location struct {
				Name string `json:"name"`
			} `json:"location"`
			Suggestion []struct {
				Name    string `json:"name"`
				Brief   string `json:"brief"`
				Details string `json:"details"`
			} `json:"suggestion"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应
	if len(apiResp.Results) == 0 {
		return nil, fmt.Errorf("未获取到生活指数数据")
	}

	result := apiResp.Results[0]
	lifeIndex := &LifeIndexData{
		Location:   result.Location.Name,
		Suggestion: result.Suggestion,
	}

	// 缓存数据
	if lifeJSON, err := json.Marshal(lifeIndex); err == nil {
		redis.Set(ctx, cacheKey, string(lifeJSON), lifeCacheTTL)
	}

	return lifeIndex, nil
}
