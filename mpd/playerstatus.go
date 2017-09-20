package mpd

import "time"

// PlayerStatus contains information about MPD's player status.
type PlayerStatus struct {
	Audio             string
	Bitrate           int
	Consume           bool
	Elapsed           float64
	ElapsedPercentage float64
	Err               string
	MixRampDB         float64
	Playlist          int
	PlaylistLength    int
	Random            bool
	Repeat            bool
	Single            bool
	Song              int
	SongID            int
	State             string
	Time              int
	Volume            int

	updateTime time.Time
}

// Strings found in the PlayerStatus.State variable.
const (
	StatePlay    string = "play"
	StateStop    string = "stop"
	StatePause   string = "pause"
	StateUnknown string = "unknown"
)

func (p *PlayerStatus) SetTime() {
	p.updateTime = time.Now()
}

func (p *PlayerStatus) Since() time.Duration {
	return time.Since(p.updateTime)
}

func (p PlayerStatus) Tick() PlayerStatus {
	if p.State != StatePlay {
		return p
	}
	diff := p.Since()
	p.SetTime()
	p.Elapsed += diff.Seconds()
	if p.Time == 0 {
		p.ElapsedPercentage = 0.0
	} else {
		p.ElapsedPercentage = float64(100) * p.Elapsed / float64(p.Time)
	}
	return p
}
