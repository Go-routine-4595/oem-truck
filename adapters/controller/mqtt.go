package controller

import (
	"Go-routine-4594/oem-truck/service"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	pmqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
	uuid "github.com/satori/go.uuid"
	"os"
	"time"
)

type MqttConf struct {
	Connection string `yaml:"Connection"`
	Topic      string `yaml:"Topic"`
}

type Mqtt struct {
	Topic    string
	MgtUrl   string
	logger   zerolog.Logger
	opt      *pmqtt.ClientOptions
	ClientID uuid.UUID
	client   pmqtt.Client
	srv      service.IService
}

func NewMqtt(conf MqttConf, logl int, ctx context.Context, srv service.IService) (*Mqtt, error) {
	var (
		err error
		l   zerolog.Logger
		cid uuid.UUID
		//opt *pmqtt.ClientOptions
	)

	l = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).Level(zerolog.InfoLevel+zerolog.Level(logl)).With().Timestamp().Int("pid", os.Getpid()).Logger()
	cid = uuid.NewV4()
	c := &Mqtt{
		Topic:    conf.Topic,
		MgtUrl:   conf.Connection,
		logger:   l,
		ClientID: cid,
		opt: pmqtt.NewClientOptions().
			AddBroker(conf.Connection).
			SetClientID("oem-truck-monitor-" + cid.String()).
			SetCleanSession(true).
			SetAutoReconnect(true).
			SetTLSConfig(&tls.Config{
				InsecureSkipVerify: true,
			}).
			SetConnectionLostHandler(ConnectLostHandler(l)).
			SetOnConnectHandler(ConnectHandler(l)),
		srv: srv,
	}

	go func() {
		<-ctx.Done()
		c.client.Disconnect(250)
		c.logger.Warn().Msg("Mqtt disconnect")
	}()

	err = c.Connect()
	if err == nil {
		c.test()
	}

	return c, err
}

func (m *Mqtt) test() {

	dump := struct {
		Message string `json:"message"`
		Uuidc   string `json:"uuid_client"`
		Tm      string `json:"tm"`
	}{
		Message: "oem-bridge-mqtt test Message",
		Uuidc:   m.ClientID.String(),
		Tm:      time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.Marshal(dump)
	if err != nil {
		m.logger.Error().Err(err).Msg("Mqtt test message error while marshaling")
	}
	token := m.client.Publish("topic/test", 0, false, b)
	m.logger.Info().Str("test", string(b)).Str("topic", "topic/test").Msg("test message send on topic")
	token.Wait()

}

func (m *Mqtt) ProcessMessage(_ pmqtt.Client, msg pmqtt.Message) {
	m.srv.ProcessMsg(msg.Payload())
}

// SendAlarmRaw sends a raw alarm message to the MQTT broker and returns an error if the sending fails.
func (m *Mqtt) SendAlarmRaw(b []byte) error {
	var (
		token pmqtt.Token
	)

	token = m.client.Publish(m.Topic, 1, false, b)
	if token.WaitTimeout(200*time.Millisecond) && token.Error() != nil {
		m.logger.Error().Err(token.Error()).Str("event", fmt.Sprintf("%v", string(b))).Msg("Timeout exceeded during publishing")
	}
	return nil
}

// Disconnect terminates the connection to the MQTT broker and logs the disconnection event.
func (m *Mqtt) Disconnect() {
	m.client.Unsubscribe(m.Topic)
	m.client.Disconnect(500)
	m.logger.Info().Msg("Disconnected from mqtt broker")
	m.client = nil
}

func (m *Mqtt) Connect() error {
	m.client = pmqtt.NewClient(m.opt)
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		m.logger.Error().Err(token.Error()).Msg("error connecting to mqtt broker")
		return errors.Join(token.Error(), errors.New("error connecting to mqtt broker"))
	}
	m.client.Subscribe(m.Topic, 0, m.ProcessMessage)
	return nil
}

// ConnectHandler returns a function to handle successful connections to the MQTT broker.
// The returned function logs an informational message indicating a successful connection.
func (m *Mqtt) ConnectHandler() func(client pmqtt.Client) {
	return func(client pmqtt.Client) {
		m.logger.Info().Msg("Connected to mqtt broker")
	}
}

// ConnectLostHandler returns a function to handle lost connections to the MQTT broker.
// The returned function logs a warning message indicating a lost connection along with the error encountered.
func (m *Mqtt) ConnectLostHandler() func(client pmqtt.Client, err error) {
	return func(client pmqtt.Client, err error) {
		m.logger.Warn().Err(err).Msg("Connection Lost")
	}
}

func ConnectHandler(logger zerolog.Logger) func(client pmqtt.Client) {
	return func(client pmqtt.Client) {
		logger.Info().Msg("Connected to mqtt broker")
	}
}

func ConnectLostHandler(logger zerolog.Logger) func(client pmqtt.Client, err error) {
	return func(client pmqtt.Client, err error) {
		logger.Warn().Err(err).Msg("Connection Lost")
	}
}
