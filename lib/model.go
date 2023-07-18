package lib

import (
	"errors"
	"strings"
	"time"
)

// Returns good tags and bad tags
// as a special case, empty tags are ignored
// also cleans up tags (lowercase and trim whitespace)
func SortTagList(str string) ([]string, []string) {
    lst := strings.Split(str, ",")
    bad := make([]string, 0)
    good := make([]string, 0)
    for i, t := range lst {
        lst[i] = strings.ToLower(strings.Trim(t, " "))
    }
    outer:
    for _, t := range lst {
        if len(t) == 0 { continue }
        for _, c := range t {
            if (c < 97 || c > 122) && c != 45 {
                bad = append(bad, t)
                continue outer
            }
        }
        good = append(good, t)
    }
    return good, bad
}

type Resource struct {
	Id        string
	Mimetype  string
	CreatedAt time.Time
	Tags      []string
}
