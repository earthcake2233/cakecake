package model

import "fmt"

// FormatCakeID returns the immutable public account id (cake_ + zero-padded numeric id).
func FormatCakeID(id uint64) string {
	return fmt.Sprintf("cake_%011d", id)
}
