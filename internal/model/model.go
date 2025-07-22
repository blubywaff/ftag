package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Resource struct {
	Id        string
	Mimetype  string
	CreatedAt time.Time
	Tags      TagSet
}

type TagSet struct {
	Inner []string
}

func (ts *TagSet) String() string {
	var sb strings.Builder
	for i, t := range ts.Inner {
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(t)
	}
	return sb.String()
}

func (ts TagSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(ts.Inner)
}

func (ts *TagSet) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &ts.Inner)
}

func (ts *TagSet) Duplicate() *TagSet {
	du := TagSet{make([]string, len(ts.Inner))}
	copy(du.Inner, ts.Inner)
	return &du
}

func (ts *TagSet) Union(rhs TagSet) *TagSet {
	for _, t := range rhs.Inner {
		ts.add(t)
	}
	return ts
}

func (ts *TagSet) Difference(rhs TagSet) *TagSet {
	for _, t := range rhs.Inner {
		ts.remove(t)
	}
	return ts
}

func (ts *TagSet) Len() int {
	return len(ts.Inner)
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

// returns the index and whether the item was present
// if item is not present, then the index is where the item would be if it were present
func (ts *TagSet) index(str string) (int, bool) {
	l, r := 0, len(ts.Inner)
	for l != r {
		m := (l + r) / 2
		if ts.Inner[m] == str {
			return m, true
		}
		if str > ts.Inner[m] {
			l = m + 1
		} else {
			r = m
		}
	}
	return l, false
}

// adds a pre-checked tag string
// returns error iff tag is already present
func (ts *TagSet) add(str string) error {
	if len(ts.Inner) == 0 {
		ts.Inner = append(ts.Inner, str)
		return nil
	}
	l, p := ts.index(str)
	if p {
		return errors.New("tag already present in set")
	}
	if l == len(ts.Inner) {
		ts.Inner = append(ts.Inner, str)
		return nil
	}
	ts.Inner = append(ts.Inner[:l+1], ts.Inner[l:]...)
	ts.Inner[l] = str
	return nil
}

// return error iff the item is not present
func (ts *TagSet) remove(str string) error {
	if len(ts.Inner) == 0 {
		return errors.New("tagset is empty, nothing to remove")
	}
	if l, p := ts.index(str); p {
		ts.Inner = append(ts.Inner[:l], ts.Inner[l+1:]...)
	}
	return nil
}

func (ts *TagSet) FromSlice(sstr []string) error {
	// use insertion sort b/c a single set is not expected
	// to have more than a couple dozen tags
	for i := 1; i < len(sstr); i++ {
		for j := i - 1; j > 0; j-- {
			if sstr[j] < sstr[j+1] {
				sstr[j], sstr[j+1] = sstr[j+1], sstr[j]
			}
		}
	}
	ts.Inner = sstr
	return nil
}

type Query struct {
	Include TagSet
	Exclude TagSet
	Offset  int
	Limit   int
}
