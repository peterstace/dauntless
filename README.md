# LV

LV is a log file viewer. Will LV even be the final name? Who knows.

## Status

Proof-of-concept. Completely incomplete.

## It crashed, and now my terminal is screwed up...

Enter (blindly) the command `stty sane` to restore the terminal to a useable
state.

## TODO List

### Most Important

* Rebrand to "Dauntless".

* Configuration file.

### Important

* Timeout for displaying loading screen.

* Bisect file. Only consider lines matching custom regexp.

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

* Edit over scp.

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
