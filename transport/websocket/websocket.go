package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/subscription"
	"github.com/thesyncim/faye/transport"
	"log"

	"strconv"
	"sync"
	"sync/atomic"
)

const transportName = "websocket"

func init() {
	transport.RegisterTransport(&Websocket{})
}

//Websocket represents an websocket transport for the faye protocol
type Websocket struct {
	topts *transport.Options

	connMu   sync.Mutex
	conn     *websocket.Conn
	clientID string
	msgID    *uint64
	once     sync.Once
	advice   atomic.Value //type message.Advise

	stopCh chan error //todo replace wth context

	subscriptionsMu sync.Mutex //todo thread safe radix tree
	subscriptions   map[string][]*subscription.Subscription

	onPubResponseMu   sync.Mutex //todo sync.Map
	onPublishResponse map[string]func(message *message.Message)
}

var _ transport.Transport = (*Websocket)(nil)

//Init initializes the transport with the provided options
func (w *Websocket) Init(endpoint string, options *transport.Options) error {
	var (
		err   error
		msgID uint64
	)
	w.topts = options
	w.msgID = &msgID
	//w.subs = map[string]chan *message.Message{}
	w.subscriptions = map[string][]*subscription.Subscription{}
	w.onPublishResponse = map[string]func(message *message.Message){}
	w.stopCh = make(chan error)
	w.conn, _, err = websocket.DefaultDialer.Dial(endpoint, options.Headers)
	if err != nil {
		return err
	}
	return nil
}

func (w *Websocket) readWorker() error {
	for {
		select {
		case err := <-w.stopCh:
			return err
		default:
		}
		var payload []message.Message
		err := w.conn.ReadJSON(&payload)
		if err != nil {
			return err
		}
		//dispatch
		msg := &payload[0]
		w.topts.Extensions.ApplyInExtensions(msg)

		if msg.Advice != nil {
			w.handleAdvise(msg.Advice)
		}

		if transport.IsMetaMessage(msg) {
			//handle it
			switch msg.Channel {
			case transport.MetaSubscribe:
				//handle MetaSubscribe resp
				w.subscriptionsMu.Lock()
				subscriptions, ok := w.subscriptions[msg.Subscription]
				if !ok {
					panic("BUG: subscription not registered `" + msg.Subscription + "`")
				}
				if !msg.Successful {
					if msg.GetError() == nil {
						//inject the error if the server returns unsuccessful without error
						msg.Error = fmt.Sprintf("susbscription `%s` failed", msg.Subscription)
					}
					var si = -1
					for i := range subscriptions {
						if subscriptions[i].ID() == msg.Id {
							si = i
							select {
							case subscriptions[i].SubscriptionResult() <- msg.GetError():
								close(subscriptions[i].MsgChannel())
							default:
								log.Println("subscription has no listeners") //todo remove*/
							}
						}
					}
					//remove subscription
					if si > -1 {
						subscriptions = subscriptions[:si+copy(subscriptions[si:], subscriptions[si+1:])]
					}

					w.subscriptions[msg.Subscription] = subscriptions
					//v2
				} else {
					for i := range subscriptions {
						if subscriptions[i].ID() == msg.Id {
							select {
							case subscriptions[i].SubscriptionResult() <- nil:
							default:
								log.Println("subscription has no listeners") //todo remove*/
							}
						}
					}
				}
				w.subscriptionsMu.Unlock()

			}

			continue
		}
		//is Event Message
		//there are 2 types of Event Message
		// 1. Publish
		// 2. Delivery
		if transport.IsEventDelivery(msg) {
			w.subscriptionsMu.Lock()
			subscriptions, ok := w.subscriptions[msg.Channel]

			if ok {
				//send to all listeners
				for i := range subscriptions {
					if subscriptions[i].MsgChannel() != nil {
						select {
						case subscriptions[i].MsgChannel() <- msg:
						default:
							log.Println("subscription has no listeners") //todo remove

						}
					}
				}
			}
			w.subscriptionsMu.Unlock()

			continue
		}

		if transport.IsEventPublish(msg) {
			w.onPubResponseMu.Lock()
			onPublish, ok := w.onPublishResponse[msg.Channel]
			w.onPubResponseMu.Unlock()
			if ok {
				onPublish(msg)
			}
		}

	}
}

//Name returns the transport name (websocket)
func (w *Websocket) Name() string {
	return transportName
}

func (w *Websocket) sendMessage(m *message.Message) error {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	w.topts.Extensions.ApplyOutExtensions(m)
	var payload []message.Message
	payload = append(payload, *m)

again:
	err := w.conn.WriteJSON(payload)
	if websocket.IsUnexpectedCloseError(err) {
		advise := w.advice.Load().(*message.Advise)
		if advise.Reconnect == message.ReconnectNone {
			return err
		}
		//reconnect
		//we should re-register again subscriptions again
		goto again
	}
	return nil
}
func (w *Websocket) nextMsgID() string {
	return strconv.Itoa(int(atomic.AddUint64(w.msgID, 1)))
}

//Options return the transport Options
func (w *Websocket) Options() *transport.Options {
	return w.topts
}

//Handshake initiates a connection negotiation by sending a message to the /meta/handshake channel.
func (w *Websocket) Handshake() (err error) {
	m := message.Message{
		Channel:                  transport.MetaHandshake,
		Version:                  "1.0", //todo const
		SupportedConnectionTypes: []string{transportName},
	}
	err = w.sendMessage(&m)
	if err != nil {
		return err
	}

	var hsResps []message.Message
	if err = w.conn.ReadJSON(&hsResps); err != nil {
		return err
	}

	resp := &hsResps[0]
	w.topts.Extensions.ApplyInExtensions(resp)
	if resp.GetError() != nil {
		return err
	}
	w.clientID = resp.ClientId
	return nil
}

//Connect is called  after a client has discovered the server’s capabilities with a handshake exchange,
//a connection is established by sending a message to the /meta/connect channel
func (w *Websocket) Connect() error {
	m := message.Message{
		Channel:        transport.MetaConnect,
		ClientId:       w.clientID,
		ConnectionType: transportName,
		Id:             w.nextMsgID(),
	}
	go w.readWorker()
	return w.sendMessage(&m)
}

//Disconnect closes all subscriptions and inform the server to remove any client-related state.
//any subsequent method call to the client object will result in undefined behaviour.
func (w *Websocket) Disconnect() error {
	m := message.Message{
		Channel:  transport.MetaDisconnect,
		ClientId: w.clientID,
		Id:       w.nextMsgID(),
	}

	w.stopCh <- nil
	close(w.stopCh)
	w.subscriptionsMu.Lock()
	for i := range w.subscriptions {
		//close all listeners
		for j := range w.subscriptions[i] {
			close(w.subscriptions[i][j].MsgChannel())
		}
		delete(w.subscriptions, i)
	}
	w.subscriptionsMu.Unlock()

	return w.sendMessage(&m)
}

//Subscribe informs the server that messages published to that channel are delivered to itself.
func (w *Websocket) Subscribe(channel string) (*subscription.Subscription, error) {
	id := w.nextMsgID()
	m := &message.Message{
		Channel:      transport.MetaSubscribe,
		ClientId:     w.clientID,
		Subscription: channel,
		Id:           id,
	}

	if err := w.sendMessage(m); err != nil {
		return nil, err
	}

	inMsgCh := make(chan *message.Message, 0)
	subRes := make(chan error)

	var pub = func(data message.Data) (string, error) {
		return w.Publish(channel, data)
	}
	sub := subscription.NewSubscription(id, channel, w.Unsubscribe, pub, inMsgCh, subRes)

	w.subscriptionsMu.Lock()
	w.subscriptions[channel] = append(w.subscriptions[channel], sub)
	w.subscriptionsMu.Unlock()

	//todo timeout here
	err := <-subRes
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println(sub)
	return sub, nil
}

//Unsubscribe informs the server that the client will no longer listen to incoming event messages on
//the specified channel/subscription
func (w *Websocket) Unsubscribe(subscription *subscription.Subscription) error {
	//https://docs.cometd.org/current/reference/#_bayeux_meta_unsubscribe
	w.subscriptionsMu.Lock()
	defer w.subscriptionsMu.Unlock()
	subs, ok := w.subscriptions[subscription.Channel()]
	if ok {
		var si = -1
		for i := range subs {
			if subs[i] == subscription {
				close(subs[i].MsgChannel())
				si = i
			}
		}
		if si > -1 {
			//remove the subscription
			subs = subs[:si+copy(subs[si:], subs[si+1:])]
		}
		w.subscriptions[subscription.Channel()] = subs
		//if no more listeners to this subscription send unsubscribe to server
		if len(subs) == 0 {
			delete(w.subscriptions, subscription.Channel())
			//remove onPublishResponse handler
			w.onPubResponseMu.Lock()
			delete(w.onPublishResponse, subscription.Channel())
			w.onPubResponseMu.Unlock()
			m := &message.Message{
				Channel:      transport.MetaUnsubscribe,
				Subscription: subscription.Channel(),
				ClientId:     w.clientID,
				Id:           w.nextMsgID(),
			}
			return w.sendMessage(m)
		}
	}

	return nil
}

//Publish publishes events on a channel by sending event messages, the server MAY  respond to a publish event
//if this feature is supported by the server use the OnPublishResponse to get the publish status.
func (w *Websocket) Publish(subscription string, data message.Data) (id string, err error) {
	id = w.nextMsgID()
	m := &message.Message{
		Channel:  subscription,
		Data:     data,
		ClientId: w.clientID,
		Id:       id,
	}
	if err = w.sendMessage(m); err != nil {
		return "", err
	}
	return id, nil
}

//OnPublishResponse sets the handler to be triggered if the server replies to the publish request
//according to the spec the server MAY reply to the publish request, so its not guaranteed that this handler will
//ever be triggered
//can be used to identify the status of the published request and for example retry failed published requests
func (w *Websocket) OnPublishResponse(subscription string, onMsg func(message *message.Message)) {
	w.onPubResponseMu.Lock()
	w.onPublishResponse[subscription] = onMsg
	w.onPubResponseMu.Unlock()
}

func (w *Websocket) handleAdvise(m *message.Advise) {
	//todo actually handle the advice
	w.advice.Store(m)
}
