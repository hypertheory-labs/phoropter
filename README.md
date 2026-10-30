# Phoropter TUI 

I'd like to explore an experiment in creating a TUI (probably Go?) for a human exploring the substrate presented at http://localhost:4173/ right now.

It's REST, microdata, etc. The repo for that tool is at ../phoropter and it's generating that HTML from ../mock-control 

We are in the early stages of building this. So part of what I'm looking for here is to identify shortcomings in the HTML API. 

## Roughly, what I'm envisioning 

A tool that will be used in the context of a repo that is serving the substrate. A list of the plans will be presented. Developer can select a plan, see the details, and for each step, see (maybe in another pane) the code - so the source code in "layers" somehow - maybe navigable by keyboard shortcuts - what it was at the time of the plan under observation, what it is now, and a list of other plans that involve this file.

This will be completely read only. There is an "edge" feature coming that will have some write ability, but never to the substrate itself.
