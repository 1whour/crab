package utils

import (
	"math/rand"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func SliceRandOne(s []string) string {

	sort.Strings(s)
	index := rand.Int31n(int32(len(s)))
	return s[index]
}
