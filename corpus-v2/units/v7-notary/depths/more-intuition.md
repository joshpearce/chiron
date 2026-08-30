## The empty chair

Imagine a law requiring every restaurant to publish a list of its suppliers.
The list must be honest, it must be updated twice a year, and lying on it costs
you the business. Reasonable law, real teeth.

Now read the form. It asks for the top ten per cent of suppliers by weight of
goods delivered. You are a family cheesemaker who sells them forty kilos a
month, and the top ten per cent is dominated by the flour wholesaler and the
potato merchant. Your name is not on the form and never will be, no matter how
honest the restaurant is.

Then, at the bottom of the same law, a clause: the inspectors will check that
the form is filled in, and will not check whether any particular supplier's
goods actually went through the kitchen.

That is the situation exactly. There is a mandatory document, a real penalty, a
recurring deadline, and a stated refusal to answer the one question that matters
to anyone who is not enormous. The chair where an inspector of individual claims
would sit has been drawn on the floor plan and left empty, and the whole rest of
this unit is about who sits in it and what they carry with them.

## Constructing a null

Someone shows you a coin and says it lands heads more often than it should. They
flip it fifty times and get thirty-one heads. Is that suspicious?

You cannot answer without knowing what an ordinary coin does. Not what an
ordinary coin does on average - everyone knows it is about half - but the whole
spread: how often an ordinary coin gives you thirty-one, or thirty-five, or
twenty-two. Only against that spread does thirty-one mean anything. If ordinary
coins routinely land thirty-one in fifty, you have seen nothing. If they almost
never do, you have seen something.

That spread is the thing this unit calls the null, and everything hangs on
whether you can get at it. With coins it is easy: pick up an ordinary coin and
flip it. With a trained model it is not, because the ordinary coin you would need
is the same model trained without your data, and that coin does not exist and
never will. This is what the previous unit walked into and could not walk out of.

Here is the move. Stop trying to find an ordinary coin. Mint your own.

Print a hundred banknotes from a plate only you own. Put twenty-five of them into
circulation and lock seventy-five in a drawer. Some months later, a bank tells
you it has been seeing notes from your plate. You ask which serial numbers.

If the numbers are all from the twenty-five you spent, that is the story fitting
together. If the bank is also reporting numbers from the drawer, then whatever
process is generating those reports has nothing to do with your notes having been
spent, because those particular notes have never left the drawer. The locked
seventy-five are the ordinary coin. They tell you the background rate of being
told a thing that cannot be true.

Three things about that arrangement carry the whole rest of the unit, and each
one sounds obvious until you see somebody get it wrong.

The notes in the drawer have to come off the same plate. If you filled the drawer
with somebody else's currency, any difference between what the bank reports about
your notes and about theirs could be a difference between the two currencies
rather than a difference between spent and unspent.

The drawer has to have been filled first, and locked. A drawer you fill after the
bank calls you tells you nothing about anything, and worse, invites the suspicion
that you chose what to put in it.

And you have to say in advance what would count as a match. Otherwise, faced with
a list of near misses, you will find yourself deciding after the fact which
resemblances count - and you will decide in your own favour, not because you are
dishonest but because that is what people do.

## The test, end to end

You are trying to hear whether one particular instrument is playing in a
recording of a very large orchestra.

You cannot pick it out by listening. So instead you ask a hundred specific
questions about what that instrument played - was there a rising figure at bar
forty, a held note at bar sixty-three - and count how many times the recording
agrees with your score.

Twenty-five agreements out of a hundred. Is that a lot?

By itself the number means nothing, because agreements happen by accident. Two
instruments playing the same kind of music will coincide often. So you ask the
same hundred-style questions about a score you wrote and never gave to anyone. If
the recording agrees with that one ten times in a hundred, you have learned the
accident rate. Ten per cent of the time, agreement happens for no reason.

Now the twenty-five means something: it is fifteen more than the accident rate
would produce. And to know whether fifteen extra is a lot, you need one more
thing - how much the accident count itself wobbles from one batch of questions to
the next. Not every unplayed score gets exactly ten. Some get seven, some get
thirteen. If the typical wobble is three, then fifteen extra is five wobbles'
worth, and five wobbles is not something you see by luck. If the typical wobble
had been fifteen, twenty-five would be unremarkable.

That is the entire calculation, and it is three quantities: the accident rate,
the size of the wobble, and how many wobbles you are away from the accident rate.
Everything else in the arithmetic is bookkeeping.

One thing about the wobble worth carrying. It grows more slowly than the count
does. Ask four times as many questions and the accident count quadruples while
the wobble only doubles, so you end up twice as far out in wobble-units. More
questions always help, and they help less and less. That single fact is why the
design cares about how many watermarks you plant, and why it does not care
infinitely.

## Canary design is the whole game

The mistake is thinking of this as hiding something. It is not hiding, it is
smuggling - and smuggling has two failure modes, not one.

Try to smuggle a gold bar through customs by putting it in your bag. It is seen
immediately, because a solid brick of metal is exactly what the scanner is
looking for. That is the random string, the base64 blob, the forty-character
identifier. Conspicuousness is not a feature here; the pipeline is a scanner, and
being unusual is what it scans for.

So try the other extreme. Smuggle it by melting it down and pouring it into the
shape of an ordinary belt buckle, indistinguishable from every other belt buckle.
It sails through. And now, on the other side, you cannot find it either, because
it looks like every other belt buckle. That is the ordinary sentence, the plain
phrase inserted into forty articles: it survives perfectly and there is nothing
left to detect.

The trick that works sits in neither place. Make something that looks entirely
ordinary from the outside and is impossible from the inside. A perfectly
convincing passport for a country that does not exist. It passes every check that
looks at the shape of the thing, because the shape is right. And the moment
someone tries to use the knowledge in it - the moment somebody knows the name of
that country's capital - you know exactly where they learned it, because there is
nowhere else on earth to learn it from.

That is the design. Not a hidden mark, not a strange string. A fluent, sober,
plausible account of a research instrument that has never existed, indexed by the
one thing a data pipeline never checks: whether the content is true.

And the detection method follows from the same idea. You do not go looking for
your document inside the model. You ask the model a question that only somebody
who read your document could answer. The instrument is a conversation.

The homoglyph trick deserves a separate note because it is the one clever people
propose most confidently. Swapping a Latin letter for its Cyrillic twin is like
signing a document in invisible ink and then handing it to a clerk whose first
action, before anything else, is to photocopy it. The photocopier does not know
about your ink and does not care. Normalization runs before anybody looks at
anything.

## How many watermarks, and how often

Put a drop of dye in a bathtub and the water turns visibly coloured. Put the same
drop in a swimming pool and nothing happens. The drop did not change; the thing
it went into did.

That is the whole scaling law. What decides whether the model notices your
content is not how many copies you planted but how many copies relative to
everything else it read. Your twenty-five documents in a hundred-million-token
corpus are a bathtub. The same twenty-five in a trillion-token corpus are a
swimming pool, and the water is clear.

Two things follow, and one of them is uncomfortable.

The first is a rule you can act on. If they build a pool five times bigger, you
need five times the dye. This is not a metaphor for a rough trend; it is the
operating rule, and it means the amount of watermarking you must do is set by
somebody else's compute budget and rises every generation. Watermarking is not
something you do once. It is a subscription you pay in fabricated content.

The second is the uncomfortable one. The amount of dye you are allowed to use is
capped by the size of your own bathtub, because you can only put fabricated
articles into your own archive and only so many before your archive stops being
worth anything. A large publisher has a big tub and can afford a lot of dye. A
small one cannot, and no cleverness closes the gap, because the constraint is
arithmetic rather than technical.

So the parties with the least leverage in the licensing market also have the
weakest evidence available to them, which is a familiar shape and not a
coincidence. The only structural answer is that a hundred small publishers can
share one plate, and between them own a tub the size of a large publisher's. That
is a business - and it is also the reason the notary is a service rather than a
tool you download.

There is one piece of good news buried in the arithmetic. The relationship
between how much dye you use and how visible it is is a gentle one: doubling the
dye does not double the visibility, it adds a step. Which means the small
publisher's disadvantage, while real, is far smaller than the ratio of their
archives would suggest - and it means there is a sensible amount to use, above
which you are fabricating content for very little return.

## Your own private control

Two ways to find out whether a runner has been training.

The first way: time a hundred runners over a mile and compare the average against
a hundred other runners. The trouble is that runners differ enormously for
reasons that have nothing to do with training - age, build, whether they had a
cold that week - and those differences swamp the couple of seconds that training
buys. You are trying to hear a whisper in a room full of shouting.

The second way: time each runner twice, before and after, and look at each one's
own change. Age, build, and build quality all cancel, because they are the same
person both times. What is left is the change, and the change is the thing you
wanted to measure. The whisper is now in a quiet room.

The paired watermark design is the second way, and the private siblings are the
"before". You write several versions of the same document, publish one, and keep
the others. Every version has the same topic, the same length, the same register,
the same difficulty - they are the same runner. When you compare the published
version against its own siblings, everything that makes documents differ from
each other cancels, and only the effect of having been read survives.

The size of the win is worth feeling rather than just knowing. In the worked
example, the same documents and the same effect gave you a result that was barely
worth reporting when measured the first way, and an overwhelming one when
measured the second. Nothing about the content changed. The comparison changed.

And there is a version of this you can hold in one sentence. Under the null, the
model has read none of the versions, so which one it happens to find most
familiar is a coin toss among them. If the published one wins far more often than
its share, that is the finding, and it needs no assumptions about how surprising
documents usually are. Just: it should win one time in five, and it won two times
in five, across a hundred documents.

If you had refused to pair and tried to make up the difference by publishing more
documents instead, you would have needed twenty-five times as many. That is not a
refinement. That is the difference between an affordable programme and an
impossible one.

## The pre-commitment ledger

Two laboratories run the same experiment and get the same number. One of them
posted, publicly and with a timestamp, exactly what they were going to measure
and what would count as a positive result, before they began. The other wrote it
up afterwards. The numbers are identical and the evidence is not, and everybody
knows why.

That is the whole product, and it has nothing to do with statistics.

Think about what a notary actually does. A notary does not verify that your
document is true, or fair, or wise. A notary attests that this document existed,
in this form, on this date, and that this person signed it. It is an
extraordinarily narrow service and an entire profession has been built on it for
centuries, because "it existed before" is a fact that cannot be established after
the fact by any amount of cleverness, and an astonishing amount of what people
need to prove reduces to exactly that.

The three attacks on any statistical claim of this kind are all attacks on
timing, wearing different clothes.

The first says: you have been fishing. You ran this test on forty clients and are
showing me the one that came back interesting. The answer is arithmetic - if the
test is run forty times, the bar has to be higher, and here is the higher bar and
here is the result clearing it anyway.

The second says: your forty tests are not forty separate throws of the dice.
Those clients quote each other and all forty tests were run against one model.
This one has no clean answer, and the correct response is to say so, describe the
overlap you can measure, and note that the correction you used errs on the
cautious side. An expert who concedes a point precisely is believed on the ones
they do not concede.

The third says: you decided what counts as a hit after you saw the result. This
one has no statistical answer at all. There is only a document with a timestamp
on it, or there is not.

Which is why the defensible thing here is not the test. Anyone can run the test;
it fits on the back of an envelope and the method is published. What cannot be
copied in a hurry is a public, append-only record going back years, kept by
somebody with no stake in any particular outcome, containing promises made before
the models in question existed. A competitor with ten times your funding cannot
buy that, because the only way to have started in 2026 is to have started in
2026.

## What a court will actually credit

Say the hard part first, out loud, before anyone asks.

A watermark is a mark you make on your own property before it is taken. It
protects things you publish from today onward. It does absolutely nothing about
what was taken from you three years ago, and no improvement in the technique will
ever change that, because the whole method works by having arranged something in
advance and you cannot arrange something in the past.

The models people most want to sue were trained on material collected years ago.
So the product that produces proof speaks only about the future, and the past -
which is where the anger is, and much of the money - is served by something
weaker and honestly labelled: screening, which produces leads rather than
exhibits. It tells you where to look. It does not go in front of a judge.

As for what a judge has actually credited, the ranking is not the one an engineer
expects. Top of the list is paperwork: records showing a company downloaded a
pirated library. That is evidence about a transaction, not about a model, and it
is what drove the largest settlement in the field. Second is the model saying the
thing back to you - prompt it, get the work out, put the two side by side. Third
is everything else, and statistical inference about model behaviour is not on the
list at all, because no court has ever been asked to rest a finding on it.

Which is where the keyed watermark sits: untested, and shaped unusually well for
the test whenever it comes. A court asked to admit a novel technique wants to
know the chance it is wrong and whether the method has rules governing how it is
run. The construction has both, and it has them in an unusual form - the chance
of being wrong comes from a set of documents the other side's own expert can
regenerate and check for themselves, and the rules governing the run are in a
document whose timestamp predates the model. Most statistical evidence asks a
court to trust an assumption. This asks it to check a hash.

## The audit market, and its graveyard

Read the regulation as a builder rather than as a lawyer and it stops being law
and starts being a specification with a launch date on it.

Somebody has mandated a form. They have mandated who fills it in, what fields it
has, how precise the answers must be, how often it is refreshed, and what
happens if you skip it. They have shipped the deadline. And they have written
down, in the same document, that nobody will check the answers against reality.
That is a specification for a market: a compulsory annual process, produced by
people who would rather not be doing it, with no verification layer, and
somebody has to build the verification layer.

The order in which to sell into it is counterintuitive and worth internalizing.
The obvious product - go to a rightsholder, tell them you can prove their work
was used - is the last one, not the first, because the customer is angry, the
counterparty is hostile, and nobody in the chain is obliged to answer the phone.

The first product is the mirror image, and it is friendly. Go to a developer and
sell them a certificate that they did NOT train on something. They want it. They
will help you get it. They have a compliance budget, they have a deadline, and
the deadline repeats every six months. The technical work is closely related and
the sales conversation is completely different, which is the sort of asymmetry
worth noticing early.

Now the graveyard, because two of these will come up in every technical
conversation you have.

Zero-knowledge proofs of training are the beautiful idea. Prove, mathematically,
that you ran this training on this data, without revealing the data. It is
elegant, it is exactly what everybody wants, and it is off by a factor that is
hard to hold in your head: proving the training of a model a thousand times
smaller than a frontier one would take longer than recorded history. Not "we need
better hardware." Not "in five years." The gap is four or more orders of
magnitude on a curve that has not moved anything like that fast.

Hardware attestation is the one that sounds boring and is real, and the confusion
around it is precise enough to name. An attested training run proves that a
specific program ate a specific sealed box of data and produced specific weights.
Every part of that is trustworthy. And it says nothing whatsoever about what was
in the box. Someone who tells you attestation solves provenance has heard "sealed
box" and understood "inspected box", and those are opposite things. It is a
useful guarantee between parties who already trust each other enough to be
checked - which is a real product, with a real buyer, and is structurally never
the answer to somebody who does not want to be checked.

## The strongest case against the notary

Four objections, and none of them are answered by working harder on the
technique.

The past is not protectable, and the past is where the grievance is. Everything
already scraped is out of reach forever. A business whose good product matures in
several years needs to eat in the meantime.

The other side gets a move. Today the watermarks survive because nobody is
looking for them. If this became standard, somebody would look. The good news is
that finding them means telling apart invented entities from genuinely obscure
real ones across the whole internet, and getting that wrong means deleting
enormous amounts of real content you wanted - so the arithmetic is against the
filterer as long as watermarks are rare and their owners are not enumerable. The
bad news is that a lab does not have to filter the whole internet; it only has to
filter the corpora belonging to publishers who have publicly announced they are
doing this. That risk is real and it is not solved.

Nobody has bought this yet. Not this specifically - anything in this space. The
neutral marketplaces have no licensees, the crawl-payment products never left
beta, and the share of content deals that even include training rights is
shrinking. The niche is open partly because nobody has proven there is money in
it, and "the regulator created a deadline" is a better argument than most, but it
is still an argument rather than a customer.

And the last one is the one nobody in the technical conversation raises. To do
this, a publisher has to publish things that are not true. At the density a small
publisher needs, that is nearly one article in a hundred. For a scholarly
publisher, whose entire value is that what they print is true, this is not an
operational inconvenience; it is a threat to the asset being protected. Every way
of softening it - confining the fabrications to a labelled section, or to
metadata, or to a content type where invention is expected - weakens the design,
because the design works by being indistinguishable from everything else. This
unit does not resolve that, and any honest deployment has to.

What survives all four is narrow and real. It is the only method in the field
that produces a chance of being wrong that a hostile expert can reconstruct
without trusting you. It works through a chat window, which is the only access
anyone will ever have to the models that matter. Its mathematics is simple enough
that everything that can go wrong is a procedure rather than a mystery. And the
thing it accumulates - a public record of promises made early and kept - is the
one asset a better-funded competitor cannot buy.

## At the bench: the notary run

The build is a rehearsal, and the point of a rehearsal is the part that would be
embarrassing to get wrong in front of an audience.

You plant fabricated articles in one source of your corpus, having first posted a
sealed promise about what you will measure and what would count. You train the
model. You ask it the questions. You get a number. All of that is the pleasant
part and it will almost certainly work.

Then you do the thing that makes it a scientific result rather than a demo. You
train a second model, on the same corpus with the fabrications removed, and you
run the exact same questions against it. That model has never seen a single one
of your invented instruments. If your test fires on it, then your idea of "what
happens when the model has not read this" is wrong, and every other number you
produced is decoration.

You will never have that second model in a real engagement. That is the entire
reason this field is hard: the comparison you most want is a model trained
without your data, and it belongs to somebody else. Here you have it, because you
own both training runs, and having it once - feeling what it is like to check
your null directly rather than construct it - is worth more than a successful
detection.

Two disciplines to carry out of it. Decide before you train that you will run
both models and publish both results, because a control you only report when it
agrees with you is not a control. And notice the asymmetry in what the second run
can tell you: if it fires, you have definitely learned something is broken; if it
does not fire, you have learned very little, because a single quiet result is
weak evidence about a claim that the quiet outcome happens virtually always. The
cheap fix is to run the check many times over the control material you have
already generated, which costs nothing but a few minutes of inference and turns
one anecdote into a picture.

## What you can now do

Four things, and they build on each other.

You can stop trying to find an ordinary coin and start minting your own. When the
comparison you need does not exist in the world, manufacture both halves of it
from one plate, spend one half and lock the other away. The locked half is the
answer to "what does this look like when nothing happened", and having that
answer is the difference between a number and a chance of being wrong.

You can design the thing you spend so that it survives the journey and can still
be found at the other end. Not something strange, which gets thrown away, and not
something ordinary, which vanishes into the background - something entirely
plausible on the outside and impossible on the inside, found by asking a question
only a reader could answer. And you can size it: enough of it relative to your
own archive, enough copies relative to the pool it is going into, and more of
both as the pool grows.

You can turn the result into a document rather than a claim. Promise first, in
public, with a date on it, exactly what you will measure and what would count.
Then measure, unchanged. Then hand somebody the plate and let them check every
step themselves. Say how many times you ran the test and raise the bar
accordingly. Concede precisely the thing you cannot fully answer. Refuse, in
writing, to describe a result that missed the bar as encouraging.

And you can place it honestly. Sell the friendly version first, to the customer
who wants the certificate and has a budget and a recurring deadline. Build the
adversarial version for the long game, knowing that the moat is the record and
not the mathematics. Say the limitation before anyone asks it, because it is the
first thing a serious person will find, and because saying it first is what makes
everything else you say worth listening to.
