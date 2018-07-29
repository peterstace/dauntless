package dauntless

type control struct {
	Keys   []Key
	Desc   string
	action func(*app)
}

var Controls = []control{
	control{
		Keys:   []Key{"q"},
		Desc:   "quit",
		action: func(a *app) { a.model.StartCommandMode(QuitCommand) },
	},
	control{
		Keys:   []Key{"?"}, // TODO: Add F1
		Desc:   "show help",
		action: func(a *app) { a.model.showHelp = !a.model.showHelp },
	},

	control{
		Keys:   []Key{"j", DownArrowKey},
		Desc:   "move down by one line",
		action: func(a *app) { a.model.moveDown() },
	},
	control{
		Keys:   []Key{"k", UpArrowKey},
		Desc:   "move up by one line",
		action: func(a *app) { a.model.moveUp() },
	},
	control{
		Keys:   []Key{"d", PageDownKey},
		Desc:   "move down by one screen",
		action: func(a *app) { a.model.moveDownByHalfScreen() },
	},
	control{
		Keys:   []Key{"u", PageUpKey},
		Desc:   "move up by one screen",
		action: func(a *app) { a.model.moveUpByHalfScreen() },
	},

	control{
		Keys:   []Key{LeftArrowKey},
		Desc:   "scroll left horizontally",
		action: func(a *app) { a.model.reduceXPosition() },
	},
	control{
		Keys:   []Key{RightArrowKey},
		Desc:   "scroll right horizontally",
		action: func(a *app) { a.model.increaseXPosition() },
	},

	control{
		Keys:   []Key{"r"},
		Desc:   "force screen refresh",
		action: func(a *app) { a.discardBufferedInputAndRepaint() },
	},

	control{
		Keys:   []Key{"g"},
		Desc:   "move to start of file",
		action: func(a *app) { a.model.moveTop() },
	},
	control{
		Keys:   []Key{"G"},
		Desc:   "move to end of file",
		action: func(a *app) { a.moveBottom() },
	},

	control{
		Keys:   []Key{"/"},
		Desc:   "enter a new search regex",
		action: func(a *app) { a.model.StartCommandMode(SearchCommand) },
	},
	control{
		Keys:   []Key{"n"},
		Desc:   "jump to next regex match",
		action: func(a *app) { a.jumpToMatch(false) },
	},
	control{
		Keys:   []Key{"N"},
		Desc:   "jump to previous regex match",
		action: func(a *app) { a.jumpToMatch(true) },
	},

	control{
		Keys:   []Key{"w"},
		Desc:   "toggle line wrap mode",
		action: func(a *app) { a.model.toggleLineWrapMode() },
	},

	control{
		Keys:   []Key{"c"},
		Desc:   "change regex highlight colour",
		action: func(a *app) { a.model.startColourCommand() },
	},
	control{
		Keys:   []Key{"\t"},
		Desc:   "cycle forward through regexes",
		action: func(a *app) { a.model.cycleRegexp(true) },
	},
	control{
		Keys:   []Key{ShiftTab},
		Desc:   "cycle backward though regexes",
		action: func(a *app) { a.model.cycleRegexp(false) },
	},
	control{
		Keys:   []Key{"x"},
		Desc:   "delete regex",
		action: func(a *app) { a.model.deleteRegexp() },
	},

	control{
		Keys:   []Key{"s"},
		Desc:   "seek to a percentage",
		action: func(a *app) { a.model.StartCommandMode(SeekCommand) },
	},
	control{
		Keys:   []Key{"b"},
		Desc:   "bisect line prefix",
		action: func(a *app) { a.model.StartCommandMode(BisectCommand) },
	},

	control{
		Keys:   []Key{"`"},
		Desc:   "toggle debug mode",
		action: func(a *app) { a.model.debug = !a.model.debug },
	},
}
