## Two transcripts, one model

Picture a lawn between two buildings, and a paved path that goes the long way
around it.

Most people take the path. Some cut across the grass. If two people cut across
in a year, the grass does not care - it grows back, and by spring there is no
evidence anyone walked there. If ten thousand people cut across, a bare brown
line appears, and it is so clear that a stranger arriving at midnight can follow
it without a torch.

Now ask the question this unit is about. You want to know whether a particular
person crossed the lawn. You look for the worn line. If they were one of the ten
thousand, the line is there - but the line does not belong to them; it belongs
to the ten thousand. If they were one of the two, there is nothing to see, and
the absence of a mark is not the absence of a walk.

That is both transcripts. The novel is the worn line: it came back near-verbatim
because thousands of copies of it walked through the training corpus, in
reviews, excerpts, study guides, and pirated scans. The workshop paper is the
two crossings: it was there, it left no visible trace, and the lawn is telling
you nothing about it.

The thing to feel, before any of the machinery, is that these two facts are not
in tension. A model that reproduces a famous book word for word and produces
nothing recognizable from 99.9% of its corpus is behaving exactly as expected.
Anyone arguing from one of those observations while ignoring the other is
telling you half of a story, and both halves are load-bearing for opposite
conclusions.

## What memorization is, and what it is not

Three different things get called memorization, and the clearest way to separate
them is to think about a witness.

**The witness who can recite it.** You ask them to repeat a phone number and
they do, digit for digit. This is verbatim continuation. It is the most
convincing form of testimony and the easiest to check, and it is also the
easiest to defeat: ask them to describe the number's owner instead and their
knowledge is intact while the recitation is gone. Blocking the recitation is not
removing the knowledge, and this is why a filter that catches exact quotes
catches only the form of what is retained, never the retention.

**The witness who could recite it if you asked enough times.** Ask them once and
they fumble. Ask them a hundred times in slightly different ways and forty of
those times the number comes out complete. Are they a witness or not? Under the
first standard, no; they failed. Under an honest standard, they are a witness
forty percent of the time, and reporting that number is more useful than
reporting a yes or a no, because it is comparable between witnesses and it does
not depend on which single question you happened to ask first.

**The witness who knows it because of this document, rather than in general.**
Here is the one that matters and the one nobody can afford to measure. A witness
who recites the standard closing formula of a legal letter is not testifying
about your letter; they have seen ten thousand letters. To know whether *your*
letter is why they can recite it, you would have to run their whole life again
without your letter in it and see whether they still could. That comparison -
with the document, without the document - is the only version that means what
people think "memorized" means, and it is why the causal question is real and
almost never answerable.

The practical residue is a single question you can ask about any claimed match,
and it survives every technicality: **could someone who never saw this document
have produced this?** An audit tool reporting four thousand matches on a
journal's copyright footer has answered yes four thousand times without
noticing. That is not a strict standard or a lenient one; it is the difference
between evidence and coincidence, and it applies before any statistics.

## The arithmetic of what can be stored

Think about a filing cabinet.

You have a warehouse of documents and one cabinet. The cabinet has a fixed
number of drawers and each drawer holds a fixed amount. If the warehouse
contains fifteen times what the cabinet can hold, then no matter how clever your
filing system, the cabinet cannot contain the warehouse. Two different
warehouses will end up producing identical cabinets, and once that happens
nobody can look at the cabinet and say which warehouse it came from. This is not
a claim about filing technique. It is a claim about counting, and it is the one
argument in this whole subject that cannot be improved on by anyone.

So the "the weights are a compressed archive of the internet" position is dead
on arrival, and the arithmetic that kills it is a division you can do on a
napkin.

But now look at what the counting argument does *not* say, because this is where
people run off the road in the other direction. It says the cabinet cannot hold
the warehouse. It says nothing about how the space in the cabinet gets used. If
one document from the warehouse arrived four thousand times and everything else
arrived once, a sensible filing clerk would keep that document and discard the
rest - and the cabinet would contain one complete document and no trace of the
other million. The aggregate bound is satisfied. The specific document is fully
recoverable. Both facts hold at once, and any argument that treats the counting
bound as proof that nothing is stored has confused a ceiling with a floor.

Two more pictures for the dials.

**Why each doubling buys the same increment.** Think of learning a phone number
by hearing it in passing. Hearing it twice instead of once helps a great deal.
Hearing it a thousand and one times instead of a thousand helps not at all. The
increments come from ratios, not from counts, which is why the honest way to
plot any of this is with a logarithmic axis, and why "repeated a hundred times"
means nothing without knowing what it is a hundred out of.

**Why relative frequency is what matters.** A drop of dye in a glass of water is
visible. The same drop in a swimming pool is not. Nothing about the drop
changed. If you want the pool to look the way the glass looked, you need enough
dye to match the ratio, and the pool is somebody else's and it keeps getting
bigger. That is the entire design problem for anyone planting evidence: the
denominator is set by a party who is not consulting you, and it grows every
year.

And the sting from the previous unit: the dial that makes a small corpus support
a real training run - reading it several times over - is the same dial that
makes the corpus recoverable from the finished model. You do not get to turn one
without turning the other. Whether that is a hazard or a feature depends only on
which side of the table you sit on.

## Extraction as evidence

Fingerprints.

A fingerprint at a scene is the strongest thing an investigator can have. Nobody
needs to accept a statistical model to understand it. It is not an inference
that something happened; it is a physical trace of it having happened, and the
argument is over as soon as it is shown.

Now hold the other half. Almost nobody leaves fingerprints. Most surfaces do not
take them, most contact does not transfer them, and most people who were in the
room left nothing at all. So the absence of a print is not evidence of absence -
it is the ordinary condition, the thing you would expect whether or not the
person was there. An investigator who concluded from a clean surface that nobody
had been in the room would be laughed out of the building, and yet that is the
inference people make constantly about models: no extraction, therefore no
training.

Two more things the picture gets right.

**The refusal is not a clean surface.** When a model declines to continue your
text, that is not "no fingerprint found." That is a guard at the door telling you
he would rather you did not look. The trace may be sitting there untouched
behind him. Investigators who accept the guard's word file bad reports, and the
first person to walk around the guard produces the transcript that destroys
them.

**Why you budget for it anyway.** A fingerprint is cheap to look for and
occasionally ends the case. That is a fantastic ratio, and it is why extraction
belongs in every engagement even though you expect it to come up empty. What it
cannot be is the product. You cannot open a business that sells "we will find a
fingerprint," because for the overwhelming majority of clients there is nothing
to find, and telling them that after taking the fee is a business that lasts one
cycle.

## Why membership inference cannot prove training

Here is the picture, and it is a picture about what a control group is for.

A drug trial without a placebo arm is not a weak trial. It is not a trial. You
give a hundred people a pill, seventy get better, and you have learned nothing,
because you do not know how many would have got better anyway. The seventy is
not a small effect or a noisy effect; it is an uninterpretable number, and no
amount of care in measuring who got better repairs it. The missing thing is not
precision. It is the comparison.

Membership inference has no placebo arm and cannot have one. The comparison it
needs is: this same model, trained the same way, on the same corpus, without my
document. That model does not exist. Building it means running the whole trial
again - the corpus, the recipe, the machines, several times over because the
result wobbles from run to run - and nobody outside the lab can do that, and the
lab has no reason to.

So people substitute. Instead of a model that did not see the document, they use
documents the model did not see. That is not the same substitution it appears to
be, and here is the picture for why.

Suppose you want to know whether a bakery has been using a particular supplier's
flour. You cannot compare this bakery to an identical bakery that never bought
from them - there is no such bakery. So instead you compare loaves. Loaves made
with the supplier's flour against loaves made with other flour. Reasonable. But
you gather your comparison loaves from a different city, a different month, and
a different season of wheat, and now every difference you find might be the
flour, or might be the water, the humidity, the oven, the month. You have a
difference. You cannot say what it is a difference in.

That is exactly what happened, and the way it was caught is the most satisfying
result in this literature. Somebody built a tester that never tasted the bread -
it only looked at the date on the wrapper - and it did better than every tester
that actually tasted. If a taster who cannot taste beats every taster who can,
then the thing being detected was never in the taste. It was on the wrapper.

Three more things, each independently fatal, and all three easier to feel than
to derive.

**The verdict is a coin.** Run the whole trial again with nothing changed but
the shuffle of the deck, and individual documents flip from flagged to
unflagged. A verdict that changes when nothing about the world changed is not a
verdict about the world.

**The signal is thinnest exactly where the money is.** The regime where any
membership signal survives is one where the model saw relatively little text per
unit of its own size. Frontier models sit far past that, and they move further
past it every year, because that is where the returns are. The technique is
getting worse against the targets people care about, on a trend, for reasons
nobody can reverse.

**It is cheap to defeat.** An operator who does not want to be audited can
retrain briefly on reworded versions of their own data. Nothing is lost - the
model knows everything it knew - and every one of these attacks stops working.
An audit that an adversary can switch off for pocket change is not an audit.

None of that means statistics cannot help. It means this particular question,
asked after the fact, has no control group, and the answer is to change the
question or to arrange for a control group before the fact. Both of those are
coming.

## Dataset inference: the collection as the unit of evidence

A single grainy photograph of a dark room shows nothing. Stack a thousand
photographs of the same room and average them, and a shape appears.

That is the whole idea, and it is worth understanding why it works rather than
treating it as a trick. Each photograph has the same faint shape in it plus
random grain. When you average, the shape - being the same every time - stays
exactly as bright as it was. The grain, being different every time, cancels
against itself. It does not cancel completely; it cancels at a rate governed by
how many photographs you stacked, and the rate is the square root, so a hundred
photographs cut the grain by ten and ten thousand cut it by a hundred.

Apply that to documents. The signal in any one document is far too faint to see.
The signal in ten thousand documents from the same catalogue is the same faint
signal ten thousand times over, and the noise mostly cancels. The catalogue is
visible where the document was not. Nothing about the attack improved. The unit
of evidence changed.

Two things this picture insists on, and both of them are where audits break.

**The photographs have to be of the same room.** Your comparison stack -
documents the model did not train on - has to differ from your evidence stack in
exactly one respect. Same subject, same lens, same lighting, same era. Photograph
one room in summer and the other in winter and you will resolve a beautiful,
significant difference that is the season. This is the same trap as the bakery,
and stacking makes it worse rather than better: the more photographs you average,
the more confidently you resolve the wrong thing.

**Photographs of the same wall are not independent photographs.** If your ten
thousand images are forty shots each of two hundred and fifty walls, you have
two hundred and fifty photographs, not ten thousand, and the noise cancels at
the rate for two hundred and fifty. Counting the shutter clicks instead of the
walls inflates your confidence by a factor of six without adding any
information. This is the cheapest place for an opponent to attack an audit
report, and the defence is to say out loud, in the report, what you counted as a
wall.

And the honest limit of the whole approach: the averaged photograph shows that
something is in the room. It does not show which object it was. A
collection-level result supports a claim about the collection and no claim
whatsoever about any single document in it, and the moment a report crosses that
line it is claiming more than its own method delivers.

## The discipline that converts a number into evidence

Four pictures, one per requirement, because each requirement is a different
failure and they are usually confused with one another.

**Multiple testing: the lottery winner.** Someone wins the lottery. The odds
against that specific person winning were fifty million to one. Nobody concludes
the draw was rigged, because fifty million people played. Now run an audit that
tests eight thousand documents and reports the four hundred that came up at
"one in twenty." Four hundred is what eight thousand at one in twenty produces
when nothing is happening. The finding is the ticket, and the question that
dissolves it is: how many tickets did you buy? An opposing expert does not
dispute your arithmetic. They ask how many tests you ran, and that question is
the whole cross-examination.

**Dependence: the echo.** You claim ten witnesses independently reported the
same thing. It turns out all ten heard it from one person at the same dinner.
You do not have ten reports; you have one report and nine echoes, and treating
the echoes as testimony inflates your confidence enormously while adding no
information. Chapters of a book, papers from one lab, pages from one site - all
echoes. Somebody has to decide what counts as an independent voice, that
decision is arguable, and the report that states its own answer out loud is
harder to attack than the report that leaves it implicit.

**AUC versus the operating point: the smoke alarm.** An alarm that goes off for
every meal you cook detects every fire, and it is worthless, because you take the
battery out in week two. The number that matters is not how good it is on
average across all sensitivities - it is how many fires it catches at a
sensitivity where it does not scream at your toast. A detector summarized by an
average across every setting, including all the useless ones, is being flattered
by the settings you would never use. Ask what it catches when it is set strictly
enough to be believed, and a great many impressive-sounding detectors turn out
to catch almost nothing.

**Pre-registration: the sharpshooter and the barn.** A man fires a shotgun at a
barn wall, walks over, draws a target around the tightest cluster, and shows you
a bullseye. Every hole is real. The photograph is not doctored. And the display
proves nothing, because the target was drawn afterward. Choosing the statistic
after seeing which one separates, the threshold after seeing the scores, the
documents after seeing all of them - every one of those is painting the target.
Nothing in the numbers reveals it. Nothing in the report shows it. The only cure
is to paint the target first, in public, with a date on it, and that is why the
timestamped commitment is a business rather than a technique: it is the one part
a counterparty cannot manufacture after the fact.

## Where the null comes from

Everything in this unit and the next collapses to one question, and it is not
"how good is your method."

**Compared to what?**

A number is not evidence. A number compared against what that number would have
been if you were wrong is evidence. Every method in this book is a different
answer to where that comparison comes from, and the methods sort cleanly by how
much control you have over it.

Extraction sidesteps the question by not making a comparison. It shows you the
thing. The trace either exists or it does not, and there is nothing to be
uncertain about - which is why it is the strongest evidence and why it says
nothing at all when it comes up empty.

Membership inference needs the comparison and cannot get it. The world it needs
to compare against - the same model without your document - was never built.

Dataset inference borrows the comparison from somewhere else: a matched set of
documents the model did not see. It works exactly as well as the match, and
every advance in this family over the last two years has been an advance in
constructing the match rather than in sharpening the test. That is the tell. When
a field's progress is all in the control group, the control group was the
problem.

And then the move the next unit makes, which is so simple it feels like cheating.
Write two versions of something. Publish one. Lock the other in a drawer with a
dated receipt. The one in the drawer was never scraped by anyone, so it is
exactly what "not trained on" looks like - not an estimate of it, not a proxy
for it, the thing itself - and you can make as many as you want. You did not
find a comparison. You manufactured one, in advance, and you can prove when.

The catch is in the tense. It only works for what you publish from now on. The
models people most want to challenge were built from text collected years ago,
when nobody had put anything in a drawer, and no cleverness closes that gap.
State it first, before anyone else does, and then explain that this is precisely
why the borrowed-comparison tooling still has customers: it is the only thing
that speaks to everything published before you arrived.

## At the bench: three measurements on your own model

The reason to run all three, on a model you trained yourself, is that you already
know the answers and you will still be surprised by how they feel.

You know the heavily duplicated document will come back and the ordinary one
will not. Watching it happen is different from knowing it, because when the
duplicated one comes back it comes back completely, and you find yourself
thinking "so it IS in there" - and then the ordinary document, which is equally
in there, produces nothing at all, and the two feelings sit next to each other
and refuse to combine into a single opinion. That refusal is the correct mental
state and it is very hard to reach by reading.

You know the membership attack will hover near chance. What you cannot predict
is how convincing its output looks anyway. The scores separate a little. The
curve bends the right way. Some documents are clearly at the top. It looks like
a working detector until you set it strictly enough to accuse someone, at which
point it catches almost nothing. Then run it against your second training seed -
the same data, the same recipe, a different shuffle - and watch a third of the
individual verdicts change their minds. That number, on your own bench, from
your own model, is the thing to carry into every conversation where someone
proposes selling this.

And you know the collection-level test will work. It is worth doing last,
because after two failures the fact that changing the unit of evidence rescues
the same weak signal reads as a genuine result rather than an obvious one. Then
compute the honest effective sample size, watch your beautiful p-value get worse
by orders of magnitude, and decide which of the two numbers you would stand
behind in front of someone paid to break it. Deciding that, before anyone asks,
is the skill this unit is for.

## What you can now do

Three questions, and they will carry you through every claim anyone makes about
what a model remembers.

**Could this have happened anyway?** Applied to a verbatim match, it separates
evidence from boilerplate. Applied to a low loss, it separates membership from
the date. Applied to a detection at one in a thousand, it asks how many
detections were attempted. It is the same question every time, and it is the
question a null distribution is a formal answer to.

**Compared to what, and who chose it?** A number without a comparison is not
evidence, a comparison assembled after seeing the data is not a comparison, and
a comparison you built and dated in advance is the only kind nobody can take
apart. Everything in the next unit is a technique for being able to answer this
one confidently.

**At what setting, and what does it catch there?** Detectors are quoted at their
flattering average. Ask what happens at the strictness an accusation requires,
and most impressive numbers deflate to nothing. This is also the honest thing to
do to your own work, and doing it first is worth more than any result you would
have been protecting.

The last thing to carry is the shape of the argument rather than any of its
parts. Every fix in this unit was a fix to the comparison, not to the
measurement. Better attacks did not help. Bigger datasets did not help. What
helped, every single time, was somebody arranging to have a valid thing to
compare against - by aggregating to a level where a matched sample exists, by
finding a corner of the data whose randomness is known, or by writing two
versions and locking one in a drawer. That is the whole of the next unit, and
you now know why it has to exist.
