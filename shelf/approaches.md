# Seed: approaches, written up fairly, and a catalog of them

From the .NET Web API course (ended 2026-11-18). Each topic in the class's `plans/` had
a `plan.md` and a folder of `approaches/`, one file per way a reasonable team might do
it. Each approach got the same fair treatment, whether or not it was picked, so a reader
understands *when they'd pick it*. Jeff named them live in class, the class built one,
and an agent fleshed out the record afterwards, as review. A "classroom decision"
callout said when a pick was made for teaching and what a real project would do instead
(close to `shelf/re-check.md`'s **Holds while:**).

**An approach file is a Decline written well enough to choose elsewhere.** Phoropter's
Declines are a paragraph. Whether a Decline should be able to link a fuller write-up is
a format question. Muslin (`init`) declined it until a phoropter Decline is seen failing
a reader.

**A catalog across projects (Jeff):** "which projects in our company have used this
approach?" Point an agent at a URI: "find me approaches to error handling in Angular,
and add any you can think of." Jeff is building one for the class first. Phoropter is
per-repository on purpose, so this is a later question or a different product.

**The CVE example (Jeff, keep this one):** an agent reading the catalog says "I found this
approach, but for reasons X, Y and Z, and here are three CVEs, I don't think it should be
kept." Approaches amended over time, by agents, with evidence. What makes that
trustworthy is what phoropter's sources do: who said it, and on what (`measured`,
`Claude, not asked`). A catalog that agents edit without provenance drifts confidently.
It came from Jeff's homelab: he asked another Claude whether a docker-compose app could
run on k3s, adding "just thinking about it". That hedge left a gap, and the answer was
"you could, but I wouldn't", with three CVEs (it was webhook-driven). Muslin's "wondering
isn't asking" convention is the same gap, inside one repository.
