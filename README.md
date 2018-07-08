# Dauntless

Dauntless is a [terminal pager](https://en.wikipedia.org/wiki/Terminal_pager)
inspired by [GNU less](https://www.gnu.org/software/less/). It contains
additional features that make it well suited to viewing and analysing log files.

## Status

It's still under active development, however is stable and ready for every day
use.

## Usage

To use dauntless to view a file, use `dauntless <filename>`.

Dauntless can also accept input via stdin. For example, `echo "hello world" | dauntless`.

## Key Controls

The key controls used to control dauntless are inspired by vim and less:

    q - quit

    ? - show help

    j, <down-arrow> - move down by one line

    k, <up-arrow> - move up by one line

    d, <page-down> - move down by one screen

    u, <page-up> - move up by one screen

    <left-arrow> - scroll left horizontally

    <right-arrow> - scroll right horizontally

    r - force the screen to be repainted

    g - move to the beginning of the file

    G - move to the end of the file

    / - enter a new regex to search for

    n - jump to the next line matching the current regex

    N - jump to the previous line matching the current regex

    w - toggle line wrap mode

    c - change the colour of the current regex

    <tab> - cycle forward through saved regexes

    <shift-tab> - cycle backward though saved regexes

    x - delete the current regex

    s - seek to a percentage through the file

    b - bisect the file to search for line prefix

    ` - toggle debug mode

## Dauntless Crashed (and now my terminal is messed up!)

When Dauntless starts up, it enters [`cbreak`
mode](https://en.wikipedia.org/wiki/Cooked_mode). If it crashes, then it may
not exit `cbreak` mode before exiting. To manually leave `cbreak` mode, enter
(blindly!) the command `stty sane`.

## TODO List

#### Most Important

* Substitute command.

* Should not be able to see past end of file if the file is bigger than 1
  screen.

#### Important

* Persistence of key model state.

* Offset history (can go back through history).

* Custom disable/enable regexp colour choices.

* Predefined (config file) and pre-loaded regexes. E.g. to highlight errors
  that would always appear the same way.

* Copy/paste friendly mode. Toggle indent away, show all lines, no spaces at
  end of lines.

* Show search progress.

* Seek should be a 'long file op'.

* Bisect has an off-by-one error when landing at the target line.

#### Least Important

* Don't fatal on any errors. Instead, just show them in the info bar.

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

#### Known Bugs

* Bisect past EOF is fatal. Noticed that the last line in the file was partial,
  so that may have something to do with it.
