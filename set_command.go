package failsafe

import (
    "github.com/goraft/raft"
)

type SetCommand struct {
    Path  string      `json:"path"`
    Value interface{} `json:"value"`
    CAS   float64     `json:"CAS"`
}

func NewSetCommand(path string, value interface{}, cas float64) *SetCommand {
    return &SetCommand{path, value, cas}
}

// The name of the command in the log.
func (c *SetCommand) CommandName() string {
    return "set"
}

// Writes a value to a key.
func (c *SetCommand) Apply(context raft.Context) (interface{}, error) {
    sd := context.Server().Context().(*SafeDict)
    return sd.Set(c.Path, c.Value, c.CAS)
}
