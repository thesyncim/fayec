package transport

import (
	"github.com/thesyncim/faye/message"
	"time"
)

// handshake, connect, disconnect, subscribe, unsubscribe and publish

type Options struct {
	Url           string
	RetryInterval time.Duration

	InExt  []message.Extension
	OutExt []message.Extension
	//todo dial timeout
	//todo read/write deadline
}

type Transport interface {
	Name() string
	Init(options *Options) error
	Options() *Options
	Handshake() error
	Connect() error
	Disconnect() error
	Subscribe(subscription string, onMessage func(message message.Data)) error
	Unsubscribe(subscription string) error
	Publish(subscription string, message message.Data) error
}

type Meta = string

const (
	MetaSubscribe   Meta = "/meta/subscribe"
	MetaConnect     Meta = "/meta/connect"
	MetaDisconnect  Meta = "/meta/disconnect"
	MetaUnsubscribe Meta = "/meta/unsubscribe"
	MetaHandshake   Meta = "/meta/handshake"
)

type Reconnect = string

const (
	//ReconnectRetry indicates that a client MAY attempt to reconnect with a /meta/connect message,
	//after the interval (as defined by interval advice field or client-default backoff), and with the same credentials.
	ReconnectRetry Reconnect = "retry"

	//ReconnectHandshake indicates that the server has terminated any prior connection status and the client MUST reconnect
	// with a /meta/handshake message.
	//A client MUST NOT automatically retry when a reconnect advice handshake has been received.
	ReconnectHandshake Reconnect = "handshake"

	//ReconnectNone indicates a hard failure for the connect attempt.
	//A client MUST respect reconnect advice none and MUST NOT automatically retry or handshake.
	ReconnectNone Reconnect = "none"
)

var MetaEvents = []Meta{MetaSubscribe, MetaConnect, MetaUnsubscribe, MetaHandshake, MetaDisconnect}

func IsMetaEvent(channel string) bool {
	for i := range MetaEvents {
		if channel == MetaEvents[i] {
			return true
		}
	}
	return false
}

var registeredTransports = map[string]Transport{}

func RegisterTransport(t Transport) {
	registeredTransports[t.Name()] = t //todo validate
}

func GetTransport(name string) Transport {
	return registeredTransports[name]
}
