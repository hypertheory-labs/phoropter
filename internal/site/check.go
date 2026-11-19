package site

import (
	"fmt"
	"regexp"
	"strings"
)

// `phoropter index --check <program>` (seersucker): runs an indexer on a
// repository at HEAD and checks what it prints against indexers/README.md,
// so whoever writes an indexer, person or agent, can tell when it's done.

// LastName is the last name in a symbol's descriptors: …/docScreen#annotate().
// is annotate, …/`mock-panel.ts`/MockPanel# is MockPanel, and
// …/OrderService#`.ctor`(). is .ctor. A backticked name is read whole, with a
// doubled backtick standing for one, as SCIP escapes them.
func LastName(symbol string) string {
	parts := strings.Split(symbol, " ")
	if len(parts) < 5 {
		return ""
	}
	d := strings.Join(parts[4:], " ")
	var names []string
	isName := func(c byte) bool {
		return c == '_' || c == '$' || c == '+' || c == '-' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	}
	for i := 0; i < len(d); {
		switch {
		case d[i] == '`':
			var name strings.Builder
			for i++; i < len(d); i++ {
				if d[i] == '`' && i+1 < len(d) && d[i+1] == '`' {
					name.WriteByte('`')
					i++
				} else if d[i] == '`' {
					break
				} else {
					name.WriteByte(d[i])
				}
			}
			names = append(names, name.String())
			i++
		case isName(d[i]):
			j := i
			for j < len(d) && isName(d[j]) {
				j++
			}
			names = append(names, d[i:j])
			i = j
		default:
			i++
		}
	}
	if len(names) == 0 {
		return ""
	}
	return names[len(names)-1]
}

// The name check applies to names a source file spells as written. Anything
// else (.ctor, this[], <constructor> once its brackets are off) is a name an
// indexer made up, and isn't looked for.
var identifierRE = regexp.MustCompile(`^[A-Za-z_$][\w$]*$`)

const allowedMisses = 0.05

var (
	bracketsRE = regexp.MustCompile(`^<(.*)>$`)
	digitsRE   = regexp.MustCompile(`\d+$`)
)

// spellings is how a name can appear in the source. Indexers name some things
// the source never spells: scip-typescript's <constructor> is constructor, and
// an anonymous object's properties get a numeric suffix (encoding0).
func spellings(name string) []string {
	out := []string{name}
	bare := bracketsRE.ReplaceAllString(name, "$1")
	if bare != name {
		out = append(out, bare)
	}
	if undigited := digitsRE.ReplaceAllString(bare, ""); undigited != "" && undigited != bare {
		out = append(out, undigited)
	}
	return out
}

func CheckIndexer(command []string, cwd string, repo *Repo) (bool, []string) {
	var problems, report []string
	about, err := Describe(command, cwd)
	if err != nil {
		return false, []string{err.Error()}
	}
	if about.Indexer == "" {
		problems = append(problems, "--describe has no indexer")
	}
	if about.Version == "" {
		problems = append(problems, "--describe has no version")
	}
	if about.Scheme == "" {
		problems = append(problems, "--describe has no scheme")
	}
	if len(about.Extensions) == 0 {
		problems = append(problems, "--describe lists no extensions, so phoropter symbols can never pick it")
	}
	for _, n := range about.Needs {
		if n.What == "" || n.Install == "" {
			problems = append(problems, "--describe has a need without both what and install, so phoropter doctor can't say how to fix it")
		}
	}
	report = append(report, fmt.Sprintf("%s %s, scheme %s, for %s", about.Indexer, about.Version, about.Scheme, strings.Join(about.Extensions, " ")))
	if len(about.Needs) > 0 {
		var needs []string
		for _, n := range about.Needs {
			needs = append(needs, n.What)
		}
		report = append(report, "it says it needs "+strings.Join(needs, ", ")+", so the run below may fail")
	}

	sha := repo.Resolve("HEAD")
	tree, err := repo.Checkout(sha)
	if err != nil {
		return false, append(report, err.Error())
	}
	out, err := RunIndexer(command, cwd, tree, nil)
	RemoveDir(tree)
	if err != nil {
		return false, append(report, fmt.Sprintf("running it on %s failed: %s", repo.Short(sha), err))
	}
	if out.Scheme != about.Scheme {
		problems = append(problems, fmt.Sprintf("it prints scheme %s, but --describe says %s", out.Scheme, about.Scheme))
	}
	if out.Indexer == "" || out.Version == "" {
		problems = append(problems, "its output has no indexer or version")
	}
	if out.Packages == nil {
		return false, append(report, "its output has no packages object")
	}

	count, named := 0, 0
	var misses []string
	lines := map[string][]string{}
	for pkg, p := range out.Packages {
		for symbol, value := range p.Definitions {
			count++
			say := func(why string) {
				if len(problems) < 200 {
					problems = append(problems, symbol+": "+why)
				}
			}
			parts := strings.Split(symbol, " ")
			switch {
			case !IsScipSymbol(symbol):
				say("isn't a symbol in SCIP's grammar (scheme manager package version descriptors)")
			case parts[0] != about.Scheme:
				say(fmt.Sprintf("scheme %s, not %s", parts[0], about.Scheme))
			case parts[2] != pkg:
				say(fmt.Sprintf("package %s, but it's listed under %s", parts[2], pkg))
			case parts[3] != ".":
				say(`its version should be written as "."`)
			}
			file, okF := value[0].(string)
			startF, okS := value[1].(float64)
			endF, okE := value[2].(float64)
			if !okF || !okS || !okE {
				say("should be [file, startLine, endLine]")
				continue
			}
			start, end := int(startF), int(endF)
			text, seen := lines[file]
			if !seen {
				if t, ok := repo.Show(sha, file); ok {
					text = strings.Split(t, "\n")
				}
				lines[file] = text
			}
			if text == nil {
				say(fmt.Sprintf("%s isn't a file at %s; paths are relative to the repository's root", file, repo.Short(sha)))
				continue
			}
			if float64(start) != startF || float64(end) != endF || start < 1 || end < start || end > len(text) {
				say(fmt.Sprintf("lines %v–%v aren't a range in %s (%d lines; 1-based, inclusive)", value[1], value[2], file, len(text)))
				continue
			}
			name := LastName(symbol)
			rng := strings.Join(text[start-1:end], "\n")
			// A namespace (a package or a file, ending in /) is named by its path,
			// which the source doesn't spell.
			var plain []string
			for _, n := range spellings(name) {
				if identifierRE.MatchString(n) {
					plain = append(plain, n)
				}
			}
			if name == "" || strings.HasSuffix(symbol, "/") || len(plain) == 0 {
				continue
			}
			named++
			hit := false
			for _, n := range plain {
				if strings.Contains(rng, n) {
					hit = true
				}
			}
			if !hit {
				misses = append(misses, fmt.Sprintf(`%s: "%s" isn't on lines %d–%d of %s`, symbol, name, start, end, file))
			}
		}
	}
	report = append(report, fmt.Sprintf("%d package(s), %d definition(s) at %s", len(out.Packages), count, repo.Short(sha)))
	// A few names spelled differently in source (an operator's op_Addition is
	// `operator +`) are notes. Many are a range bug, which misses nearly all.
	if float64(len(misses)) > float64(named)*allowedMisses {
		problems = append(problems, misses...)
	} else if len(misses) > 0 {
		report = append(report, fmt.Sprintf("%d of %d name(s) aren't spelled in their ranges, within the %g%% allowed for names a language spells differently:", len(misses), named, allowedMisses*100))
		for i, m := range misses {
			if i == 10 {
				break
			}
			report = append(report, "  "+m)
		}
	}
	if count == 0 {
		problems = append(problems, "it found no definitions at all")
	}
	if len(problems) > 0 {
		report = append(report, fmt.Sprintf("%d problem(s):", len(problems)))
		for i, p := range problems {
			if i == 20 {
				break
			}
			report = append(report, "  "+p)
		}
	} else {
		report = append(report, "passes: every definition is a symbol in its scheme, in a file at HEAD, on lines that hold its name.")
	}
	return len(problems) == 0, report
}
