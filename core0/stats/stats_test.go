package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"math/rand"
)

func TestSampleCalculations(t *testing.T) {
	var s Sample

	var t1, t2 int64 = 0, 50
	var duration int64 = 50

	if !assert.Nil(t, s.Feed(10, t1, duration)) {
		t.Fatal()
	}

	if !assert.Nil(t, s.Feed(10, t1+1, duration)) {
		t.Fatal()
	}

	if !assert.Nil(t, s.Feed(15, t1+2, duration)) {
		t.Fatal()
	}

	if !assert.Nil(t, s.Feed(30, t1+3, duration)) {
		t.Fatal()
	}

	feed := s.Feed(100, t2, duration)
	if !assert.NotNil(t, feed) {
		t.Fail()
	}

	if !assert.Equal(t, t1, feed.Start) {
		t.Fail()
	}

	if !assert.Equal(t, float64(10+10+15+30)/4., feed.Avg) {
		t.Fail()
	}

	if !assert.Equal(t, 30., feed.Max) {
		t.Fail()
	}

	if !assert.Equal(t, float64(10+10+15+30), feed.Total) {
		t.Fail()
	}
}

func assertUpdates(t *testing.T, period int64, updates Samples, total float64, count int) {
	sample, ok := updates[period]
	if !ok {
		t.Fatal()
	}

	if !assert.Equal(t, float64(total)/float64(count), sample.Avg) {
		t.Fail()
	}

	if !assert.Equal(t, total, sample.Total) {
		t.Fail()
	}
}

type TestTotal struct {
	Total float64
	Count int
}

func TestStateAvg(t *testing.T) {
	var p int64 = 50
	state := NewState(Average, p, p*2) //50 100
	var total float64
	var count int

	for i := int64(0); i <= p*2; i += 5 {
		n := rand.Float64() * 10
		updates := state.FeedOn(int64(i), n)
		if i != 0 && i%p == 0 {
			if !assert.NotNil(t, updates) {
				t.Fatal()
			}

			if !assert.Len(t, updates, int(i/p)) {
				t.Fatal()
			}

			assertUpdates(t, i, updates, total, count)
		} else if !assert.Len(t, updates, 0) {
			t.Fatal()
		}

		total += n
		count += 1
	}
}

func TestStateDiff(t *testing.T) {
	var p int64 = 50
	state := NewState(Differential, p)

	state.FeedOn(0, 1)
	state.FeedOn(10, 2)
	state.FeedOn(20, 3)
	state.FeedOn(30, 4)
	state.FeedOn(40, 5)

	//flush (at time 50, and start using a step of 2
	updates := state.FeedOn(50, 7)
	if !assert.NotNil(t, updates) {
		t.Fatal()
	}

	sample, ok := updates[p]
	if !ok {
		t.Fatal()
	}

	if !assert.Equal(t, 0.1, sample.Avg) {
		t.Fatal()
	}

	state.FeedOn(60, 9)
	state.FeedOn(70, 11)
	state.FeedOn(80, 13)
	state.FeedOn(90, 15)

	updates = state.FeedOn(100, 11)

	if !assert.NotNil(t, updates) {
		t.Fatal()
	}

	sample, ok = updates[p]
	if !ok {
		t.Fatal()
	}

	if !assert.Equal(t, 0.2, sample.Avg) {
		t.Fatal()
	}
}
