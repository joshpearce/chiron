## Three Properties That Fail Independently

Picture a long row of dials, one per byte of your message. Each dial has 256 positions. Sending a byte means rotating its dial away from the true value by a secret random amount that only you and the receiver know. An eavesdropper sees the final position of every dial. Because each offset is random and unknown, a final position of 87 tells them nothing about where that dial started. The confidentiality here is real, not a fudge.

Now notice what the eavesdropper can still do without knowing a single offset: reach in and rotate one dial eight positions further. They do not know where it started. They do not know where it will land in plaintext terms either. They know one thing, which is enough: the *change* they introduced. The receiver removes the secret offset, and the plaintext byte comes out eight positions away from what the sender wrote.

That is the entire failure of the lockbox intuition, in one picture. This style of encryption is a **translation**: a rigid shift of everything by a secret amount. Shifts hide absolute positions and preserve differences. An attacker who cannot see any absolute value can still inject an exact, predictable difference.

The same picture with a physical instrument: suppose every temperature in a log has been recorded with an unknown constant added to it. You cannot recover any true temperature from the log. But if someone adds 5 to a logged number, the true temperature that the log now claims is exactly 5 higher than the real reading. The unknown constant cancels out of the change. Secrecy of the constant buys nothing against tampering, because tampering lives in the differences.

XOR is not literally a rotation of a dial, but it has the property that matters: the ciphertext is the plaintext offset by a secret pattern, and the pattern cancels out of any difference. XOR is the version of this where offsetting twice by the same pattern brings you home.

So think of three separate safeguards on a parcel, because that is what the three properties are:

- **Confidentiality** is opaque wrapping. Nobody can see the contents. It says nothing about whether the parcel can be opened, edited, and re-wrapped.
- **Integrity** is a tamper-evident seal. Break it and the receiver knows. This is a physically different device from the wrapping, and adding one does not add the other.
- **Authenticity** is your signet ring pressed into the wax. Anyone can put *a* seal on a parcel. Only you can put *your* seal on it.

Every combination occurs in the wild. A one-time pad is opaque wrapping with no seal at all. A signed but unencrypted git commit is a clear bag with your signet ring on it: everyone reads it, nobody can forge it. Neither property implies its neighbor.

The distinction that trips people is the difference between a seal that survives *accidents* and a seal that survives *an adversary*. A CRC32 is a contents count written on the outside of the box in pen. It catches the box being dropped, because damage does not bother to update the count. It catches nothing an attacker does, because the attacker owns a pen.

WEP tried to hide the pen by putting the count inside the wrapping. Two things go wrong. First, hiding a value an attacker can compute for themselves does not make it secret in any useful sense. Second, and worse, the checksum lived in the same shifted coordinate system as the data: the attacker computes the correction that their edit implies, injects the edit into the data region and the correction into the checksum region, and both survive the unwrapping together. The shadow was adjusted consistently with the object. A seal only helps if producing a matching seal requires something the attacker does not have, which means the seal must be keyed.

## Threat Models: What the Attacker Gets to Do

"This is secure" with no attacker named is like "this beam is strong enough" with no load named. A footbridge rated for pedestrians is not defective when a tank drives across it; it was specified against a different load. A threat model is the load rating.

Start from the strictest assumption, which is also the cheapest to defend: assume the attacker owns the blueprints. They have the algorithm, the mode, the padding rules, the source, the state machine. The only thing they lack is the key. This is not pessimism, it is bookkeeping. Anything you were quietly relying on being unknown is functioning as a key while getting none of a key's protection, and it will leak with the next binary, spec, or ex-employee.

With that fixed, attackers differ only in what they get to do to the traffic. It is a ladder of access:

- **Passive.** Someone reading postcards as they pass through the sorting office. They see everything and touch nothing.
- **Active.** Someone in the same sorting office who also holds a pen, a bin, and a stack of duplicate envelopes. They rewrite, drop, delay, reorder, and re-send. The tampering attack in the previous section needs exactly this and nothing more. Most real network positions are here: a rogue access point, a compromised middlebox, a hijacked route.
- **Chosen plaintext.** The attacker gets to slip their own postcards into your outgoing mail and watch the sealed versions come out. This sounds artificial until you notice that a web page can cause your browser to send a request whose URL the page chose, sealed in the same record as your session cookie. Attacker-chosen data sitting beside secret data under one key is the ordinary case.
- **Chosen ciphertext.** The attacker can additionally hand the receiving clerk a doctored envelope and watch the clerk's face. Accepted or rejected. Rejected quickly or rejected slowly. "The contents did not parse" versus "the seal was wrong." Any server that opens attacker-supplied envelopes and reacts observably differently depending on what it finds inside is handing out that information. The clerk's face is the leak, and it is the whole story behind the padding-oracle attacks later on.

The formal definition of confidentiality is a carnival test rather than a formula. A referee holds the key. You are allowed to hand over as many postcards as you like and inspect the sealed results, for as long as you like. Then you write two postcards of your own, the same size, and hand both over. The referee flips a coin, seals one of them, and gives you the sealed envelope. You say which one it was.

Guessing wins half the time by construction, so the only interesting quantity is your **margin above a coin flip**. Zero margin means the envelope told you nothing. A full margin means you win every time. The scheme passes if no realistic amount of work gets you a meaningful margin.

"Negligible" is a claim about a race between shrinking quantities. Some things shrink politely, like one over a large power of the input size; you can always outrun them by paying a bit more. Exponential decay does not merely shrink, it laps every polynomial budget you can name. In numbers you can feel: a margin of $2^{-30}$ is an afternoon of retries, so it is not negligible; a margin of $2^{-128}$ is a specific grain of sand among all the beaches on Earth, several times over, and no amount of retrying converts it into an attack.

The test has an immediate and brutal consequence for anything deterministic. If sealing the same postcard always produces the same envelope, then the envelope is a fingerprint of its contents. You fingerprint both of your candidate postcards yourself, using the free access you were granted, and then compare. You win every single time, with no cleverness and no weakness in the underlying cipher. This is why fresh randomness or a fresh nonce per message is not decoration bolted onto a cipher: without it, the scheme fails the definition of confidentiality outright. The same fact condemns ECB mode and textbook RSA later.

One concession is baked into the test: the two postcards must be the same size. You can make a letter unreadable; you cannot make it weightless. The sealed thing has a length, and that length tracks the original. Without the same-size rule you would submit a one-line note against a ten-page letter, weigh the result, and defeat every scheme ever built, including a one-time pad, which would make the definition describe nothing at all. So length is conceded up front and everything else is demanded. That concession is a live attack surface rather than a technicality: sizes alone have been enough to recover secrets from compressed-then-encrypted traffic and to identify which page someone loaded, the way a courier can often name a package's contents from its shape and weight. Hiding length takes a separate mechanism, which is why padding exists as an explicit facility.

## XOR and the One-Time Pad

Geometry first. Take three-bit strings and place each one at a corner of a cube: eight strings, eight corners, and moving along an edge means flipping one bit. Longer strings put you on a higher-dimensional cube with the same rule.

XOR with a fixed pattern is a **reflection of that cube**. The pattern names the axes to flip along, and every corner moves to another corner in the same way. Two facts fall out of the picture with no algebra:

- Applying the same reflection twice returns every corner to where it started. A mirror is its own inverse. This is why encryption and decryption are the same code path, and why nothing in a stream cipher needs an "undo" routine.
- A reflection is rigid. It relabels the corners without distorting the cube, which is the same difference-preserving structure that made the tampering attack work.

Now make the pattern secret and uniformly random: you pick one of the cube's reflections with every one equally likely. Whatever corner you start from, you land on a uniformly random corner. The destination is uniform *regardless of the starting point*, so the destination carries no trace of it.

That is the whole of perfect secrecy, and it is a counting statement rather than a hardness statement. Take the corner you observed and any candidate plaintext corner you care to name. There is exactly one reflection carrying that candidate to that observation, and every reflection was equally likely. So each candidate is exactly as consistent with the observation as every other candidate, and observing the ciphertext cannot rank them. Not "cannot rank them cheaply": there is no ranking to find. An attacker with infinite time and infinite hardware finishes exactly where they started, holding a list of candidates with their original probabilities untouched.

This is worth dwelling on because it is the only place in the corpus where that is true. Everything afterward is computational: the observation *does* single out the right answer, and the defense is that finding it costs more than the universe affords. A wall you cannot climb versus a wall that is not there.

The price is stated in the name, and each of the three conditions is load-bearing: the pad must be as long as the message, uniformly random, and used exactly once. The third is the one people negotiate with, so look at what happens when you do.

Two photographs taken through the same sheet of warped glass. Neither shows a usable scene. Lay one over the other and take the difference, and the warp appears in both copies identically, so it cancels; what survives is the difference between the two scenes. The glass is gone from the result and was never identified. Encrypt two messages under one pad and the attacker holds the difference of the two plaintexts with no key material involved anywhere.

A difference of two unknowns is not a plaintext, which is why this sounds survivable and is not. Anything you can guess about one message is an entry point into the other at that offset: a fixed header, a common word, a field name that has to be there. Guess a word, slide it along the difference like a transparency, and look for readable text falling out on the other side. Where it does, you have both messages at that spot, and the recovered fragments suggest the next guess. The pad was never recovered and was never needed.

One sentence ties this section to the first one. A secret offset hides absolute values and never protects differences. Tampering exploits that fact by injecting a difference; pad reuse exploits it by harvesting one.

## Modular Arithmetic and Groups, Refreshed

A clock face. Marks around a circle, a hand that steps forward, and no memory of how many times it has been round. Seventeen hours after midnight the hand points at 5, and 5 o'clock is not an approximation of 17: on a twelve-mark dial they are the same position. Reduction means "which mark," and congruence means "same mark, different number of laps."

Two consequences of the dial having no memory.

First, laps can be discarded whenever it is convenient, before or after any addition or multiplication, and the final mark comes out the same. A dial with seven marks does not care whether you walk 391 steps or notice partway that you have gone round and keep walking from the mark you are on. Both land on the same place, and only one of them ever built a big number. That is why public-key code never materializes an enormous intermediate value: it drops laps at every opportunity, and the numbers it holds stay about as wide as the dial.

Second, negative numbers are walking backwards. Three steps back from the top of a seven-mark dial lands on mark 4, so on that dial minus three and four are the same position. Several languages report the *lap-relative* answer instead and hand back a negative number for a negative input; the dial-position answer is what the math means, and turning one into the other means adding one full lap when the sign comes out wrong.

**Getting far around the dial cheaply.** Suppose your only cheap move is doubling what you already have. Starting from one step, doubling gives you 1, 2, 4, 8 steps and so on. To travel thirteen steps you combine the stones 8, 4, and 1. That is four cheap doublings and two combinations instead of thirteen single paces. The advantage is nothing at this size and everything at real sizes: a fold-and-double process reaches the Moon in around forty folds, and an exponent with two thousand bits needs about two thousand doublings where single paces would need more steps than there are atoms available to count them. Every large modular exponentiation you will read is this doubling ladder, with laps discarded after each rung so nothing grows.

**Groups** name the minimum structure that makes any of this safe to reason about. Think of the legal turns of a Rubik's cube. Do two of them and you have another legal state, never something off the manifold. "Do nothing" counts as a move. Every turn has an undo. And it does not matter how you parenthesize a sequence of turns, only their order. A dial is the same structure with one dial and one kind of step. There is no depth hiding behind the definition; it is a checklist that says the system is closed and every move is reversible.

**Generators** are about touring the dial. On a twelve-mark dial, step five marks at a time and you visit every mark before you return to the start. Step four at a time and you visit only four marks forever, a small orbit inside a bigger dial. A generator is a step size that tours everything.

Multiplying by a fixed value on a prime-sized dial is the same idea with the marks shuffled. Repeatedly multiplying by three on the dial of nonzero remainders mod seven visits every one of the six positions and then lands home. Picture that as a dial whose labels have been scrambled: the tour is complete, but consecutive stops sit nowhere near each other.

And the tour always lands home after exactly as many steps as there are positions, like a dance figure that returns dancers to their starting places on the last beat. That regularity is the reason two different dials appear in one formula in later units: the value being multiplied lives on the dial of all remainders, while the *count of steps* lives on the smaller dial of tour lengths. Two clocks with different faces in the same expression, which is a standard source of implementation bugs.

**The one-way street.** Walking the tour forward is the doubling ladder: cheap, even for enormous dials. Now the reverse question. Someone hands you a mark and asks how many steps produced it. On an ordinary clock you would estimate from the position and correct; the positions are ordered, so you can steer. On the shuffled multiplicative dial there is no such handle. Neighboring step counts land on unrelated marks, so nothing about the mark you are looking at tells you whether the true answer is larger or smaller than a guess. There is no ordering to bisect, no gradient to follow, no getting warmer. Stirring cream into coffee is easy in the forward direction for the same reason it is hopeless in reverse.

At toy size you win by trying all six possibilities, so nothing here is hard on a small dial. The hardness is purely a matter of scale, and it is the asymmetry that all of public-key crypto is built on. One note of typography for later: elliptic-curve groups write the same tour with additive language, so "multiply the point by a number of steps" replaces "raise to a power." Same dance, different name for the move.

## Keyspace, Bits of Security, and the Exponent

A combination padlock with dials on it. Each dial you add does not add its positions to the count of combinations, it multiplies. A bit is a dial with two positions, so every bit you add to a key doubles the number of combinations an attacker must work through.

That single sentence dissolves the linear intuition. The distance between a 128-bit key and a 256-bit key is not "twice," it is *one hundred and twenty-eight doublings*. Folding a sheet of paper doubles its thickness, and about forty folds reach the Moon; a hundred and twenty-eight is a number of doublings with no physical referent left to compare against.

Put hardware behind it and the shape of the problem shows up. Imagine a search farm running about a billion billion key trials every second, which is far beyond anything that exists. A 128-bit keyspace occupies that farm for something like seven hundred times the age of the universe. Nothing about that changes usefully with better engineering, and here is the cleanest way to see why: making the farm a thousand times faster removes about ten bits of the problem, because a thousand is roughly ten doublings. A million times faster removes twenty. Hardware progress advances along the exponent one small notch at a time, while adding key bits advances along it for free. The attacker is climbing a ladder you can extend faster than they can climb.

The 256-bit case is that seven-hundred-universes number, doubled another hundred and twenty-eight times. There is no meaningful sense in which one of these is twice the other.

**Bits of security are not key length.** The key length is the width of the vault door. The bits of security are the strength of whatever is actually weakest, and if there is a window beside the door then the window is the number. A cipher with a 128-bit key and a known shortcut costing a trillion operations offers about forty bits of security, no matter what the datasheet says about key width. Key length is a ceiling and never a promise. Unit 2 shows a single hash function with two different security levels for two different tasks, from the same output size, which settles the point: "bits" describes the best known attack, not the parameter.

**Entropy is the haystack you actually built, not the field you wrote it in.** Deriving a 128-bit key from an eight-character password gives you a value that is 128 bits wide and a search space of about fifty-two bits, because that is how many bits of dice were rolled. Picture a sixteen-digit combination lock where six of the digits are printed on the case: the lock is still sixteen digits wide and the attacker still only has to work through the rest. Key derivation functions do not roll additional dice. They make each guess expensive, taking a whole second instead of a microsecond, which is worth roughly twenty doublings. Twenty bits back is worth having and is nowhere near the gap being papered over.

**So why 256-bit keys, when 128 already outlasts the universe.** Three reasons, none of them "twice as strong."

First, the attacker often does not need *your* key. Searching for any one needle among a few billion independent haystacks is billions of times cheaper than searching one haystack for one specific needle, because every haystack you pass gives another chance to win. Many targets under separate keys eat directly into the margin.

Second, a large quantum computer would square-root the haystack rather than shrink it a bit, roughly halving the exponent. Halving 256 leaves a comfortable number. Halving 128 leaves one that a well-funded attacker can contemplate.

Third, attacks improve in one direction only. Spare doublings cost close to nothing today and cannot be retrofitted onto a deployed system tomorrow.

The standing convention for the rest of the corpus: 128 bits of security is the floor for anything shipping, and every "is this enough?" question becomes the same comparison of the best known attack cost against that floor.
