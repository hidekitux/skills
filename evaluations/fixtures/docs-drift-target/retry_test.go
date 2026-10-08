package retry

import (
	"errors"
	"testing"
)

func TestDoStopsAfterSuccess(t *testing.T) {
	calls := 0
	err := Do(3, 0, func() error {
		calls++
		if calls < 2 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("err = %v, calls = %d", err, calls)
	}
}
