package auth

import (
	"strings"
	"sync"
)

type waiter struct {
	List map[int64]struct{}
	Mu   sync.Mutex
	// ?
}

func IsValidSnils(s string) bool {
	if len(s) == 14 {
		for i := range s {
			switch i {
			case 3, 7:
				if s[i] != '-' {
					return false
				}
			case 11:
				if s[i] != ' ' {
					return false
				}
			default:
				if !strings.Contains("0123456789", string(s[i])) {
					return false
				}
			}
		}
		return true
	}
	return false
}

func NewWaiter() *waiter {
	m := make(map[int64]struct{})
	return &waiter{List: m}
}
