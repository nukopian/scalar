package scalar

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
)

// The JSON benchmarks read and write a feed in the shape of the USGS's
// GeoJSON summary, generated from a fixed seed: 2,000 events, each with
// the feed's properties and coordinates. In the big variant, each event
// also has a seq field above 2⁶³, as some APIs' unsigned ids are, which
// int64 can't hold.

func usgsFeed(big bool) []byte {
	r := rand.New(rand.NewPCG(3, 4))
	nets := []string{"ak", "ci", "hv", "nc", "nn", "pr", "se", "us", "uw"}
	var b strings.Builder
	b.WriteString(`{"type":"FeatureCollection","metadata":{"generated":1729900000000,"count":2000,"status":200},"features":[`)
	for i := range 2000 {
		if i > 0 {
			b.WriteByte(',')
		}
		net := nets[r.IntN(len(nets))]
		code := fmt.Sprintf("%08d", r.IntN(1e8))
		t := 1729800000000 + r.Int64N(1e8)
		fmt.Fprintf(&b, `{"type":"Feature","properties":{"mag":%.2f,"place":"%d km %s of Station %s","time":%d,"updated":%d,"felt":%d,"cdi":%.1f,"tsunami":0,"sig":%d,"net":%q,"code":%q,"ids":",%s%s,","nst":%d,"dmin":%.4f,"rms":%.2f,"gap":%d,"magType":"ml","type":"earthquake"`,
			r.Float64()*6, r.IntN(100), []string{"N", "NNE", "E", "SW", "W"}[r.IntN(5)], station(r.IntN(553)),
			t, t+r.Int64N(1e6), r.IntN(500), r.Float64()*9, r.IntN(1000), net, code, net, code,
			r.IntN(100), r.Float64(), r.Float64(), r.IntN(360))
		if big {
			fmt.Fprintf(&b, `,"seq":%d`, uint64(1)<<63+r.Uint64N(1<<63))
		}
		fmt.Fprintf(&b, `},"geometry":{"type":"Point","coordinates":[%.4f,%.4f,%.2f]},"id":"%s%s"}`,
			r.Float64()*360-180, r.Float64()*180-90, r.Float64()*700, net, code)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

var (
	feedPlain = sync.OnceValue(func() []byte { return usgsFeed(false) })
	feedBig   = sync.OnceValue(func() []byte { return usgsFeed(true) })
)

func benchRead(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		if _, err := FromJSON(data); err != nil {
			b.Fatal(err)
		}
	}
}

func benchWrite(b *testing.B, data []byte) {
	s, err := FromJSON(data)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		if _, err := json.Marshal(s); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadJSON(b *testing.B)     { benchRead(b, feedPlain()) }
func BenchmarkReadJSONBig(b *testing.B)  { benchRead(b, feedBig()) }
func BenchmarkWriteJSON(b *testing.B)    { benchWrite(b, feedPlain()) }
func BenchmarkWriteJSONBig(b *testing.B) { benchWrite(b, feedBig()) }

// BenchmarkSumMag reads every event's magnitude as a number, as a program
// summing them would: the arithmetic path a new numeric type could slow.
func BenchmarkSumMag(b *testing.B) {
	s, err := FromJSON(feedBig())
	if err != nil {
		b.Fatal(err)
	}
	feats := s.Get("features").AV()
	for b.Loop() {
		var t float64
		for _, f := range feats.All() {
			p := f.Get("properties")
			t += p.Get("mag").Num() + p.Get("seq").Num()
		}
		_ = t
	}
}
