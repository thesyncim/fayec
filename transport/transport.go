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
	Publish(subscription string, message message.Data) (id string, err error)
	//OnPublishResponse sets the handler to be triggered if the server replies to the publish request
	//according to the spec the server MAY reply to the publish request, so its not guaranteed that this handler will
	//ever be triggered
	//can be used to identify the status of the published request and for example retry failed published requests
	OnPublishResponse(subscription string, onMsg func(message *message.Message))
}

type MetaMessage = string

const (
	MetaSubscribe   MetaMessage = "/meta/subscribe"
	MetaConnect     MetaMessage = "/meta/connect"
	MetaDisconnect  MetaMessage = "/meta/disconnect"
	MetaUnsubscribe MetaMessage = "/meta/unsubscribe"
	MetaHandshake   MetaMessage = "/meta/handshake"
)

type EventMessage = int

const (
	EventPublish EventMessage = iota
	EventDelivery
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

var metaMessages = []MetaMessage{MetaSubscribe, MetaConnect, MetaUnsubscribe, MetaHandshake, MetaDisconnect}

func IsMetaMessage(msg *message.Message) bool {
	for i := range metaMessages {
		if msg.Channel == metaMessages[i] {
			return true
		}
	}
	return false
}

func IsEventDelivery(msg *message.Message) bool {
	if msg.Data != nil {
		return true
	}
	return false
}

func IsEventPublish(msg *message.Message) bool {
	return !IsEventDelivery(msg)
}

var registeredTransports = map[string]Transport{}

func RegisterTransport(t Transport) {
	registeredTransports[t.Name()] = t //todo validate
}

func GetTransport(name string) Transport {
	return registeredTransports[name]
}
