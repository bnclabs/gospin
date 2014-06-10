// HTTP client API to access fail-safe dictionary - GetCAS(), Get(), Set(),
// Delete().
//
// Example client {
//      client := NewSafeDictClient(servAddr)
//      CAS, err := client.GetCAS()
//      CAS, err := client.SetCAS("/users/[0]/eyeColor", "brown", CAS)
//      if err != nil {
//          t.Fatal(err)
//      }
//      value, CAS, err := client.Get("/users/[0]/eyeColor")
//      CAS, err := client.DeleteCAS("/users/[0]/eyeColor", CAS)
// }
//
// Above example will set the first user's eyeColor as brown and subsequently
// delete the `eyeColor` field from user's property.
//
// Get() and Set() allows full jsonpointer spec. to access SafeDict, while
// Delete() allows does not allow the final element to be member of an array.
//
// Variants of Set and Delete calls
//
//                      SET         DELETE
//  sync                 *            *
//  sync with CAS        *            *
//
// TODO: support full jsonpointer spec. for Delete().
// TODO: figure out asynchronous operation for SET and DELETE.

package failsafe

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "strconv"
)

// SafeDictClient instance
type SafeDictClient struct {
    serverAddr string
    httpc      *http.Client
    reqJSON    map[string]interface{} // reusable
    respJSON   map[string]interface{} // reusable
}

// NewSafeDictClient return reference to a new instance of SafeDictClient
func NewSafeDictClient(serverAddr string) *SafeDictClient {
    return &SafeDictClient{
        serverAddr: serverAddr,
        httpc:      http.DefaultClient,
        reqJSON:    make(map[string]interface{}),
        respJSON:   make(map[string]interface{}),
    }
}

// GetLeader for this cluster
func (c *SafeDictClient) GetLeader() (leader string, leaderAddr string, err error) {
    htresp, err := c.doHTTP(nil, nil, "HEAD")
    if err != nil {
        return
    }
    leader     = htresp.Header.Get(HttpHdrNameLeader)
    leaderAddr = htresp.Header.Get(HttpHdrNameLeaderAddr)
    return
}

// GetCAS from fail-safe dictionary.
func (c *SafeDictClient) GetCAS() (CAS uint64, err error) {
    htresp, err := c.doHTTP(nil, nil, "HEAD")
    if err != nil {
        return uint64(nullCAS), err
    }
    cas, err := strconv.ParseFloat(htresp.Header.Get("ETag"), 64)
    if err != nil {
        return uint64(nullCAS), err
    }
    return uint64(cas), nil
}

// Get value of the field located by `path` jsonpointer.
func (c *SafeDictClient) Get(path string) (value interface{}, CAS uint64, err error) {
    defer func() { c.clean() }()

    c.reqJSON["path"] = path
    if _, err := c.doHTTP(c.reqJSON, c.respJSON, "GET"); err != nil {
        return nil, uint64(nullCAS), err
    } else if errstr := c.respJSON["err"].(string); errstr != "" {
        return nil, uint64(nullCAS), fmt.Errorf(errstr)
    }
    return c.respJSON["value"], uint64(c.respJSON["CAS"].(float64)), nil
}

// Set value of the field located by `path` jsonpointer.
func (c *SafeDictClient) Set(path string, value interface{}) (nextCAS uint64, err error) {
    defer func() { c.clean() }()

    c.reqJSON["path"], c.reqJSON["value"] = path, value
    c.reqJSON["CAS"] = nullCAS
    if _, err := c.doHTTP(c.reqJSON, c.respJSON, "PUT"); err != nil {
        return uint64(nullCAS), err
    } else if errstr := c.respJSON["err"].(string); errstr != "" {
        return uint64(nullCAS), fmt.Errorf(errstr)
    }
    return uint64(c.respJSON["CAS"].(float64)), nil
}

// SetCAS value of the field located by `path` jsonpointer, for matching CAS.
func (c *SafeDictClient) SetCAS(path string, value interface{}, CAS uint64) (nextCAS uint64, err error) {
    defer func() { c.clean() }()

    c.reqJSON["path"], c.reqJSON["value"], c.reqJSON["CAS"] = path, value, CAS
    if _, err := c.doHTTP(c.reqJSON, c.respJSON, "PUT"); err != nil {
        return uint64(nullCAS), err
    } else if errstr := c.respJSON["err"].(string); errstr != "" {
        return uint64(nullCAS), fmt.Errorf(errstr)
    }
    return uint64(c.respJSON["CAS"].(float64)), nil
}

// Delete field located by `path` jsonpointer.
func (c *SafeDictClient) Delete(path string) (nextCAS uint64, err error) {
    defer func() { c.clean() }()

    c.reqJSON["path"], c.reqJSON["CAS"] = path, nullCAS
    if _, err := c.doHTTP(c.reqJSON, c.respJSON, "DELETE"); err != nil {
        return uint64(nullCAS), err
    } else if errstr := c.respJSON["err"].(string); errstr != "" {
        return uint64(nullCAS), fmt.Errorf(errstr)
    }
    return uint64(c.respJSON["CAS"].(float64)), nil
}

// DeleteCAS field located by `path` jsonpointer with matching CAS.
func (c *SafeDictClient) DeleteCAS(path string, CAS uint64) (nextCAS uint64, err error) {
    defer func() { c.clean() }()

    c.reqJSON["path"], c.reqJSON["CAS"] = path, CAS
    if _, err := c.doHTTP(c.reqJSON, c.respJSON, "DELETE"); err != nil {
        return uint64(nullCAS), err
    } else if errstr := c.respJSON["err"].(string); errstr != "" {
        return uint64(nullCAS), fmt.Errorf(errstr)
    }
    return uint64(c.respJSON["CAS"].(float64)), nil
}

// doHTTP post a request to server and get back a response for client APIs.
func (c *SafeDictClient) doHTTP(
    reqJSON, respJSON map[string]interface{},
    method string) (resp *http.Response, err error) {

    // marshal json
    body := []byte{}
    if reqJSON != nil {
        body, err = json.Marshal(&reqJSON)
        if err != nil {
            return nil, err
        }
    }
    // make request
    bodybuf := bytes.NewBuffer(body)
    url := c.serverAddr + "/dict"
    req, err := http.NewRequest(method, url, bodybuf)
    if err != nil {
        return nil, err
    }
    req.Header.Add("Content-Type", "application/json")
    // access server
    htresp, err := c.httpc.Do(req)
    if err != nil {
        return nil, err
    }
    // process response
    defer htresp.Body.Close()
    body, err = ioutil.ReadAll(htresp.Body)
    if err != nil {
        return nil, err
    }
    // unmarshal response
    if respJSON != nil {
        if err := json.Unmarshal(body, &respJSON); err != nil {
            return nil, err
        }
    }
    return htresp, nil
}

// clean and reuse the structure for next request/response.
func (c *SafeDictClient) clean() {
    // clean request
    delete(c.reqJSON, "path")
    delete(c.reqJSON, "value")
    delete(c.reqJSON, "CAS")
    // clean response
    delete(c.respJSON, "value")
    delete(c.respJSON, "CAS")
    delete(c.respJSON, "err")
}
