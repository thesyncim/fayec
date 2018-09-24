package dispatcher

import (
	"fmt"
	"github.com/thesyncim/faye/internal/store"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/subscription"
	"github.com/thesyncim/faye/transport"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
)

type Dispatcher struct {
	endpoint string
	//transports map[string]transport.Transport
	transport     transport.Transport
	transportOpts transport.Options

	msgID *uint64

	extensions message.Extensions

	//map requestID
	pendingSubs   map[string]chan error //todo wrap in structure
	pendingSubsMu sync.Mutex
	store         *store.SubscriptionsStore

	publishACKmu sync.Mutex
	publishACK   map[string]chan error

	clientID string
}

func NewDispatcher(endpoint string, tOpts transport.Options, ext message.Extensions) *Dispatcher {
	var msgID uint64
	return &Dispatcher{
		endpoint:      endpoint,
		msgID:         &msgID,
		store:         store.NewStore(100),
		transportOpts: tOpts,
		extensions:    ext,
		publishACK:    map[string]chan error{},
		pendingSubs:   map[string]chan error{},
	}
}

//todo allow multiple transports
func (d *Dispatcher) Connect() error {
	var err error
	if err = d.transport.Init(d.endpoint, &d.transportOpts); err != nil {
		return err
	}

	if err = d.metaHandshake(); err != nil {
		return err
	}
	return d.metaConnect()
}

func (d *Dispatcher) metaHandshake() error {
	m := &message.Message{
		Channel:                  message.MetaHandshake,
		Version:                  "1.0",                        //todo const
		SupportedConnectionTypes: []string{d.transport.Name()}, //todo list all tranports
	}
	d.extensions.ApplyOutExtensions(m)
	handshakeResp, err := d.transport.Handshake(m)
	if err != nil {
		return err
	}
	d.extensions.ApplyInExtensions(handshakeResp)
	if handshakeResp.GetError() != nil {
		return err
	}
	d.clientID = handshakeResp.ClientId
	return nil
}

func (d *Dispatcher) metaConnect() error {
	m := &message.Message{
		Channel:        message.MetaConnect,
		ClientId:       d.clientID,
		ConnectionType: d.transport.Name(),
		Id:             d.nextMsgID(),
	}
	return d.transport.Connect(m)
}

func (d *Dispatcher) Disconnect() error {
	m := &message.Message{
		Channel:  message.MetaDisconnect,
		ClientId: d.clientID,
		Id:       d.nextMsgID(),
	}
	return d.transport.Disconnect(m)
}

func (d *Dispatcher) dispatchMessage(msg *message.Message) {
	d.extensions.ApplyInExtensions(msg)

	if message.IsMetaMessage(msg) {
		//handle it
		switch msg.Channel {
		case message.MetaSubscribe:
			//handle MetaSubscribe resp
			d.pendingSubsMu.Lock()
			confirmCh, ok := d.pendingSubs[msg.Id]
			d.pendingSubsMu.Unlock()
			if !ok {
				panic("BUG: subscription not registered `" + msg.Subscription + "`")
			}

			if !msg.Successful {
				if msg.GetError() == nil {
					//inject the error if the server returns unsuccessful without error
					msg.Error = fmt.Sprintf("susbscription `%s` failed", msg.Subscription)
				}
				confirmCh <- msg.GetError()
				//v2
			} else {
				confirmCh <- nil
			}
			return
		}
	}
	//is Event Message
	//there are 2 types of Event Message
	// 1. Publish
	// 2. Delivery
	if message.IsEventDelivery(msg) {
		subscriptions := d.store.Match(msg.Channel)
		//send to all listeners
		log.Println(subscriptions, msg)
		for i := range subscriptions {
			if subscriptions[i].MsgChannel() != nil {
				select {
				case subscriptions[i].MsgChannel() <- msg:
				default:
					log.Println("subscription has no listeners") //todo remove
				}
			}
		}
		return
	}

	if message.IsEventPublish(msg) {
		d.publishACKmu.Lock()
		publishACK, ok := d.publishACK[msg.Id]
		d.publishACKmu.Unlock()
		if ok {
			publishACK <- msg.GetError()
			close(publishACK)
		}
	}

}

func (d *Dispatcher) SetTransport(t transport.Transport) {
	t.SetOnMessageReceivedHandler(d.dispatchMessage)
	d.transport = t
}

func (d *Dispatcher) nextMsgID() string {
	return strconv.Itoa(int(atomic.AddUint64(d.msgID, 1)))
}

//sendMessage send applies the out extensions and sends a message throught the transport
func (d *Dispatcher) sendMessage(m *message.Message) error {
	d.extensions.ApplyOutExtensions(m)
	return d.transport.SendMessage(m)
}

func (d *Dispatcher) Subscribe(channel string) (*subscription.Subscription, error) {
	id := d.nextMsgID()
	m := &message.Message{
		Channel:      message.MetaSubscribe,
		ClientId:     d.clientID,
		Subscription: channel,
		Id:           id,
	}

	if err := d.transport.SendMessage(m); err != nil {
		return nil, err
	}

	inMsgCh := make(chan *message.Message, 0)
	subscriptionConfirmation := make(chan error)

	d.pendingSubsMu.Lock()
	d.pendingSubs[id] = subscriptionConfirmation
	d.pendingSubsMu.Unlock()

	sub, err := subscription.NewSubscription(channel, d.Unsubscribe, inMsgCh)
	if err != nil {
		return nil, err
	}

	//todo timeout here
	err = <-subscriptionConfirmation
	if err != nil {
		//log.Println(err)
		return nil, err
	}
	d.store.Add(sub)
	return sub, nil

}

func (d *Dispatcher) Unsubscribe(sub *subscription.Subscription) error {
	//https://docs.cometd.org/current/reference/#_bayeux_meta_unsubscribe
	d.store.Remove(sub)
	//if this is last subscription we will send meta unsubscribe to the server
	if d.store.Count(sub.Name()) == 0 {
		d.publishACKmu.Lock()
		delete(d.publishACK, sub.Name())
		d.publishACKmu.Unlock()

		m := &message.Message{
			Channel:      message.MetaUnsubscribe,
			Subscription: sub.Name(),
			ClientId:     d.clientID,
			Id:           d.nextMsgID(),
		}
		return d.transport.SendMessage(m)
	}

	return nil
}

func (d *Dispatcher) Publish(subscription string, data message.Data) (err error) {
	id := d.nextMsgID()

	m := &message.Message{
		Channel:  subscription,
		Data:     data,
		ClientId: d.clientID,
		Id:       id,
	}

	//ack from server
	ack := make(chan error)
	d.publishACKmu.Lock()
	d.publishACK[id] = ack
	d.publishACKmu.Unlock()

	if err = d.sendMessage(m); err != nil {
		return err
	}

	select { //todo timeout
	case err = <-ack:
	}

	d.publishACKmu.Lock()
	delete(d.publishACK, id)
	d.publishACKmu.Unlock()

	if err != nil { //todo retries
		return err
	}

	return nil
}
