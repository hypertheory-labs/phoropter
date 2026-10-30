# Phoropter TUI 

I'd like to explore an experiment in creating a TUI (probably Go?) for a human exploring the substrate presented at http://localhost:4173/ right now.

It's REST, microdata, etc. The repo for that tool is at ../phoropter and it's generating that HTML from ../mock-control 

We are in the early stages of building this. So part of what I'm looking for here is to identify shortcomings in the HTML API. 

## Roughly, what I'm envisioning 

A tool that will be used in the context of a repo that is serving the substrate. A list of the plans will be presented. Developer can select a plan, see the details, and for each step, see (maybe in another pane) the code - so the source code in "layers" somehow - maybe navigable by keyboard shortcuts - what it was at the time of the plan under observation, what it is now, and a list of other plans that involve this file.

This will be completely read only. There is an "edge" feature coming that will have some write ability, but never to the substrate itself.

## Running it

```sh
go build -o bin/phoropter-tui .
bin/phoropter-tui                                # http://localhost:4173/, phoropter's npm run dev
bin/phoropter-tui testdata/phoropter-e8107f3     # or a built site on disk
bin/phoropter-tui gaps [site]                    # crawl, and list where it read past the profile
go test ./...
```

It reads only the substrate: no git. In a plan, `1` `2` `3` flip an anchor between
base, as-built head, and generated; `enter` follows the rels out of an entry; `g`
shows the gaps the session has hit so far.

`testdata/` holds sites built by `../phoropter` (`node src/build.ts <repo> <commit> --out …`)
from mock-control at `f02528f` and phoropter at `e8107f3`, so tests don't need a server.
Rebuild them when the profile changes.
