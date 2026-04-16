package proxy

import (
    "encoding/json"
    "net/http"
    "time"
    "math"
    "errors"
    "gorm.io/gorm"
    "kwintaj.com/drizzle/model"
)

// db connection
type WeatherProxy struct {
    DB *gorm.DB
}

func NewWeatherProxy(db *gorm.DB) *WeatherProxy {
    return &WeatherProxy{DB: db}
}

type ApiResponse struct {
    Hourly struct {
        Time        []string  `json:"time"`
        Temperature []float64 `json:"temperature_2m"`
        WeatherCode []int     `json:"weather_code"`
    } `json:"hourly"`
}

// weather codes translation
func translateWMO(code int) string {
    switch {
    case code == 0:  return "Clear sky"
    case code <= 1:  return "Partly cloudy"
    case code <= 3:  return "Overcast"
    case code <= 48: return "Fog"
    case code <= 55: return "Drizzle"
    case code <= 65: return "Rain"
    case code <= 67: return "Freezing rain"
    case code <= 71: return "Light snow"
    case code <= 73: return "Snow"
    case code <= 75: return "Heavy snow"
    case code <= 81: return "Showers"
    case code <= 82: return "Heavy showers"
    case code <= 95: return "Thunderstorm"
    case code <= 99: return "Thunderstorm with hail"
    default: return "Unknown"
    }
}

// process 24h of info into day
func processDay(data ApiResponse, startIdx int, city string) model.Weather {
    maxTemp := -100.0
    minTemp := 100.0
    conditionCode := -1
    
    // search for min/max values
    for i := startIdx; i < startIdx + 24; i++ {
        t := data.Hourly.Temperature[i]
        if t > maxTemp { maxTemp = t }
        if t < minTemp { minTemp = t }
        c := data.Hourly.WeatherCode[i]
        if c > conditionCode { conditionCode = c}
    }
    
    dayTime, _ := time.Parse("2006-01-02T15:04", data.Hourly.Time[startIdx])

    return model.Weather{
        City:           city,
        Conditions:     translateWMO(conditionCode),
        TemperatureMax: int(math.Round(maxTemp)),
        TemperatureMin: int(math.Round(minTemp)),
        Day:            dayTime,
    }
}

func locate(city string) (string, error) {
    switch {
    case city == "Cracow":  return "https://api.open-meteo.com/v1/forecast?latitude=50.0614&longitude=19.9366&hourly=temperature_2m,weather_code&timezone=Europe%2FBerlin", nil
    case city == "Warsaw":  return "https://api.open-meteo.com/v1/forecast?latitude=52.2298&longitude=21.0118&hourly=temperature_2m,weather_code&timezone=Europe%2FBerlin", nil
    case city == "Zakopane":  return "https://api.open-meteo.com/v1/forecast?latitude=49.299&longitude=19.9489&hourly=temperature_2m,weather_code&timezone=Europe%2FBerlin", nil
    default: return "", errors.New("Nieznana lokalizacja")
    }
}

func (p *WeatherProxy) queryApi(queryCity string) ([]model.Weather, error) {
    url, err := locate(queryCity)
    if err != nil {
        return nil, err
    }

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var apiData ApiResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
        return nil, err
    }

    today := processDay(apiData, 0, queryCity)
    tomorrow := processDay(apiData, 24, queryCity)

    return []model.Weather{today, tomorrow}, nil
}

func (p *WeatherProxy) updateDatabase(city string, data []model.Weather) {
    p.DB.Where("city = ?", city).Delete(&model.Weather{})
    p.DB.Create(&data)
}

// get weather for today and tomorrow
// cache-aside
func (p *WeatherProxy) GetData(queryCity string) ([]model.Weather, error) {
    var cachedWeather []model.Weather
    threeHoursAgo := time.Now().Add(-3 * time.Hour)
    today := time.Now().Truncate(24 * time.Hour)

    p.DB.Where("city = ? AND day >= ? AND updated_at > ?", queryCity, today, threeHoursAgo).Find(&cachedWeather)
    if len(cachedWeather) == 2 {
        return cachedWeather, nil
    }

    freshData, err := p.queryApi(queryCity)
    if err != nil {
        return nil, err
    }

    p.updateDatabase(queryCity, freshData)

    return freshData, nil
}


