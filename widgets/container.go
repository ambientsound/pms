package widgets

import (
	"github.com/ambientsound/pms/api"
	"github.com/ambientsound/pms/log"
	"github.com/ambientsound/pms/multibar"
	"github.com/ambientsound/pms/style"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

type widgets struct {
	layout   *views.BoxLayout
	topbar   *Topbar
	multibar *Multibar
	table    *Table
	active   views.Widget
}

type Application struct {
	api     api.API
	events  chan tcell.Event
	screen  tcell.Screen
	widgets widgets
	style.Styled
}

var _ tcell.EventHandler = &Application{}

var _ api.UI = &Application{}

func NewApplication(a api.API) (*Application, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	err = screen.Init()
	if err != nil {
		return nil, err
	}

	screen.Clear()
	screen.Show()

	return &Application{
		api:    a,
		events: make(chan tcell.Event, 1024),
		screen: screen,
	}, nil
}

func (app *Application) Init() {
	app.widgets.topbar = NewTopbar()
	app.widgets.table = NewTable(app.api)
	app.widgets.multibar = NewMultibarWidget(app.api)

	app.widgets.layout = views.NewBoxLayout(views.Vertical)
	app.widgets.layout.AddWidget(app.widgets.topbar, 0)
	app.widgets.layout.AddWidget(app.widgets.table, 1)
	app.widgets.layout.AddWidget(app.widgets.multibar, 0)
	app.widgets.layout.SetView(app.screen)

	app.widgets.active = app.widgets.table

	app.ActivateWindow(api.WindowLogs)
}

func (app *Application) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventResize:
		cols, rows := e.Size()
		log.Debugf("terminal resize: %dx%d", cols, rows)
		app.screen.Sync()
		app.widgets.layout.Resize()
		app.widgets.layout.SetView(app.screen)
		return true
	case *tcell.EventKey:
		return false
	default:
		log.Debugf("unrecognized input event: %T %+v", e, e)
		app.widgets.layout.HandleEvent(ev)
		return false
	}
}

func (app *Application) Draw() {
	app.widgets.layout.Draw()
	app.updateCursor()
	app.screen.Show()
}

func (app *Application) Poll() {
	for {
		app.events <- app.screen.PollEvent()
	}
}

func (app *Application) Events() <-chan tcell.Event {
	return app.events
}

func (app *Application) Finish() {
	app.screen.Fini()
}

func (app *Application) Refresh() {
	app.screen.Sync()
}

func (app *Application) TableWidget() api.TableWidget {
	return app.widgets.table
}

func (app *Application) ActivateWindow(window api.Window) {
	var widget views.Widget

	switch window {
	case api.WindowLogs:
		widget = app.widgets.table
		app.widgets.table.SetList(log.List(log.InfoLevel))
		app.widgets.table.SetColumns([]string{"timestamp", "logLevel", "logMessage"})
	case api.WindowMusic:
		widget = app.widgets.table
		app.widgets.table.SetList(nil)
	case api.WindowPlaylists:
		widget = app.widgets.table
		app.widgets.table.SetList(nil)
	default:
		panic("widget not implemented")
	}

	log.Debugf("want to activate widget %#v", widget)
	log.Debugf("first deactivating widget %#v", app.widgets.active)

	app.widgets.layout.RemoveWidget(app.widgets.active)
	app.widgets.layout.InsertWidget(1, widget, 1.0)

	app.widgets.active = widget
}

func (app *Application) updateCursor() {
	switch app.api.Multibar().Mode() {
	case multibar.ModeInput, multibar.ModeSearch:
		_, ymax := app.screen.Size()
		x := app.api.Multibar().Cursor() + 1
		app.screen.ShowCursor(x, ymax-1)
	default:
		app.screen.HideCursor()
	}
}