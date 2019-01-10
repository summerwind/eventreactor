package v1alpha1

import (
	"fmt"
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

// NewName returns a resource name based on ULID.
func NewName() string {
	t := ulid.MaxTime() - ulid.Now()

	id, err := ulid.New(t, entropy)
	if err != nil {
		panic(fmt.Sprintf("Unable to generate ULID: %s", err))
	}

	return strings.ToLower(id.String())
}
