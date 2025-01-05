package main

import (
	"Go-routine-4594/oem-truck/adapters/controller"
	"Go-routine-4594/oem-truck/adapters/presenter"
	"Go-routine-4594/oem-truck/service"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		srv     *service.Service
		monitor *presenter.Presenter
		conf    controller.MqttConf
		ctx     context.Context
		cancel  context.CancelFunc
		err     error
	)

	ctx, cancel = context.WithCancel(context.Background())

	conf = controller.MqttConf{
		Connection: "ssl://backend.christophe.engineering:8883",
		Topic:      "UAS",
	}

	monitor = presenter.NewPresenter()

	srv = service.NewService(monitor)
	_, err = controller.NewMqtt(conf, -1, ctx, srv)

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM) // Catch SIGINT and SIGTERM
		<-sigChan                                                             // Wait for a signal
		log.Println("Interrupt signal received, shutting down gracefully...")
		cancel() // Cancel the context, which can be used to stop dependent processes.
		os.Exit(0)
	}()

	monitor.Start(cancel, ctx)
	fmt.Println("Exiting...")
	os.Exit(0)
}
