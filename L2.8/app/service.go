package app

import (
	"github.com/beevik/ntp"
)

// GetTime получает текущее время с NTP-сервера
func GetTime() (string, error) {
	// Получаем время от NTP-сервера
	time, err := ntp.Time("0.beevik-ntp.pool.ntp.org")
	if err != nil {
		return "", err
	}
	// Возвращаем время в виде строки и nil в случае успеха
	return time.String(), nil
}