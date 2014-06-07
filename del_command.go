package failsafe

import (
    "github.com/goraft/raft"
)

type DeleteCommand struct {
    Path  string      `json:"path"`
    CAS   float64     `json:"CAS"`
}

func NewDeleteCommand(path string, cas float64) *DeleteCommand {
    return &DeleteCommand{path, cas}
}

// The name of the command in the log.
func (c *DeleteCommand) CommandName() string {
    return "delete"
}

// Writes a value to a key.
func (c *DeleteCommand) Apply(context raft.Context) (interface{}, error) {
    sd := context.Server().Context().(*SafeDict)
    return sd.Delete(c.Path, c.CAS)
}
