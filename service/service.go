package service

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"

	"Go-routine-4594/oem-truck/model"
)

// Refactored code
const (
	EquipmentKey = "EquipmentName"
	KeyNameField = "KeyNameAsString"
	ValueField   = "ValueAsString"
)

type IService interface {
	ProcessMsg(msg []byte)
}

type IMonitor interface {
	SendTrucks(data model.TrucksInfo)
}

type Service struct {
	trucksMapLock     *sync.RWMutex
	trucksMap         map[string]model.Truck
	globalCounterLock *sync.RWMutex
	globalCounter     int
	screen            tcell.Screen
	monitor           IMonitor
}

func NewService(m IMonitor) *Service {
	var (
		err    error
		screen tcell.Screen
	)
	screen, err = tcell.NewScreen()
	if err != nil {
		log.Fatalf("Error creating tcell screen: %v", err)
	}

	err = screen.Init()
	if err != nil {
		log.Fatalf("Error initializing tcell: %v", err)
	}

	return &Service{
		trucksMapLock:     new(sync.RWMutex),
		globalCounterLock: new(sync.RWMutex),
		trucksMap:         make(map[string]model.Truck),
		screen:            screen,
		monitor:           m,
	}
}

// Helper function: Processes a single annotation
func (service *Service) processAnnotation(annotation map[string]interface{}, timeAlarm time.Time) {
	// Extract "KeyNameAsString" and "ValueAsString"
	keyName, keyNameOk := annotation[KeyNameField].(string)
	valueName, valueNameOk := annotation[ValueField].(string)

	if !keyNameOk || !valueNameOk {
		fmt.Println("invalid annotation format")
		return
	}

	// Check if it's the EquipmentName and increment count
	if keyName == EquipmentKey {
		// lock the map to update the data
		service.trucksMapLock.Lock()
		// and then release the lock once done
		defer service.trucksMapLock.Unlock()

		if _, ok := service.trucksMap[valueName]; ok {
			t := service.trucksMap[valueName]
			t.Count++
			t.Date = timeAlarm
			service.trucksMap[valueName] = t
		} else {
			t := service.trucksMap[valueName]
			t.Count = 1
			t.Date = timeAlarm
			service.trucksMap[valueName] = t
		}
	}
}

func (service *Service) ProcessMsg(msg []byte) {
	var (
		message    map[string]interface{}
		timeAlarm  time.Time
		timeString string
		layout     string
		err        error
	)

	// lock global counter
	service.globalCounterLock.Lock()
	service.globalCounter++
	service.globalCounterLock.Unlock()

	// Unmarshal JSON message
	err = json.Unmarshal(msg, &message)
	if err != nil {
		fmt.Println("json unmarshal error:", err)
		return
	}

	// get the time of the alarm
	layout = "2006-01-02T15:04:05.999999999Z"
	timeString = message["timestamp"].(string)
	timeAlarm, err = time.Parse(layout, timeString)
	if err != nil {
		timeAlarm = time.Now()
	}
	// Type assert to retrieve annotations
	annotations, ok := message["annotations"].([]interface{})
	if !ok {
		fmt.Println("invalid annotations field")
		return
	}

	// Process each annotation
	for _, annotation := range annotations {
		service.processAnnotation(annotation.(map[string]interface{}), timeAlarm)
	}
	trucksInfo := model.TrucksInfo{
		Trucks:            service.trucksMap,
		GlobalAlarmsCount: service.globalCounter,
	}
	service.monitor.SendTrucks(trucksInfo)
}
