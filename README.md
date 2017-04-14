# Dauntless

Dauntless is a log viewer inspired by GNU less. It contains additional features
that make analysing log files easier.

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

* Bisect file.

* Configuration file.

* Only consider lines matching custom regexp during bisection.

### Important

* Timeout for displaying loading screen.

* Managed cursor position (currently goes off the end of the screen in tmux).

* Help screen. Application name, author, copyright notice. Then a list of key
  mappings.

* Arrow keys in command mode (at least for search?).

* Cursor should follow current position in command mode.

* Custom indentation in split lines mode.

* Custom disable/enable regexp colour choices.

* Predefined (config file) and pre-loaded regexes. E.g. to highlight errors
  that would always appear the same way.

### Least Important

* Bookmarks.

* View bz2 files in-place.

* View over scp.

* Signal for term size change. This would be more efficient than running `stty
  size` externally once per second to detect a term size change.

* Restore terminal upon panic/crash. Not sure how to actually implement this.
  Can use defers to restore the term state if the panic occurs in the main
goroutine. But if the panic occurs in another goroutine, we're out of luck.

### Technical Debt

* Inefficiency in finding next match (use std lib line reader)

* Should use backward line reader when finding jump-to-bottom offset.

* Add assertions back in for main data structure.

### Bugs

* Seeking to zero percentage is fatal.

None known (yet).
