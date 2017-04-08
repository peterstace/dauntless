# LV

LV is a log file viewer. Will LV even be the final name? Who knows.

## Status

Proof-of-concept. Completely incomplete.

## TODO List

### MVP (Essential)

* Search backwards.

* Split lines mode.

* Move left/right when in non-split mode.

* Change regexp colour using command interface.

### Important

* Bisect file. Only consider lines matching custom regexp.

* Bookmarks.

* Custom indentation in split lines mode.

* Custom disable/enable regexp colour choices.

* Allow multiple regexps.

* Cycle between regexps.

* Change colour of existing regexp.

### Nice To Have

* Signal for term size change. This would be more efficient than running `stty
  size` externally once per second to detect a term size change.

* Restore terminal upon panic/crash. Not sure how to actually implement this.
  Can use defers to restore the term state if the panic occurs in the main
goroutine. But if the panic occurs in another goroutine, we're out of luck.

* Refactoring and tests.
