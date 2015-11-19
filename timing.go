package timing

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"
)

var percentiles = []int{90, 95, 99}

func New() *Timing {
	t := &Timing{
		begin:     time.Now(),
		results:   make(chan record, 10),
		collected: make(map[string][]record),
	}
	go t.collector()
	return t
}

type Timing struct {
	begin     time.Time
	results   chan record
	done      int32
	collected map[string][]record
}

func (t *Timing) collector() {
	for result := range t.results {
		t.collected[result.key] = append(t.collected[result.key], result)
	}
	atomic.AddInt32(&t.done, 1)
}

func (t *Timing) Log(elapsed time.Duration, key string) {
	t.results <- record{
		key:     key,
		elapsed: elapsed,
	}
}

func (t *Timing) LogSince(begin time.Time, key string) {
	t.Log(time.Since(begin), key)
}

func (t *Timing) Report() Report {
	dest := make(Report)
	t.ReportInto(dest)
	return dest
}

func (t *Timing) ReportInto(dest Report) {
	dest["elapsed"] = time.Since(t.begin).String()
	if atomic.LoadInt32(&t.done) == 0 {
		close(t.results)
	}
	for k, records := range t.collected {
		sort.Sort(recordsByTime(records))

		// calculate mean
		var total time.Duration
		for _, record := range records {
			total += record.elapsed
		}
		dest[fmt.Sprintf("%s_samples", k)] = len(records)
		dest[fmt.Sprintf("%s_avg", k)] = (total / time.Duration(len(records))).String()

		// calculate all the percentiles
		for _, p := range percentiles {
			// TODO do we need a ceiling operator?
			target := len(records) * p / 100
			if target >= len(records) {
				target = len(records) - 1
			}
			dest[fmt.Sprintf("%s_%dth", k, p)] = records[target].elapsed.String()
		}
	}
}

type Report map[string]interface{}

type record struct {
	key     string
	elapsed time.Duration
}

type recordsByTime []record

func (r recordsByTime) Len() int           { return len(r) }
func (r recordsByTime) Less(i, j int) bool { return r[i].elapsed < r[j].elapsed }
func (r recordsByTime) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
