# Seed: a Select's chosen options as properties

From the TUI's brief (`tui/docs/for-phoropter.md`, section 5). A Select's answer
renders its `- [x]` list as disabled checkboxes, so which options were chosen is only
each `<input>`'s `checked`, which the profile doesn't define. The TUI reads it anyway
and notes `answer-choices-unprofiled`. Ask: a `chosen` value for each option, inside the
`answer`. This is also where ramie's guess (reading the draft needs nothing beyond the
profile) was disproved: ramie only tested unanswered prompts.
