package store

import (
	"github.com/thesyncim/faye/subscription"
	"sync"
)

type SubscriptionsStore struct {
	mutex sync.Mutex
	subs  map[string][]*subscription.Subscription

	//cache for expanded channel names
	cache map[string]*SubscriptionName
}

func NewStore(size int) *SubscriptionsStore {
	return &SubscriptionsStore{
		subs:  make(map[string][]*subscription.Subscription, size),
		cache: map[string]*SubscriptionName{},
	}
}

func (s *SubscriptionsStore) Add(sub *subscription.Subscription) {
	s.mutex.Lock()
	s.subs[sub.Name()] = append(s.subs[sub.Name()], sub)
	s.mutex.Unlock()
}

//Match returns the subscriptions that match with the specified channel name
//Wildcard subscriptions are matched
func (s *SubscriptionsStore) Match(channel string) []*subscription.Subscription {
	var (
		matches []*subscription.Subscription
		name    *SubscriptionName
		ok      bool
	)
	s.mutex.Lock()
	if name, ok = s.cache[channel]; !ok {
		name = NewName(channel)
		s.cache[channel] = name
	}
	for _, subs := range s.subs {
		for i := range subs {
			if name.Match(subs[i].Name()) {
				matches = append(matches, subs[i])
			}
		}
	}
	s.mutex.Unlock()
	return matches
}

func (s *SubscriptionsStore) Remove(sub *subscription.Subscription) {
	close(sub.MsgChannel())
	s.mutex.Lock()
	for channel, subs := range s.subs {
		for i := range subs {
			if subs[i] == sub {
				//delete the subscription
				subs = subs[:i+copy(subs[i:], subs[i+1:])]
				if len(subs) == 0 {
					delete(s.subs, channel)
				}
				goto end
			}
		}
	}

end:
	s.mutex.Unlock()
}

//RemoveAll removel all subscriptions and close all channels
func (s *SubscriptionsStore) RemoveAll() {
	s.mutex.Lock()
	for i := range s.subs {
		//close all listeners
		for j := range s.subs[i] {
			s.subs[i][j].Unsubscribe()
			close(s.subs[i][j].MsgChannel())
		}
		delete(s.subs, i)
	}
	s.mutex.Unlock()
}

//Count return the number of subscriptions associated with the specified channel
func (s *SubscriptionsStore) Count(channel string) int {
	return len(s.Match(channel))
}
