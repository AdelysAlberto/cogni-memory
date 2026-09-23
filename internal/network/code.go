package network

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// Curated pool of alphanumeric variations around the word "cogni"
var cogniVariations = []string{
	"c0gn1",
	"c0gni",
	"cogn1",
	"k0gni",
	"k0gn1",
	"c09n1",
	"cg2ni",
	"c09ni",
	"c9gni",
	"k09n1",
	"c0g2i",
	"c1gni",
	"c0gn2",
	"c09n2",
	"k0g2i",
	"c0gnx",
}

var codePattern = regexp.MustCompile(`^[0-9]{3}-[a-z0-9]{4,6}-[0-9]{2}$`)

// GeneratePairCode generates an ephemeral 3-slot pairing code:
// Slot 1: 3 numbers (100-999)
// Slot 2: Alphanumeric variation of "cogni" (e.g., c0gn1, k0gni)
// Slot 3: 2 numbers (10-99)
// Example output: "381-c0gn1-42"
func GeneratePairCode() string {
	// 1. Slot 1: 3 digits (100 to 999)
	n1, err := rand.Int(rand.Reader, big.NewInt(900))
	slot1 := 100
	if err == nil {
		slot1 = int(n1.Int64()) + 100
	}

	// 2. Slot 2: Cogni variation
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(cogniVariations))))
	slot2 := "c0gn1"
	if err == nil {
		slot2 = cogniVariations[int(idx.Int64())]
	}

	// 3. Slot 3: 2 digits (10 to 99)
	n3, err := rand.Int(rand.Reader, big.NewInt(90))
	slot3 := 10
	if err == nil {
		slot3 = int(n3.Int64()) + 10
	}

	return fmt.Sprintf("%03d-%s-%02d", slot1, slot2, slot3)
}

// ValidatePairCode validates that a code adheres to the 3-slot Cogni Network standard.
func ValidatePairCode(code string) bool {
	clean := strings.ToLower(strings.TrimSpace(code))
	return codePattern.MatchString(clean)
}
