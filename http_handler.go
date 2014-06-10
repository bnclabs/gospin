package failsafe

import (
    "encoding/json"
    "fmt"
    "github.com/goraft/raft"
    "io/ioutil"
    "log"
    "net/http"
)

func (s *Server) joinHandler(w http.ResponseWriter, req *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("%v, error: %v", s.logPrefix, r)
        }
    }()

    command := &raft.DefaultJoinCommand{}

    if err := json.NewDecoder(req.Body).Decode(&command); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if _, err := s.raftServer.Do(command); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (s *Server) leaveHandler(w http.ResponseWriter, req *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("%v, error: %v", s.logPrefix, r)
        }
    }()

    command := &raft.DefaultLeaveCommand{}

    if err := json.NewDecoder(req.Body).Decode(&command); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if _, err := s.raftServer.Do(command); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (s *Server) dbHandler(w http.ResponseWriter, req *http.Request) {
    var m map[string]interface{}

    defer func() {
        if r := recover(); r != nil {
            log.Printf("%v, error: %v", s.logPrefix, r)
        }
    }()

    tracef("%v, %v %q\n", s.logPrefix, req.Method, req.URL)
    switch req.Method {
    case "HEAD":
        w.Header().Set("ETag", fmt.Sprintf("%v", uint64(s.db.GetCAS())))
        x := s.GetLeader()
        w.Header().Set(HttpHdrNameLeader, x[0])
        w.Header().Set(HttpHdrNameLeaderAddr, x[1])

    case "GET":
        jsonreq, err := parseRequest(req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        } else {
            value, CAS, err := s.DBGet(jsonreq["path"].(string))
            w.Header().Set("ETag", fmt.Sprintf("%v", uint64(CAS)))
            m = map[string]interface{}{
                "value": value, "CAS": CAS, "err": errorString(err),
            }
        }

    case "PUT":
        jsonreq, err := parseRequest(req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        } else {
            path, value := jsonreq["path"].(string), jsonreq["value"]
            CAS := jsonreq["CAS"].(float64)
            nextCAS, err := s.DBSetCAS(path, value, CAS)
            m = map[string]interface{}{"CAS": nextCAS, "err": errorString(err)}
        }

    case "DELETE":
        jsonreq, err := parseRequest(req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        } else {
            path, CAS := jsonreq["path"].(string), jsonreq["CAS"].(float64)
            nextCAS, err := s.DBDeleteCAS(path, CAS)
            m = map[string]interface{}{"CAS": nextCAS, "err": errorString(err)}
        }

    default:
        log.Fatalf("%v, uknown method %q\n", s.logPrefix, req.Method)
    }

    if m != nil {
        if data, err := json.Marshal(&m); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        } else {
            w.Write(data)
        }
    }
}

func parseRequest(req *http.Request) (jsonreq map[string]interface{}, err error) {
    b, err := ioutil.ReadAll(req.Body)
    if err != nil {
        return nil, err
    }
    jsonreq = make(map[string]interface{})
    err = json.Unmarshal(b, &jsonreq)
    return jsonreq, err
}

func errorString(err error) string {
    if err == nil {
        return ""
    }
    return err.Error()
}
