package log

import "testing"

func TestLevel_String(t *testing.T) {
	t.Run("debug level", func(t *testing.T) {
		l := LevelDebug
		if got, want := l.String(), "debug"; got != want {
			t.Errorf("l.String() = %s, want %s", got, want)
		}
	})
}
