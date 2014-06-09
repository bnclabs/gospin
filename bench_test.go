package failsafe

import (
	"github.com/dustin/go-jsonpointer"
	"io/ioutil"
	"testing"
)

var mediumJSON, _ = ioutil.ReadFile("./testdata/medium.json")

func TestJsonpointer(t *testing.T) {
	jptrs, _ := jsonpointer.ListPointers(mediumJSON)
	for _, ptr := range jptrs {
		if _, err := jsonpointer.Find(mediumJSON, ptr); err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkJsonpointer(b *testing.B) {
	jptrs, _ := jsonpointer.ListPointers(mediumJSON)
	l := len(jptrs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonpointer.Find(mediumJSON, jptrs[i%l])
	}
}
