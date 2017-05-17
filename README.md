# Dauntless

Dauntless is a log viewer inspired by [GNU
less](https://www.gnu.org/software/less/). It contains additional features that
make analysing log files easier.

## Status

It's currently at feature-parity with `less`. There are still some planned
features that haven't been implemented yet.

## Dauntless Crashed (and now my terminal is messed up!)

Dauntless is still in active development, and may crash. When Dauntless starts
up, it enters [`cbreak` mode](https://en.wikipedia.org/wiki/Cooked_mode). If it
crashes, then it may not exit `cbreak` mode before exiting. To manually leave
`cbreak` mode, enter (blindly!) the command `stty sane`.

## TODO List

### Most Important

* Up/Down keys in command mode to use history. Should be per command.

* Show regex in red if it doesn't compile.

* Show regex highlighting as it's being typed in.

* Only allow one search at a time. Show progress. Allow search to be cancelled.
  Modal search?

* Offset history.

* Shift-tab cycles through regexp backwards.

* Debug screen. Lines loaded forwards, backwards, GC stats, memory usage, cycle
  number etc. Put on a toggle, and overlay the info.

### Important

* Help screen. Application name, author, copyright notice. Then a list of key
  mappings.

* Custom disable/enable regexp colour choices.

* Predefined (config file) and pre-loaded regexes. E.g. to highlight errors
  that would always appear the same way.

* Don't fatal on any errors. Instead, just show them in the info bar.

* Copy/paste friendly mode. Toggle indent away, show all lines, no spaces at
  end of lines.

* Highlight regex in status bar with its own style.

### Least Important

* Bookmarks.

* View bz2 files in-place.

* View over scp.

* Signal for term size change. This would be more efficient than running `stty
  size` externally once per second to detect a term size change.

* Restore terminal upon panic/crash. Not sure how to actually implement this.
  Can use defers to restore the term state if the panic occurs in the main
goroutine. But if the panic occurs in another goroutine, we're out of luck.

* Scrolling support when entering a command. Currently, the user cannot see
  what they're entering past the end of the screen if they're entering
something long.

### Technical Debt

* Add assertions back in for main data structure.

### Known Bugs

* Bisect past EOF is fatal. Noticed that the last line in the file was partial,
  so that may have something to do with it.
