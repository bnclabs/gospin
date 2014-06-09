package failsafe

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
)

// error codes

// ErrorInvalidPath
var ErrorInvalidPath = fmt.Errorf("errorInvalidPath")

// ErrorInvalidType
var ErrorInvalidType = fmt.Errorf("errorInvalidType")

// ErrorInvalidCAS
var ErrorInvalidCAS = fmt.Errorf("errorInvalidCAS")

const nullCAS = float64(0)

// SafeDict is a failsafe data-structure similar to JSON property.
type SafeDict struct {
	mu  sync.Mutex             `json:"-"`
	m   map[string]interface{} `json:"m"`   // JSON decoded data-structure
	CAS float64                `json:"CAS"` // monotonically increasing CAS
}

// NewSafeDict returns a reference to new failsafe dictionary. Can be
// initialized to support CAS and/or initialized with initial JSON encoded
// string or JSON decode map[string]interface{} data.
func NewSafeDict(data interface{}, cas bool) (*SafeDict, error) {
	sd := &SafeDict{}
	switch arg := data.(type) {
	case map[string]interface{}:
		sd.m = arg

	case []byte:
		if arg != nil {
			sd.m = make(map[string]interface{})
			if err := json.Unmarshal(arg, &sd.m); err != nil {
				return nil, err
			}
		}
	}
	if cas {
		sd.CAS = float64(1)
	}
	return sd, nil
}

// MarshalJSON implements encoding/json.Marshaler interface.
func (sd *SafeDict) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		M   map[string]interface{} `json:"m"`
		CAS float64                `json:"CAS"`
	}{sd.m, sd.CAS})
}

// UnmarshalJSON implements encoding/json.Unmarshaler interface.
func (sd *SafeDict) UnmarshalJSON(data []byte) error {
	t := struct {
		M   map[string]interface{} `json:"m"`
		CAS float64                `json:"CAS"`
	}{}

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	sd.m = t.M
	sd.CAS = t.CAS
	return nil
}

// GetCAS returns the current CAS value.
func (sd *SafeDict) GetCAS() float64 {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.CAS
}

// Get field value located by `path` jsonpointer, full json-pointer spec is
// allowed.
func (sd *SafeDict) Get(path string) (rv interface{}, CAS float64, err error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.get(path)
}

func (sd *SafeDict) get(path string) (rv interface{}, CAS float64, err error) {
	var ok bool

	m := sd.m
	switch path {
	case "":
		return m, sd.CAS, nil

	case "/":
		if rv, ok = m[""]; ok {
			return rv, sd.CAS, nil
		}
		return nil, nullCAS, ErrorInvalidPath

	default:
		parts := parseJSONPointer(path)
		rv = m
		for _, p := range parts {
			switch v := rv.(type) {
			case map[string]interface{}:
				if rv, ok = v[p]; !ok {
					return nil, nullCAS, ErrorInvalidPath
				}

			case []interface{}:
				if idx, err := strconv.Atoi(p); err == nil && idx < len(v) {
					rv = v[idx]
				} else {
					return nil, nullCAS, ErrorInvalidPath
				}

			default:
				return nil, nullCAS, ErrorInvalidPath
			}
		}
		return rv, sd.CAS, nil
	}
}

// Set value at the specified path, full json-pointer spec. is allowed. If CAS
// is specied as nullCAS, CAS is ignored.
func (sd *SafeDict) Set(path string, value interface{}, CAS float64) (nextCAS float64, err error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if CAS == nullCAS {
		CAS = sd.CAS
	}
	return sd.set(path, value, CAS)
}

func (sd *SafeDict) set(path string, value interface{}, CAS float64) (nextCAS float64, err error) {
	m := sd.m
	if sd.checkCAS(CAS) == false {
		return nullCAS, ErrorInvalidCAS
	}

	switch path {
	case "":
		if m, ok := value.(map[string]interface{}); ok {
			sd.m = m
			return sd.incrementCAS(), nil
		}
		return nullCAS, ErrorInvalidType

	case "/":
		m[""] = value
		return sd.incrementCAS(), nil

	default:
		parts := parseJSONPointer(path)
		l := len(parts)
		hs, last := parts[:l-1], parts[l-1]

		container, _, err := sd.get(encodeJSONPointer(hs))
		if err != nil {
			return nullCAS, err
		}

		switch v := container.(type) {
		case map[string]interface{}:
			v[last] = value // idempotent mutation
			return sd.incrementCAS(), nil

		case []interface{}:
			if idx, err := strconv.Atoi(last); err == nil && idx < len(v) {
				v[idx] = value // idempotent mutation
				return sd.incrementCAS(), nil
			}
			return nullCAS, ErrorInvalidPath

		default:
			return nullCAS, ErrorInvalidPath
		}
	}
}

// Delete value at the specified path, last segment shall always index
// into json property. If CAS is specied as nullCAS, CAS is ignored.
func (sd *SafeDict) Delete(path string, CAS float64) (nextCAS float64, err error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if CAS == nullCAS {
		CAS = sd.CAS
	}
	return sd.delete(path, CAS)
}

func (sd *SafeDict) delete(path string, CAS float64) (nextCAS float64, err error) {
	m := sd.m
	if sd.checkCAS(CAS) == false {
		return nullCAS, ErrorInvalidCAS
	}

	switch path {
	case "":
		return nullCAS, ErrorInvalidPath

	case "/":
		delete(m, "")
		return sd.incrementCAS(), nil

	default:
		parts := parseJSONPointer(path)
		l := len(parts)
		hs, last := parts[:l-1], parts[l-1]

		container, _, err := sd.get(encodeJSONPointer(hs))
		if err != nil {
			return nullCAS, err
		}

		switch v := container.(type) {
		case map[string]interface{}:
			delete(v, last) // idempotent mutation
			return sd.incrementCAS(), nil

		default:
			return nullCAS, ErrorInvalidPath
		}
	}
}

// monotonically increasing CAS.
func (sd *SafeDict) incrementCAS() float64 {
	if sd.CAS != nullCAS {
		sd.CAS += 1.0
	}
	return sd.CAS
}

// compare local CAS with API supplied CAS.
func (sd *SafeDict) checkCAS(CAS float64) bool {
	if sd.CAS != nullCAS && sd.CAS == CAS {
		return true
	}
	return false
}

// save will marshal the dictionary and persist on disk.
func (sd *SafeDict) save(filename string) error {
	data, err := json.Marshal(sd)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

// restore will unmarshal the dictionary persisted on disk.
func (sd *SafeDict) restore(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &sd)
}
