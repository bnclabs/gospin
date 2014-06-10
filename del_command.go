package failsafe

import (
    "github.com/goraft/raft"
)

// DeleteCommand to delete a field from SafeDict.
type DeleteCommand struct {
    Path string  `json:"path"`
    CAS  float64 `json:"CAS"`
}

// NewDeleteCommand creates a new instance of DeleteCommand.
// TODO: figure out a way to resue the command, to reduce GC overhead.
func NewDeleteCommand(path string, cas float64) *DeleteCommand {
    return &DeleteCommand{path, cas}
}

// CommandName implements raft.Command interface.
func (c *DeleteCommand) CommandName() string {
    return "delete"
}

// Apply implements raft.CommandApply interface.
func (c *DeleteCommand) Apply(context raft.Context) (interface{}, error) {
    s := context.Server().Context().(*Server)
    nextCAS, err := s.db.Delete(c.Path, c.CAS)
    return nextCAS, err
}
