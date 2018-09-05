package extensions

import (
	"github.com/thesyncim/faye/message"
	"github.com/thesyncim/faye/transport"
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

func (gt GetStream) OutExtension(message *message.Message) {
	if message.Channel == string(transport.Subscribe) {
		//get useriID
		gt.UserID = message.Subscription[1:]
		message.Ext = gt
	}
}
