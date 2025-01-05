package presenter

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"Go-routine-4594/oem-truck/model"

	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Presenter struct {
	screen      tcell.Screen
	dataCh      chan model.TrucksInfo
	titleStyle  tcell.Style
	textStyle   tcell.Style
	log         zerolog.Logger
	loggingFile *os.File
}

func initLoggingFile() *os.File {
	file, err := os.OpenFile(
		"oem-truck.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0664,
	)
	if err != nil {
		panic(err)
	}
	return file
}

func LogInint(file *os.File) zerolog.Logger {
	return zerolog.New(file).With().Timestamp().Logger()

}

func NewPresenter() *Presenter {
	var (
		err    error
		screen tcell.Screen
	)
	screen, err = tcell.NewScreen()
	if err != nil {
		log.Fatal().Msgf("Error creating tcell screen: %v", err)
	}

	err = screen.Init()
	if err != nil {
		log.Fatal().Msgf("Error initializing tcell: %v", err)
	}

	loggingFile := initLoggingFile()

	return &Presenter{
		screen:      screen,
		dataCh:      make(chan model.TrucksInfo, 5),
		titleStyle:  tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true),
		textStyle:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		log:         LogInint(loggingFile),
		loggingFile: loggingFile,
	}
}

func (p *Presenter) Start(cancel func(), ctx context.Context) {
	var (
		truckCount int
		alarmCount int
	)

	go p.listenKey(cancel)

	p.log.Info().Msg("Starting")

	for {
		// clear the screen
		p.screen.Clear()
		p.title(alarmCount, truckCount)

		select {
		case data := <-p.dataCh:
			p.log.Info().Msgf("Received data size: %d", len(data.Trucks))
			dataString := sortMapByKey(data.Trucks)
			p.displayMap(dataString)
			truckCount = len(data.Trucks)
			alarmCount = data.GlobalAlarmsCount
		//	_ = dataString
		//	p.log.Debug().Msg("leaving select")
		//	p.log.Trace().Msgf("Data: %s", dataString)
		case <-time.After(5 * time.Second):
			p.log.Info().Msg("Timeout occurred, no data received")

		case <-ctx.Done():
			p.log.Info().Msg("Context Done")
			return

		default:
			p.log.Debug().Msg("No data available, skipping")
		}

		// Update screen
		p.screen.Show()

		time.Sleep(1 * time.Second)

	}
}

func (p *Presenter) listenKey(cancel func()) {
	for {
		// Poll event
		evP := p.screen.PollEvent()

		switch ev := evP.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				// closing everything
				p.screen.Fini()
				p.log.Info().Msg("Exiting")
				p.loggingFile.Close()
				cancel()
				return
			} else if ev.Key() == tcell.KeyCtrlL {
				p.screen.Sync()
			} else if ev.Rune() == 'C' || ev.Rune() == 'c' {
				p.screen.Clear()
			}
		}
	}
}

func (p *Presenter) Stop() {

}

func (p *Presenter) SendTrucks(data model.TrucksInfo) {
	p.log.Debug().Msgf("Sending data: %d", len(data.Trucks))
	if p.dataCh != nil {
		if len(p.dataCh) < 5 {
			p.dataCh <- data
			p.log.Debug().Msg("Channel successful send data")
			return
		}
		p.log.Debug().Msg("Channel is full")
		return
	}
	p.log.Debug().Msg("Channel is nil")
}

func (p *Presenter) title(counter int, trucks int) {
	// Current time
	currentTime := time.Now().Local()
	// Format the time up to the second
	formattedTime := currentTime.Format("2006-01-02 15:04:05")

	writeText(p.screen, 0, 0, p.titleStyle, fmt.Sprintf("Truck Alarm Monitor"))
	writeText(p.screen, 0, 1, p.titleStyle, fmt.Sprintf("Time: %-24s -- Alarm Global Counter: %-4d", formattedTime, counter))
	writeText(p.screen, 31, 2, p.titleStyle, fmt.Sprintf("--          Truck Count: %-4d", trucks))
}

func (p *Presenter) debug(msg string) {
	writeText(p.screen, 0, 3, p.titleStyle, msg)
}

func (p *Presenter) displayMap(data []string) {

	// Display the data in the map
	row := 4
	dataStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	for _, truck := range data {
		//writeText(screen, 0, row, dataStyle, fmt.Sprintf("Truck: %s - Alarms: %d", truck, count))
		writeText(p.screen, 0, row, dataStyle, truck)
		row++
	}

}

func writeText(screen tcell.Screen, x, y int, style tcell.Style, text string) {
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

func sortMapByKey(data map[string]model.Truck) []string {
	// Get all keys from the map
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	// Sort the keys
	sort.Strings(keys)

	// Build the ordered result
	result := make([]string, 0, len(data))
	for _, key := range keys {
		result = append(result, fmt.Sprintf("Truck: %-9s - Alarms: %4d - Last alarm: %s-24", key, data[key].Count, data[key].Date.Local().String()))
	}

	return result
}
