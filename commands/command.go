// Package commands contains all functionality that is triggered by the user,
// either through keyboard bindings or the command-line interface. New commands
// such as 'sort', 'add', etc. must be implemented here.
package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ambientsound/pms/api"
	"github.com/ambientsound/pms/input/lexer"
	"github.com/ambientsound/pms/parser"
	"github.com/ambientsound/pms/song"
	"github.com/ambientsound/pms/utils"
)

// Verbs contain mappings from strings to Command constructors.
// Make sure to add commands here when implementing them.
var Verbs = map[string]func(api.API) Command{
	"add":       NewAdd,
	"bind":      NewBind,
	"copy":      NewYank,
	"cursor":    NewCursor,
	"cut":       NewCut,
	"inputmode": NewInputMode,
	"isolate":   NewIsolate,
	"list":      NewList,
	"next":      NewNext,
	"paste":     NewPaste,
	"pause":     NewPause,
	"play":      NewPlay,
	"previous":  NewPrevious,
	"prev":      NewPrevious,
	"print":     NewPrint,
	"q":         NewQuit,
	"quit":      NewQuit,
	"redraw":    NewRedraw,
	"seek":      NewSeek,
	"select":    NewSelect,
	"se":        NewSet,
	"set":       NewSet,
	"single":    NewSingle,
	"sort":      NewSort,
	"stop":      NewStop,
	"style":     NewStyle,
	"unbind":    NewUnbind,
	"update":    NewUpdate,
	"viewport":  NewViewport,
	"volume":    NewVolume,
	"yank":      NewYank,
}

// Command must be implemented by all commands.
type Command interface {
	// Execute parses the next input token.
	// FIXME: Execute is deprecated
	Execute(class int, s string) error

	// Exec executes the AST generated by the command.
	Exec() error

	// SetScanner assigns a scanner to the command.
	// FIXME: move to constructor?
	SetScanner(*lexer.Scanner)

	// Parse and make an abstract syntax tree. This function MUST NOT have any side effects.
	Parse() error

	// TabComplete returns a set of tokens that could possibly be used as the next
	// command parameter.
	TabComplete() []string

	// Scanned returns a slice of tokens that have been scanned using Parse().
	Scanned() []parser.Token
}

// command is a helper base class that all commands may use.
type command struct {
	cmdline string
}

// newcommand is an abolition which implements workarounds so that not
// everything in commands/ has to be refactored right away.
// FIXME
type newcommand struct {
	parser.Parser
	cmdline     string
	tabComplete []string
}

// New returns the Command associated with the given verb.
func New(verb string, a api.API) Command {
	ctor := Verbs[verb]
	if ctor == nil {
		return nil
	}
	return ctor(a)
}

// Keys returns a string slice with all verbs that can be invoked to run a command.
func Keys() []string {
	keys := make(sort.StringSlice, 0, len(Verbs))
	for verb := range Verbs {
		keys = append(keys, verb)
	}
	keys.Sort()
	return keys
}

// setTabComplete defines a string slice that will be used for tab completion
// at the current point in parsing.
func (c *newcommand) setTabComplete(filter string, s []string) {
	c.tabComplete = utils.TokenFilter(filter, s)
}

// setTabCompleteTag sets the tab complete list to a list of tag keys in a specific song.
func (c *newcommand) setTabCompleteTag(lit string, song *song.Song) {
	if song == nil {
		c.setTabCompleteEmpty()
		return
	}
	c.setTabComplete(lit, song.TagKeys())
}

// setTabCompleteEmpty removes all tab completions.
func (c *newcommand) setTabCompleteEmpty() {
	c.setTabComplete("", []string{})
}

// ParseTags parses a set of tags until the end of the line, and maintains the
// tab complete list according to a specified song.
func (c *newcommand) ParseTags(song *song.Song) ([]string, error) {
	c.setTabCompleteEmpty()
	tags := make([]string, 0)
	tag := ""

	for {
		tok, lit := c.Scan()

		switch tok {
		case lexer.TokenWhitespace:
			if len(tag) > 0 {
				tags = append(tags, strings.ToLower(tag))
			}
			tag = ""
		case lexer.TokenEnd, lexer.TokenComment:
			if len(tag) > 0 {
				tags = append(tags, strings.ToLower(tag))
			}
			if len(tags) == 0 {
				return nil, fmt.Errorf("Unexpected END, expected tag")
			}
			return tags, nil
		default:
			tag += lit
		}

		c.setTabCompleteTag(tag, song)
	}
}

//
// These functions belong to the old implementation.
// FIXME: remove everything below.
//

// Execute implements Command.Execute.
func (c *newcommand) Execute(class int, s string) error {
	return nil
}

// TabComplete implements Command.TabComplete.
func (c *newcommand) TabComplete() []string {
	if c.tabComplete == nil {
		// FIXME
		return make([]string, 0)
	}
	return c.tabComplete
}

// Parse implements Command.Parse.
func (c *command) SetScanner(s *lexer.Scanner) {
}

// Parse implements Command.Parse.
func (c *command) Parse() error {
	return nil
}

// Scanned implements Command.Scanned.
func (c *command) Scanned() []parser.Token {
	return make([]parser.Token, 0)
}

// TabComplete implements Command.TabComplete.
func (c *command) TabComplete() []string {
	return []string{}
}

// Exec implements Command.TabComplete.
func (c *command) Exec() error {
	return nil
}
