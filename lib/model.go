package lib

import (
    "time"
)

// Verifies and return true if a tag is conformant.
// Iff the tag is noncomformant, doesTagConform will also return the index of the noncormant rune
// In the case that the tag is not long enough, it will return the length of the tag, this should be checked for because it is not a valid index.
func DoesTagConform(tag string) (bool, int) {
	if l := len(tag); l < 3 {
		return false, l
	}
	for i, c := range tag {
		if (c < 97 || c > 122) && c != 45 {
			return false, i
		}
	}
	return true, 0
}

type Resource struct {
    id string
    mimetype string
    createdAt time.Time
    tags []string
}
