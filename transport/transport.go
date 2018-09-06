package transport

import "github.com/thesyncim/faye/message"

// handshake, connect, disconnect, subscribe, unsubscribe and publish

type Options struct {
	Url    string
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
	Subscribe(subscription string, onMessage func(message *message.Message)) error
	Unsubscribe(subscription string) error
	Publish(subscription string, message *message.Message) error
}

type Event = string

const (
	Subscribe   Event = "/meta/subscribe"
	Connect     Event = "/meta/connect"
	Unsubscribe Event = "/meta/unsubscribe"
	Handshake   Event = "/meta/handshake"
	Disconnect  Event = "/meta/disconnect"
)

var ControlEvents = []Event{Subscribe, Connect, Unsubscribe, Handshake, Disconnect}

func IsControlMsg(channel string) bool {
	for i := range ControlEvents {
		if channel == ControlEvents[i] {
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
