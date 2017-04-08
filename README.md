# LV

LV is a log file viewer. Will LV even be the final name? Who knows.

## Status

Proof-of-concept. Completely incomplete.

## TODO List

### MVP (Essential)

* Help screen.

* Split lines mode.

* Move left/right when in non-split mode.

* Change regexp colour using command interface.

### Important

* Ctrl-C should cancel command inputs.

* Prune fwd and bck.

* Display message to user.

* Bisect file. Only consider lines matching custom regexp.

* Bookmarks.

* Custom indentation in split lines mode.

* Custom disable/enable regexp colour choices.

* Allow multiple regexps.

* Cycle between regexps.

* Change colour of existing regexp.

* Some buffer sizes and chunk sizes are quite small, and would lead to bad
  performance. These should ideally be configurable.

### Nice To Have

* Signal for term size change. This would be more efficient than running `stty
  size` externally once per second to detect a term size change.

* Restore terminal upon panic/crash. Not sure how to actually implement this.
  Can use defers to restore the term state if the panic occurs in the main
goroutine. But if the panic occurs in another goroutine, we're out of luck.

### Technical Debt

* `endOffset` method on `line`

* Inefficiency in finding next match.

* Inefficiency in finding bottom of file offset.

* Add assertions back in for main data structure.

### Bugs

* Logging outside of reactor when jumping to bottom.

* Not closing some opened files.
