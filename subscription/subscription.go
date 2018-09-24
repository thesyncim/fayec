package subscription

import (
	"errors"
	"github.com/thesyncim/faye/message"
	"regexp"
)

var ErrInvalidChannelName = errors.New("invalid channel channel")

type Unsubscriber func(subscription *Subscription) error

type Subscription struct {
	channel string
	unsub   Unsubscriber
	msgCh   chan *message.Message
}

//todo error
func NewSubscription(chanel string, unsub Unsubscriber, msgCh chan *message.Message) (*Subscription, error) {
	if !IsValidSubscriptionName(chanel) {
		return nil, ErrInvalidChannelName
	}
	return &Subscription{
		channel: chanel,
		unsub:   unsub,
		msgCh:   msgCh,
	}, nil
}

func (s *Subscription) OnMessage(onMessage func(channel string, msg message.Data)) error {
	var inMsg *message.Message
	for inMsg = range s.msgCh {
		if inMsg.GetError() != nil {
			return inMsg.GetError()
		}
		onMessage(inMsg.Channel, inMsg.Data)
	}
	return nil
}

func (s *Subscription) MsgChannel() chan *message.Message {
	return s.msgCh
}

func (s *Subscription) Name() string {
	return s.channel
}

//Unsubscribe ...
func (s *Subscription) Unsubscribe() error {
	return s.unsub(s)
}

//validChannelName channel specifies is the channel is in the format /foo/432/bar
var validChannelName = regexp.MustCompile(`^\/(((([a-z]|[A-Z])|[0-9])|(\-|\_|\!|\~|\(|\)|\$|\@)))+(\/(((([a-z]|[A-Z])|[0-9])|(\-|\_|\!|\~|\(|\)|\$|\@)))+)*$`)

var validChannelPattern = regexp.MustCompile(`^(\/(((([a-z]|[A-Z])|[0-9])|(\-|\_|\!|\~|\(|\)|\$|\@)))+)*\/\*{1,2}$`)

func IsValidSubscriptionName(channel string) bool {
	return validChannelName.MatchString(channel) || validChannelPattern.MatchString(channel)
}

//isValidPublishName
func IsValidPublishName(channel string) bool {
	return validChannelName.MatchString(channel)
}
