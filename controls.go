package main

type control struct {
	keys   []Key
	desc   string
	action func(*app)
}

var controls = []control{
	control{
		keys:   []Key{"q"},
		desc:   "quit",
		action: func(a *app) { a.model.StartCommandMode(QuitCommand) },
	},

	control{
		keys:   []Key{"j", DownArrowKey},
		desc:   "move down by one line",
		action: func(a *app) { a.moveDown() },
	},
	control{
		keys:   []Key{"k", UpArrowKey},
		desc:   "move up by one line",
		action: func(a *app) { a.moveUp() },
	},
	control{
		keys:   []Key{"d", PageDownKey},
		desc:   "move down by one screen",
		action: func(a *app) { a.moveDownByHalfScreen() },
	},
	control{
		keys:   []Key{"u", PageUpKey},
		desc:   "move up by one screen",
		action: func(a *app) { a.moveUpByHalfScreen() },
	},

	control{
		keys:   []Key{LeftArrowKey},
		desc:   "scroll left horizontally",
		action: func(a *app) { a.reduceXPosition() },
	},
	control{
		keys:   []Key{RightArrowKey},
		desc:   "scroll right horizontally",
		action: func(a *app) { a.increaseXPosition() },
	},

	control{
		keys:   []Key{"r"},
		desc:   "force the screen to be repainted",
		action: func(a *app) { a.discardBufferedInputAndRepaint() },
	},

	control{
		keys:   []Key{"g"},
		desc:   "move to the beginning of the file",
		action: func(a *app) { a.moveTop() },
	},
	control{
		keys:   []Key{"G"},
		desc:   "move to the end of the file",
		action: func(a *app) { a.moveBottom() },
	},

	control{
		keys:   []Key{"/"},
		desc:   "enter a new regex to search for",
		action: func(a *app) { a.model.StartCommandMode(SearchCommand) },
	},
	control{
		keys:   []Key{"n"},
		desc:   "jump to the next line matching the current regex",
		action: func(a *app) { a.jumpToMatch(false) },
	},
	control{
		keys:   []Key{"N"},
		desc:   "jump to the previous line matching the current regex",
		action: func(a *app) { a.jumpToMatch(true) },
	},

	control{
		keys:   []Key{"w"},
		desc:   "toggle line wrap mode",
		action: func(a *app) { a.toggleLineWrapMode() },
	},

	control{
		keys:   []Key{"c"},
		desc:   "change the colour of the current regex",
		action: func(a *app) { a.startColourCommand() },
	},
	control{
		keys:   []Key{"\t"},
		desc:   "cycle forward through saved regexes",
		action: func(a *app) { a.cycleRegexp(true) },
	},
	control{
		keys:   []Key{ShiftTab},
		desc:   "cycle backward though saved regexes",
		action: func(a *app) { a.cycleRegexp(false) },
	},
	control{
		keys:   []Key{"x"},
		desc:   "delete the current regex",
		action: func(a *app) { a.deleteRegexp() },
	},

	control{
		keys:   []Key{"s"},
		desc:   "seek to a percentage through the file",
		action: func(a *app) { a.model.StartCommandMode(SeekCommand) },
	},
	control{
		keys:   []Key{"b"},
		desc:   "bisect the file to search for line prefix",
		action: func(a *app) { a.model.StartCommandMode(BisectCommand) },
	},

	control{
		keys:   []Key{"`"},
		desc:   "toggle debug mode",
		action: func(a *app) { a.model.debug = !a.model.debug },
	},
}
