package store

import (
	"github.com/thesyncim/faye/subscription"
	"reflect"
	"testing"
)

var (
	wildcardSubscription, _ = subscription.NewSubscription("a", "/wildcard/*", nil, nil, nil)
	simpleSubscription, _   = subscription.NewSubscription("b", "/foo/bar", nil, nil, nil)
)

func TestStore_Add(t *testing.T) {
	type args struct {
		name string
		subs []*subscription.Subscription
	}

	tests := []struct {
		name     string
		s        *SubscriptionsStore
		args     args
		expected *SubscriptionsStore
	}{

		{
			name: "add one",
			s:    NewStore(10),
			args: args{
				name: "/wildcard/*",
				subs: []*subscription.Subscription{wildcardSubscription},
			},
			expected: &SubscriptionsStore{
				subs: map[string][]*subscription.Subscription{
					"/wildcard/*": {wildcardSubscription},
				},
			},
		},
		{
			name: "add three",
			s:    NewStore(10),
			args: args{
				name: "/wildcard/*",
				subs: []*subscription.Subscription{wildcardSubscription, wildcardSubscription, wildcardSubscription},
			},
			expected: &SubscriptionsStore{
				subs: map[string][]*subscription.Subscription{
					"/wildcard/*": {wildcardSubscription, wildcardSubscription, wildcardSubscription},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for i := range tt.args.subs {
				tt.s.Add(tt.args.subs[i])
			}

			if !reflect.DeepEqual(tt.expected, tt.s) {
				t.Fatalf("expecting :%v got: %v", tt.expected, tt.s)
			}
		})
	}
}

func TestStore_Match(t *testing.T) {
	type args struct {
		name string
	}
	simpleStore := NewStore(0)
	simpleStore.Add(simpleSubscription)
	wildcardStore := NewStore(0)
	wildcardStore.Add(wildcardSubscription)
	tests := []struct {
		name string
		s    *SubscriptionsStore
		args args
		want []*subscription.Subscription
	}{
		{
			name: "match simple",
			s:    simpleStore,
			want: []*subscription.Subscription{simpleSubscription},
			args: args{
				name: "/foo/bar",
			},
		},
		{
			name: "match wildcard 1",
			s:    wildcardStore,
			want: []*subscription.Subscription{wildcardSubscription},
			args: args{
				name: "/wildcard/a",
			},
		},
		{
			name: "match wildcard 2",
			s:    wildcardStore,
			want: []*subscription.Subscription{wildcardSubscription},
			args: args{
				name: "/wildcard/ccc",
			},
		},
		{
			name: "match non existent",
			s:    wildcardStore,
			want: nil,
			args: args{
				name: "/wildcardsadasd",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Match(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SubscriptionsStore.Match() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStore_Remove(t *testing.T) {
	type args struct {
		sub *subscription.Subscription
	}
	tests := []struct {
		name string
		s    *SubscriptionsStore
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.Remove(tt.args.sub)
		})
	}
}
