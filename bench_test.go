package failsafe

import (
    "github.com/dustin/go-jsonpointer"
    "testing"
    "io/ioutil"
)

var mediumJson, _ = ioutil.ReadFile("./testdata/medium.json")

func TestJsonpointer(t *testing.T) {
    jptrs, _ := jsonpointer.ListPointers(mediumJson)
    for _, ptr := range jptrs {
        if _, err := jsonpointer.Find(mediumJson, ptr); err != nil {
            t.Fatal(err)
        }
    }
}

func BenchmarkJsonpointer(b *testing.B) {
    jptrs, _ := jsonpointer.ListPointers(mediumJson)
    l := len(jptrs)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        jsonpointer.Find(mediumJson, jptrs[i%l])
    }
}
