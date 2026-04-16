package controller

import (
    "net/http"
    "kwintaj.com/drizzle/proxy"
    "github.com/labstack/echo/v4"
)

type WeatherController struct {
    Proxy *proxy.WeatherProxy
}

func (ctrl *WeatherController) GetWeather(c echo.Context) error {
    city := c.QueryParam("city")
    if city == "" {
        city = "Krakow"
    }

    data, err := ctrl.Proxy.GetData(city)

    if err != nil {
        return c.JSON(http.StatusBadGateway, map[string]string{"error": "Nie udało się pobrać danych"})
    }
    
    return c.JSON(http.StatusOK, data)
}