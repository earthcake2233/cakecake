package hotsearch

import "testing"

func TestDedupKey(t *testing.T) {
	tests := []struct {
		name      string
		userID    uint64
		clientKey string
		norm      string
		want      string
	}{
		{"logged in user", 42, "1.2.3.4", "hello", "hotsearch:dedup:u:42:hello"},
		{"anonymous with ip", 0, "1.2.3.4", "world", "hotsearch:dedup:ip:1.2.3.4:world"},
		{"anonymous no ip", 0, "", "test", "hotsearch:dedup:anon:test"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := prefixHotSearchDed + dedupIdentity(tc.userID, tc.clientKey) + ":" + tc.norm
			if got != tc.want {
				t.Errorf("dedup key(%d, %q, %q) = %q, want %q",
					tc.userID, tc.clientKey, tc.norm, got, tc.want)
			}
		})
	}
}
