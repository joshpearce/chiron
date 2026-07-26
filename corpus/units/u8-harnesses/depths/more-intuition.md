---
unit: u8
depth: more-intuition
---

## It is all one token stream

Picture a single long strip of paper. Every word of the system prompt, every
tool description, everything you typed, everything every tool returned - all of
it written on that one strip, in order, with no gaps and no margins. Some marks
on the strip are special glyphs that mean "a new speaker starts here". Those
glyphs are ink on the same strip as everything else.

The model is handed the whole strip, from the beginning, every single time it is
asked to produce anything. It has no notepad, no marginalia, no memory of having
seen the strip before. It reads the whole thing and writes one more mark at the
end. Then it is handed the strip again, one mark longer, and reads the whole
thing again.

This is why "the model forgot my instruction" is the wrong picture. Nothing was
forgotten, because nothing was ever held. The instruction is still there on the
strip, near the beginning, and the model still reads it. But there are now
ninety thousand words between it and the end of the strip, and the model's
attention is a fixed quantity of ink being spread over all of them. An
instruction near the start of a very long strip is not forgotten. It is
outvoted.

And the reason a web page can hijack your agent: the glyph that means "a new
speaker starts here" cannot be drawn by anyone but the harness - that part is
airtight. But nothing stops a web page from *sounding* like the boss. The model
has learned, from enormous amounts of text, that certain phrasings tend to be
followed by compliance. It learned that from text, not from envelopes. So
bossy-sounding text gets some of the deference that bossy text usually earns,
wherever on the strip it happens to be written.

## The tool loop: the agent is the loop

The thing you should picture is not an assistant with hands. It is a very good
autocomplete that occasionally writes down a note that reads "someone please go
get me the contents of config.ex", and then stops writing.

Stops completely. Puts the pen down. Goes away.

Then you - the harness - read the note, walk to the filing cabinet, fetch the
file, and copy its contents onto the end of the strip in your own handwriting.
Then you hand the strip back and say "continue".

The autocomplete has no idea that time passed. It has no idea whether you
actually went to the filing cabinet or whether you made the contents up on the
spot. It cannot tell, because the only thing it ever sees is the strip, and text
you invented looks exactly like text you fetched. It resumes with no memory of
having asked - it simply reads the strip, which now happens to contain both a
request and a plausible answer, and continues from there.

So the "agent" is not the autocomplete. The agent is you: the one who decides
which notes to honor, which filing cabinets exist, whether to ask permission
before opening the drawer, how many round trips to allow before giving up, and
what to write when the drawer is locked. Change none of the model and all of
those policies and you have a different product.

## What the loop costs

The strip picture makes the cost obvious once you see it. Each round trip, the
model reads the *whole strip* again. Round one it reads four pages. Round two it
reads four and a half. Round six it reads seven. Add up all six readings and it
has read thirty-three pages of material to produce a seven-page strip. The
reading, not the writing, is the bill.

That is what makes long agent sessions expensive, and it is why the expense
grows faster than the session does. Doubling the number of round trips does not
double the reading. It roughly quadruples it, because each of the twice-as-many
readings is also twice as long.

The prefix cache is the fix, and here is the picture for it. Imagine the model
can leave a bookmark: "I have already digested everything up to here, and I kept
my notes." Next round, if the strip up to the bookmark is *character for
character identical*, the notes are still good, and the model can start reading
at the bookmark. Since the strip only ever grows at the end, the bookmark is
always valid, and each page gets read exactly once in its life. Thirty-three
pages of reading collapses to seven.

The fragility is equally visual. Erase one word near the beginning of the strip
and every note past that word is worthless, because those notes were made while
looking at the old word. It does not matter that you only changed one word, and
it does not matter that everything after it is unchanged. The bookmark moves all
the way back to your edit. This is why a live timestamp in your system prompt is
so costly: you are erasing and rewriting a word on page one, every single turn,
and throwing away the notes for the entire rest of the strip each time. And it
is why compaction - which rewrites a large early stretch of the strip - is
followed by one very expensive turn.

## Context is the scarce resource

Two different things run out, and they run out for different reasons.

The first is the reading time, above: a longer strip takes longer to read, and
worse than proportionally.

The second is desk space. Those "notes" the model keeps so it can start at the
bookmark are not free - they are physical, they live in the accelerator's
memory, and they are surprisingly bulky. A rough feel for the scale: roughly one
hundred kilobytes of notes per token of strip. That is a hundred kilobytes to
remember one word. A very long session's notes are tens of gigabytes - most of a
very expensive card, occupied by one conversation. This is the real reason
context limits exist and the real reason an idle session's cache gets thrown
away.

The third cost is the one people miss because it has no invoice. Attention is a
fixed budget spread across everything on the strip - it always sums to one,
whether the strip is short or long. Pasting in fifty thousand tokens of log
output does not buy more attention. It buys more competitors for the attention
that already exists. Junk in the context is not neutral. It is actively bidding
against the parts you need.

Once you hold those three pictures, every context-management technique is
obviously the same technique:

**Compaction** is burning most of the strip and writing a paragraph about what
was on it. The paragraph is shorter, which helps the reading time and the desk
space. But a machine chose what to burn, the choice is unrecoverable from the
strip, and the burning happened near the beginning - so the bookmark just moved
all the way back.

**Sub-agents** are handing a colleague a fresh strip and a task. They read
fifty pages of code onto their strip, figure out the answer, and hand you back a
sticky note. Their strip goes in the bin. Yours grew by a sticky note instead of
fifty pages - and, crucially, you do not carry those fifty pages for the rest of
the day. The cost is that your colleague only knows what you wrote on their
strip when you handed it over, and you only learn what fits on the sticky note.

**Reading a file, searching, retrieval** - all the same move. The library exists,
it is not on your strip, and you copy over only the paragraph you need.

## Putting the three claims together

One strip. One reader who has never seen it before and never will again. One
mark added at the end, then the whole thing handed back.

Everything else in the experience - the sense of a conversation with someone who
remembers you, the sense of a thing that went off and ran your tests - is
assembled by the harness out of that one repeated motion. The conversation is
the harness re-sending the strip. The memory is the strip itself. The tool use is
the harness reading a note, doing the work, and writing the answer down in its
own hand. The agent is the loop that keeps handing the strip back.

