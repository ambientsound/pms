package pms

import (
	"github.com/ambientsound/pms/console"
	"github.com/ambientsound/pms/message"
)

// Main does (eventually) read, evaluate, print, loop
func (pms *PMS) Main() {
	for {
		select {
		case <-pms.Connection.Connected:
			go pms.handleConnected()
		case subsystem := <-pms.Connection.IdleEvents:
			pms.handleEventIdle(subsystem)
		case <-pms.QuitSignal:
			pms.handleQuitSignal()
			return
		case <-pms.EventLibrary:
			pms.handleEventLibrary()
		case <-pms.EventQueue:
			pms.handleEventQueue()
		case <-pms.EventPlayer:
			pms.handleEventPlayer()
		case key := <-pms.EventOption:
			pms.handleEventOption(key)
		case msg := <-pms.EventMessage:
			pms.handleEventMessage(msg)
		case ev := <-pms.ui.EventKeyInput:
			pms.KeyInput(ev)
		case s := <-pms.ui.EventInputCommand:
			pms.Execute(s)
		}

		// Draw missing parts after every iteration
		pms.ui.App.PostFunc(func() {
			pms.ui.App.Update()
		})
	}
}

func (pms *PMS) handleQuitSignal() {
	console.Log("Received quit signal, exiting.")
	pms.ui.Quit()
}

func (pms *PMS) handleEventLibrary() {
	console.Log("Song library updated in MPD, assigning to UI")
	pms.ui.App.PostFunc(func() {
		pms.database.Panel().Replace(pms.database.Library())
	})
}

func (pms *PMS) handleEventQueue() {
	console.Log("Queue updated in MPD, assigning to UI")
	pms.ui.App.PostFunc(func() {
		pms.database.Panel().Replace(pms.database.Queue())
	})
}

func (pms *PMS) handleEventOption(key string) {
	console.Log("Option '%s' has been changed", key)

	switch key {
	case "topbar":
		pms.setupTopbar()
	case "columns":
		// list changed, FIXME
	case "consume":
		client := pms.CurrentMpdClient()
		if client == nil {
			return
		}
		client.Consume(pms.Options.BoolValue(key))
	case "random":
		client := pms.CurrentMpdClient()
		if client == nil {
			return
		}
		client.Random(pms.Options.BoolValue(key))
	case "single":
		client := pms.CurrentMpdClient()
		if client == nil {
			return
		}
		client.Single(pms.Options.BoolValue(key))
	case "repeat":
		client := pms.CurrentMpdClient()
		if client == nil {
			return
		}
		client.Repeat(pms.Options.BoolValue(key))
	}
}

func (pms *PMS) handleEventPlayer() {
}

func (pms *PMS) handleEventMessage(msg message.Message) {
	message.Log(msg)
	pms.ui.App.PostFunc(func() {
		pms.ui.Multibar.SetMessage(msg)
	})
}

// handleEventIdle triggers actions based on IDLE events.
func (pms *PMS) handleEventIdle(subsystem string) {
	var err error

	console.Log("MPD says it has IDLE events on the following subsystem: %s", subsystem)

	switch subsystem {
	case "database":
		err = pms.SyncLibrary()
	case "playlist":
		err = pms.SyncQueue()
	case "player":
		err = pms.UpdatePlayerStatus()
		if err != nil {
			break
		}
		err = pms.UpdateCurrentSong()
	case "options":
		err = pms.UpdatePlayerStatus()
	case "mixer":
		err = pms.UpdatePlayerStatus()
	default:
		console.Log("Ignoring updates by subsystem %s", subsystem)
	}

	if err != nil {
		pms.Error("Lost sync with MPD; reconnecting.")
		pms.Connection.Close()
	}
}
