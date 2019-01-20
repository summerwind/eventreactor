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

// NewID returns a ULID.
func NewID() string {
	id, err := ulid.New(ulid.Now(), entropy)
	if err != nil {
		panic(fmt.Sprintf("Unable to generate ULID: %s", err))
	}

	return strings.ToLower(id.String())
}
