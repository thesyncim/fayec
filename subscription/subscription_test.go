package subscription

import "testing"

/*
 assertEqual( ["/**", "/foo", "/*"],
                   Channel.expand("/foo") )

      assertEqual( ["/**", "/foo/bar", "/foo/*", "/foo/**"],
                   Channel.expand("/foo/bar") )

      assertEqual( ["/**", "/foo/bar/qux", "/foo/bar/*", "/foo/**", "/foo/bar/**"],
*/
func TestIsValidChannel(t *testing.T) {
	type args struct {
		channel string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "single asterisk",
			args: args{
				channel: "/*",
			},
			want: true,
		},
		{
			name: "double asterisk",
			args: args{
				channel: "/**",
			},
			want: true,
		},
		{
			name: "regular channel",
			args: args{
				channel: "/foo",
			},
			want: true,
		},
		{
			name: "regular channel 2",
			args: args{
				channel: "/foo/bar",
			},
			want: true,
		},
		{
			name: "invalid slash ending",
			args: args{
				channel: "/foo/",
			},
			want: false,
		},
		{
			name: "invalid asterisk at the middle",
			args: args{
				channel: "/foo/**/bar",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidChannel(tt.args.channel); got != tt.want {
				t.Errorf("IsValidChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}
