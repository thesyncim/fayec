package subscription

import (
	"github.com/thesyncim/faye/message"
	"regexp"
)

type Unsubscriber func(subscription *Subscription) error

type Publisher func(msg message.Data) (string, error)

type Subscription struct {
	id      string //request Subscription ID
	channel string
	ok      chan error //used by
	unsub   Unsubscriber
	pub     Publisher
	msgCh   chan *message.Message
}

func NewSubscription(id string, chanel string, unsub Unsubscriber, pub Publisher, msgCh chan *message.Message, ok chan error) *Subscription {
	return &Subscription{
		pub:     pub,
		ok:      ok,
		id:      id,
		channel: chanel,
		unsub:   unsub,
		msgCh:   msgCh,
	}
}

func (s *Subscription) OnMessage(onMessage func(msg message.Data)) error {
	var inMsg *message.Message
	for inMsg = range s.msgCh {
		if inMsg.GetError() != nil {
			return inMsg.GetError()
		}
		onMessage(inMsg.Data)
	}
	return nil
}

func (s *Subscription) ID() string {
	return s.id
}

func (s *Subscription) MsgChannel() chan *message.Message {
	return s.msgCh
}

func (s *Subscription) Channel() string {
	return s.channel
}

//todo remove
func (s *Subscription) SubscriptionResult() chan error {
	return s.ok
}

func (s *Subscription) Unsubscribe() error {
	return s.unsub(s)
}

func (s *Subscription) Publish(msg message.Data) (string, error) {
	return s.pub(msg)
}

var validChannelName = regexp.MustCompile(`^\/(((([a-z]|[A-Z])|[0-9])|(\-|\_|\!|\~|\(|\)|\$|\@)))+(\/(((([a-z]|[A-Z])|[0-9])|(\-|\_|\!|\~|\(|\)|\$|\@)))+)*$`)
var validChannelPattern = regexp.MustCompile(`^(\/(((([a-z]|[A-Z])|[0-9])|(\-|\_|\!|\~|\(|\)|\$|\@)))+)*\/\*{1,2}$`)

func IsValidChannel(channel string) bool {
	return validChannelName.MatchString(channel) || validChannelPattern.MatchString(channel)
}
