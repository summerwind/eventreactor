package v1alpha1

import (
	"math/rand"
	"strings"
	"time"

	"github.com/oklog/ulid"
)

var (
	entropy *rand.Rand
)

func init() {
	t := time.Now()
	entropy = rand.New(rand.NewSource(t.UnixNano()))
}

// NewEventName returns a event name based on ULID.
func NewEventName() (string, error) {
	t := ulid.MaxTime() - ulid.Now()

	id, err := ulid.New(t, entropy)
	if err != nil {
		return "", err
	}

	return strings.ToLower(id.String()), nil
}
