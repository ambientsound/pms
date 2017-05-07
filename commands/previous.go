package commands

import (
	"fmt"

	"github.com/ambientsound/pms/input/lexer"
)

// Previous switches to the previous song in MPD's queue.
type Previous struct {
	api API
}

func NewPrevious(api API) Command {
	return &Previous{
		api: api,
	}
}

func (cmd *Previous) Execute(t lexer.Token) error {
	switch t.Class {
	case lexer.TokenEnd:
		client := cmd.api.MpdClient()
		if client == nil {
			return fmt.Errorf("Unable to play previous song: cannot communicate with MPD")
		}
		return client.Previous()

	default:
		return fmt.Errorf("Unknown input '%s', expected END", t.String())
	}

	return nil
}