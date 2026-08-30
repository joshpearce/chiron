## The screen

Picture a courtroom exhibit table with four objects laid out on it, in a row,
each with a card in front of it.

The first object is a photocopy of a page with four sentences highlighted in
yellow, and the card says: *these words appear in this document.* Anyone can
check it in ten seconds by reading the page. It is the most persuasive object on
the table and it is making the smallest claim.

The second object is a ranked list of twelve names, and the card says: *these
people influenced the outcome, in this order, and when we checked our ranking
method against a case where we knew the true order, it agreed about six times in
ten.* That card is unusual because it grades itself. Notice that the card is
what makes the list admissible; without it the list is one person's opinion in
table form.

The third object is a pie chart, printed in grey, with a card saying: *if this
pool were divided by a rule that four stated fairness principles uniquely
determine, the slices would be these sizes - and the slices are not far enough
apart to distinguish given how much they wobble when we redo the measurement.*
It is the least visually convincing object and it carries the most reasoning.

The fourth object is a sealed envelope with a date stamp on it, opened in front
of the room, containing a prediction that turned out to be right. The card says
one word: *proven.* The envelope's power comes entirely from the date stamp. If
it had been sealed after the fact, the identical contents would be worthless.

The reason all four sit on the same table is that a room shown any one of them
alone will assume it is the others. Shown only the highlighted photocopy,
people conclude influence. Shown only the ranked list, people conclude proof.
The layout is the argument.

## The retrieval panel, built honestly

A suffix array is the index at the back of a book, if the index had an entry for
every possible phrase rather than for the words an editor chose.

Think of the whole corpus as one enormously long ribbon of text. Now imagine
cutting a copy of that ribbon at every single position, so you have as many
strips as there are characters, each strip running from its cut point to the end.
Sort all the strips alphabetically and stack them. That is the suffix array.

The payoff is immediate and physical. Every strip beginning with "document-masked
packing" is now sitting in one contiguous stack, because alphabetical sorting put
them together. To count occurrences of a phrase you find the top and bottom of
that stack and subtract two positions. To find them you do what you do with a
sorted stack: open it in the middle, see whether you are too high or too low,
halve again. Twenty or thirty halvings gets you anywhere in a billion-token
corpus.

Nothing about that structure knows anything about a model. It is a filing system
over text. Which is exactly the point: when a panel built on it tells you that
these four documents contain these words, it is reporting a property of the
filing cabinet.

**Why rarity and not length.** Two matches. One is twenty words of "the results
were analyzed using standard statistical methods," which appears in three
thousand papers. The other is nine words that appear in exactly one. The long
match is longer and tells you nothing, because it is the sort of sentence anyone
writes. The short one is a fingerprint. Ranking by length would put the
boilerplate on top and bury the fingerprint, so the panel would consist mostly of
the phrases every paper shares.

And then the sting: fingerprints are cheap to manufacture. If you pay people for
producing text that nobody else wrote, you have made "nobody else wrote it" the
product. Somebody will write a great deal of text nobody else wrote. It does not
need to be read, or true, or good - it needs to sit in an index and occasionally
turn up in a match.

**Why verbatim matching is a floor, not a test.** Imagine an apprentice who
studied under a master for years and now works in the master's manner - the same
instincts about proportion, the same way of handling a difficult joint - but has
never copied a single one of the master's pieces. Search their work for
identical components and you find nothing. Conclude from that search that the
master had no influence and you have made exactly the error the verbatim panel
invites. What travelled was not the components. It was the way of working, and
the search you ran was a search for components.

The uncomfortable part is that this gets worse as the student gets better. A
beginner traces. A journeyman quotes. A master absorbs and produces something
that shares no surface with anything they learned from. Larger models are
documented to behave the same way: their influence becomes more abstract and
less token-shaped. So the panel is blind to precisely the contribution that
matters most, in precisely the regime where the money is.

**The legibility trap.** Imagine a museum room with three exhibits. One is a
skeleton, mounted, lit, with the visitor able to walk around it. One is a chart
of measurements. One is a paragraph of careful reasoning about what the
measurements do and do not establish. Every visitor leaves the room talking
about the skeleton, and every visitor describes it as proving whatever the room
was about.

The curator's fix is not a longer label under the skeleton. It is to put the
chart *inside* the skeleton's case, at the same size, where the eye cannot avoid
it - and to arrange the room so the visitor's path crosses the reasoning before
they reach the bones. Your fix is structurally identical: the ground-truth
disagreement goes inside the influence panel at full size, the allocation goes
grey when its noise says so, and no panel renders without its error bar. Honesty
becomes a property of the room's construction, which is the only kind that
survives the visitor.

## Assembling the pipeline

Think of the whole build as a relay, where each runner must physically hand over
a baton and cannot start until they are holding one.

The corpus pipeline hands over a *ledger* - a list saying which text came from
whom, which copies were duplicates of which, and where every piece sits in the
shards. Nothing downstream can be done without it, and no amount of later work
recreates it, because the questions it answers are about decisions that were
already made.

The training run hands over a *model and a wobble*. The wobble is the
interesting half. Train the same thing twice with different random starts and
you get two slightly different models, and how different they are is the ruler
everything downstream is measured against. Without that ruler every later
difference is a number with no scale, like being told two objects differ by
"three" without being told three of what.

The counterfactual sweep hands over a *truth*, obtained the expensive way: build
it, break it, see what changed. Everything cheap later gets graded against this,
and it exists only because the models are small enough that breaking and
rebuilding them is an evening rather than a research program.

The split hands over a *rule and a verdict* - here is the share each source
gets, here is how much the shares wobble, and here is whether the shares are far
enough apart to be worth distinguishing.

The memorization work hands over a *wall*, which is a strange kind of baton: a
demonstration that a whole family of methods cannot do what people want it to
do. It is what makes the next runner turn and go a different way.

The notary hands over a *sealed envelope with a date on it*, and the date is the
whole object.

The gradient tracer hands over a *fast estimate and its report card*.

And the last runner assembles what the others produced. Notice the shape: no
runner can go before their predecessor, and two of the handovers are one-way
doors. You cannot seal an envelope retroactively, and you cannot un-collapse
duplicates you already collapsed without recording who was in them. Those two
are the ones that will end a project quietly, months later, when someone asks a
question the files cannot answer.

## What the demo claims and what it does not

**Where the wobble lives.** Picture four instruments on a bench.

The first is a ruler. Given the same object it gives the same reading every
time - it has no wobble of its own. It can only be wrong about *what you pointed
it at*, which is a real limitation and a different kind.

The second is a scale on a slightly unsteady table, being used to weigh
something against a reference weight that is itself known only approximately.
Two sources of uncertainty stacked: the reading moves, and the thing you are
comparing to also moves. When you report how well the scale agrees with the
reference, some of the disagreement is the scale and some of it is the
reference, and the honest report says so.

That second point deserves dwelling on, because it explains a phenomenon that
otherwise looks like cheating. If you measure your reference more carefully -
more repeats, tighter bounds - your scale will *appear* to agree with it better,
without anything about the scale having changed. So a stated agreement figure
means nothing without knowing how carefully the reference was pinned down.
That is why the correlation on the screen always travels with the number of
repeats behind it.

The third instrument is a set of balance beams that share components. Their
readings move together, which sounds bad but is actually helpful: averaging
across many arrangements averages the wobble down too, so the shares end up
steadier than the raw differences underneath them. Steadier, not steady. The
question the grey pie answers is whether the slices are further apart than the
wobble, and here they are not.

The fourth instrument is a thermometer that was calibrated in ice water this
morning, in front of witnesses, before anyone brought it near the thing being
measured. Its error is not smaller than the others'. Its error is *known*,
because the calibration was performed on purpose in advance rather than
estimated afterwards from the reading itself.

**The scale gap, and why saying it out loud helps.** There is a version of this
conversation that everyone in the room has had before. A founder is asked about
a limitation, and they minimize it, and everyone notices, and the rest of the
meeting is about whether the other limitations were minimized too.

There is a different version. The founder states the limitation before it is
asked, in specific terms, with the boundary drawn where a listener could go and
check it. What happens next is not that the listener stops worrying about the
limitation. It is that they stop auditing everything else, because they have
just been given a calibration reading on the speaker.

That is the entire mechanism by which "this is a small-model existence proof
plus a measurement methodology" beats "our attribution is accurate." The first
sentence gives the listener a way to check you. The second gives them a reason
to start.

## The economics of the pitch

**The map, as a picture of where water flows.** Imagine the money in this
industry as water running downhill, and ask where the streams actually are.

There is a wide, fast river: labs paying for expert human work - people writing
and grading and demonstrating. It is far larger than anything else in the
picture and it has nothing to do with published articles.

There is a modest, well-photographed stream: bilateral content deals, negotiated
one at a time between parties with leverage, paid as a lump. It gets nearly all
the press and it is a fraction of the river.

There is a toll booth on the road above both: the gatekeeper metering crawler
access, owning the gate, the identity papers, the price list and the till.

There are several small channels: answer engines paying for placement in their
own results.

And there is a dry riverbed, carefully engineered, with signposts and a
standards body and fifteen hundred landowners along it, and no water in it at
all. That is the training-data marketplace. Somebody built a beautiful
irrigation system and nobody upstream has agreed to open a valve.

A product that only earns money when water flows down the dry bed is a bet on
the valve. A product that meters, verifies or certifies wherever water already
runs is a business.

**Why pro-rata misallocates, without arithmetic.** Three cooks. Two of them make
the same dish, well. The third makes a sauce that is unremarkable on its own and
transforms either of the other two dishes into something people talk about.

Pay by what each dish sells for alone and the sauce cook gets the smallest
share, because the sauce, alone, is not much. But remove the sauce and the
restaurant's takings fall further than removing either main dish would, because
each main dish only sells as well as it does *with the sauce on it*. The value
lives in the combination, and a rule that only ever tastes things separately
cannot find it.

The two main-dish cooks have the opposite problem in reverse: they substitute for
each other, so each one's real marginal value is less than their solo dish
suggests. A payment rule reading solo performance overpays substitutes and
underpays complements, every time, structurally. In a corpus assembled from
complementary specialist sources, almost everything is the sauce.

There is a second problem, which is what the pool does to the payer.
Pro-rata pooling means your payment is divided according to *everyone else's*
usage, not yours. Spend your entire month reading one obscure journal and your
money goes mostly to whatever everyone else read. That is not an implementation
bug; it is what dividing a pool by platform-wide share means. And the moment
payment depends on share of a pool rather than on what any individual did,
manufacturing volume becomes the dominant strategy - which in this market costs
almost nothing, because plausible text needs no audience.

**The threshold, physically.** Imagine trying to sort a bag of apparently
identical ball bearings by weight, on a scale that jitters. If the bearings
genuinely differ by less than the jitter, no amount of care in reading the dial
sorts them. A better dial does not help; the dial was never the problem. Either
get bearings that actually differ, or take more readings of each and average
until the jitter shrinks below the differences.

And crucially: while the jitter exceeds the differences, the *right* thing to do
is charge everyone the same price, and it is right in a strong sense rather than
a resigned one. Pretending to sort them means charging people different prices
for a difference you cannot demonstrate, which is the thing a regulator, an
auditor or a counterparty's lawyer will eventually ask you to demonstrate.

**Why the negative result is the asset.** Everyone selling in this space has a
confident number. Nobody has a number with a stated uncertainty, because
producing one requires the counterfactual experiment and at large scale the
experiment is not affordable. So the comparison in the room is not between your
modest measured number and someone else's better measured number. It is between
a measurement and an assertion. Bringing the only measurement to a room full of
assertions is a strong position even when the measurement says "not yet."

## The strongest case against the whole venture

Take the objection seriously by picturing it as a weather forecast you cannot
influence.

Three kinds of weather. In the first, the courts hold their current line and the
labs simply buy clean copies of what they need. In the second, one of the
unresolved theories lands and ongoing payment becomes a real obligation. In the
third, nothing much changes and the mandatory disclosure regime grinds on,
coarse and unchecked.

Now ask what you would want to be holding under each.

If you built the marketplace, you are holding an umbrella that is useful in
exactly one of the three, and you cannot make it rain.

If you built the measurement layer, you are holding something useful in all
three. Clean-copy buying creates a need to establish what was bought. Ongoing
payment creates a need for an allocation somebody can audit. And the status quo
leaves a legally required document that the regulator has said in writing it will
not check - which is a mandatory annual claim with nobody verifying it, in a
market with real fines attached.

This is not optimism. It is the difference between a bet that pays in one
outcome and a position that pays in all of them. And the second one is the
easier thing to say to a skeptic, because agreeing with every fact they raised
costs you nothing.

The sharpest form of the objection is different: the gatekeeper already owns
everything else in this chain, so why not this too. The answer is the one
structural reason independence has value anywhere. An auditor employed by the
party being audited is not an auditor, and everyone in the room already knows
that from every other industry they have worked in. The gatekeeper's vertical
integration is what makes verification uniquely hard for them to occupy, which
is the mirror image of why it is hard for them to occupy anything else.

## What to build next

Four roads out, and the useful picture is that they are sorted by how much
mathematics they demand - and that the sorting runs almost exactly backwards to
how well they work.

The road that needs nothing but data structures leads to the verbatim panel:
honest, exact, and answering the wrong question.

The road that needs one chapter of introductory statistics leads to the notary:
the only place in this whole landscape where the word "proof" is available.

The road that needs combinatorics and no calculus at all leads to the fair
split: the only method whose output comes with an argument a non-technical
person can agree to.

The road that needs gradients leads to the cheap tracer: fast, scalable, and
requiring a report card because on its own nobody should believe it.

And the road that needs second-order calculus, Hessians, and eigendecompositions
of enormous matrices leads to influence functions, which four independent groups
have found unreliable on language models, and whose classical derivation assumes
three things about the loss surface that a language model does not satisfy.

That inversion is not a coincidence and it is worth carrying as a general
instinct. The methods that are sound here are sound because their reasoning is
simple enough to check and their comparison group is constructed rather than
assumed. Mathematical sophistication in this space has mostly been the price of
avoiding an experiment somebody could not afford to run - and you can afford to
run it, which is the whole reason the small scale was chosen.

## At the bench: the demo and the memo

Two documents, and the useful mental image is that one of them is a machine and
the other is a testimony.

The screen is a machine: it takes files as input and refuses to run if a file is
missing. That refusal is the design. If the correlation is absent the influence
panel does not render a ranking with a blank space where the number should be -
it does not render. If the noise figure is absent the pie does not appear in
colour with an empty caption. Build it so that the only way to produce a
confident-looking screen is to have actually produced the confidence, because
six months from now, tired, the evening before a meeting, you will want to
demo something that is not quite finished, and the machine should be the thing
that says no.

The memo is a testimony, and testimony has an order: the thing you measured,
the attack you ran against yourself, the thing you proved, and the map of where
you sit. Notice that the second block is the unusual one. Volunteering the
shell-company attack - split one source into three, watch its share inflate,
apply the fix, watch it return - is showing the room a way to cheat your own
system before anyone asks. It reads as confidence rather than as weakness for
the same reason a surgeon describing complication rates reads as competent: only
someone who has looked knows the number.

## What you can now do

The whole unit reduces to one habit, and it is a habit rather than a technique.

When somebody shows you a number about training data, ask what would have to
happen for that number to change. Not "how was it computed" - the answer to that
is a method name and method names are marketing. What would move it.

If rebuilding a search index would move it, it is a statement about text
containment.

If retraining the model would move it, it is a statement about the model.

If changing what you are optimizing for would move it, it is a statement about a
rule you chose.

And if nothing would move it, because it was fixed and sealed before the thing
being tested existed, then it is the only kind of statement in this field that
gets to use the word proof.

Four questions, ten seconds, and you can classify any product in this market
including your own. The reason the book took nine units to get here is not that
the classification is hard. It is that being able to *produce* an artifact that
survives all four questions took every one of the files along the way.
