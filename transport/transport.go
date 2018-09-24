package transport

import (
	"github.com/thesyncim/faye/message"
	"net/http"
	"time"
)

//Options represents the connection options to be used by a transport
type Options struct {
	Headers http.Header
	Cookies http.CookieJar

	RetryInterval time.Duration
	DialDeadline  time.Duration
	ReadDeadline  time.Duration
	WriteDeadline time.Duration
}

//Transport represents the transport to be used to comunicate with the faye server
type Transport interface {
	//name returns the transport name
	Name() string
	//Init initializes the transport with the provided options
	Init(endpoint string, options *Options) error
	//Options return the transport Options
	Options() *Options
	//Handshake initiates a connection negotiation by sending a message to the /meta/handshake channel.
	Handshake(msg *message.Message) (*message.Message, error)
	//Init is called  after a client has discovered the server’s capabilities with a handshake exchange,
	//a connection is established by sending a message to the /meta/connect channel
	Connect(msg *message.Message) error
	//Disconnect closes all subscriptions and inform the server to remove any client-related state.
	//any subsequent method call to the transport object will result in undefined behaviour.
	Disconnect(msg *message.Message) error
	//SendMessage sens a message through the transport
	SendMessage(msg *message.Message) error

	SetOnMessageReceivedHandler(onMsg func(msg *message.Message))

	//SetOnTransportUpHandler is called when the transport is connected
	SetOnTransportUpHandler(callback func())

	//SetOnTransportDownHandler is called when the transport goes down
	SetOnTransportDownHandler(callback func(error))

	//handled by dispatcher
	SetOnErrorHandler(onError func(err error))
}

var registeredTransports = map[string]Transport{}

func RegisterTransport(t Transport) {
	registeredTransports[t.Name()] = t //todo validate
}

func GetTransport(name string) Transport {
	return registeredTransports[name]
}
