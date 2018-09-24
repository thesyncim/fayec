package extensions

import (
	"github.com/thesyncim/faye/message"
)

type GetStream struct {
	UserID    string `json:"user_id,omitempty"`
	ApiKey    string `json:"api_key,omitempty"`
	Signature string `json:"signature,omitempty"`
}

func NewGetStream(apiKey string, signature string) GetStream {
	return GetStream{
		ApiKey:    apiKey,
		Signature: signature,
	}
}

func (gt GetStream) OutExtension(msg *message.Message) {
	if msg.Channel == string(message.MetaSubscribe) {
		//get useriID
		gt.UserID = msg.Subscription[1:]
		msg.Ext = gt
	}
}
