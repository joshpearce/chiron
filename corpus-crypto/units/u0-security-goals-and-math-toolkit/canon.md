---
unit: u0
title: Security Goals and the Math Toolkit
concepts:
  - c-cia-triad
  - c-threat-model
  - c-modular-arithmetic
  - c-xor-properties
  - c-key-entropy
assumes: []
---

This unit fixes vocabulary. Everything downstream - block ciphers, MACs, key
exchange, TLS - is built out of the five ideas here: three security properties
that fail independently, a language for describing attackers, XOR, modular
arithmetic, and the arithmetic of brute force. No crypto is assumed. Bit
manipulation, big-O reasoning, and protocol state machines are assumed, and are
leaned on hard.

## Three Properties That Fail Independently

<!-- refutes: M1 -->

You probably think that if an attacker cannot read your data, they cannot
meaningfully change it. Encryption feels like a lockbox: contents unreadable,
therefore contents untouchable.

Here is the specific prediction that model makes, and here is where it fails.

Most modern stream ciphers, and the counter mode you will meet in unit 1, work
like this. A key $k$ and a nonce are stretched into a **keystream** $K$, a
pseudorandom byte string as long as the message. The ciphertext is

$$C = M \oplus K$$

where $M$ is the plaintext (a byte string of length $L$), $K$ is the keystream
(same length $L$), and $\oplus$ is bitwise XOR applied byte by byte. The
receiver regenerates $K$ from $k$ and the nonce and computes $C \oplus K$ to get
$M$ back. Nothing here is broken; this is what AES-CTR does, and its
confidentiality is well founded.

Now suppose the plaintext is a wire instruction:

```
P A Y   0 1 0 0   U S D
0 1 2 3 4 5 6 7 8 9 ...
```

Byte 5 holds the ASCII character `'1'`, which is `0x31`. An attacker sitting on
the network sees only $C$. They do not have $k$, they cannot compute $K$, and
they cannot read a single byte of $M$. They do know the message format, because
formats are public.

```beat
id: u0-b1
type: predict
concept: c-cia-triad
prompt: |
  The attacker wants the recipient to decrypt "PAY 0900 USD" instead of
  "PAY 0100 USD". They cannot read the ciphertext and cannot derive the
  keystream $K$. Before reading on: can they do it, and if so, what exactly do
  they modify? State what the receiver observes.
answer: |
  Yes, with certainty and without touching the key. ASCII '1' is 0x31 and '9' is
  0x39, so the difference is the single mask byte 0x31 XOR 0x39 = 0x08. The
  attacker XORs 0x08 into ciphertext byte 5 in flight. On decryption:
  (C_5 XOR 0x08) XOR K_5 = (M_5 XOR K_5 XOR 0x08) XOR K_5 = M_5 XOR 0x08 = 0x39.
  The receiver observes nothing wrong - decryption succeeds and produces
  well-formed plaintext saying 0900.
rubric: |
  Must identify: (1) yes, the attack works, (2) the attacker XORs a mask into the
  ciphertext, (3) the flip propagates one-for-one into the plaintext with no error
  raised. Any 2 of 3 = pass. "They would need the key" or "the receiver would see
  garbage" = M1, fail. "It corrupts the rest of the message" = fail; XOR stream
  modes have no error propagation, each byte is independent.
check: llm
```

<!-- fade: xor-tamper-mask -->

The arithmetic, fully:

1. Target byte, current: `0x31` = `0011 0001`. Target byte, desired: `0x39` =
   `0011 1001`.
2. Mask $\delta = \texttt{0x31} \oplus \texttt{0x39} = \texttt{0000 1000} =
   \texttt{0x08}$.
3. Attacker replaces $C_5$ with $C_5 \oplus \delta$.
4. Receiver computes $(C_5 \oplus \delta) \oplus K_5$. Substituting
   $C_5 = M_5 \oplus K_5$ and reordering (XOR is commutative and associative):
   $M_5 \oplus K_5 \oplus \delta \oplus K_5 = M_5 \oplus \delta$ because
   $K_5 \oplus K_5 = 0$.
5. $M_5 \oplus \delta = \texttt{0x31} \oplus \texttt{0x08} = \texttt{0x39} =
   \texttt{'9'}$.

The keystream cancels itself out. The attacker never learns it and never needs
to. Confidentiality held completely - and the attacker still changed 100 to 900,
deterministically, with no error signal anywhere.

Why the wrong model is appealing: confidentiality and integrity both compress in
your head to "the attacker can't mess with my data," and one property
intuitively covers the other. It does not. What is actually true is that there
are three separate properties, each needing its own mechanism:

- **Confidentiality.** The ciphertext reveals nothing about the plaintext except
  its length. Mechanism: a cipher.
- **Integrity.** Any modification of the message is detected. Mechanism: a MAC
  or an AEAD mode (unit 2, unit 3).
- **Authenticity.** The message originated from the party you believe it did.
  Mechanism: a MAC (shared key) or a signature (public key, unit 5).

Integrity and authenticity are close relatives, and in practice the same
primitive delivers both: a MAC tag verifies only if the sender held the key, so
"unmodified" and "from the keyholder" arrive together. The distinction that
matters is that integrity against *noise* and integrity against an *adversary*
are different problems. A CRC32 detects random bit flips. It detects nothing an
attacker does, because a CRC is unkeyed - the attacker recomputes it.

```beat
id: u0-b2
type: self-explain
concept: c-cia-triad
prompt: |
  WEP (the original 802.11 encryption) appended a CRC32 checksum to the plaintext
  and then encrypted the whole thing with an XOR keystream, reasoning that an
  attacker could not tamper with a checksum they could not see. Explain in your
  own words why this fails. Two independent reasons exist; find at least one.
answer: |
  Reason one: CRC32 is unkeyed. Any property an attacker needs to satisfy that
  they can compute themselves is not a security property - it is a formality.
  Reason two, the one that actually killed WEP: CRC32 is linear over XOR, so
  CRC(M XOR d) = CRC(M) XOR CRC(d) for a same-length delta d. The cipher is also
  XOR. So an attacker who wants to flip bits d in the plaintext computes CRC(d)
  offline, XORs d into the data region of the ciphertext and CRC(d) into the
  checksum region of the ciphertext, and the decrypted message carries the
  attacker's edit with a perfectly valid checksum. No key, no plaintext knowledge.
  Integrity requires a keyed primitive whose output an attacker cannot predict or
  patch.
rubric: |
  Pass requires at least one of: (a) CRC is unkeyed so it certifies nothing an
  attacker cannot forge, (b) CRC is linear over XOR and the cipher is linear over
  XOR, so the checksum fix-up composes through the ciphertext. Full credit
  mentions both plus the conclusion that integrity needs a keyed primitive.
  "Encryption hides the checksum so the attacker cannot target it" = M1 exactly,
  fail. "CRC32 is too short" = partial only; length is not the flaw here, being
  unkeyed and linear is.
check: llm
```

These three properties fail independently in both directions. A one-time pad has
mathematically perfect confidentiality and zero integrity. A signed but
unencrypted git commit has perfect authenticity and zero confidentiality. Nothing
about having one gives you the other.

## Threat Models: What the Attacker Gets to Do

A claim like "this is secure" is meaningless without saying secure against whom,
with what access. A **threat model** is that specification. Crypto papers state
it as a game, and once you read the game as a state machine with the attacker
driving the inputs, the formalism stops being decorative.

**Kerckhoffs's principle.** Everything about the system is public except the key.
The algorithm, the mode, the padding, the source code, the protocol state
machine - assume the attacker has all of it. This is not paranoia; it is the only
assumption that survives a shipped binary or a leaked spec. Anything you were
relying on being unknown is not a key.

**Passive vs active.** A passive attacker observes traffic and nothing else. An
active attacker also injects, modifies, drops, reorders, and replays. The
tampering attack in the previous section requires an active attacker; the
one-time pad's perfect secrecy holds against an unbounded passive one. Most real
network positions (rogue AP, compromised middlebox, BGP hijack) are active.

**CPA, chosen-plaintext attack.** The attacker can obtain encryptions of
plaintexts of their choosing under the target key. This sounds like a
contrivance until you notice that any web request encrypts an attacker-chosen
URL path in the same TLS record as your session cookie. Attacker-controlled data
sitting next to secret data under one key is the normal case, not the exotic one.

**CCA, chosen-ciphertext attack.** The attacker can additionally submit
ciphertexts of their choosing and learn something about the result - whether
decryption succeeded, whether padding was well formed, how long the response
took. Any server that decrypts attacker-supplied bytes and behaves observably
differently on failure is a CCA oracle. This is the setting that produced the
padding-oracle attacks in unit 3.

**The IND-CPA game, from scratch.** IND means indistinguishability. The
definition is a game between a challenger holding the key $k$ and an adversary
$\mathcal{A}$:

1. The challenger generates $k$ at random and keeps it.
2. $\mathcal{A}$ may ask for $\mathrm{Enc}(k, m)$ for any plaintexts $m$ it
   likes, as many times as it likes. This is the chosen-plaintext oracle.
3. $\mathcal{A}$ submits two plaintexts $m_0$ and $m_1$ with
   $|m_0| = |m_1|$ (equal lengths in bytes).
4. The challenger flips a fair coin $b \in \{0, 1\}$ and returns the challenge
   ciphertext $c^* = \mathrm{Enc}(k, m_b)$.
5. $\mathcal{A}$ may keep querying the oracle, then outputs a guess
   $b' \in \{0,1\}$.

$\mathcal{A}$ wins if $b' = b$. Guessing blindly wins half the time, so the
meaningful quantity is the **advantage**:

$$\mathrm{Adv}(\mathcal{A}) = \left| \Pr[b' = b] - \tfrac{1}{2} \right|$$

where $\Pr[b' = b]$ is the probability the adversary guesses the coin correctly.
Advantage ranges from $0$ (learned nothing) to $1/2$ (always right). The scheme
is **IND-CPA secure** if every efficient $\mathcal{A}$ has negligible advantage.

**Negligible, from scratch.** A function $\varepsilon(n)$ of the security
parameter $n$ (think: key length in bits) is *negligible* if it shrinks faster
than the inverse of every polynomial - for any polynomial $p$, eventually
$\varepsilon(n) < 1/p(n)$. In big-O terms you are used to: $1/n^{100}$ is not
negligible, because it is beaten by the polynomial $n^{101}$; $2^{-n}$ is
negligible, because exponential decay outruns every polynomial. Concretely, an
advantage of $2^{-128}$ is negligible and an advantage of $2^{-30}$ is not,
because $2^{30}$ trials is an afternoon. "Efficient" means running in time
polynomial in $n$; concretely, bounded by something like $2^{80}$ to $2^{128}$
operations.

```beat
id: u0-b3
type: predict
concept: c-threat-model
prompt: |
  Let Enc be *deterministic*: encrypting the same plaintext under the same key
  always yields the same ciphertext, with no nonce or randomness anywhere. Before
  reading on: give a concrete adversary strategy in the IND-CPA game above, and
  state its advantage.
answer: |
  Strategy: pick any two distinct equal-length plaintexts m0, m1. First query the
  oracle on m0 and record the answer c0. Then submit (m0, m1) as the challenge
  pair and receive c*. If c* == c0, output b' = 0; otherwise output b' = 1.
  Determinism makes this always correct, so Pr[b'=b] = 1 and the advantage is
  |1 - 1/2| = 1/2, the maximum. No deterministic scheme can be IND-CPA secure,
  regardless of how strong the underlying cipher is.
rubric: |
  Must contain: (1) query the oracle on one of the two challenge plaintexts first,
  (2) compare the challenge ciphertext to the recorded oracle output, (3) advantage
  = 1/2, i.e. total break. Any 2 of 3 = pass. An answer claiming a strong enough
  block cipher rescues determinism is M3-flavoured (the cipher's strength does not
  transfer to the mode) and fails. An answer that only says "repeats leak" without
  the oracle query is partial credit.
check: llm
```

The consequence of b3 is structural and worth naming now: **encryption must be
randomized or nonced**. Fresh randomness per message is not an implementation
detail bolted onto a cipher; it is a requirement of the security definition
itself. This is the same fact that condemns ECB mode in unit 1 and textbook RSA
in unit 5.

```beat
id: u0-b4
type: self-explain
concept: c-threat-model
prompt: |
  The IND-CPA game requires $|m_0| = |m_1|$. Explain why the definition would be
  useless without that restriction, and name one real-world consequence of the
  leak the restriction concedes.
answer: |
  Standard encryption does not hide plaintext length: ciphertext length is a
  simple, usually affine, function of plaintext length. Without the equal-length
  requirement the adversary submits a 1-byte m0 and a 1000-byte m1, looks at
  len(c*), and wins with advantage 1/2 against every scheme in existence -
  including a one-time pad. The definition would be satisfiable by nothing and
  would therefore distinguish nothing. So the definition concedes length up front
  and demands secrecy of everything else. Real consequence: length leakage is a
  live attack surface - compression-plus-encryption side channels (CRIME, BREACH)
  recover secrets purely from ciphertext lengths, and traffic analysis identifies
  visited pages from response size patterns. Hiding length requires a separate
  mechanism (padding), which is why TLS 1.3 has a record padding facility.
rubric: |
  Pass requires: (1) ciphertext length reveals plaintext length, so unequal lengths
  give a trivial win, and (2) the recognition that this would make the definition
  vacuous / unsatisfiable rather than merely inconvenient. Full credit adds a real
  consequence (CRIME/BREACH, traffic analysis, or the need for explicit padding).
  An answer claiming encryption does hide length = fail. An answer claiming the
  restriction exists to be "fair to the scheme" without the vacuity point = partial.
check: llm
```

## XOR and the One-Time Pad

XOR is the operation the rest of the corpus is built on. You know it as `^`.
What matters here are its algebraic properties, written for single bits and
extended bitwise to strings of equal length:

- $a \oplus a = 0$ (self-cancelling)
- $a \oplus 0 = a$ (identity)
- $a \oplus b = b \oplus a$ (commutative)
- $(a \oplus b) \oplus c = a \oplus (b \oplus c)$ (associative)

Combining the first two gives the property everything depends on:
$(m \oplus k) \oplus k = m \oplus (k \oplus k) = m \oplus 0 = m$. XOR is its own
inverse. Encryption and decryption are the same code path.

The fifth property is the one that is not obvious from bit twiddling: if $k$ is
uniformly random on $\{0,1\}^n$ (every one of the $2^n$ strings equally likely)
and independent of $m$, then $m \oplus k$ is uniformly random on $\{0,1\}^n$,
*whatever $m$ is*. XOR with a uniform value destroys all structure.

<!-- fade: otp-encrypt-decrypt -->

Worked, on one byte. Plaintext $m = \texttt{0x6C}$ (ASCII `'l'`), key
$k = \texttt{0x3A}$:

```
encrypt:   m  0110 1100   (0x6C)
           k  0011 1010   (0x3A)
           c  0101 0110   (0x56)

decrypt:   c  0101 0110   (0x56)
           k  0011 1010   (0x3A)
           m  0110 1100   (0x6C)
```

Step by step, algebraically: $c = m \oplus k = \texttt{0x56}$. Then
$c \oplus k = (m \oplus k) \oplus k = m \oplus (k \oplus k) = m \oplus
\texttt{0x00} = \texttt{0x6C}$.

```beat
id: u0-b5
type: completion
concept: c-xor-properties
prompt: |
  Fill the blanks in this one-time pad round trip. Plaintext $m = \texttt{0x2F}$,
  key $k = \texttt{0x9D}$.

    step 1.  c = m XOR k = 0x2F XOR 0x9D = ____
    step 2.  c XOR k = (m XOR k) XOR k = m XOR (k XOR k) = m XOR ____
    step 3.  = ____

  Answer with the three missing values in order, each as two uppercase hex digits
  with a 0x prefix, comma-separated, no spaces. Example format: 0xAB,0xCD,0xEF
# variants: blank steps 1 and 3 only for a warmup version (mechanical XOR);
# blank step 2 alone for the concept-carrying version (k XOR k = 0 is what makes
# decryption work, and is the same cancellation the tampering attack in the first
# section exploits).
answer: "0xB2,0x00,0x2F"
rubric: |
  0x2F = 0010 1111, 0x9D = 1001 1101, XOR = 1011 0010 = 0xB2. Step 2 blank is
  0x00 because k XOR k = 0 for any k. Step 3 recovers 0x2F. Answering step 2 with
  0x9D indicates the reader thinks the second XOR reintroduces the key rather than
  cancelling it.
check: exact
```

**Perfect secrecy.** Fix any plaintext $m \in \{0,1\}^n$ and any ciphertext
$c \in \{0,1\}^n$. Exactly one key produces that pair, namely $k = m \oplus c$.
Since $k$ is uniform, that key has probability $2^{-n}$, so

$$\Pr[C = c \mid M = m] = 2^{-n} \quad \text{for every } m$$

The probability of seeing ciphertext $c$ is the same no matter which plaintext
was sent. Therefore observing $c$ changes nothing about your belief in $m$:
$\Pr[M = m \mid C = c] = \Pr[M = m]$. This is Shannon's perfect secrecy, and it
holds against an attacker with unlimited computing power and unlimited time.
It is the only scheme in this corpus with that property; everything after this is
computational, meaning secure only against attackers who cannot afford $2^{128}$
operations.

The price is stated in the name. The key must be (a) as long as the message,
(b) uniformly random, and (c) **used once**. Requirement (c) is not a stylistic
preference.

**Two-time pad.** Encrypt $m_1$ and $m_2$ under the same key $k$:

$$c_1 \oplus c_2 = (m_1 \oplus k) \oplus (m_2 \oplus k) = m_1 \oplus m_2 \oplus (k \oplus k) = m_1 \oplus m_2$$

The key vanishes. The attacker holds the XOR of the two plaintexts with no key
material involved at all. If any part of either plaintext is known or guessable -
a fixed header, an English word, a JSON key name - it peels the other message
open at that offset, and you iterate (crib-dragging). This is the exact failure
you will meet again as CTR nonce reuse in unit 1, where it is a total break
rather than a degradation.

```beat
id: u0-b6
type: compute
concept: c-xor-properties
prompt: |
  Two single-byte messages were encrypted under the same one-byte one-time pad
  key $k$. You capture $c_1 = \texttt{0x6D}$ and $c_2 = \texttt{0x49}$, and you
  learn from context that $m_1 = \texttt{0x41}$ (ASCII 'A'). Compute $m_2$ and
  give it as a decimal integer.
answer: 101
rubric: |
  c1 XOR c2 = 0x6D XOR 0x49 = 0x24 = m1 XOR m2. Then m2 = 0x24 XOR m1 =
  0x24 XOR 0x41 = 0x65 = 101 decimal (ASCII 'e'). Answering 36 (0x24) means the
  reader stopped at m1 XOR m2 and did not finish. Answering 109 (0x6D) or 65
  (0x41) means they never cancelled the key. Note that k was never recovered and
  was never needed.
check: numeric(0.5)
```

## Modular Arithmetic and Groups, Refreshed

Public-key crypto is arithmetic in finite sets. This section rebuilds the parts
of it that units 4 and 5 assume.

**Reduction.** $a \bmod n$ is the remainder of $a$ divided by $n$, always taken
in the range $[0, n)$. The mathematical convention differs from C, Java, and Go,
where `%` on a negative left operand returns a negative result: `-3 % 7` is `-3`
there, while $-3 \bmod 7 = 4$. The portable fix is `((a % n) + n) % n`. Python's
`%` already matches the mathematical convention.

**Congruence.** Write $a \equiv b \pmod n$ to mean $n$ divides $a - b$, that is,
$a$ and $b$ leave the same remainder. This partitions the integers into $n$
classes, and the set of those classes is written $\mathbb{Z}_n = \{0, 1, \ldots,
n-1\}$. Congruence is an equality of classes, not of integers: $17 \equiv 5
\pmod{12}$.

**Reduce whenever you like.** Addition and multiplication respect congruence:

$$(a \cdot b) \bmod n = \big( (a \bmod n) \cdot (b \bmod n) \big) \bmod n$$

So $17 \cdot 23 \bmod 12$ can be computed as $391 \bmod 12 = 7$, or as
$5 \cdot 11 \bmod 12 = 55 \bmod 12 = 7$. Same answer, and the second never built
the big number. This is the property that keeps intermediate values bounded in
every public-key implementation you will read.

<!-- fade: modular-exponentiation -->

**Modular exponentiation by repeated squaring.** Computing $g^e \bmod n$ by
multiplying $g$ into an accumulator $e$ times costs $O(e)$ multiplications, which
for a 2048-bit exponent is $2^{2048}$ operations and therefore not a plan. Square
repeatedly instead and cost drops to $O(\log e)$.

Compute $3^{13} \bmod 17$. Write the exponent in binary:
$13 = 8 + 4 + 1 = \texttt{1101}_2$.

1. $3^1 \equiv 3$
2. $3^2 = 9$
3. $3^4 = 9^2 = 81 = 4 \cdot 17 + 13 \equiv 13$
4. $3^8 = 13^2 = 169 = 9 \cdot 17 + 16 \equiv 16$
5. Multiply the pieces the exponent selects: $3^{13} = 3^8 \cdot 3^4 \cdot 3^1
   \equiv 16 \cdot 13 \cdot 3$
6. $16 \cdot 13 = 208 = 12 \cdot 17 + 4 \equiv 4$; then $4 \cdot 3 = 12$

So $3^{13} \equiv 12 \pmod{17}$. Four squarings and two multiplications instead
of twelve multiplications, and the gap widens exponentially with exponent size.
Every number in the trace stayed under $17^2$.

```beat
id: u0-b7
type: completion
concept: c-modular-arithmetic
prompt: |
  Fill the blanks. Compute $5^{11} \bmod 19$ by repeated squaring, using
  $11 = 8 + 2 + 1$.

    5^1 = 5
    5^2 = 25 mod 19 = 6
    5^4 = 6^2 = 36 mod 19 = ____
    5^8 = (previous)^2 mod 19 = ____
    5^11 = 5^8 * 5^2 * 5^1 = (that) * 6 * 5 mod 19 = ____

  Answer with the three missing values in order as integers in [0,19),
  comma-separated, no spaces.
# variants: blank the 5^4 and 5^8 lines only to drill the squaring chain;
# blank the final line only to drill the binary decomposition of the exponent,
# which is the step readers skip.
answer: "17,4,6"
rubric: |
  5^4 = 36 mod 19 = 17. 5^8 = 17^2 = 289 = 15*19 + 4 = 4. 5^11 = 4*6*5 = 120 =
  6*19 + 6 = 6. A wrong final value with correct intermediates usually means the
  reader decomposed 11 incorrectly (e.g. used 5^4 instead of 5^2) - the exponent
  bits, not the squaring, are the concept here.
check: exact
```

**Groups.** A group is a set $G$ with a binary operation satisfying four things:
closure (combining two elements of $G$ gives an element of $G$), associativity,
an identity element $e$ with $e \cdot a = a$ for all $a$, and an inverse for
every element ($a \cdot a^{-1} = e$). The **order** of the group is $|G|$, the
number of elements. That is the entire definition; there is no hidden depth to
recover here.

Two groups matter downstream:

- $(\mathbb{Z}_n, +)$, the integers mod $n$ under addition. Identity $0$, inverse
  of $a$ is $n - a$. Order $n$. Always a group.
- $(\mathbb{Z}_p^*, \times)$, the nonzero residues $\{1, 2, \ldots, p-1\}$ under
  multiplication mod $p$, for $p$ prime. Identity $1$. Order $p - 1$. Primality
  is what guarantees every element has a multiplicative inverse.

A group is **cyclic** with **generator** $g$ if repeatedly applying the operation
to $g$ produces every element. For $\mathbb{Z}_7^* = \{1,2,3,4,5,6\}$, try
$g = 3$:

$$3^1 = 3, \quad 3^2 = 2, \quad 3^3 = 6, \quad 3^4 = 4, \quad 3^5 = 5, \quad 3^6 = 1 \pmod 7$$

All six elements appear, so $3$ generates $\mathbb{Z}_7^*$. The powers cycle with
period $6 = |G|$, and they land back on the identity exactly at the group order.
That is not a coincidence: for prime $p$ and any $a$ not divisible by $p$,

$$a^{p-1} \equiv 1 \pmod p$$

which is Fermat's little theorem. Its practical consequence is that exponents
live mod $p-1$ while bases live mod $p$ - two different moduli in the same
formula, which is a standard source of implementation bugs.

**The discrete logarithm problem.** Given $p$, a generator $g$, and
$y = g^x \bmod p$, find $x$. Forward is $O(\log x)$ multiplications by repeated
squaring. Backward is believed hard for large $p$: for a 3072-bit prime, the best
known algorithms (index calculus, the number field sieve) run in *subexponential*
time, which is much better than brute force but still infeasible. In
$\mathbb{Z}_7^*$ you would find $x$ by trying six values, so nothing here is hard
at toy scale - the hardness is entirely a function of size. Unit 4 builds
Diffie-Hellman on exactly this asymmetry, and shows why elliptic-curve groups,
where the best known attack is *fully exponential*, need far smaller parameters
for the same strength.

One notational warning for unit 4: elliptic-curve groups are conventionally
written additively, so $g^x$ becomes $xG$ and "exponentiation" becomes "scalar
multiplication." Same group structure, different typography.

```beat
id: u0-b8
type: compute
concept: c-modular-arithmetic
prompt: |
  Compute $7^5 \bmod 13$ by repeated squaring. Give the result as an integer in
  $[0, 13)$.
answer: 11
rubric: |
  7^2 = 49 = 3*13 + 10 = 10. 7^4 = 10^2 = 100 = 7*13 + 9 = 9. 5 = 4 + 1, so
  7^5 = 9 * 7 = 63 = 4*13 + 11 = 11. Answering 10 or 9 means the reader stopped at
  an intermediate square. Answering 7 means the exponent decomposition was dropped
  entirely.
check: numeric(0.4)
```

## Keyspace, Bits of Security, and the Exponent

<!-- refutes: M2 -->

You probably think a 256-bit key is about twice as strong as a 128-bit key.
Doubling RAM, cores, or bandwidth buys roughly proportional benefit, so doubling
key length reads as a proportional security upgrade.

The prediction that model makes: brute-forcing AES-256 takes roughly twice as
long as brute-forcing AES-128. Here is what the arithmetic actually says.

<!-- fade: bits-of-security-work-factor -->

An $n$-bit key has a **keyspace** of $2^n$ possible values. Exhaustive search
tries them until one decrypts correctly, so the work factor is $2^n$ trials
(expected $2^{n-1}$ if you stop at the average - one bit of savings, which
changes nothing). Take an implausibly large attacker: a farm doing $2^{60}$ key
trials per second, about $10^{18}$ per second, far beyond anything that exists.

For $n = 128$:

1. Trials required: $2^{128}$.
2. Time in seconds: $2^{128} / 2^{60} = 2^{68}$ seconds.
3. Seconds in a year: about $3.15 \times 10^7 \approx 2^{25}$.
4. Time in years: $2^{68} / 2^{25} = 2^{43} \approx 9 \times 10^{12}$ years.

That is roughly 700 times the current age of the universe, on hardware that does
not exist. For $n = 256$ the same farm needs $2^{256}/2^{60} = 2^{196}$ seconds,
or $2^{171}$ years.

The ratio between the two is $2^{256} / 2^{128} = 2^{128} \approx 3.4 \times
10^{38}$. Not 2. Key length is the *exponent* in the attacker's work function, so
each additional bit doubles the attacker's cost: going from 128 to 129 bits
already achieves the "twice as strong" the wrong model attributes to going from
128 to 256.

```beat
id: u0-b9
type: completion
concept: c-key-entropy
prompt: |
  Fill the blanks. Three-key 3DES offers about 112 bits of security. Attacker farm
  runs at $2^{50}$ trials per second. Use $2^{25}$ seconds per year.

    step 1.  trials required                = 2^112
    step 2.  seconds = 2^112 / 2^50         = 2^____
    step 3.  years   = (that) / 2^25        = 2^____
    step 4.  cost ratio, 128-bit vs 112-bit key, same farm = 2^____

  Answer with the three missing exponents as integers in order, comma-separated,
  no spaces.
# variants: blank steps 2 and 3 only for a mechanical version (dividing powers of
# two subtracts exponents); blank step 4 alone for the concept-carrying version -
# the 16-bit difference being a 65536x cost difference is the whole point.
answer: "62,37,16"
rubric: |
  112 - 50 = 62 seconds-exponent. 62 - 25 = 37 years-exponent, about 1.4e11 years.
  Step 4 is 2^(128-112) = 2^16 = 65536x, not 128/112 = 1.14x. A step-4 answer
  expressed as a small ratio (1.14, or "about 14 percent stronger") is M2 exactly:
  treating the exponent as if it were the quantity being compared.
check: exact
```

**Bits of security is not key length.** A scheme has $b$ bits of security if the
*best known attack* costs about $2^b$ operations. Key length is an upper bound on
$b$, never a guarantee. A cipher with a 128-bit key and a $2^{40}$ cryptanalytic
shortcut has 40 bits of security. In unit 2 you will see hash functions where the
relevant bound is half the output size for one property and the full output size
for another, from the same function - concrete proof that "bits" is a property of
the attack, not of the parameter.

**Entropy of what you actually have.** If a key is drawn uniformly from a set
$\mathcal{K}$, its entropy in bits is

$$H = \log_2 |\mathcal{K}|$$

where $|\mathcal{K}|$ is the number of equally likely possibilities. For a
password of length $L$ drawn uniformly from an alphabet of $A$ symbols,
$|\mathcal{K}| = A^L$ and so $H = L \log_2 A$. An 8-character password over the
95 printable ASCII characters gives $8 \times \log_2 95 = 8 \times 6.57 = 52.6$
bits - and that is the *uniform* case, which human-chosen passwords are not. A
128-bit AES key derived from that password has 52.6 bits of entropy, not 128. The
key is 128 bits wide; the keyspace an attacker must search is $2^{52.6}$. Key
derivation functions (PBKDF2, scrypt, Argon2) do not create entropy - they raise
the per-guess cost, buying back maybe 20 bits of effective work factor, no more.

```beat
id: u0-b10
type: compute
concept: c-key-entropy
prompt: |
  A passphrase is generated by drawing 12 symbols uniformly at random from a
  64-symbol alphabet. How many bits of entropy does it have?
answer: 72
rubric: |
  H = L * log2(A) = 12 * log2(64) = 12 * 6 = 72 bits. Answering 64 or 12 means the
  reader reported a parameter instead of computing the entropy; answering 768
  (12*64) means multiplying by alphabet size rather than its log. Note that 72 bits
  is well short of the 128-bit floor, which is the point.
check: numeric(0.5)
```

**Why 256-bit keys exist at all**, given that 128 already outlasts the universe.
Three reasons, none of them "twice as strong":

- *Multi-target attacks.* Searching for any one key out of $N$ independently
  chosen targets costs about $2^n / N$, not $2^n$. Against $2^{32}$ targets, a
  128-bit key gives 96 bits of margin.
- *Grover's algorithm.* A large quantum computer would reduce exhaustive key
  search from $2^n$ to about $2^{n/2}$. That puts AES-256 at $2^{128}$ quantum
  work and AES-128 at $2^{64}$, which is uncomfortable.
- *Margin against cryptanalysis.* Attacks improve monotonically. Spare bits are
  cheap; a broken deployment is not.

The standing convention for the rest of this corpus: 128 bits of security is the
floor for anything shipping, and every "is this enough?" question resolves to
comparing $\log_2(\text{best known attack cost})$ against that floor.
