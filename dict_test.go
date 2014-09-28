package failsafe

import (
	"github.com/prataprc/go-jsonpointer"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestSafeDict(t *testing.T) {
	// test with bytes
	data := `{"path": 10}`
	if _, err := NewSafeDict([]byte(data), true); err != nil {
		t.Errorf("SafeDict on []byte(%v)", data)
	}

	// test with map
	m := map[string]interface{}{"path": 10}
	if _, err := NewSafeDict(m, true); err != nil {
		t.Errorf("SafeDict on map[string]interface{}(%v)", m)
	}

	// test for wrong input
	data := `{path: 10}`
	if _, err := NewSafeDict([]byte(data), true); err == nil {
		t.Errorf("SafeDict expected to fail on []byte(%v)", m)
	}

	sd, _ := NewSafeDict([]byte(`{"path": 10}`), true)
	if val, _, err := sd.Get("/path"); err != nil {
		t.Errorf("On Get(`/path`) %v", err)
	} else if val.(float64) != float64(10) {
		t.Errorf("On Get(`/path`) return type %v", val)
	} else {
		if _, err := sd.Set("/path", float64(20), sd.GetCAS()); err != nil {
			t.Errorf("On Set(`/Path`, float64(20)) %v", err)
		}
		if _, err := sd.Set("/path", float64(20), sd.GetCAS()); err != nil {
			t.Fatal(err)
		}
		if val, _, err := sd.Get("/path"); err != nil {
			t.Fatal(err)
		} else if val.(float64) != float64(20) {
			t.Fatal("failed safedict")
		}
	}
}

func TestSaveRestore(t *testing.T) {
	sd, err := NewSafeDict(smallJSON, true)
	if err != nil {
		t.Fatal(err)
	}
	refValue := []interface{}{float64(1), float64(2)}
	if _, err := sd.Set("/balance", refValue, sd.GetCAS()); err != nil {
		t.Fatal(err)
	}
	m1 := sd.m

	data, err := sd.Save()
	if err != nil {
		t.Fatal(err)
	}
	ioutil.WriteFile(dummyFile, data, 0644)

	sd, err = NewSafeDict(nil, true)
	data, err = ioutil.ReadFile(dummyFile)
	if err != nil {
		t.Fatal(err)
	}
	err = sd.Recovery(data)
	if err != nil {
		t.Fatal(err)
	}

	m2 := sd.m
	if reflect.DeepEqual(m1, m2) == false {
		t.Fatal("failed save / recovery for SafeDict")
	}
	if value, CAS, err := sd.Get("/balance"); err != nil {
		t.Fatal(err)
	} else if CAS != float64(2) {
		t.Fatal("failed Set() SafeDict")
	} else if reflect.DeepEqual(refValue, value) == false {
		t.Fatal("failed save / recovery for SafeDict")
	}

	os.Remove(dummyFile)
}

func TestSafeDictCAS(t *testing.T) {
	if _, err := NewSafeDict([]byte(`{path: 10}`), true); err == nil {
		t.Fatal("SafeDict expected error for wrong json doc")
	}
	if sd, err := NewSafeDict([]byte(`{"path": 10}`), true); err != nil {
		t.Fatal(err)
	} else {
		if val, cas, err := sd.Get("/path"); err != nil {
			t.Fatal(err)
		} else if val.(float64) != float64(10) {
			t.Fatal("failed safedict")
		} else {
			if _, err := sd.Set("/path", float64(20), cas+1); err == nil {
				t.Fatal("expected error, for larger cas")
			}
			if _, err := sd.Set("/path", float64(20), cas); err != nil {
				t.Fatal(err)
			}
			if _, err := sd.Set("/path", float64(20), cas); err == nil {
				t.Fatal("expected error, for smaller cas")
			}
			if val, _, err := sd.Get("/path"); err != nil {
				t.Fatal(err)
			} else if val.(float64) != float64(20) {
				t.Fatal("failed safedict")
			}
		}
	}
}

func TestGetSetSafeDict(t *testing.T) {
	var err error

	jptrs, _ := jsonpointer.ListPointers(smallJSON)
	sd, err := NewSafeDict(smallJSON, true)
	if err != nil {
		t.Fatal(err)
	}
	if sd.GetCAS() != 1 {
		t.Fatal("failed cas is not initialized to 1")
	}

	// testing getting and setting
	for _, ptr := range jptrs[1:] {
		val, cas, err := sd.Get(ptr)
		if err != nil {
			t.Fatal(err)
		}
		switch val.(type) {
		case []interface{}, map[string]interface{}:
			continue
		default:
			nextCas, err := sd.Set(ptr, float64(20), cas)
			if err != nil {
				t.Fatal(err)
			}
			if val, cas, err := sd.Get(ptr); err != nil {
				t.Fatal(err)
			} else if val.(float64) != float64(20) {
				t.Fatalf("failed getting %q:%v\n", ptr, val)
			} else if cas != nextCas {
				t.Fatal("mismatch in CAS")
			}
		}
	}

	// test getting root
	val, cas, err := sd.Get("")
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(val, sd.m) == false {
		t.Fatal("mismatch in safedict")
	}

	// test setting empty property.
	m1, err := NewSafeDict([]byte(`{"key": "value"}`), true)
	if err != nil {
		t.Fatal(err)
	}
	cas, err = sd.Set("/", m1, cas)
	if err != nil {
		t.Fatal(err)
	}
	if val, cas, err = sd.Get("/"); err != nil {
		t.Fatal(err)
	} else if val == nil {
		t.Fatal("mismatch in safedict")
	} else if reflect.DeepEqual(val, m1) == false {
		t.Fatal("failed Setting `/` from SafeDict")
	}
}

func TestDeleteSafeDict(t *testing.T) {
	jptrs, _ := jsonpointer.ListPointers(smallJSON)
	sd, err := NewSafeDict(smallJSON, true)
	if err != nil {
		t.Fatal(err)
	}
	cas := sd.GetCAS()
	for _, ptr := range jptrs[1:] {
		if ncas, err := sd.Delete(ptr, cas); err == nil {
			cas = ncas
		}
	}
	sd1, err := NewSafeDict([]byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	sd1.CAS = float64(22)
	if reflect.DeepEqual(sd, sd1) == false {
		t.Fatal("failed delete safedict")
	}
}

func BenchmarkGetSafeDict1(b *testing.B) {
	sd, _ := NewSafeDict(smallJSON, true)
	for i := 0; i < b.N; i++ {
		sd.Get("/friends")
	}
}

func BenchmarkGetSafeDict3(b *testing.B) {
	sd, _ := NewSafeDict(smallJSON, true)
	path := "/friends/2/name"
	for i := 0; i < b.N; i++ {
		sd.Get(path)
	}
}

func BenchmarkSetSafeDict1(b *testing.B) {
	sd, _ := NewSafeDict(smallJSON, true)
	cas := sd.GetCAS()
	path := "/friends"
	for i := 0; i < b.N; i++ {
		cas, _ = sd.Set(path, 20, cas)
	}
}

func BenchmarkSetSafeDict3(b *testing.B) {
	sd, _ := NewSafeDict(smallJSON, true)
	cas := sd.GetCAS()
	path := "/friends/2/name"
	for i := 0; i < b.N; i++ {
		cas, _ = sd.Set(path, 20, cas)
	}
}

func BenchmarkDelSafeDict1(b *testing.B) {
	sd, _ := NewSafeDict(smallJSON, true)
	cas := sd.GetCAS()
	path := "/friends"
	for i := 0; i < b.N; i++ {
		cas, _ = sd.Set(path, 20, cas)
		cas, _ = sd.Delete(path, cas)
	}
}

func BenchmarkDelSafeDict3(b *testing.B) {
	sd, _ := NewSafeDict(smallJSON, true)
	cas := sd.GetCAS()
	path := "/friends/2/name"
	for i := 0; i < b.N; i++ {
		cas, _ = sd.Set(path, 20, cas)
		cas, _ = sd.Delete(path, cas)
	}
}
