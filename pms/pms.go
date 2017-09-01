package pms

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ambientsound/pms/api"
	"github.com/ambientsound/pms/console"
	"github.com/ambientsound/pms/index"
	"github.com/ambientsound/pms/input"
	"github.com/ambientsound/pms/input/keys"
	"github.com/ambientsound/pms/message"
	pms_mpd "github.com/ambientsound/pms/mpd"
	"github.com/ambientsound/pms/options"
	"github.com/ambientsound/pms/song"
	"github.com/ambientsound/pms/songlist"
	"github.com/ambientsound/pms/style"
	"github.com/ambientsound/pms/widgets"
	"github.com/ambientsound/pms/xdg"
	"github.com/gdamore/tcell"

	"github.com/ambientsound/gompd/mpd"
)

// PMS is a kitchen sink of different objects, glued together as a singleton class.
type PMS struct {
	mpdStatus   pms_mpd.PlayerStatus
	currentSong *song.Song
	Index       *index.Index
	CLI         *input.CLI
	ui          *widgets.UI
	Queue       *songlist.Queue
	Library     *songlist.Library
	clipboards  map[string]songlist.Songlist
	Options     *options.Options
	Sequencer   *keys.Sequencer
	stylesheet  style.Stylesheet
	mutex       sync.Mutex

	// MPD connection object
	Connection *Connection

	// Local versions of MPD's queue and song library, in addition to the song library version that was indexed.
	queueVersion   int
	libraryVersion int
	indexVersion   int

	// EventIndex receives a signal when the search index has been updated.
	EventIndex chan int

	// EventList receives a signal when current songlist has been changed.
	EventList chan int

	// EventIndex receives a signal when MPD's library has been updated and retrieved.
	EventLibrary chan int

	// EventMessage is used to display text in the statusbar.
	EventMessage chan message.Message

	// EventOption receives a signal when options have been changed.
	EventOption chan string

	// EventPlayer receives a signal when MPD's "player" status changes in an IDLE event.
	EventPlayer chan int

	// EventPlayer receives a signal when MPD's "playlist" status changes in an IDLE event.
	EventQueue chan int

	// EventPlayer receives a signal when PMS should quit.
	QuitSignal chan int
}

func createDirectory(dir string) error {
	dirMode := os.ModeDir | 0755
	return os.MkdirAll(dir, dirMode)
}

func makeAddress(host, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}

func indexDirectory(host, port string) string {
	cacheDir := xdg.CacheDirectory()
	indexDir := path.Join(cacheDir, host, port, "index")
	return indexDir
}

func indexStateFile(host, port string) string {
	cacheDir := xdg.CacheDirectory()
	stateFile := path.Join(cacheDir, host, port, "state")
	return stateFile
}

func (pms *PMS) writeIndexStateFile(version int) error {
	path := indexStateFile(pms.Connection.Host, pms.Connection.Port)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	str := fmt.Sprintf("%d\n", version)
	file.WriteString(str)
	return nil
}

func (pms *PMS) readIndexStateFile() (int, error) {
	path := indexStateFile(pms.Connection.Host, pms.Connection.Port)
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		version, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return 0, err
		}
		return version, nil
	}

	return 0, fmt.Errorf("No data in index file")
}

func (pms *PMS) Message(format string, a ...interface{}) {
	pms.EventMessage <- message.Format(format, a...)
}

func (pms *PMS) Error(format string, a ...interface{}) {
	pms.EventMessage <- message.Errorf(format, a...)
}

func (pms *PMS) Wait() {
	pms.ui.Wait()
}

// handleConnected (re)synchronizes MPD's state with PMS.
func (pms *PMS) handleConnected() {
	var err error

	console.Log("New connection to MPD.")

	console.Log("Updating current song...")
	err = pms.UpdateCurrentSong()
	if err != nil {
		goto errors
	}

	console.Log("Synchronizing queue...")
	err = pms.SyncQueue()
	if err != nil {
		goto errors
	}

	console.Log("Synchronizing library...")
	err = pms.SyncLibrary()
	if err != nil {
		goto errors
	}

	pms.Message("Ready.")

	return

errors:

	pms.Error("ERROR: %s", err)
	pms.Connection.Close()
}

// CurrentMpdClient ensures there is a valid MPD connection, and returns the MPD client object.
func (pms *PMS) CurrentMpdClient() *mpd.Client {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		pms.Error("%s", err)
	}
	return client
}

// CurrentQueue returns the queue songlist.
func (pms *PMS) CurrentQueue() *songlist.Queue {
	return pms.Queue
}

// CurrentPlayerStatus returns a copy of the current MPD player status as seen by PMS.
func (pms *PMS) CurrentPlayerStatus() pms_mpd.PlayerStatus {
	return pms.mpdStatus
}

// CurrentIndex returns the Bleve search index.
func (pms *PMS) CurrentIndex() *index.Index {
	return pms.Index
}

// CurrentSonglistWidget returns the current songlist.
func (pms *PMS) CurrentSonglistWidget() api.SonglistWidget {
	return pms.ui.Songlist
}

// Stylesheet returns the global stylesheet.
func (pms *PMS) Stylesheet() style.Stylesheet {
	return pms.stylesheet
}

// Stylesheet returns the global stylesheet.
func (pms *PMS) Multibar() api.MultibarWidget {
	return pms.ui.Multibar
}

// UI returns the tcell UI widget.
func (pms *PMS) UI() api.UI {
	return pms.ui
}

// RunTicker starts a ticker that will increase the elapsed time every second.
func (pms *PMS) RunTicker() {
	ticker := time.NewTicker(time.Millisecond * 1000)
	defer ticker.Stop()
	for range ticker.C {
		pms.mpdStatus.Tick()
		pms.EventPlayer <- 0
	}
}

// Sync retrieves the MPD library and stores it as a Songlist in the
// PMS.Library variable. Furthermore, the search index is opened, and if it is
// older than the database version, a reindex task is started.
//
// If the Songlist or Index is cached at the correct version, that part goes untouched.
func (pms *PMS) SyncLibrary() error {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return err
	}

	stats, err := client.Stats()
	if err != nil {
		return fmt.Errorf("Error while retrieving library stats from MPD: %s", err)
	}

	libraryVersion, err := strconv.Atoi(stats["db_update"])
	console.Log("Sync(): server reports library version %d", libraryVersion)
	console.Log("Sync(): local version is %d", pms.libraryVersion)

	if libraryVersion != pms.libraryVersion {
		pms.Message("Retrieving library metadata, %s songs...", stats["songs"])
		library, err := pms.retrieveLibrary()
		if err != nil {
			return fmt.Errorf("Error while retrieving library from MPD: %s", err)
		}
		pms.Library = library
		pms.libraryVersion = libraryVersion
		pms.Message("Library metadata at version %d.", pms.libraryVersion)
		pms.EventLibrary <- 1
	}

	console.Log("Sync(): opening search index")
	err = pms.openIndex()
	if err != nil {
		return fmt.Errorf("Error while opening index: %s", err)
	}
	console.Log("Sync(): index at version %d", pms.indexVersion)
	pms.EventIndex <- 1

	if libraryVersion != pms.indexVersion {
		console.Log("Sync(): index version differs from library version, reindexing...")
		err = pms.ReIndex()
		if err != nil {
			return fmt.Errorf("Failed to reindex: %s", err)
		}

		err = pms.writeIndexStateFile(pms.indexVersion)
		if err != nil {
			console.Log("Sync(): couldn't write index state file: %s", err)
		}
		console.Log("Sync(): index updated to version %d", pms.indexVersion)
	}

	console.Log("Sync(): finished.")

	return nil
}

func (pms *PMS) SyncQueue() error {
	if err := pms.UpdatePlayerStatus(); err != nil {
		return err
	}
	if pms.queueVersion == pms.mpdStatus.Playlist {
		return nil
	}
	console.Log("Retrieving changed songs in queue...")
	queueChanges, err := pms.retrieveQueue()
	if err != nil {
		return fmt.Errorf("Error while retrieving queue from MPD: %s", err)
	}
	console.Log("Total of %d changed songs in queue.", queueChanges.Len())
	newQueue, err := pms.Queue.Merge(queueChanges)
	if err != nil {
		return fmt.Errorf("Error while merging queue changes: %s", err)
	}
	if err := newQueue.Truncate(pms.mpdStatus.PlaylistLength); err != nil {
		return fmt.Errorf("Error while truncating queue: %s", err)
	}

	// Replace list while preserving cursor position, either at song ID, or if
	// that failed, place it at the nearest position.
	song := pms.Queue.CursorSong()
	cursor := pms.Queue.Cursor()
	pms.Queue = newQueue
	if err := pms.Queue.CursorToSong(song); err != nil {
		pms.Queue.SetCursor(cursor)
	}

	pms.queueVersion = pms.mpdStatus.Playlist
	console.Log("Queue at version %d.", pms.queueVersion)
	pms.EventQueue <- 1
	return nil
}

func (pms *PMS) retrieveLibrary() (*songlist.Library, error) {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return nil, err
	}

	timer := time.Now()
	list, err := client.ListAllInfo("/")
	if err != nil {
		return nil, err
	}
	console.Log("ListAllInfo in %s", time.Since(timer).String())

	s := songlist.NewLibrary()
	s.AddFromAttrlist(list)
	return s, nil
}

func (pms *PMS) retrieveQueue() (*songlist.Queue, error) {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return nil, err
	}

	timer := time.Now()
	list, err := client.PlChanges(pms.queueVersion, -1, -1)
	if err != nil {
		return nil, err
	}
	console.Log("PlChanges in %s", time.Since(timer).String())

	s := songlist.NewQueue(pms.CurrentMpdClient)
	s.AddFromAttrlist(list)
	return s, nil
}

func (pms *PMS) openIndex() error {
	timer := time.Now()
	indexDir := indexDirectory(pms.Connection.Host, pms.Connection.Port)
	err := createDirectory(indexDir)
	if err != nil {
		return fmt.Errorf("Unable to create index directory %s!", indexDir)
	}

	if pms.Index != nil {
		pms.Index.Close()
	}

	pms.Index, err = index.New(indexDir, pms.Library)
	if err != nil {
		return fmt.Errorf("Unable to acquire index: %s", err)
	}

	pms.indexVersion, err = pms.readIndexStateFile()
	if err != nil {
		console.Log("Sync(): couldn't read index state file: %s", err)
	}

	console.Log("Opened search index in %s", time.Since(timer).String())

	return nil
}

func (pms *PMS) CurrentSong() *song.Song {
	return pms.currentSong
}

// UpdateCurrentSong stores a local copy of the currently playing song.
func (pms *PMS) UpdateCurrentSong() error {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return err
	}

	attrs, err := client.CurrentSong()
	if err != nil {
		return err
	}

	console.Log("MPD current song: %s", attrs["file"])

	s := song.New()
	s.SetTags(attrs)
	pms.currentSong = s

	pms.EventPlayer <- 0

	return nil
}

// UpdatePlayerStatus populates pms.mpdStatus with data from the MPD server.
func (pms *PMS) UpdatePlayerStatus() error {
	client, err := pms.Connection.MpdClient()
	if err != nil {
		return err
	}

	attrs, err := client.Status()
	if err != nil {
		return err
	}

	pms.mpdStatus.SetTime()

	console.Log("MPD player status: %s", attrs)

	pms.mpdStatus.Audio = attrs["audio"]
	pms.mpdStatus.Err = attrs["err"]
	pms.mpdStatus.State = attrs["state"]

	// The time field is divided into ELAPSED:LENGTH.
	// We only need the length field, since the elapsed field is sent as a
	// floating point value.
	split := strings.Split(attrs["time"], ":")
	if len(split) == 2 {
		pms.mpdStatus.Time, _ = strconv.Atoi(split[1])
	} else {
		pms.mpdStatus.Time = -1
	}

	pms.mpdStatus.Bitrate, _ = strconv.Atoi(attrs["bitrate"])
	pms.mpdStatus.Playlist, _ = strconv.Atoi(attrs["playlist"])
	pms.mpdStatus.PlaylistLength, _ = strconv.Atoi(attrs["playlistlength"])
	pms.mpdStatus.Song, _ = strconv.Atoi(attrs["song"])
	pms.mpdStatus.SongID, _ = strconv.Atoi(attrs["songid"])
	pms.mpdStatus.Volume, _ = strconv.Atoi(attrs["volume"])

	pms.mpdStatus.Elapsed, _ = strconv.ParseFloat(attrs["elapsed"], 64)
	pms.mpdStatus.ElapsedPercentage, _ = strconv.ParseFloat(attrs["elapsedpercentage"], 64)
	pms.mpdStatus.MixRampDB, _ = strconv.ParseFloat(attrs["mixrampdb"], 64)

	pms.mpdStatus.Consume, _ = strconv.ParseBool(attrs["consume"])
	pms.mpdStatus.Random, _ = strconv.ParseBool(attrs["random"])
	pms.mpdStatus.Repeat, _ = strconv.ParseBool(attrs["repeat"])
	pms.mpdStatus.Single, _ = strconv.ParseBool(attrs["single"])

	pms.EventPlayer <- 0

	// Make sure any error messages are relayed to the user
	if len(attrs["error"]) > 0 {
		pms.Error(attrs["error"])
	}

	return nil
}

func (pms *PMS) ReIndex() error {
	timer := time.Now()
	if err := pms.Index.IndexFull(); err != nil {
		return err
	}
	pms.indexVersion = pms.libraryVersion
	pms.Message("Song library index complete, took %s", time.Since(timer).String())
	pms.EventIndex <- 1
	return nil
}

// KeyInput receives key input signals, checks the sequencer for key bindings,
// and runs commands if key bindings are found.
func (pms *PMS) KeyInput(ev *tcell.EventKey) {
	matches := pms.Sequencer.KeyInput(ev)
	seqString := pms.Sequencer.String()
	statusText := seqString

	input := pms.Sequencer.Match()
	if !matches || input != nil {
		// Reset statusbar if there is either no match or a complete match.
		statusText = ""
	}

	pms.EventMessage <- message.Sequencef(statusText)

	if input == nil {
		return
	}

	//console.Log("Input sequencer matches bind: '%s' -> '%s'", seqString, input.Command)
	pms.ui.EventInputCommand <- input.Command
}

func (pms *PMS) Execute(cmd string) {
	console.Log("Execute command: '%s'", cmd)
	err := pms.CLI.Execute(cmd)
	if err != nil {
		pms.Error("%s", err)
	}
}

// Clipboard returns the default clipboard.
func (pms *PMS) Clipboard() songlist.Songlist {
	return pms.clipboards["default"]
}
