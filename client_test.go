package failsafe

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

var _ = fmt.Sprintf("dummy") // TODO: remove this later.

func TestSafeDictClient(t *testing.T) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)

	client := NewSafeDictClient(servAddr)

	CAS, err := client.GetCAS()
	if err != nil {
		t.Fatal(err)
	} else if CAS != uint64(1) {
		t.Fatal("failed GetCAS() SafeDictClient", CAS)
	}
}

func TestClientGet(t *testing.T) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)

	client := NewSafeDictClient(servAddr)

	value, _, err := client.Get("")
	if err != nil {
		t.Fatal(err)
	}
	if len(value.(map[string]interface{})) != 0 {
		t.Fatal("Expected nil")
	}

	_, sd := populate(client, smallJSON, t)
	if value, _, err = client.Get(""); err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(value, sd.m) == false {
		t.Fatal("Set empty failed")
	}
}

func TestClientSet(t *testing.T) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)

	client := NewSafeDictClient(servAddr)
	CAS, _ := populate(client, smallJSON, t)

	CAS, err := client.SetCAS("/eyeColor", "weird", CAS)
	if err != nil {
		t.Fatal(err)
	}
	value, CAS, err := client.Get("/eyeColor")
	if err != nil {
		t.Fatal(err)
	} else if CAS != 4 {
		t.Fatal("failed expected CAS as 4", CAS)
	} else if value.(string) != "weird" {
		t.Fatal("failed", value.(string))
	}
}

func TestClientDelete(t *testing.T) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)

	client := NewSafeDictClient(servAddr)
	CAS, _ := populate(client, smallJSON, t)
	CAS, err := client.DeleteCAS("/eyeColor", CAS)
	if err != nil {
		t.Fatal(err)
	}
	value, CAS, err := client.Get("/eyeColor")
	if err.Error() != ErrorInvalidPath.Error() {
		t.Fatal("failed expected ErrorInvalidPath")
	}
	if CAS, err = client.GetCAS(); err != nil {
		t.Fatal(err)
	} else if value != nil {
		t.Fatal("failed expected value as nil")
	}
	if CAS != 6 {
		t.Fatal("failed expected CAS as nullCAS")
	}
}

func BenchmarkClientGetCAS(b *testing.B) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)
	client := NewSafeDictClient(servAddr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetCAS()
	}
}

func BenchmarkClientGet(b *testing.B) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)
	client := NewSafeDictClient(servAddr)
	populate(client, smallJSON, b)
	ks := addKeys(client, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Get(ks[i])
	}
}

func BenchmarkClientSet(b *testing.B) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)
	client := NewSafeDictClient(servAddr)
	startCAS, _ := populate(client, smallJSON, b)
	ks := addKeys(nil, b.N)
	CAS := startCAS

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CAS, _ = client.Set(ks[i], i)
	}
	if CAS != startCAS+uint64(b.N) {
		b.Fatal("mismatch", CAS, startCAS)
	}
}

func BenchmarkClientSetCAS(b *testing.B) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)
	client := NewSafeDictClient(servAddr)
	startCAS, _ := populate(client, smallJSON, b)
	ks := addKeys(nil, b.N)
	CAS := startCAS

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CAS, _ = client.SetCAS(ks[i], i, CAS)
	}
	if CAS != startCAS+uint64(b.N) {
		b.Fatal("mismatch", CAS, startCAS)
	}
}

func BenchmarkClientDelete(b *testing.B) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)
	client := NewSafeDictClient(servAddr)
	populate(client, smallJSON, b)
	ks := addKeys(client, b.N)
	startCAS, _ := client.GetCAS()
	CAS := startCAS

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CAS, _ = client.Delete(ks[i])
	}
	if CAS != startCAS+uint64(b.N) {
		b.Fatal("mismatch", CAS, startCAS)
	}
}

func BenchmarkClientDeleteCAS(b *testing.B) {
	startTestServer(testRaftdir)
	time.Sleep(10 * time.Millisecond)
	client := NewSafeDictClient(servAddr)
	populate(client, smallJSON, b)
	ks := addKeys(client, b.N)
	startCAS, _ := client.GetCAS()
	CAS := startCAS

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CAS, _ = client.DeleteCAS(ks[i], CAS)
	}
	if CAS != startCAS+uint64(b.N) {
		b.Fatal("mismatch", CAS, startCAS)
	}
}

func populate(client *SafeDictClient, data []byte, tb testing.TB) (uint64, *SafeDict) {
	sd, err := NewSafeDict(smallJSON, true)
	if err != nil {
		tb.Fatal(err)
	} else {
		if CAS, err := client.GetCAS(); err != nil {
			tb.Fatal(err)
		} else {
			if _, err := client.SetCAS("", sd.m, CAS); err != nil {
				tb.Fatal(err)
			}
		}
	}
	CAS, _ := client.GetCAS()
	return CAS, sd
}

func addKeys(client *SafeDictClient, N int) []string {
	ks := make([]string, 0, N)
	for i := 0; i < N; i++ {
		k := fmt.Sprintf("/key%v", i)
		ks = append(ks, k)
		if client != nil {
			client.Set(k, i)
		}
	}
	return ks
}
