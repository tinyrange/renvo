package main

import "testing"

func TestAirQualityColor(t *testing.T) {
	tests := []struct {
		name             string
		tvoc             uint16
		red, green, blue uint8
	}{
		{name: "green", tvoc: 0, red: 0, green: 64, blue: 0},
		{name: "orange", tvoc: 300, red: 64, green: 32, blue: 0},
		{name: "red", tvoc: 600, red: 64, green: 0, blue: 0},
		{name: "clamped", tvoc: 5000, red: 64, green: 0, blue: 0},
	}
	for _, test := range tests {
		red, green, blue := airQualityColor(test.tvoc)
		if red != test.red || green != test.green || blue != test.blue {
			t.Errorf("%s: (%d, %d, %d), want (%d, %d, %d)",
				test.name, red, green, blue, test.red, test.green, test.blue)
		}
	}
}
