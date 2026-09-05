---
name: chiron
description: Send work to Matt's Chiron bookshelf - a summary or description of a passage, a primer, or a smart book - and read what it wrote. Use when asked to "send to Chiron", "make a primer about", "create a smart book on", or to check the bookshelf.
allowed-tools: Bash, Read, Write
user-invocable: true
---

# Chiron

Chiron is Matt's adaptive textbook: a server that writes primers and
books, read on his iPad. The `chiron` command drives it from this
machine. Everything it makes lands on his bookshelf.

```
chiron shelf                          the bookshelf and where each row stands
chiron capture -scale S -prompt Q [-title T] [-url U] [-app A] [-text T | -file F]
chiron plan ID [-say TEXT]            the planning conversation of a draft
chiron build ID [-brief B | -brief-file F]
chiron discard ID                     drop a draft (only drafts)
chiron status ID                      plan, job, or shelf row as JSON
chiron wait ID [-timeout D]           until ready or failed; prints the row
chiron read ID [-unit U]              the text of a primer or a book unit
chiron teach -title T -brief-file F [-slug S]   a book from a brief alone
```

`chiron` with no arguments prints this usage. Replies are JSON except
`shelf` and `read`. A non-zero exit means it did not happen; the reason
is on stderr.

The server and key come from `~/.config/chiron-dev/config`. The key is
a 1Password reference resolved with `op read` at run time; if a command
sits for more than a few seconds with no output, `op` is waiting for
Matt's approval - tell him rather than retrying. Never copy the key
anywhere.

## The four scales

| scale | what comes back | where |
|---|---|---|
| `summary` | a few paragraphs answering the prompt | printed, in the reply's `answer_md` |
| `description` | a page or so | printed, in `answer_md` |
| `primer` | a short self-contained document | the bookshelf, after a build |
| `book` | a smart book with units and checks | the bookshelf, after a build; takes tens of minutes |

A summary or description is done when the command returns. A primer or
book comes back as a **draft** in `planning`: the reply carries its
`subject` id and the tutor's first question in `reply_md`. The draft is
built when you say so, with a brief.

## Recipe: "create a primer from first principles about FOO"

You write the material and the brief; Chiron writes the primer.

1. **Write the passage** to a scratch file: your own notes on FOO, a
   page or two, the facts and the relationships that a primer should
   build on. This is the source the writer grows from, so put the real
   substance in it, not a table of contents.
2. **Write the brief** to a second file. Say who the reader is (Matt: a
   strong engineer, assume gaps in this particular field), that the
   primer derives everything from first principles at the point of use
   and compresses rather than skips, what question it answers, and what
   to leave out. Ask for any acronym to be expanded on first use.
3. **Capture it**:
   ```
   chiron capture -scale primer -title "FOO" -prompt "<the question the primer answers>" -file notes.md
   ```
   Note the `subject` in the reply.
4. **Build it with the brief**, skipping the planning turns:
   ```
   chiron build <subject> -brief-file brief.md
   chiron wait <subject>
   ```
   `wait` prints the row when the primer is ready, and exits 1 with the
   reason if the write failed.
5. **Check it**: `chiron read <subject>` prints the text. Read it once
   for the things a first-principles primer must not do: skip a step,
   lean on an unexpanded acronym, assume the reader knows the field.
   If it falls short, the primer cannot be rewritten in place; say so,
   and offer to capture again with a sharper brief.
6. Tell Matt the title and that it is on the shelf.

The same recipe with `-scale book` makes a smart book; `wait` follows
the draft to the book it builds and returns when the book is on the
shelf. `chiron teach -title T -brief-file brief.md` makes a book from a
brief with no passage at all.

## Planning instead of a brief

When the ask is vague, or Matt wants to be asked, let the tutor plan:

```
chiron plan <subject>                    the conversation so far
chiron plan <subject> -say "<answer>"    one turn; done=true when the brief is settled
chiron build <subject>                   build with the brief the plan reached
```

A draft can be left in `planning` for as long as needed. It shows on
the iPad's shelf with a hammer, and Matt can pick the conversation up
there.

## Sending a passage as it is

For "send this to Chiron" with text in hand (a file, a page, a quote),
capture it unchanged at the scale asked for, with `-url` and `-app`
naming where it came from if known. A summary or description prints
the answer; relay it in full.

## Care

- The server holds Matt's real learner record. `read` counts as opening
  that unit for him. Do not `discard` a draft you did not make in this
  conversation, and never `POST /reset` or call the API directly for
  anything the command does not offer.
- One build at a time: a book's generation is heavy, and the same
  server is serving the book he is reading.
- Titles become shelf cards; keep them short and specific.
