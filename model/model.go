package model

import "time"

type Truck struct {
	Date  time.Time
	Count int
}

type TrucksInfo struct {
	Trucks            map[string]Truck
	GlobalAlarmsCount int
}
