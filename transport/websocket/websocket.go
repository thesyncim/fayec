package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/transport"

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
	TransportOpts *transport.Options
	conn          *websocket.Conn
	clientID      string
	msgID         *uint64
	once          sync.Once
	advice        atomic.Value //type message.Advise

	stopCh chan error

	subsMu sync.Mutex //todo sync.Map
	subs   map[string]chan *message.Message

	onPubResponseMu   sync.Mutex //todo sync.Map
	onPublishResponse map[string]func(message *message.Message)
}

var _ transport.Transport = (*Websocket)(nil)

//Init initializes the transport with the provided options
func (w *Websocket) Init(options *transport.Options) error {
	var (
		err   error
		msgID uint64
	)
	w.TransportOpts = options
	w.msgID = &msgID
	w.subs = map[string]chan *message.Message{}
	w.stopCh = make(chan error)
	w.conn, _, err = websocket.DefaultDialer.Dial(options.Url, nil)
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

		if msg.Advice != nil {
			w.handleAdvise(msg.Advice)
		}

		if transport.IsMetaMessage(msg) {
			//handle it
			switch msg.Channel {
			case transport.MetaSubscribe:
				//handle MetaSubscribe resp
				if !msg.Successful {
					w.subsMu.Lock()
					subscription, ok := w.subs[msg.Subscription]
					w.subsMu.Unlock()
					if !ok {
						panic("BUG: subscription not registered `" + msg.Subscription + "`")
					}
					if msg.GetError() != nil {
						//inject the error
						msg.Error = fmt.Sprintf("susbscription `%s` failed", msg.Subscription)
					}
					subscription <- msg
					close(subscription)
					w.subsMu.Lock()
					delete(w.subs, msg.Channel)
					w.subsMu.Unlock()
				}
			case transport.MetaUnsubscribe:
				//handle MetaUnsubscribe resp
			case transport.MetaConnect:
				//handle MetaConnect resp

			case transport.MetaDisconnect:
				//handle MetaDisconnect resp

			case transport.MetaHandshake:
				//handle MetaHandshake resp
			}

			continue
		}
		//is Event Message
		//there are 2 types of Event Message
		// 1. Publish
		// 2. Delivery

		if transport.IsEventDelivery(msg) {
			w.subsMu.Lock()
			subscription := w.subs[msg.Channel]
			w.subsMu.Unlock()

			w.applyInExtensions(msg)

			if subscription != nil {
				subscription <- msg
			}
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
	w.applyOutExtensions(m)

	var payload []message.Message
	payload = append(payload, *m)
	return w.conn.WriteJSON(payload)
}
func (w *Websocket) nextMsgID() string {
	return strconv.Itoa(int(atomic.AddUint64(w.msgID, 1)))
}

//Options return the transport Options
func (w *Websocket) Options() *transport.Options {
	return w.TransportOpts
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
	w.applyInExtensions(resp)
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
	//todo expect connect resp from server
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
	w.subsMu.Lock()
	for i := range w.subs {
		close(w.subs[i])
		delete(w.subs, i)
	}
	w.subsMu.Unlock()

	return w.sendMessage(&m)
}

//Subscribe informs the server that messages published to that channel are delivered to itself.
func (w *Websocket) Subscribe(subscription string, onMessage func(data message.Data)) error {
	m := &message.Message{
		Channel:      transport.MetaSubscribe,
		ClientId:     w.clientID,
		Subscription: subscription,
		Id:           w.nextMsgID(),
	}

	if err := w.sendMessage(m); err != nil {
		return err
	}

	//todo validate
	inMsgCh := make(chan *message.Message, 0)

	w.subsMu.Lock()
	w.subs[subscription] = inMsgCh
	w.subsMu.Unlock()

	var inMsg *message.Message
	for inMsg = range inMsgCh {
		if inMsg.GetError() != nil {
			return inMsg.GetError()
		}
		onMessage(inMsg.Data)
	}
	//we we got were means that the subscription was closed
	// return nil for now
	return nil
}

//Unsubscribe informs the server that the client will no longer listen to incoming event messages on
//the specified channel/subscription
func (w *Websocket) Unsubscribe(subscription string) error {
	//https://docs.cometd.org/current/reference/#_bayeux_meta_unsubscribe
	m := &message.Message{
		Channel:      transport.MetaUnsubscribe,
		Subscription: subscription,
		ClientId:     w.clientID,
		Id:           w.nextMsgID(),
	}
	w.subsMu.Lock()
	sub, ok := w.subs[subscription]
	if ok {
		close(sub)
		delete(w.subs, subscription)
	}
	w.subsMu.Unlock()

	return w.sendMessage(m)
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

func (w *Websocket) applyOutExtensions(m *message.Message) {
	for i := range w.TransportOpts.OutExt {
		w.TransportOpts.OutExt[i](m)
	}
}

func (w *Websocket) applyInExtensions(m *message.Message) {
	for i := range w.TransportOpts.InExt {
		w.TransportOpts.InExt[i](m)
	}
}

func (w *Websocket) handleAdvise(m *message.Advise) {
	//todo actually handle the advice
	w.advice.Store(m)
}
