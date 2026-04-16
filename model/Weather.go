package model

import (
    "time"
    "gorm.io/gorm"
)

type Weather struct {
    gorm.Model
    City            string      `json:"city"`
    Conditions      string      `json:"conditions"`
    TemperatureMax  int         `json:"temperature_max"`
    TemperatureMin  int         `json:"temperature_min"`
    Day             time.Time   `json:"day"`
}
