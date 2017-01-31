package history

import (
	"fmt"
	"sync"
	"time"
)

type History struct {
	length time.Duration             // constraint on newest time - oldest time
	t      map[time.Time]HistoryItem // time -> item
	times  []time.Time               // sorted slice of keys in map
	mux    sync.Mutex                // for thread-safeness
}

func MakeHistory(d time.Duration) *History {
	return &History{
		length: d,
		t:      make(map[time.Time]HistoryItem),
		times:  make([]time.Time, 0),
	}
}

func (l *History) UpdateDuration(d time.Duration) {
	l.mux.Lock()
	defer l.mux.Unlock()

	l.length = d
}

func (l *History) Len() int {
	l.mux.Lock()
	defer l.mux.Unlock()

	return len(l.times)
}

func (l *History) Add(t time.Time, it HistoryItem) {
	l.mux.Lock()
	defer l.mux.Unlock()

	l.times = append(l.times, t)
	l.t[t] = it

	if len(l.times) != len(l.t) {
		err := fmt.Errorf("History in inconsistent state: %v %v", len(l.times), len(l.t))
		panic(err)
	}

	lastTime := l.times[len(l.times)-1]
	// remove older, keep at least 100
	for len(l.times) > 100 && lastTime.Sub(l.times[0]) > l.length {
		rem := l.times[0]
		delete(l.t, rem)
		l.times = l.times[1:]
	}
}

// last item before given time and time it was logged
func (l *History) Before(wanted time.Time) (HistoryItem, time.Time, error) {
	l.mux.Lock()
	defer l.mux.Unlock()

	var then time.Time

	if len(l.times) == 0 {
		return nil, time.Now(), fmt.Errorf("empty log")
	}

	if wanted.Before(l.times[0]) {
		return l.t[l.times[0]], l.times[0], fmt.Errorf("wanted time before log start")
	}

	for _, t := range l.times {
		if t.After(wanted) {
			return l.t[then], then, nil
		} else {
			then = t
		}
	}

	lastTime := l.times[len(l.times)-1]
	return l.t[lastTime], lastTime, nil
}

func (l *History) AvgBetween(
	from time.Time,
	to time.Time,
	zero HistoryItem,
	sum func(a HistoryItem, b HistoryItem) HistoryItem,
	div func(a HistoryItem, n int) HistoryItem,
) (HistoryItem, error) {
	l.mux.Lock()
	defer l.mux.Unlock()

	cum := zero
	count := 0
	// TODO binary search
	for _, t := range l.times {
		if t.After(from) {
			if t.Before(to) {
				cum = sum(cum, l.t[t])
				count++
			} else {
				break
			}
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("timed log: no values to avg")
	}

	return div(cum, count), nil
}

func (l *History) NumItemsBetween(start time.Time, end time.Time) (int, error) {
	l.mux.Lock()
	defer l.mux.Unlock()

	count := 0
	for _, t := range l.times {
		if !t.Before(end) {
			return count, nil
		} else if !t.Before(start) {
			count++
		}
	}

	return count, nil
}
