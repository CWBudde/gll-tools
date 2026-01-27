package gll

import "testing"

func TestSystemTypeString(t *testing.T) {
	cases := []struct {
		value SystemType
		want  string
	}{
		{SystemTypeLineArray, "LineArray"},
		{SystemTypeCluster, "Cluster"},
		{SystemTypeLoudspeaker, "Loudspeaker"},
		{SystemType(99), "Unknown(99)"},
	}

	for _, tc := range cases {
		if got := tc.value.String(); got != tc.want {
			t.Fatalf("SystemType.String(%d) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
