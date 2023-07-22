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
		if len(t) == 0 {
			continue
		}
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

type TagSet struct {
	inner []string
}

func (ts *TagSet) Len() int {
	return len(ts.inner)
}

// Returns tags that do not conform
// ignores empty tags and duplicate tags
func (ts *TagSet) FillFromString(str string) []string {
	lst := strings.Split(str, ",")
	bad := make([]string, 0)
	for i, t := range lst {
		lst[i] = strings.ToLower(strings.Trim(t, " "))
	}
outer:
	for _, t := range lst {
		if len(t) == 0 {
			continue
		}
		if len(t) < 3 {
			bad = append(bad, t)
			continue
		}
		for _, c := range t {
			if (c < 97 || c > 122) && c != 45 {
				bad = append(bad, t)
				continue outer
			}
		}
		ts.add(t)
	}
	return bad
}

func (ts *TagSet) Add(str string) error {
	tag := strings.ToLower(strings.Trim(str, " "))
	if l := len(tag); l < 3 {
		return errors.New("tag is too short")
	}
	for _, c := range tag {
		if (c < 97 || c > 122) && c != 45 {
			return errors.New("tag has invalid character")
		}
	}
	ts.add(tag)
	return nil
}

// adds a pre-checked tag string
func (ts *TagSet) add(str string) error {
	if len(ts.inner) == 0 {
		ts.inner = append(ts.inner, str)
		return nil
	}
	l, r := 0, len(ts.inner)
	for l != r {
		m := (l + r) / 2
		if ts.inner[m] == str {
			return errors.New("tag already present in set")
		}
		if str > ts.inner[m] {
			l = m + 1
		} else {
			r = m
		}
	}
	if l == len(ts.inner) {
		ts.inner = append(ts.inner, str)
		return nil
	}
	ts.inner = append(ts.inner[:l+1], ts.inner[l:]...)
	ts.inner[l] = str
	return nil
}
