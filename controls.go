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
		keys:   []Key{"?"}, // TODO: Add F1
		desc:   "show help",
		action: func(a *app) { a.model.showHelp = !a.model.showHelp },
	},

	control{
		keys:   []Key{"j", DownArrowKey},
		desc:   "move down by one line",
		action: func(a *app) { a.model.moveDown() },
	},
	control{
		keys:   []Key{"k", UpArrowKey},
		desc:   "move up by one line",
		action: func(a *app) { a.model.moveUp() },
	},
	control{
		keys:   []Key{"d", PageDownKey},
		desc:   "move down by one screen",
		action: func(a *app) { a.model.moveDownByHalfScreen() },
	},
	control{
		keys:   []Key{"u", PageUpKey},
		desc:   "move up by one screen",
		action: func(a *app) { a.model.moveUpByHalfScreen() },
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
		desc:   "force screen refresh",
		action: func(a *app) { a.discardBufferedInputAndRepaint() },
	},

	control{
		keys:   []Key{"g"},
		desc:   "move to start of file",
		action: func(a *app) { a.model.moveTop() },
	},
	control{
		keys:   []Key{"G"},
		desc:   "move to end of file",
		action: func(a *app) { a.moveBottom() },
	},

	control{
		keys:   []Key{"/"},
		desc:   "enter a new search regex",
		action: func(a *app) { a.model.StartCommandMode(SearchCommand) },
	},
	control{
		keys:   []Key{"n"},
		desc:   "jump to next regex match",
		action: func(a *app) { a.jumpToMatch(false) },
	},
	control{
		keys:   []Key{"N"},
		desc:   "jump to previous regex match",
		action: func(a *app) { a.jumpToMatch(true) },
	},

	control{
		keys:   []Key{"w"},
		desc:   "toggle line wrap mode",
		action: func(a *app) { a.model.toggleLineWrapMode() },
	},

	control{
		keys:   []Key{"c"},
		desc:   "change regex highlight colour",
		action: func(a *app) { a.startColourCommand() },
	},
	control{
		keys:   []Key{"\t"},
		desc:   "cycle forward through regexes",
		action: func(a *app) { a.cycleRegexp(true) },
	},
	control{
		keys:   []Key{ShiftTab},
		desc:   "cycle backward though regexes",
		action: func(a *app) { a.cycleRegexp(false) },
	},
	control{
		keys:   []Key{"x"},
		desc:   "delete regex",
		action: func(a *app) { a.deleteRegexp() },
	},

	control{
		keys:   []Key{"s"},
		desc:   "seek to a percentage",
		action: func(a *app) { a.model.StartCommandMode(SeekCommand) },
	},
	control{
		keys:   []Key{"b"},
		desc:   "bisect line prefix",
		action: func(a *app) { a.model.StartCommandMode(BisectCommand) },
	},

	control{
		keys:   []Key{"`"},
		desc:   "toggle debug mode",
		action: func(a *app) { a.model.debug = !a.model.debug },
	},
}
