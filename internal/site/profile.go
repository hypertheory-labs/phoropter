package site

import (
	"fmt"
	"strings"
)

// The profile defines every rel and microdata property on these pages. Its
// text is the public API's documentation, carried over word for word from
// the TypeScript version (organdy).

var profileRels = [][3]string{
	{`profile`, `registered`, `What do the rels and properties on this page mean? This document.`},
	{`help`, `registered`, `How are plans written? From every page to the guide, which is the same on every site.`},
	{`draft`, `phoropter`, `Is a plan being drafted? From every page to <code>draft/</code>, the repository’s one draft, while <code>plan.draft.md</code> exists. Only the dev server adds it; a built site never has a draft.`},
	{`item`, `registered`, `What does this collection hold? From the home page to each plan and anchored place.`},
	{`collection`, `registered`, `What is this a member of? From a plan or place to the home page.`},
	{`up`, `registered`, `What is this part of? From an as-built page to its plan.`},
	{`as-built`, `phoropter`, `What changed while this plan was being built? From a plan to its as-built file.`},
	{`anchored-at`, `phoropter`, `Where does this entry point? From an anchor on an entry to the page for its symbol or file, or to another plan’s entry, or to a whole plan.`},
	{`anchored-by`, `phoropter`, `Which plans point here? From a symbol or file page to each entry anchored there, and from an entry to each entry in another plan whose anchor names it.`},
	{`already-here`, `phoropter`, `What has already been said where this draft points? From an anchor on the draft page to each landed entry at the same place: an open guess first (unanswered, or answered not tested), then declines, then decisions, then the rest, newest first within each (poplin). Only the draft page has it.`},
	{`lenses`, `phoropter`, `Where are a reader’s lenses? From every page to <code>lenses/</code>, which lists those in <code>drafts/lenses/</code>, and says how to start one when there are none. Only the dev server adds it (percale).`},
	{`lens-item`, `phoropter`, `What does this lens hold? From a lens’s item to the entry, or the whole plan, its reference names.`},
	{`revisits`, `phoropter`, `Whose ground is this plan standing on? From a plan to the one plan its <code>revisits:</code> names (bodkin).`},
	{`revisited-by`, `phoropter`, `Has this plan’s ground been revisited? From a plan to each plan that revisits it.`},
	{`answers`, `phoropter`, `Which guess does this answer? From an as-built’s <code>Answered:</code> entry to the plan’s <code>Guess:</code> with the same title.`},
	{`answered-by`, `phoropter`, `Has this guess been answered? From a plan’s <code>Guess:</code> to the <code>Answered:</code> entry in its as-built.`},
}

var profileProps = [][2]string{
	{`(header)`, `Every page’s provenance, the item in its <code>&lt;header&gt;</code>: <code>repository</code> and <code>commit</code> (full sha in <code>value</code>).`},
	{`plan`, `A plan: <code>title</code>, <code>name</code> and <code>date</code> (from its filename), <code>added</code> (the commit that added it, which orders plans), <code>cart</code> as written, <code>base</code>, <code>file</code>, <code>intro</code>, <code>asBuilt</code>, <code>revisits</code> and <code>revisitedBy</code> links, and its <code>entry</code> items. On the home page, a plan has <code>entryCount</code> instead of its entries. When a <code>base</code> (or an as-built’s <code>head</code>) names a commit here, it’s a <code>&lt;data&gt;</code> whose value is the full sha. Otherwise its value is the text as written, like “none, a new repo”, or a commit a squash merge replaced; a missing base is “not given”.`},
	{`asBuilt`, `An as-built file: <code>title</code>, <code>plan</code>, <code>head</code>, <code>file</code>, and its <code>entry</code> items.`},
	{`entry`, `One entry: <code>kind</code> (Decision, Decline, Guess; in an as-built, Changed, Chose, Answered), <code>title</code>, <code>anchor</code> items, <code>reason</code>, <code>source</code> (the parenthesis after <em>Because</em> or <em>Answer</em>, exactly as written), <code>verdict</code> (an answer’s opening word: held, disproved, partly disproved, or not tested; on a guess, inside its <code>answeredBy</code> link’s sentence), <code>answers</code> or <code>answeredBy</code> links, and <code>anchoredBy</code> links from other plans’ entries.`},
	{`anchor`, `Where an entry points: <code>where</code> as written, <code>quote</code> if any, <code>new</code> (content <code>true</code>) for <code>where (new)</code>, in the same shape wherever it appears, <code>note</code> when it is shown but not resolved, <code>place</code> (a symbol or file page) or <code>target</code> (another plan’s entry, or a whole plan), and one <code>view</code> per commit. An anchor at an entry has no views unless it has a quote, and then one, at the generated commit: plans and landed as-builts don’t change.`},
	{`view`, `An anchor at one commit: <code>role</code> (base, as-built head, generated), <code>commit</code>, <code>status</code>, <code>file</code>, <code>startLine</code>/<code>endLine</code> for a symbol, <code>matches</code>, one <code>match</code> item per match with the <code>startLine</code> and <code>endLine</code> the quote covers, <code>note</code>, <code>code</code> (each one preceded by <code>codeStart</code>, the line number of its first line; the nth <code>codeStart</code> belongs to the nth <code>code</code>), and <code>sameAs</code> when a generated view matches the as-built head’s exactly and its code isn’t repeated. A plan’s anchors have up to three views: base, as-built head (where the plan’s own work landed), and generated.`},
	{`status`, `<code>found</code>: the quote matched. <code>not-found</code>: it didn’t. <code>no-quote</code>: nothing to search for. <code>no-file</code>: the file isn’t at that commit. <code>no-symbol</code>: the symbol isn’t in that commit’s index. <code>no-index</code>: the commit couldn’t be indexed for the symbol’s package, so whether it’s there isn’t known; the note names the indexer (batik). None of these judges a difference between commits.`},
	{`(draft header)`, `On the draft page, the header says it’s the working tree, not a commit: <code>file</code> (<code>drafts/plan.draft.md</code>), <code>repository</code>, <code>readAt</code>, the draft’s <code>sha256</code>, and the <code>commit</code> its anchors resolve at.`},
	{`draft`, `The draft page’s item: <code>title</code>, <code>base</code>, <code>revisits</code>, <code>notesSha256</code> and <code>annotates</code> (the draft version the notes were written against, when it isn’t this one), <code>intro</code>, <code>entry</code> items as on a plan (but never listed on places or other plans’ entries), <code>prompt</code> items, and <code>annotation</code> items. Each anchor on the draft can have <code>alreadyHere</code> links, to the landed entries at its place (poplin).`},
	{`prompt`, `An open question in a draft, in the annotating agreement’s form: <code>kind</code> (Question, Select, Affirm), <code>text</code>, <code>option</code> for each of a Select’s choices, <code>fallback</code> (what happens if it goes unanswered), and <code>answer</code> when the notes restate it. It follows the entry (or intro) it belongs to.`},
	{`annotation`, `A note from <code>drafts/plan.draft.notes.md</code>, or, on a landed plan’s page as the dev server serves it, from that plan’s notes in <code>drafts/</code> (tarlatan): <code>heading</code> as written, <code>quote</code> (the source text it quotes), and <code>comment</code>. It follows the entry whose source holds its quote; a note without a quote, or whose quote isn’t there anymore, comes first. A built site never has a landed plan’s notes.`},
	{`lens`, `On <code>lenses/</code>: one of a reader’s lenses, from <code>drafts/lenses/&lt;name&gt;.md</code>, with <code>url</code>, <code>title</code>, <code>file</code> and <code>itemCount</code>. On a lens’s own page, the page’s item: <code>title</code>, <code>itemCount</code>, <code>file</code>, <code>repository</code> as written (the first commit’s sha) and <code>repositoryNote</code> when it isn’t this repository’s, <code>intro</code>, <code>lensItem</code> items, and <code>closing</code>, the lens’s own prose after them (lawn). A lens is someone’s way of reading the plans, never testimony; only the dev server serves lenses (percale).`},
	{`lensItem`, `One line of a lens: <code>reference</code> as written (<code>plans/&lt;file&gt;.md</code>, or with <code>› &lt;entry title&gt;</code>), <code>note</code> when it has one (what follows the reference on its line, then the indented lines under it, as markdown), and either an <code>entry</code> link (<code>lens-item</code>) with <code>kind</code> (an entry’s, or Plan for a whole plan), <code>title</code>, <code>plan</code>, and for a guess its <code>status</code> (open, meaning unanswered or answered not tested, or its verdict; for an answer, its verdict), or a <code>problem</code> saying why it didn’t resolve.`},
	{`firstCommit`, `On <code>lenses/</code>: this repository’s root commit, full sha in <code>value</code>, which a lens’s <code>repository</code> names.`},
	{`noteCount`, `On the home page as the dev server serves it, on a plan with notes in <code>drafts/</code>: how many (tarlatan; organza renamed it from <code>requestCount</code>).`},
	{`place`, `A symbol or file that plans anchor to: <code>name</code>, <code>kind</code>, <code>symbol</code> or <code>file</code>, a symbol’s whole definition at the generated commit as <code>code</code> with its <code>codeStart</code>, <code>entryCount</code> on the home page, and <code>anchoredBy</code> items.`},
}

func profile(s *Site) Page {
	here := "profile/"
	var rels, props []string
	for _, r := range profileRels {
		rels = append(rels, fmt.Sprintf(`<dt id="%s"><code>%s</code> <span class="meta">%s</span></dt><dd>%s</dd>`, r[0], r[0], r[1], r[2]))
	}
	for _, p := range profileProps {
		props = append(props, fmt.Sprintf(`<dt id="prop-%s"><code>%s</code></dt><dd>%s</dd>`, p[0], p[0], p[1]))
	}
	return layout(s, here, "Profile", fmt.Sprintf(`
<nav><a href="%s">%s at %s</a></nav>
<h1>Profile</h1>
<p>Phoropter’s pages are its API. A person and an agent read the same HTML; structure is in <code>rel</code> links and microdata (items have no <code>itemtype</code>; property names are defined here). The top-level item in each page’s <code>&lt;main&gt;</code> is the plan, as-built, place, or home listing the page is about.</p>
<h2>Link relations</h2>
<dl class="defs">
%s
</dl>
<h2>Microdata properties</h2>
<dl class="defs">
%s
</dl>`, Rel(here, ""), esc(s.RepoName), esc(s.Generated.Short), strings.Join(rels, "\n"), strings.Join(props, "\n")), "")
}
