## Two transcripts, one model

**The pair.** The gap between the two transcripts is like **a cache that returns
instantly for a hot key and misses on a cold one**, and like **a vendored file
that appears in four hundred forks of a repo while another file appears in one**.

**The shared structure.** In all three, what you can observe about an item is
governed by how many times the system encountered it, and by nothing about the
item's importance. The hot key is not more valuable than the cold key; it is
more frequent, and frequency is what put it in the cache. The vendored file is
not better code; it is copied, and copying is what makes it findable by a grep
across forks. In each case an observer who mistakes the observable for the
important draws exactly the wrong conclusion about the system's contents - and,
crucially, a miss carries no information at all, because a miss is the ordinary
state for the overwhelming majority of keys.

**Where this breaks.** A cache has an eviction policy you wrote, so you can
inspect it, reason about what is resident, and enumerate the keyspace. Training
has no policy and no residency list: nothing decided that the novel would be
retained and the paper discarded, and there is no API that tells you what is in
there. The fork analogy breaks in the other direction - you can exhaustively
search four hundred forks and be certain, whereas an extraction attack samples
an output space you cannot enumerate, so "not found" is always "not found under
these prompts" rather than "absent."

---

**The pair.** Duplication as the driver of memorization is like **log sampling
at one in a thousand**, and like **a flaky test that only reproduces under
repeated runs**.

**The shared structure.** Both convert a rate into a visibility threshold. An
event that happens ten thousand times survives one-in-a-thousand sampling and
lands in your logs ten times; an event that happens twice is almost certainly
never recorded, even though it definitely happened. Same with the flake: at a
1% failure rate you see it in a hundred-run loop and never in a three-run CI
job. In both cases the thing you did not observe is not absent, and anyone
reasoning from the sampled record to the underlying population needs the
sampling rate to say anything at all.

**Where this breaks.** You chose the sampling rate and can raise it. Nobody
chose the memorization rate; it emerged from an optimization that was not trying
to store anything, and there is no knob on the finished model that turns it up.
Worse, the equivalent of your sampling rate here depends on the size of somebody
else's corpus, which you do not know and cannot query.

## What memorization is, and what it is not

**The pair.** The three definitions of memorization map onto **comparing two
builds byte-for-byte versus comparing their behaviour**, and onto **a boolean
test result versus a flake rate**.

**The shared structure.** Byte-identical binaries are a strong and brittle
signal: they prove the builds came from the same source, and they go away the
instant a timestamp or a path is embedded, even though the programs are still
identical in every respect that matters. That is verbatim continuation exactly -
a strong signal that an adversary can eliminate without changing anything real,
by rephrasing rather than by forgetting. And the second half is the flake rate:
reporting "the test passed" from a single run throws away the information that
it fails 40% of the time. Probabilistic extraction is the flake rate for
memorization, and it is more useful for the same reason a flake rate is more
useful than a green check: it is comparable across items and it does not depend
on which single run you happened to observe.

**Where this breaks.** A build comparison has a ground truth you can obtain -
you have both source trees. The document comparison has no such luxury: you have
the document and you do not have the corpus, so a match is one-sided evidence
and a non-match is nothing. And a flaky test can be root-caused by instrumenting
the code, whereas a 40% extraction rate has no accessible mechanism to
instrument; you can measure it and you cannot explain it.

---

**The pair.** Templatic reconstruction being counted as memorization is like
**a diff tool reporting that two projects share code because both contain the
same generated lockfile header**, and like **a plagiarism checker flagging the
standard licence block**.

**The shared structure.** The matched text is real, long, and identical, and it
carries zero information about the relationship between the two artifacts,
because it was produced by a generator that both parties ran independently. The
match rate looks impressive precisely because the boilerplate is long. In every
case the correct filter is the same question: would this text be here anyway? A
tool that does not ask it produces a large number of confident false positives
and its output volume is proportional to how much boilerplate the domain has.

**Where this breaks.** For lockfiles you can maintain an exclusion list, because
the set of generators is small, known, and enumerable. Training-corpus
boilerplate is open-ended - every publisher's footer, every genre's conventions,
every era's formatting - so exclusion by list does not converge. The workable
substitute is not a bigger list but a counterfactual test, which is exactly the
expensive definition nobody can afford to compute, so in practice you are
sampling and inspecting rather than filtering mechanically.

## The arithmetic of what can be stored

**The pair.** The capacity bound is like **the counting proof that no lossless
compressor shrinks every input**, and like **a Bloom filter with a fixed bit
budget**.

**The shared structure.** All three are pigeonhole arguments over a fixed state
space. A compressor mapping every $n$-bit string to a shorter string would need
more short strings than exist, so some inputs must grow - and no cleverness in
the algorithm escapes it, because the argument never mentions the algorithm. A
Bloom filter with $m$ bits cannot distinguish more than $2^m$ set-membership
states, so past a certain load, collisions are guaranteed rather than unlucky.
The model is the same shape: a fixed number of distinguishable weight
configurations cannot encode a larger space of corpora, so two corpora must
collide and neither is recoverable from the weights. In each case the bound is
about counting, holds regardless of implementation, and is therefore the one
claim in the argument that nobody can attack.

**Where this breaks.** A Bloom filter's false-positive rate is designed,
derivable in closed form from the bit budget and the hash count, and you know it
before you deploy. The model's 3.6 bits per parameter is a measured empirical
constant, not a designed one, and it comes from a specific experimental setup
that may not transfer to your architecture. The compression analogy breaks
harder: a compressor is required to be lossless, so the bound bites everywhere,
whereas a model is under no obligation to reconstruct anything, which is why the
bound tells you the corpus is unrecoverable and tells you nothing about which
individual documents got the surviving budget. Treat the counting argument as a
ceiling proof and never as a floor.

---

**The pair.** Memorization risk depending on relative rather than absolute
frequency is like **rate limiting by share of traffic rather than by requests
per second**, and like **a log line's survival probability under
volume-proportional sampling**.

**The shared structure.** A rule expressed as a share is stable under growth and
a rule expressed as a count is not. Configure a limiter at 100 requests per
second and it means one thing at a thousand requests per second of total traffic
and something entirely different at a million. The planted trap has the same
property: 90 copies is a meaningful share of a 20-billion-token corpus and
statistical noise in a 1.5-trillion-token one, and the parameter that changed
was somebody else's denominator.

**Where this breaks.** You own the limiter and can express the rule as a share
directly. You do not own the corpus, cannot observe its size, and cannot adjust
after the fact - the material was published years before the training run and
the duplication count was fixed at publication. So the design has to
over-provision against a denominator that will only grow, which is a planning
problem with no equivalent in a system whose scaling you control.

## Extraction as evidence

**The pair.** Extraction as evidence is like **finding a plaintext credential in
a core dump**, and like **finding a leaked key in git history**.

**The shared structure.** All three are existence proofs. Nobody needs to accept
a statistical model to read a credential out of a core dump; the artifact is the
argument, and the conversation ends. All three are also almost entirely silent
in the negative direction, for the same reason: the base rate of secrets
surviving into a dump or a commit is low, so a clean dump is the ordinary
outcome and tells you nothing about whether the process ever held the secret. And
in all three the cost asymmetry is identical - cheap to look, occasionally
decisive, never a basis for a guarantee.

**Where this breaks.** Git history is finite and exhaustively searchable: you can
grep every object in the repo and say with certainty that a string is not
present. A model's output space is not enumerable, so every negative extraction
result is conditional on the prompts you tried, and an adversary who finds a
better prompt tomorrow overturns your report. This is why the honest report says
"did not fire under these attacks at these prompt lengths" rather than "is not
present," and why anyone who writes the second sentence has made a claim their
method cannot support.

---

**The pair.** Treating a model's refusal as evidence of absence is like
**concluding a file is gone because `ls` does not show it**, and like
**concluding a port is closed because a firewall drops the SYN**.

**The shared structure.** In each case you observed a policy layer, not the
underlying state. The file may be there with the directory entry hidden; the
service may be listening behind a drop rule; the text may be in the weights
behind an alignment layer that declines the request. The observation is real and
it is about the wrong component. Anyone who has done incident response knows to
distinguish "the thing is not there" from "something in front of the thing said
no," and this is the same distinction with higher stakes attached.

**Where this breaks.** You can usually get past the policy layer in a system you
control - stat the inode, check from inside the VPC - and confirm the underlying
state definitively. There is no equivalent confirmation here. A divergence attack
that succeeds proves presence; a divergence attack that fails proves nothing
new, because you still cannot see the state directly. The asymmetry never closes,
which is why the operational rule is one-sided: never accept a refusal as a
negative, and never report your own model's refusal as reassurance.

## Why membership inference cannot prove training

**The pair.** The missing null is like **an A/B test with no control cell**, and
like **attributing a latency regression with no prior deploy to compare
against**.

**The shared structure.** In each case the measurement is fine and the
comparison does not exist. You shipped a change, conversion is 4.1%, and you
have learned nothing, because you do not know what conversion would have been.
p99 is 340ms and you cannot say whether that is a regression without the
baseline. The instinct in both cases - measure more carefully, gather more
traffic - does not help at all, because the missing object is not precision. It
is a second run of the system under the counterfactual condition. Membership
inference is in exactly that position: the object it needs is the same training
run without your document, and no amount of care in computing the statistic
manufactures it.

**Where this breaks.** In your own system you can usually get the control back.
Roll back, hold out a cell, replay yesterday's traffic - the counterfactual is
expensive but obtainable. For somebody else's model it is not obtainable at any
price you can pay: it requires their corpus, their recipe, their cluster, and
several runs of it. That is the whole difference between an experiment you
neglected to design and an experiment nobody can run.

---

**The pair.** The blind-classifier result is like **discovering your fraud model
is keying on a timestamp column that leaked from the label pipeline**, and like
**a cache-hit metric that turns out to be measuring which region the request
came from**.

**The shared structure.** A model achieved good numbers by learning a shortcut
that correlates perfectly with the label in your dataset and has nothing to do
with the phenomenon. The tell in both cases is the same experiment: strip the
feature you believe is doing the work, and performance does not drop - or,
sharper, keep only the suspect feature and performance goes up. That is exactly
what happened here. A classifier with no access to the model at all beat
classifiers that queried it, which means membership was never what was being
detected. Anyone who has debugged label leakage recognizes this instantly, and
the recognition is the fastest route into the whole argument.

**Where this breaks.** Label leakage in your own pipeline is fixable: find the
leaking column, drop it, retrain, and the honest number appears. Here there is
no column to drop. The leak is in how the benchmark's membership labels were
constructed - pre-cutoff text as members, post-cutoff as non-members - so the
confound is the dataset's definition of the label, not a feature alongside it.
Fixing it means constructing a differently-matched corpus, which is a
data-collection problem that may have no solution for a model whose training
data is undisclosed.

## Dataset inference: the collection as the unit of evidence

**The pair.** Aggregating documents to recover a weak signal is like **averaging
many request latencies to detect a small regression**, and like **stacking
repeated benchmark runs until the effect clears the noise**.

**The shared structure.** In all three the per-observation signal is far smaller
than the per-observation noise, and in all three the fix is not a better
measurement but more observations, with the improvement arriving as a square
root. Nobody detects a 2ms regression from one request; everybody detects it
from a hundred thousand. The intuition an engineer already has for "run the
benchmark more times" is precisely correct here, and it is the entire reason
collection-level inference works where per-document inference does not.

**Where this breaks.** Latency samples from independent requests really are
independent; documents from one catalogue are not. Worse, the failure is silent -
your aggregate looks better and better as you add correlated documents, exactly
as it would if the documents were independent, and nothing in the computation
flags it. Benchmarks have the same trap when you re-run on a warm cache and count
each run, which is a familiar enough bug to make the point land: repeating a
correlated measurement inflates confidence without adding information.

---

**The pair.** Effective sample size is like **counting retries as distinct
requests in a traffic report**, and like **counting log lines when one tight
loop emitted ten thousand of them**.

**The shared structure.** Your denominator is inflated by copies of one event.
The dashboard says a hundred thousand errors and it is one client retrying, or
one loop logging. Any statistic computed on the inflated count is confidently
wrong, and the correction is always the same: identify the independent unit -
the client, the incident, the book, the source - collapse within it, and count
those. The correction usually reduces your numbers by an order of magnitude and
the reduced numbers are the true ones.

**Where this breaks.** In telemetry you can usually recover the true unit
exactly: request IDs, trace IDs, client identifiers are all recorded. For
documents there is no identifier that says "these forty share an underlying
draw." You have to estimate the correlation from the data and defend the choice,
which means the number is arguable in a way a trace ID is not. State your choice
in the report, state what the answer becomes under the most hostile reasonable
alternative, and you have removed the attack by conceding it first.

## The discipline that converts a number into evidence

**The pair.** Uncorrected multiple testing is like **an alert that fires on
every host in a five-hundred-host fleet**, and like **a linter rule with a 5%
false-positive rate run across a million-line codebase**.

**The shared structure.** A per-unit error rate that is entirely acceptable in
isolation becomes a firehose when multiplied by the fleet. Nobody would ship a
check that is wrong one time in twenty and then run it fifty thousand times
without expecting two and a half thousand wrong results; put that way it is
obvious. Written as "p < 0.05" it stops being obvious, which is the only
difference. The fix is the same in both worlds: tighten the per-unit threshold in
proportion to how many units you are running it against, and accept the loss in
sensitivity that buys.

**Where this breaks.** Alert fatigue has cheap mitigations that do not exist
here: grouping, suppression windows, deduplication by signature. Those work
because you can afford to miss a duplicate of something you already know. An
audit cannot suppress: every document is a separate claim about a separate work,
and collapsing them is precisely the dependence error from the previous section.
So the correction has to be paid in full, in sensitivity, and the honest
consequence is that catalogue-wide sweeps mostly return nothing.

---

**The pair.** Pre-registration is like **writing the success criteria into the
canary-deploy config before the rollout**, and like **an RFC that states the
rejected alternatives before the implementation exists**.

**The shared structure.** All three remove degrees of freedom from a future
decision by spending them in public in advance. A canary whose promotion
threshold is chosen after looking at the metrics is not a canary, it is a
narrative; the config file with a timestamp is what makes the rollout decision
mean anything. Same with the RFC: the value is not in the document's prose, it
is that the reasoning is on the record before the outcome is known, so nobody
can claim afterward that this was the plan all along. In each case the artifact
is procedural rather than technical, and in each case it is worth far more than
any improvement to the thing it governs.

**Where this breaks.** An RFC can be amended, and amending it is usually healthy
- plans should respond to what you learn. A pre-registration that can be amended
after seeing the model provides exactly zero of its value, so the flexibility
that makes an RFC good is the property you must remove. The correct analogue is
narrower: a cryptographic commitment published with a timestamp, where the
amendment is visible as an amendment. That distinction is not pedantry; it is
the difference between a document and an exhibit, and it is the thing v7 sells.

## Where the null comes from

**The pair.** Constructing a null in advance is like **holding back a control
cell in a feature-flag rollout**, and like **chaos engineering with a matched
untouched cell**.

**The shared structure.** In all three you manufacture the comparison population
before you need it, by deliberately withholding something identical in every
respect except the treatment. The holdback cell is not an estimate of what
untreated users would do; it is untreated users. That is exactly what the
unpublished keyed siblings are: not a proxy for "text the model never saw," but
text the model never saw, generated by the same procedure from the same seed
space as the text it did see. The pairing is what removes the confounds that
destroyed every retrospective method in this unit, and any engineer who has run
a holdback already understands why.

**Where this breaks.** You own both cells in your own rollout, so you can assign
treatment, verify assignment, and audit the split. Here you own only one side.
You control what you publish; you do not control whether it gets scraped, when,
how many copies propagate, or whether the operator later fine-tunes on
paraphrases. So the design has to survive an assignment mechanism you cannot
observe, which is why the controls must be generated in bulk under a key rather
than chosen individually, and why the honest claim is about the collection of
planted items rather than about any one of them.

## At the bench: three measurements on your own model

**The pair.** Running two experiments you expect to fail is like **a load test
whose purpose is to find the knee**, and like **writing the failing test first**.

**The shared structure.** In each case the negative result is the deliverable,
and it is only a deliverable because it was designed. A load test that falls over
tells you the capacity number; a test that fails before the implementation exists
proves the test can detect the thing. The membership attack on your own model is
the same move: it is a designed failure that produces a number - the AUC
interval, the TPR at 1% FPR, the seed-flip rate - that you can quote for years.
An engineer who has watched a colleague "verify" a test that would have passed
against an empty implementation knows exactly why this matters.

**Where this breaks.** A failing unit test has one expected failure mode and you
can read the assertion to confirm it. An attack that comes out at chance might be
at chance because there is no signal, or because your setup is underpowered, or
because you shipped a bug - three explanations with identical output. That is why
the bench work needs the sizing calculation in advance: without it you cannot
distinguish "measured no effect" from "could not have measured an effect," and
only the first is a result. The load test analogy has the same requirement and
engineers already respect it there: nobody reports a capacity number from a load
generator that was itself the bottleneck.

## What you can now do

**The pair.** The habit this unit installs is like **the "how would we have
detected this" section of a postmortem**, and like **threat modelling before a
design review**.

**The shared structure.** Both ask what evidence would exist under each
hypothesis, before going to look for evidence. A postmortem that only explains
what happened is half a postmortem; the useful half asks what signal would have
been present, whether it was collected, and what a null reading of that signal
would have meant. Threat modelling does the same prospectively: enumerate the
adversary's paths, then ask what each one would leave behind. Applied here, the
three questions - could this have happened anyway, compared to what, and at what
operating point - are the same discipline pointed at an evidentiary claim
instead of at an outage.

**Where this breaks.** Threat modelling assumes an adversary with intent, and
most of what this unit examines has none: a training run that ingested a
publisher's catalogue was not attacking anyone, and framing it that way produces
bad predictions about behaviour. The postmortem analogy breaks on access - you
own the logs of your own incident and can add instrumentation for next time,
whereas here the system under investigation belongs to someone else and the only
instrumentation you can add is to the data you publish before it is collected.
That single constraint is why prospective evidence exists as a category, and it
is the whole of the next unit.
