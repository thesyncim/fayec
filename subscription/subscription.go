package subscription

import (
	"github.com/thesyncim/faye/message"
)

type Unsubscriber interface {
	Unsubscribe(subscription *Subscription) error
}
type Subscription struct {
	id      string //request Subscription ID
	channel string
	ok      chan error //used by
	unsub   Unsubscriber
	msgCh   chan *message.Message
}

func NewSubscription(id string, chanel string, unsub Unsubscriber, msgCh chan *message.Message, ok chan error) *Subscription {
	return &Subscription{
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

func (s *Subscription) SubscriptionResult() chan error {
	return s.ok
}

func (s *Subscription) Unsubscribe() error {
	return s.unsub.Unsubscribe(s)
}
