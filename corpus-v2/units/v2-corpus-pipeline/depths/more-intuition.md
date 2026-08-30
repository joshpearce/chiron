## What the extractor actually hands you

A PDF is not a document. It is a set of drawing instructions: put this glyph
at this coordinate, in this font, at this size. Nothing in the file says
"this is a paragraph" or "this column comes before that one." The visual
document exists only in your head, assembled from spatial cues you were
trained on since childhood - proximity, alignment, whitespace, font weight.

So picture the naive extractor as a person who can see the ink but not the
page. They are handed a stack of index cards, one per glyph, in the order the
printer laid them down, and asked to read the page aloud. They will read
across the physical page because that is the order the cards came in, and
they will produce fluent-sounding nonsense that alternates between two
unrelated topics every twelve words. They are not making a mistake. They were
never given the information that there were two columns.

The good pipeline is a person who first steps back and looks at the page as a
picture: sees two blocks of grey with a white channel between them, decides
that is two columns, and only then starts reading. That step - look at the
shape before reading the words - is the entire difference between the two
outputs in canon.

The ligature loss has a different flavour and it is worth feeling separately.
In printed text `fi` is often a single carved shape, one piece of type, going
back to metal typesetting. The file records that one shape. If the font's
translation table has no entry saying "this shape means the letters f and i,"
the extractor has an ink blob with no name and quietly drops it. Nothing
errors. You simply get `ne-tuned` and `signicant` sprinkled through your
corpus, always in the same words, because those are the words that contain
those ligatures. Your corpus now contains a systematic misspelling of the
domain's most common vocabulary, and the tokenizer will faithfully learn
merges for the misspellings.

## Extraction: three source types, three tools

Think of the three source types as three levels of decay of the same object.

The LaTeX source is the *recipe*. It says: this is a section heading, this is
an equation, this cites that paper. Every structural fact is written down
explicitly because the author had to write it down for the typesetter.

The PDF is the *baked cake*. All that structure went in and came out as
geometry. You can taste the ingredients but the recipe is gone, and
recovering it means reasoning backwards from the finished object.

The scanned PDF is a *photograph of the cake*. Now even the geometry is
approximate and you are inferring letters from pixels.

Every step down that ladder is a step from reading to guessing, and the whole
of the OCR literature is engineering to make the guessing at the bottom rung
as good as the reading at the top. This is why "prefer LaTeX source" is not a
clever trick: it is declining to descend the ladder in the first place. The
best OCR system in the world is trying to reconstruct information that, for a
large fraction of arXiv papers, is sitting in an S3 bucket in its original
form.

The HTML tradeoff between the two extractors is a different picture: a fine
sieve versus a coarse one. The fine sieve holds back the grit but also holds
back some of what you wanted. The coarse sieve lets everything through, grit
included. Neither is right in isolation - it depends on whether you have a
second sieve downstream. DCLM uses the coarse one because it has a very fine
classifier waiting; FineWeb uses the fine one because it does not.

## Quality filtering: the classifier is the lever

The instinct that more data is better comes from a picture where data is
*fuel*. Pour in more, go further. Under that picture a subset can never beat
its superset, because you have simply poured in less.

Replace it with a different picture: data is *curriculum*, and the model has a
fixed number of hours. Every document is a lesson the student sits through.
Adding an hour of a bad lesson does not add a little value; it consumes an
hour that a good lesson could have had, and it teaches something that has to
be unlearned. A student who studies from a thin excellent textbook outperforms
one who reads an enormous mediocre library, and that is not paradoxical at
all.

The 1.3T-beats-15T result stops being surprising the moment the picture
changes. It is not "less is more." It is "the model's capacity is finite and
you decide what fills it."

Now the part that should genuinely surprise you, and it is a fact about where
effort pays off rather than about learning.

Everyone building these systems has a mental hierarchy of prestige.
Architecture is glamorous. Optimizers are glamorous. Data cleaning is what you
make the intern do. The measured result inverts this completely: the largest
single factor in the leading data benchmark was the filtering classifier -
larger than any architecture change tested. And the classifier itself is
unremarkable. FineWeb-Edu's is a small regression head sitting on frozen
embeddings, trained on labels a big model wrote about educational value. There
is no idea in it.

The expensive part is not the machine learning. It is having a large model
read half a million documents and write down an opinion about each one. The
lever is *taste, applied at scale, and then cheaply imitated*. That is a
sentence about labour, not about mathematics, and it is why this unit belongs
to the systems engineer.

## Deduplication, worked by hand

Forget hashing for a moment. Here is why MinHash works, as a game.

Two people each have a bag of marbles. Some marbles are in both bags, some
only in one. You want to know what fraction of all the distinct marbles are in
both bags, and you are not allowed to look at the bags.

The trick: agree on a random ordering of every marble that could exist - a
shuffled deck listing every possible marble. Each person looks through their
own bag, finds the marble that comes earliest in the agreed ordering, and
tells you its name. Nothing else.

If they say the same name, that marble is in both bags. If they say different
names, it is not.

Now: how often do they say the same name? The marble they report is whichever
of the shared pool comes first in the ordering, and since the ordering is
random, any marble in the combined pool is equally likely to be the earliest
one. They agree exactly when that earliest marble happens to be one of the
shared ones. So the chance they agree *is* the fraction of the combined pool
that is shared. Which is the number you wanted.

That is the entire mechanism. One name each, and you get an unbiased estimate
of overlap. Play the game a hundred times with a hundred different orderings
and you get a hundred yes/no answers, and the fraction of yeses is your
estimate. The hash function is just a way of agreeing on a random ordering
without writing it down.

The signature is a hundred names. It is the same size whether the bag holds
ten marbles or ten million, which is the property that makes this scale.

**The banding trick** is a sieve with several meshes stacked. You do not
compare a pair unless some mesh catches them both in the same hole. Requiring
more positions to agree within a band makes each mesh finer; using more bands
gives more chances to be caught. Fourteen bands of eight is: eight consecutive
names must match exactly for one catch, and you get fourteen tries. Very
similar documents get caught almost certainly; documents sharing half their
content almost never do. The transition between those two regimes is sharp,
and that sharpness is what you are buying.

**Why global dedup lost.** Picture a river carrying both driftwood and
plastic. The driftwood is heavy and keeps washing up in the same places month
after month; the plastic blows through once and is gone. If you count each
distinct object exactly once, you have systematically thrown away the
information that the driftwood kept coming back - and coming back was the
signal that it was solid. Per-snapshot dedup keeps that repetition. Global
dedup flattens it, and what it flattens is a quality signal that nobody put
there on purpose.

## Dedup is a payout decision

Two archives both hold a copy of the same photograph. A magazine wants to
print it and pays a licensing fee. Somebody has to decide who receives it.

That is the situation, and the uncomfortable part is that in a normal
pipeline, nobody decides. The decision is made by whichever archive's
catalogue the software iterated over first. Not by a policy, not by a person,
not by the publication dates. By file order.

You would never accept this arrangement in a payments system. You have spent a
career making sure that money movements are explicit, logged, and idempotent.
And yet the standard dedup pass is a payment routing decision implemented as a
`seen` set.

Here is the reframe that makes the fix obvious. In every other pipeline you
have written, duplicate data has no owner. Two identical rows in a warehouse
table are not owed anything by anyone. Your instinct that dedup is neutral is
not wrong - it is correct in every context where you formed it. What changed
is not the algorithm. It is that a claimant appeared.

And the fix is not a better algorithm. It is a receipt. Keep training on one
copy, exactly as before - the model does not change and does not care. But
write down what you merged and who held each copy. The pipeline stays
identical; you have added a ledger beside it.

Notice what that separation buys. The corpus is frozen the moment you build
it, and the model trained on it is frozen too. But the payout policy is now a
function of a small table, so you can change it, argue about it, run it three
ways and show a rightsholder the difference, all without touching a GPU. The
irreversible thing and the negotiable thing have been pulled apart. That is
the design move, and it costs one table.

## Decontamination and PII

Decontamination is the same discipline as not letting your test set leak into
your training set, applied at a scale where you cannot inspect either one. The
picture is a proctor checking whether the exam questions appeared in the
textbook. You cannot read the textbook - it is a trillion words - so you check
mechanically: does any thirteen-word run from the exam appear anywhere in it?

Thirteen words is the right length for a reason you can feel. Three words
match by accident constantly ("on the other"). Thirteen consecutive words
matching is not an accident; someone copied something. The threshold is
chosen at the point where coincidence dies.

The PII picture is where intuition fails, and it fails in a specific way.

Imagine redacting a document with a marker. You black out the email addresses,
the phone numbers, the account numbers. The page now looks anonymized -
visibly so, which is part of the problem, because the redaction is the evidence
of care. And then the remaining text says: *the third-year PhD student in the
Northlake speech group who ran the 2019 pilot*.

Nothing was blacked out because nothing looked like an identifier. Every word
is an ordinary word. But there is exactly one person who fits, and everyone in
that field knows who. Identification is not a property of any single field; it
is a property of the *combination*, and combinations multiply. Institution
times subfield times year times career stage runs out of distinct people very
quickly, the way birthdays run out of distinct dates in a room of twenty-three
people much sooner than anyone expects.

A pattern matcher looks at fields one at a time. It cannot see combinations by
construction. So the honest position is not "we scrubbed the PII" - it is "we
removed formatted identifiers, and this corpus is public scholarly writing
whose authors are named on purpose." The second sentence is defensible under
questioning. The first is not.

## Training the tokenizer

BPE is a game of dominoes. Lay the text out as single letters. Look for the
pair of adjacent letters that appears most often anywhere in the pile, and
glue every occurrence of that pair into one new piece. Now look again - the
new pieces count too - and glue the most common pair again. Keep going until
you have as many distinct pieces as you decided to allow.

That is all of it. Nothing knows about words, syllables, or meaning. Frequency
does everything.

What comes out looks uncannily like linguistic structure - the algorithm
"discovers" prefixes and suffixes - but it discovered them the way water
discovers the shape of a valley. Those sequences recurred, so they got glued.
Run it on a corpus of papers and it will glue together `\begin{`, `et al.`,
`arXiv`, and the words of your field, because in *that* pile those are the
common dominoes. Run it on web text and it glues `http`, `Click`, and
`Copyright`. The tokenizer is a fingerprint of its training corpus, which is
exactly why you train your own.

**Why a bigger vocabulary stops helping.** Picture the vocabulary as shelf
space in a workshop. The first few shelves hold your most-used tools, and they
save you an enormous amount of walking. The thousandth shelf holds something
you reach for once a month. The hundred-thousandth shelf holds a tool you may
never touch.

Two things go wrong at the far end. The shelves cost the same to build
regardless of how often you use them - every vocabulary entry gets a full row
of parameters, the popular ones and the forgotten ones alike. And a tool you
never pick up never gets sharpened: an embedding row for a token that appears
a handful of times in your corpus finishes training almost exactly where
random initialization put it. It is not a learned representation. It is noise
that the model has been told to treat as a symbol.

So the useful picture is: compression gain flattens out like a curve
approaching a ceiling, while cost keeps climbing in a straight line. Somewhere
those cross. For a small model on a small corpus they cross early, around
sixteen to thirty-two thousand entries, and the frontier-scale numbers you
have seen come from a regime where the crossing point is far to the right
because there is vastly more compute and vastly more text.

## Mixing and packing

**Mixing** is deciding the diet. How much web, how much code, how much
mathematics. The state of the art is largely people with good judgement trying
combinations and measuring, which should be reassuring rather than
embarrassing: it means there is no secret you are missing.

The one thing to notice - and it is a gift - is the shape of the RegMix
method. Train a hundred tiny models, each on a random blend of the sources.
Record how well each one turned out. Then look across all hundred and ask:
when there was more of source A in the blend, did the result get better?

That question is the same question as "what is source A worth." The mixing
people are asking it to choose proportions. You will ask it to decide payment.
The experiment is identical; only the reason for running it differs. Keep this
in your pocket - v4 spends a whole unit on it and it will feel like a return
rather than an introduction.

**Packing** is seating a coach. The bus has a fixed number of seats per row
and you are boarding families of different sizes. To avoid empty seats you let
one family spill into the next row, so a row often contains the end of one
family and the start of another.

Now, the model reads a row left to right, and each passenger can see everyone
seated before them in the row. Without a partition, a passenger from the
second family is looking at, and being influenced by, the first family's
conversation. They are strangers who happen to have boarded consecutively.

The separator token is a polite gap in the seating. It is not a wall. Attention
looks straight past it.

For plain language modelling this is a small cost that people have paid for
years without much complaint. For this book it is fatal, and the reason is
worth stating slowly. Everything later in the book asks "what did *this piece
of training data* do to the model." The unit the question gets asked about is
the row - the packed sequence. If a row holds two families, then the answer
you get is about the pair, and there is no way to divide it afterwards,
because the two influences were mixed together inside the model before you
ever saw a number.

Intra-document masking installs the actual wall. Each family sees only its own
members. The bus is just as full, nothing is wasted, and every row is now
attributable to one source.

## The provenance manifest

The manifest is a library call number, and the shard is a shelf.

`shard 3, offset 44,100, length 9,800` means: third shelf, start 44,100 books
along, take the next 9,800. That is enough to find anything in constant time
and it is enough to say what you found. Every question the rest of this book
asks reduces to that lookup: which source produced this token, which spans
belong to this source, what was in this batch.

The one design decision that repays explanation is why documents never
straddle a shelf. It costs empty space at the end of every shelf - real waste,
several percent. What it buys is that a document is *one interval*. Not two
intervals with a join. Not a special case in every query. One row in a table,
one range check, no branch. Every downstream stage in the book gets simpler
because of a rule enforced once here, and that trade - pay a fixed storage
cost to eliminate a category of special case - is one you have made a hundred
times in other systems.

**Determinism, and the part that is not obvious.** Pinning the shuffle seed is
routine. The surprising requirement is pinning the arithmetic.

Adding floating point numbers is not associative. Add a large number and two
tiny ones, and whether the tiny ones survive depends on whether you added them
to each other first. So when gradients from eight GPUs are summed, the order
of the summation changes the answer in the last few bits. Run the same code on
four GPUs instead and you get different last bits.

Normally you shrug at this. Here you cannot, because training amplifies tiny
differences the way weather does: two runs that differ in the last bit of one
gradient do not stay close. They end up at genuinely different models with
about the same loss.

Now consider what you are trying to measure later: "removing this source
changed the model by this much." That is a difference between two training
runs. If the two runs also differed in the order the hardware happened to add
things up, part of your measured difference is arithmetic, not data. You
cannot tell which part. Pinning reproducibility does not make attribution
work; it removes one of the two things that could be producing your number,
which is the only way to be left with the one you care about.
