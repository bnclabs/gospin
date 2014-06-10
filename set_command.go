package failsafe

import (
    "github.com/goraft/raft"
)

// SetCommand to set value to a field in SafeDict.
type SetCommand struct {
    Path  string      `json:"path"`
    Value interface{} `json:"value"`
    CAS   float64     `json:"CAS"`
}

// NewSetCommand creates a new instance of SetCommand.
// TODO: figure out a way to resue the command, to reduce GC overhead.
func NewSetCommand(path string, value interface{}, cas float64) *SetCommand {
    return &SetCommand{path, value, cas}
}

// CommandName implements raft.Command interface.
func (c *SetCommand) CommandName() string {
    return "set"
}

// Apply implements raft.CommandApply interface.
func (c *SetCommand) Apply(context raft.Context) (interface{}, error) {
    s := context.Server().Context().(*Server)
    nextCAS, err := s.db.Set(c.Path, c.Value, c.CAS)
    return nextCAS, err
}
