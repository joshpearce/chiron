## Eight numbers and a hundred dollars

Three people cook one dinner and there is one bill to divide.

The first two both know how to make the same stew. Either one alone could put
a decent meal on the table. Having both of them there does not get you two
stews; it gets you one stew slightly faster. The third person cannot cook at
all, but brought a spice nobody else had, and the spice does nothing on its own
- eaten by itself it is unpleasant - while in the stew it turns a decent meal
into a memorable one.

Now split the bill. Notice how quickly the obvious rules embarrass themselves.

Pay each person for what they could have done alone, and the two cooks are each
billed for the whole stew, so the diners are charged twice for one pot. Pay each
person for what would have been lost without them, and both cooks come out at
almost nothing - remove either one and dinner still happens - while the spice
carrier, who is genuinely irreplaceable, collects nearly everything. Push that
second rule to its limit with two people who make the identical stew: remove
either and nothing changes, so both are worth nothing, and yet between them they
made the entire dinner. The rule has not given a harsh answer. It has stopped
giving an answer.

That is the shape of the whole problem, and it has nothing to do with models or
data. Whenever the parts overlap, "what each one could do alone" adds up to more
than what was actually done, and "what would break without each one" adds up to
less. Neither is the thing you are dividing. What you want is somewhere between
them, and the rest of the unit is about finding the specific place.

## What a coalition is worth

Think of the trained model as an instrument with a needle on it, and the needle
measures surprise.

Show the model a page of text it has never seen and ask it to guess its way
through, word by word. Every time it is caught off guard, the needle twitches
up. A model that has read widely in that subject is rarely surprised and the
needle stays low. A model that has read nothing relevant is startled constantly.

Now the measurement that matters. Pick a page and hold it fixed forever. Read it
to a model that trained on none of your sources and note where the needle
settles. Then train a model on some particular combination of sources, read it
the same page, and note the needle again. The drop between the two readings is
what that combination bought. Do this for every possible combination and you
have the table.

Two things about this that are easy to miss and that make the whole method
work. There is no answer key. You never had to decide what a good answer looks
like, mark anything right or wrong, or write a benchmark - the model tells you
how surprised it was, and surprise is a number it computes about itself. And
because the same page is used every time, the readings are on one scale, so
comparing any two combinations is meaningful even though the combinations share
nothing else.

The instrument also has a property you have to respect: it is a needle, not a
click. It moves a little for a small improvement and a lot for a large one. If
you had instead scored the model on a multiple-choice quiz, most combinations
would produce the identical score, because a modest improvement does not flip a
single answer. The table would be full of ties, and a table full of ties says
nothing about anybody.

## Four requirements a contract can cite

Forget for a moment that there is any arithmetic here at all. You are writing
four sentences that everyone signs before anyone measures anything, and the
sentences have to be the kind a person will agree to without being talked into
it.

*Everything in the pot gets handed out, and nothing more than what is in the
pot.* This one is bookkeeping and everyone agrees to it instantly, which is
also why it is worth being careful with it: agreeing that the money is fully
distributed is not the same as agreeing that it is well distributed. A rule that
hands the entire pot to one person satisfies this sentence perfectly.

*Two people who do exactly the same thing are paid the same.* This is the one
that makes the whole arrangement checkable by someone who does not trust you.
They do not need the model, the code, or the method - just the table of
measurements. If two sources make the same difference everywhere and get
different cheques, something outside the measurements moved the money, and
everyone can see it.

*Someone who changes nothing is paid nothing.* Without this, the cheapest way to
get paid is to be big. Turn a generator loose, produce half a million words of
plausible filler, hand it over, and collect a share proportional to bulk. The
sentence closes that door in advance, and it closes it in a way you can test:
add the filler to any combination, watch the needle not move, pay zero.

*Settling twice for two things gives the same result as settling once for both.*
The least intuitive of the four and the one that stops a whole class of arguing.
Without it, the answer depends on whether you invoiced quarterly or annually,
and on whether two things were measured in one sitting or two. Every party then
has a reason to argue about the calendar, and the calendar has nothing to do
with anyone's contribution.

Now the surprising part, and the reason to spend fifty minutes here. Those four
sentences, which read like the beginning of a list, are the entire list. There
is exactly one way to divide any pot that keeps all four promises. Not a family
of ways to be chosen among by taste - one. Anyone who wants a different answer
has to tell you which of the four sentences they are giving up, and say it out
loud, to your counterparty's face.

## Averaging over arrival orders

Here is the picture that produces the answer, and it is a picture about
queueing.

Imagine the sources did not all arrive at once. They walked in one at a time,
and each one was credited with however much better the model got at the moment
it walked in. The first through the door is measured against an empty room, so
it gets credit for everything it brings. The last through the door is measured
against a room that already has everything else in it, so it gets credit only
for what was still missing.

Walk the three from the dinner through one order. Send the spice carrier in
first, into an empty kitchen: a jar of spice and nothing to put it in, so a
small amount of credit. Send in a cook second: the cook now has the spice
available, and the meal that results is far better than a plain one, so the cook
is credited with a large jump. Send in the second cook last: there is already a
finished dinner, and the newcomer can improve it only slightly, so almost no
credit.

Run the same three people through a different order and everything reverses. Put
a cook first and the cook is credited only with a plain stew. Put the spice
carrier last, into a kitchen with dinner already made, and the spice is credited
with the whole improvement it causes. The same three people, the same
measurements, and completely different numbers - because arriving early into an
empty room and arriving late into a full one are different jobs.

So which order is the real one? None of them. The sources did not arrive; they
were licensed, all at once, by a contract. The order is a fiction introduced to
make the accounting possible, and the moment you notice it is a fiction the
answer becomes obvious: run every order and average.

Three sources means six orders, which you can write out on a napkin. And two
things fall out that are worth pausing over. First, no matter which order you
pick, the credits along it add up to exactly the finished dinner - each person
is credited with the gap between what existed before them and what existed
after, and those gaps tile the whole thing with no overlap and no space left
over. Average over orders and it still adds up. The pot is fully distributed
without anyone arranging for that to happen.

Second, being credited with a lot in some orders and a little in others is not
noise to be smoothed away. It is the actual structure of the problem. The two
cooks each swing between large and small credit depending on whether the other
cook got there first, and the *size of that swing* is exactly how redundant they
are. The average is a summary; the spread is the story.

## Counting the runs: 4,096, not 479 million

The objection people raise is a counting mistake, and the picture that fixes it
is a row of light switches.

Every source is a switch: on if it went into the training run, off if it did
not. Twelve sources means twelve switches, and the number of distinct patterns
is what you get by doubling once per switch - two, four, eight, and so on twelve
times, arriving at four thousand and ninety-six. That is the number of different
models there are to train. Not because anyone was clever about it; there simply
are not any more combinations than that.

The orders are a different count and a much larger one, and the trap is
assuming each order needs its own model. It does not. When a source walks
through the door, what determines its credit is *who is already in the room* -
not the sequence in which they filed in. Two orders that leave the same crowd in
the room before a given source arrives hand it the identical credit, and the
model needed to work that out is the model trained on that crowd, which you
already trained once.

So the four hundred and seventy-nine million orders are answered by four
thousand and ninety-six models plus a lookup table. Train each combination once,
write the number down, and every order in the world is answered by reading, not
by training.

What that costs, at the scale this book works at: one of those models is about a
minute on a rented graphics card. Four thousand of them is a long weekend of
machine time and a few hundred dollars. The exact answer - not an approximation,
not a sample, nothing for anyone to poke at - costs less than a plane ticket.

The catch is the doubling. Every source you add doubles the cost, and doubling
is patient right up until it is not. Twelve sources is a weekend. Fifteen is a
week. Twenty is a small research grant. The wall is at about fifteen, and it is
a genuine wall - but it is nowhere near the place the standard objection puts
it, because that objection is about valuing every individual document, and there
is no such thing as a cheque made out to a document.

## The error budget: seeds before coalitions

Every number in the table came off a scale that wobbles.

Train the identical configuration twice, changing only the random shuffle, and
you get two slightly different models with two slightly different readings.
Nothing was wrong with either run. Training is a process with randomness in it,
and the randomness lands in the final number.

This would not matter much if the numbers you cared about were large compared
to the wobble. They are not, and the reason is worth feeling. Every quantity in
this unit is a *difference* between two readings. Each reading carries its own
wobble, and subtracting two nearly-equal wobbly numbers leaves you with a small
number carrying both wobbles. That is where all the trouble lives: the model
itself is stable to several decimal places, and the differences between models
are not.

The remedy is the boring one. Train each combination several times and average
before doing anything else. Averaging four wobbly readings halves the wobble;
averaging nine cuts it to a third. It never gets to zero and it costs runs, so
the only real question is how many you need, and the pleasant surprise is that
the answer comes out of the same arithmetic you are already doing rather than
from taste.

And here is the part that runs against intuition. You might expect that
combining thousands of wobbly numbers into one share would pile up the wobble.
It does the reverse. Averaging is averaging, and the share that comes out at the
end is *steadier* than any single leave-one-out difference you could have
computed from the same table - noticeably steadier, roughly half the wobble at
three sources and better at twelve. The combinatorics that look like the
expensive part of the method are also the part that buys back precision.

One choice matters more than any of this. The needle has to move smoothly. If
you score models by counting how many multiple-choice questions they got right,
then a source that improves the model slightly changes nothing at all - not one
answer flips - and it reads as contributing exactly zero. Do that and most of
your table is ties, every difference is a coin flip, and the entire apparatus
produces confident nonsense. Picking a smooth measurement over a lumpy one
changes the answer more than picking one division rule over another.

## Gaming the split

The publisher goes to a lawyer and comes back as two companies.

Nothing else changes. The same books, the same files, the same everything -
delivered twice, under two letterheads, by two entities that happen to share a
parent. The buyer now sees four suppliers where there were three.

Watch what the rule does with that. It is scrupulously fair to the four
suppliers in front of it. Each of the two new companies is evaluated on its own
merits, credited for what it adds when it walks into each room, and paid its
honest share of a four-way split. Add the two shares together and the publisher
is up by about a fifth, for the price of an incorporation fee.

The money has to come from somewhere, because the pot did not grow, and where
it comes from is the point. It comes almost entirely out of the news wire - the
supplier whose material most overlapped the publisher's. Think about why. The
wire's position was always weak, because much of what it offered could be had
from the publisher instead. Putting a second copy of the publisher in the room
makes the wire even more replaceable, and the rule charges it for that. The
specialist journal, which overlaps nobody, barely moves.

That is the whole character of this attack. It is not theft from the buyer, who
pays the same total. It is a transfer from your closest substitutes, and the
people it hurts are small suppliers crowded into the same subject as a large
one. It also has a ceiling: the publisher can keep splitting - three companies,
ten - and the take keeps rising, but it converges on what the publisher would
have been paid if the overlap with the wire had never been noticed. Fragmenting
does not create value. It erases the discount that redundancy earned everyone
else.

The reason the four sentences from earlier do not stop this is worth stating
plainly, because it is not a flaw in the reasoning. Every one of those sentences
is a promise about *this* deal, with *these* participants. Two people who do the
same thing get the same pay - that is a promise about two people in the room.
Nothing in any of them says a word about what happens when someone walks in
wearing a second hat. The rule is not being broken; it is being applied
correctly to a different room.

Which tells you where the fix lives. Not in the arithmetic, which is behaving.
In the door. Before anyone is paid, check whether two suppliers are handing over
the same material, and if they are, treat them as one supplier. That check is
not new work - it is the duplicate detection the corpus pipeline already runs,
pointed at the people rather than at the documents. It is also why recording
which sources shared which content was never mere housekeeping. It was always
the thing that decides who gets paid.

## The honest position

Two people can run this entire procedure on the same corpus, do every step
correctly, and get different splits. Not because one made an error - because
they chose different pages to read to the model.

Everything here is measured relative to a held-out sample of text. Change what
that sample is about and you change what "valuable" means, honestly and
substantially. Add some medical writing to the sample and whichever source
carries medical writing rises, exactly as it should. That is the method working.
It is also a lever, and whoever holds it holds the outcome.

There is no clever version of the method that removes the lever, because the
lever is a business decision wearing a technical costume: what is this model
for, and therefore what should it be good at. The only honest handling is to
name the sample, describe how it was assembled and cleaned, and commit to it in
writing before anything runs - the same discipline a scientist uses when
declaring what will be tested before looking at the results. That does not make
the choice go away. It makes it a disclosed choice instead of a hidden one, and
disclosed choices are the kind counterparties can accept.

Two more limits belong in the same breath. Every number in this unit came from
actually retraining models, which is what makes it exact and also what caps it -
nobody has shown any of this works at the scale of a frontier model, and the
claim should not be stretched past the scale where the retraining was done.
And the market has not been waiting for this: nearly all the money in content
deals so far has gone to aggregators, and the industry has settled on flat fees.
The uncomfortable possibility is that the settling was correct - that nobody's
measurement was ever clean enough to justify anything else.

What survives all of it is one thing that nothing else in this book can do. At
the end of this procedure you can tell a rightsholder why their share is what it
is, in four sentences they can check, without them trusting you, your code, or
your model. No amount of gradient machinery produces that sentence, and it is
the sentence the whole conversation eventually comes down to.

## At the bench: the split

The thing to build is small and the thing to be careful about is not the code.

You will end up with one file: a value for every combination of your twelve
sources, each one the average of a few training runs. It is thirty-odd
kilobytes. Everything else in this unit - the split, the alternative splits, the
diagnostics, the attack, the fix - is arithmetic over that one file and takes
less time to run than to describe. The cost is entirely in producing it, which
is thousands of short training runs, and the correct engineering instinct is
therefore to spend zero effort making the arithmetic fast and all of it making
the runs fast.

Then do the three things that separate a demo from a design, in this order.
Check whether a single number per source is even an honest summary, by seeing
how well a simple "each source is worth a fixed amount" story fits your table -
if it fits badly, your sources interact so strongly that one number each is
hiding something and a ranking will not be stable. Attack your own split before
anyone else does, by re-running it with your largest source pretending to be
two companies and then three, which costs no machine time at all because
duplicate copies teach the model nothing new and the answers are already in the
file. And measure how far apart your sources actually are compared to how much
your scale wobbles.

That last number is the one to lead with, whichever way it lands. If the sources
are clearly distinguishable, you have a split and the measurement is the reason
to believe it. If they are not, you have learned that this corpus at this scale
should be paid by flat fee, which is a finding, is cheap to state, and is far
more credible than a pie chart from someone who did not check.

## What you can now do

You can look at a table of measurements over combinations and produce a
percentage for each participant, along with a four-sentence argument for why
that percentage and not another, checkable by a person who does not trust you
and has never seen your code.

You can say what it costs before you start - it doubles per source, it is a few
hundred dollars at a dozen, and there is a wall at about fifteen - and you can
say what to do past the wall and what you give up by doing it.

You can put an error bar on every percentage, from the same table, and say how
many more runs would tighten it and what they would cost.

And you can say when not to do any of this. When the differences between your
sources are smaller than the wobble in your scale, the right answer is a flat
fee, and saying so is a stronger position than a confident chart. That is the
sentence to practise, because it is the one that will be tested by the first
person across the table who knows the field.
