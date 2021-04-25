# Options

See the documentation on [setting options](commands.md#setting-global-options) for more information on syntax.

## Cursor position

* `set center`  
  `set nocenter`

  If set, the viewport is automatically moved so that the cursor stays in the center, if possible.


## Visual options

### Visible columns of tracklist

* `set columns=<tag>[,<tag>[...]]`

  Define which tags should be shown in the tracklist.

  A comma-separated list of tag names must be given, such as the default `artist,track,title,album,year,time`.

### Sort order

* `set sort=<tag>[,<tag>[...]]`

  Set the default sort order, for when using the [`sort` command](commands.md#manipulating-lists) without any parameters.

  A comma-separated list of tag names must be given, such as the default `file,track,disc,album,year,albumartistsort`.

### Information bar ("top bar")

* `set topbar=<spec>`

  Define the layout and visible items in the _top bar_.
  See the [styling guide](styling.md#top-bar) for information on how to configure the top bar.

  The default value is `"|$shortname $version||;${tag|artist} - ${tag|title}||${tag|album}, ${tag|year};$volume $mode $elapsed ${state} $time;|[${list|index}/${list|total}] ${list|title}||;;"`.

### Playback mode

* `set consume`
* `set random`
* `set single`
* `set repeat`

  Set the playback mode. The mode can be toggled off or inverted, e.g. `set norandom`, `set invrandom`.

  See [here](https://musicpd.org/doc/protocol/playback_option_commands.html) for more information on the different modes.
