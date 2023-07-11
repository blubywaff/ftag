package lib

import (
	"strings"
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

// Returns good tags and bad tags
// as a special case, empty tags are ignored
func SortTagList(tagstr string) ([]string, []string) {
	var tags []string
	var badtags []string
	_tags := strings.Split(tagstr, ",")
	tags = make([]string, 0)
	badtags = make([]string, 0)
	for _, t := range _tags {
		if len(t) == 0 {
			continue
		}
		if c, _ := DoesTagConform(t); c {
			tags = append(tags, t)
			continue
		}
		badtags = append(badtags, t)
	}
	return tags, badtags
}

type Resource struct {
	Id        string
	Mimetype  string
	CreatedAt time.Time
	Tags      []string
}
