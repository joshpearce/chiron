Full derivations for the claims canon states and uses. Every symbol is defined
where it appears; nothing here requires you to recall an equation from an earlier
section.

## Three Properties That Fail Independently

**Malleability as an algebraic identity.** Let $\mathrm{Enc}(k,\nu,m) = m \oplus G(k,\nu)$, where $G$ is a keystream generator that stretches key $k$ and nonce $\nu$ into $L$ bytes, and $m \in \{0,1\}^{8L}$ is the plaintext. Decryption is $\mathrm{Dec}(k,\nu,c) = c \oplus G(k,\nu)$. For any attacker-chosen $\delta \in \{0,1\}^{8L}$:

$$\mathrm{Dec}(k,\nu,\, c \oplus \delta) = (m \oplus G(k,\nu)) \oplus \delta \oplus G(k,\nu) = m \oplus \delta$$

Decryption is *equivariant* under the group $(\{0,1\}^{8L}, \oplus)$: shifting the ciphertext by $\delta$ shifts the plaintext by the same $\delta$. That equivariance is the entire attack. It holds no matter how strong $G$ is, how long $k$ is, or whether $\delta$ was chosen before $c$ was ever observed. The key cancels for exactly the reason it cancels during honest decryption.

CBC is equivariant too, one block displaced. CBC decryption is $m_i = D_k(c_i) \oplus c_{i-1}$, where $D_k$ is the block cipher's inverse and $c_{i-1}$ is the previous ciphertext block. Replace $c_{i-1}$ with $c_{i-1} \oplus \delta$:

- block $i$ becomes $D_k(c_i) \oplus c_{i-1} \oplus \delta = m_i \oplus \delta$, a surgical edit under attacker control;
- block $i-1$ becomes $D_k(c_{i-1} \oplus \delta) \oplus c_{i-2}$, uncontrolled garbage.

So CBC trades one destroyed block for one precisely edited block. Confidentiality is untouched in both modes. This is the machinery behind the padding-oracle attacks of unit 3.

**Theorem: length preservation forbids integrity.** Define the ciphertext-integrity game INT-CTXT for a scheme $(\mathrm{Enc}, \mathrm{Dec})$ where $\mathrm{Dec}$ returns either a plaintext or the rejection symbol $\bot$:

1. The challenger samples key $k$.
2. The adversary $\mathcal{A}$ queries $\mathrm{Enc}(k,\cdot)$ on plaintexts of its choice; let $Q$ be the set of ciphertexts returned.
3. $\mathcal{A}$ outputs $c^* \notin Q$ and wins if $\mathrm{Dec}(k, c^*) \ne \bot$.

$\mathrm{Adv}^{\mathrm{INT}}(\mathcal{A}) = \Pr[\mathcal{A} \text{ wins}]$.

Claim: if for every length $L$ the encryption map is a bijection from $L$-bit plaintexts to $L$-bit ciphertexts, and $\mathrm{Dec}$ never returns $\bot$, then there is an adversary making one query with advantage exactly $1$. Proof: query any $m$, receive $c$, output $c^* = c \oplus \delta$ for any $\delta \ne 0$. Then $c^* \ne c$, so it is fresh, and $\mathrm{Dec}(k,c^*) \ne \bot$ by hypothesis. No cryptographic assumption appears in the proof, so no choice of cipher weakens it. A scheme that cannot say no detects nothing. Integrity requires ciphertext expansion.

**Theorem: tag length caps integrity, independent of key length.** Suppose ciphertexts carry a $t$-bit tag, and for each (nonce, body) pair exactly one tag verifies. A blind forger picks a tag uniformly and wins with probability $2^{-t}$; over $q$ independent verification attempts,

$$\Pr[\text{some forgery accepted}] = 1 - (1 - 2^{-t})^q \ge 1 - e^{-q 2^{-t}} \approx q\,2^{-t}$$

using $1 - x \le e^{-x}$. So integrity is bounded by $t$ bits whatever the key size: a 32-bit tag falls to a $2^{32}$-query attacker with probability about $0.63$, and $2^{32}$ verification attempts against an online service is minutes of traffic. This is why truncated AEAD tags come with hard query caps (unit 3) while nobody caps queries to protect a 128-bit key.

**Why CRC-32 composed through the cipher in WEP.** Model a message's bits $b_{L-1}\ldots b_0$ as the polynomial $M(x) = \sum b_i x^i$ over $\mathbb{F}_2$, the two-element field where addition is XOR. The pure CRC remainder map is

$$R(M) = \big(x^{32} M(x)\big) \bmod G(x)$$

with $G$ the fixed degree-32 generator polynomial. Reduction modulo $G$ is a ring homomorphism onto $\mathbb{F}_2[x]/G(x)$, and polynomial addition is bitwise XOR, so $R$ is $\mathbb{F}_2$-linear:

$$R(M \oplus d) = R(M) \oplus R(d)$$

Worked at toy scale with $G(x) = x^3 + x + 1$ (so $x^3 \equiv x+1$ in the quotient), 4-bit messages and 3-bit checks. Take $M = 1101$, i.e. $M(x) = x^3 + x^2 + 1$. Then $x^3 M = x^6 + x^5 + x^3$, and reducing term by term:

- $x^3 \equiv x + 1$
- $x^4 \equiv x(x+1) = x^2 + x$
- $x^5 \equiv x(x^2+x) = x^3 + x^2 \equiv x^2 + x + 1$
- $x^6 \equiv x(x^2+x+1) = x^3 + x^2 + x \equiv x^2 + 1$

Summing $x^6 + x^5 + x^3 \equiv (x^2+1) + (x^2+x+1) + (x+1) = 1$, so $R(1101) = 001$.

Now flip one bit with $d = 0100$, i.e. $d(x) = x^2$. Then $R(d) = x^5 \bmod G \equiv x^2+x+1 = 111$. Linearity predicts $R(1001) = 001 \oplus 111 = 110$. Check directly: $M' = 1001$ gives $x^3 M' = x^6 + x^3 \equiv (x^2+1) + (x+1) = x^2 + x = 110$. It matches.

Deployed CRC-32 (init `0xFFFFFFFF`, final XOR `0xFFFFFFFF`, reflected input) is affine rather than linear: $\mathrm{CRC}(M) = R(M) \oplus f(L)$ where the constant $f$ depends only on the message length $L$. For equal-length $M$ and $M' = M \oplus d$ that constant cancels:

$$\mathrm{CRC}(M \oplus d) = R(M) \oplus R(d) \oplus f(L) = \mathrm{CRC}(M) \oplus R(d)$$

and $R(d)$ is computable from $d$ and $L$ alone. WEP transmitted $C = (M \,\|\, \mathrm{CRC}(M)) \oplus K$ for keystream $K$. An attacker who wants the receiver to accept $M \oplus d$ transmits $C \oplus (d \,\|\, R(d))$. The decrypted data region is $M \oplus d$, the decrypted check region is $\mathrm{CRC}(M) \oplus R(d) = \mathrm{CRC}(M \oplus d)$, and verification passes. Two linear operations composed, no key, no plaintext knowledge, works for every $d$. Against HMAC the same plan has no first step: the attacker cannot evaluate the function at all, so there is no offline fix-up to compute.

**Integrity, authenticity, and the property a shared key cannot deliver.** MAC unforgeability, EUF-CMA: $\mathcal{A}$ gets a $\mathrm{Mac}(k,\cdot)$ oracle and wins by outputting $(m^*, \tau^*)$ with $m^*$ never queried and $\mathrm{Vrf}(k, m^*, \tau^*)$ accepting. Both integrity (nothing was altered) and authenticity (a keyholder produced it) follow from the same winning-event definition, which is why one primitive supplies both. What does not follow is *which* keyholder. With a two-party shared key either party can compute any tag, so a MAC gives no non-repudiation: Alice cannot prove to a third party that Bob authored $m$, because Alice could have produced the tag herself. Signatures (unit 5) split the capability to produce from the capability to check, and that asymmetry is what buys non-repudiation.

**Composition theorem: IND-CPA plus INT-CTXT gives IND-CCA.** For any chosen-ciphertext adversary $\mathcal{A}$ making $q_d$ decryption queries there exist adversaries $\mathcal{B}$, $\mathcal{C}$ with

$$\mathrm{Adv}^{\mathrm{CCA}}(\mathcal{A}) \le \mathrm{Adv}^{\mathrm{CPA}}(\mathcal{B}) + q_d \cdot \mathrm{Adv}^{\mathrm{INT}}(\mathcal{C})$$

Proof by simulation. $\mathcal{B}$ runs $\mathcal{A}$ and keeps a table of every $(m,c)$ pair it obtained from its own encryption oracle. When $\mathcal{A}$ submits $c$ for decryption, $\mathcal{B}$ answers from the table if $c$ is present and returns $\bot$ otherwise. This simulation is perfect unless $\mathcal{A}$ submits a fresh ciphertext whose true decryption is not $\bot$, and that event is precisely an INT-CTXT forgery, bounded by $q_d \mathrm{Adv}^{\mathrm{INT}}$. Conditioned on the event not happening, $\mathcal{A}$'s view is exactly a CPA adversary's view. This theorem is the formal reason encrypt-then-MAC is the default construction: CCA security drops out of a CPA-secure cipher plus a secure MAC with no new assumption about either.

## Threat Models: What the Attacker Gets to Do

**Concrete security, which is what you deploy against.** A scheme is $(t, q, \sigma, \varepsilon)$-secure if every adversary running in time $t$, making at most $q$ queries totalling $\sigma$ blocks, has advantage below $\varepsilon$. Asymptotic negligibility is a summary of that table. The table is the operational artifact: AES-GCM's proof yields a bound of the shape $\varepsilon \approx \sigma^2/2^{128} + q_v \cdot 2^{-t}$ for $\sigma$ blocks encrypted and $q_v$ verification attempts against a $t$-bit tag, and the widely quoted "rekey before $2^{64}$ blocks" rule is that first term crossing an acceptable threshold, not folklore.

**The two lemmas that license every proof in the corpus.** A function $\varepsilon(n)$ is negligible if for every polynomial $p$ there is an $N$ with $\varepsilon(n) < 1/p(n)$ for all $n > N$.

*Lemma 1 (closure under addition).* If $\varepsilon_1, \varepsilon_2$ are negligible so is $\varepsilon_1 + \varepsilon_2$. Proof: given a polynomial $p$, apply the definition to the polynomial $2p$ to get $N_1$ with $\varepsilon_1(n) < 1/(2p(n))$ for $n > N_1$, and $N_2$ likewise for $\varepsilon_2$. For $n > \max(N_1,N_2)$, the sum is below $1/p(n)$.

*Lemma 2 (closure under polynomial multiples).* If $\varepsilon$ is negligible and $p$ is a polynomial, $p \cdot \varepsilon$ is negligible. Proof: given a target polynomial $q$, note $pq$ is a polynomial, so eventually $\varepsilon(n) < 1/(p(n)q(n))$, hence $p(n)\varepsilon(n) < 1/q(n)$.

Lemma 2 is the entire content of a hybrid argument: if one message leaks $\varepsilon$, then $q$ messages leak at most $q\varepsilon$, and for polynomial $q$ that is still negligible. Every multi-message claim downstream is Lemma 2 applied to a single-message bound.

**Determinism: the advantage is exactly $1/2$, derived.** Let $\mathrm{Enc}$ be a function of $(k,m)$ with no randomness or state. Pick $m_0 \ne m_1$ with $|m_0| = |m_1|$. Query the oracle on $m_0$ to get $c_0 = \mathrm{Enc}(k,m_0)$, submit $(m_0,m_1)$, receive $c^* = \mathrm{Enc}(k,m_b)$, and output $b' = 0$ if $c^* = c_0$, else $b' = 1$.

- If $b = 0$ then $c^* = \mathrm{Enc}(k,m_0) = c_0$ with probability 1, since $\mathrm{Enc}$ is a function of its inputs. Output is correct.
- If $b = 1$ then $c^* \ne c_0$, because for a fixed key $\mathrm{Enc}(k,\cdot)$ must be injective (otherwise two distinct plaintexts share a ciphertext and correct decryption is impossible), and $m_1 \ne m_0$. Output is correct.

So $\Pr[b'=b] = 1$ and $\mathrm{Adv} = |1 - 1/2| = 1/2$, the maximum. Note where injectivity was used: correctness of decryption forces it, so no cipher design can evade the argument. The requirement for fresh randomness or a nonce is a consequence of the definition, not an implementation preference.

**Randomized IVs push the failure out to the birthday bound, and here is the bound.** Model the block cipher $E_k$ on $b$-bit blocks as a random permutation. Message $j$ draws a uniform IV $\nu_j \in \{0,1\}^b$ and consumes counter values $\nu_j, \nu_j+1, \ldots, \nu_j + \ell - 1 \pmod{2^b}$ for a message of at most $\ell$ blocks. Keystream blocks look independent and random as long as no counter value is used under two different messages. Messages $i \ne j$ overlap exactly when $\nu_i - \nu_j \bmod 2^b$ lands in the window $\{-(\ell-1), \ldots, \ell-1\}$, which contains $2\ell - 1$ values, so for a fixed pair

$$\Pr[\text{overlap}] = \frac{2\ell - 1}{2^b}$$

Union bound over the $\binom{q}{2}$ pairs among $q$ messages:

$$\Pr[\text{any overlap}] \le \frac{q(q-1)}{2}\cdot\frac{2\ell-1}{2^b} < \frac{q^2 \ell}{2^b}$$

Absent overlap the output is indistinguishable from random up to the PRP advantage of $E_k$, giving

$$\mathrm{Adv}^{\mathrm{IND\text{-}CPA}} \le \mathrm{Adv}^{\mathrm{PRP}}(E) + \frac{q^2\ell}{2^b}$$

Numbers, $b = 128$, $\ell = 2^{16}$ (1 MiB messages), $q = 2^{32}$ messages: $2^{64}\cdot 2^{16}/2^{128} = 2^{-48}$, negligible. The same workload with $b = 64$ (3DES, Blowfish): $2^{64}\cdot2^{16}/2^{64} = 2^{16}$, a bound above 1, meaning no guarantee whatsoever. That is the Sweet32 arithmetic, and note which parameter drives it: the block size $b$, not the key size. 3DES's 168-bit key does not appear in the bound at all.

**CCA is a different order of magnitude, not a nuance.** Take CBC without integrity, where $m_i = D_k(c_i) \oplus c_{i-1}$, and a server that leaks one bit per submitted ciphertext (padding well formed or not). To recover the final byte of $m_i$, submit $c_{i-1} \oplus g$ for each of the 256 values $g$ in the last byte position until the padding check passes; that pins the last byte of $D_k(c_i)$, hence the plaintext byte. Cost is at most 256 queries per byte and 128 on average, so a 16-byte block costs about $2^{11}$ queries and a 1 KiB record about $2^{17}$. Set that against $2^{128}$ for key search. The key length is irrelevant to the comparison; the absent integrity check is the whole distance between $2^{128}$ and $2^{17}$. This is M9 stated quantitatively, and it is why CCA security rather than CPA security is the deployment target.

**Left-or-right versus real-or-random.** The IND-CPA game in canon is left-or-right: the challenger encrypts one of two adversary-chosen plaintexts. The real-or-random variant returns either $\mathrm{Enc}(k,m)$ for the adversary's chosen $m$ or $\mathrm{Enc}(k,r)$ for a uniform $r$ of the same length. The two are equivalent up to a factor 2: $\mathrm{Adv}^{\mathrm{ROR}} \le \mathrm{Adv}^{\mathrm{LOR}}$, and $\mathrm{Adv}^{\mathrm{LOR}} \le 2\,\mathrm{Adv}^{\mathrm{ROR}}$ by a two-step hybrid (replace $\mathrm{Enc}(m_0)$ with $\mathrm{Enc}(r)$, then $\mathrm{Enc}(r)$ with $\mathrm{Enc}(m_1)$, each step costing one ROR advantage). A factor 2 is one bit of security, which is why papers move between the definitions without comment.

**Kerckhoffs's principle, quantified.** If a design hides $s$ bits of structure on top of an $n$-bit key, the nominal work factor is $2^{n+s}$ and collapses to $2^n$ the moment a binary is disassembled or a spec leaks. The asymmetry that matters is not the size of $s$ but its lifetime: a key is rotatable, so a compromise costs you one session, while design secrecy is not renewable, so its compromise is retroactive across every past and future session under every key. That is why Kerckhoffs is a design rule rather than a statement of modesty.

## XOR and the One-Time Pad

**The structure: $\{0,1\}^n$ is a vector space over $\mathbb{F}_2$.** $\mathbb{F}_2 = \{0,1\}$ with addition = XOR and multiplication = AND is the two-element field. Then $(\{0,1\}^n, \oplus)$ is the vector space $\mathbb{F}_2^n$: vector addition is bitwise XOR and the only scalars are 0 and 1. Two consequences carry real weight.

First, every element is its own inverse ($a \oplus a = 0$), so every non-identity element has order 2. The group is elementary abelian, isomorphic to $(\mathbb{Z}_2)^n$, and for $n \ge 2$ it is not cyclic: a cyclic group of order $2^n$ contains an element of order $2^n$, and here nothing has order above 2. So "XOR is addition" holds only in the carry-free sense - it is addition in $\mathbb{Z}_2^n$, not in $\mathbb{Z}_{2^n}$. Every primitive that interleaves $+$ mod $2^{32}$ with $\oplus$ (ChaCha, SHA-2, the whole ARX family) is deliberately alternating between two incompatible group structures on the same bits, and that incompatibility is where the nonlinearity comes from.

Second, an $\mathbb{F}_2$-linear map is exactly a bit matrix, and any system of $\mathbb{F}_2$-linear equations in $n$ unknowns is solvable by Gaussian elimination in $O(n^3)$ operations. So no cipher can be $\mathbb{F}_2$-linear in the key or the plaintext. CRC is linear, which is exactly why it is not a MAC. AES's S-box exists to destroy this linearity.

**Lemma (uniform absorption).** If $K$ is uniform on $\{0,1\}^n$ and independent of $M$, then $C = M \oplus K$ is uniform on $\{0,1\}^n$ and independent of $M$. Proof: condition on $M = m$. The map $\varphi_m(x) = x \oplus m$ is a bijection on $\{0,1\}^n$ (it is its own inverse), so for any target $c$,

$$\Pr[C = c \mid M = m] = \Pr[K = c \oplus m] = 2^{-n}$$

which does not depend on $m$. A conditional distribution identical for every $m$ means $C$ is independent of $M$, and it is uniform. The only property of $\oplus$ used is that $x \mapsto x \oplus m$ is a bijection, so the identical proof works over any group: a one-time pad over $\mathbb{Z}_{26}$ with a uniform key is equally perfect.

**Perfect secrecy, full Bayes.** Definition: a scheme has perfect secrecy if for every plaintext $m$ and every ciphertext $c$ with $\Pr[C=c] > 0$, $\Pr[M = m \mid C = c] = \Pr[M = m]$ - the ciphertext leaves the prior unchanged. For the one-time pad, start from the lemma's $\Pr[C=c \mid M=m] = 2^{-n}$ for every $m$, and marginalise:

$$\Pr[C = c] = \sum_{m'} \Pr[M=m']\Pr[C=c\mid M=m'] = 2^{-n}\sum_{m'}\Pr[M=m'] = 2^{-n}$$

Then Bayes:

$$\Pr[M=m \mid C=c] = \frac{\Pr[C=c\mid M=m]\Pr[M=m]}{\Pr[C=c]} = \frac{2^{-n}\Pr[M=m]}{2^{-n}} = \Pr[M=m]$$

No computational assumption appears anywhere in that derivation: no efficient adversary, no PRF, no hard problem. It is the only unconditional security result in the corpus. In information-theoretic form, with Shannon entropy $H(X) = -\sum_x \Pr[X=x]\log_2 \Pr[X=x]$ measured in bits, the equality of posterior and prior for every $c$ gives $H(M \mid C) = H(M)$, hence mutual information $I(M;C) = H(M) - H(M\mid C) = 0$.

**Theorem (Shannon's bound: the key is at least as long as the message).** If a scheme has perfect secrecy then $|\mathcal{K}| \ge |\mathcal{M}|$, where $\mathcal{K}$ is the key set and $\mathcal{M}$ the set of plaintexts, all with nonzero probability. Proof. First note perfect secrecy is equivalent to $\Pr[C=c \mid M=m]$ being the same for all $m$, which is what the Bayes step above used. Fix any $c$ with $\Pr[C=c] > 0$, and let $S = \{\mathrm{Dec}(k,c) : k \in \mathcal{K}\}$ be the set of plaintexts that $c$ could have come from, so $|S| \le |\mathcal{K}|$. Suppose $|\mathcal{K}| < |\mathcal{M}|$. Then $|S| < |\mathcal{M}|$, so some $m_0 \in \mathcal{M}$ lies outside $S$, giving $\Pr[C=c \mid M=m_0] = 0$. But $\Pr[C=c] > 0$ forces some $m_1$ with $\Pr[C=c\mid M=m_1] > 0$. Equality across all $m$ then says $0 = \Pr[C=c\mid M=m_1] > 0$, a contradiction. Hence $|\mathcal{K}| \ge |\mathcal{M}|$, and taking $\log_2$, the key is at least as many bits as the message. The entropy version is $H(K) \ge H(M)$.

This theorem is why everything after this section is computational. Perfect secrecy is achievable and unusable at scale (a 1 GiB key per 1 GiB message, never reused), so practical schemes replace "the ciphertext distribution is identical for all plaintexts" with "the ciphertext distributions are indistinguishable to an adversary bounded by $2^{128}$ operations". The whole subject is that one substitution and its consequences.

**The one-time pad has advantage 1 in the integrity game.** Apply the equivariance identity from the first section with $G$ replaced by the pad: flipping any ciphertext bit flips the corresponding plaintext bit, and decryption never rejects. So a single scheme simultaneously holds the strongest possible confidentiality result in the field and zero integrity - the sharpest available demonstration that the properties are independent. Note also that the pad is not IND-CPA secure as canon's game is written, because the game's oracle reuses the key; the "one-time" qualifier is a restriction on the adversary's access, not only on the user's discipline.

**Two-time pad: exactly how much leaks.** With one uniform key $k$, $c_1 = m_1 \oplus k$ and $c_2 = m_2 \oplus k$, so $c_1 \oplus c_2 = m_1 \oplus m_2$ deterministically. Conditioned on the value of $m_1 \oplus m_2$, the pair $(c_1,c_2)$ is uniform over the $2^n$ pairs consistent with it, so the joint distribution of the ciphertexts given the plaintexts depends on the plaintexts only through their XOR. Therefore the leakage is exactly

$$I\big((M_1,M_2);(C_1,C_2)\big) = H(M_1 \oplus M_2)$$

and nothing beyond it. For two $n$-bit messages ($2n$ bits of payload) up to $n$ bits leak, and they are the correlated bits, which is what makes the failure fatal in practice rather than merely half-bad.

Crib dragging, with its false-positive rate. Guess a $w$-byte crib $g$ at offset $j$ in $m_1$; the implied candidate is $\hat{m}_2[j..j{+}w) = (c_1 \oplus c_2)[j..j{+}w) \oplus g$. Filter candidates by plausibility, for instance "all bytes printable ASCII" (95 of 256 byte values). A wrong guess yields effectively random bytes, so

$$\Pr[\text{wrong guess survives}] \approx (95/256)^w = 0.371^w$$

and over $L$ offsets the expected number of false positives is $L \cdot 0.371^w$. For $L = 1000$ and $w = 5$: about 7 survivors to inspect by hand. For $w = 10$: $1000 \times 4.9\times10^{-5} \approx 0.05$. Recovery is a filtering problem whose noise decays exponentially in crib length, which is why "the plaintexts were compressed, or binary, or of unknown format" buys close to nothing.

## Modular Arithmetic and Groups, Refreshed

**$\mathbb{Z}_n$ is a ring, and "reduce whenever you like" is a homomorphism.** Define the congruence class $[a] = \{a + tn : t \in \mathbb{Z}\}$, and $\mathbb{Z}_n = \{[0], [1], \ldots, [n-1]\}$, with $[a]+[b] = [a+b]$ and $[a][b] = [ab]$. Those definitions need a proof that they do not depend on the representative chosen. Suppose $a' = a + sn$ and $b' = b + tn$. Then

$$a' + b' = a + b + (s+t)n \equiv a+b \pmod n$$
$$a'b' = ab + n(at + bs + stn) \equiv ab \pmod n$$

So both operations are well defined on classes, and the reduction map $\mathbb{Z} \to \mathbb{Z}_n$, $a \mapsto a \bmod n$, is a ring homomorphism. That single fact is the licence to reduce at any intermediate point, and it is what keeps a bignum library's temporaries from doubling in width at every multiplication.

**Inverses exist exactly when $\gcd = 1$, and extended Euclid produces them.** Claim: $[a]$ has a multiplicative inverse in $\mathbb{Z}_n$ if and only if $\gcd(a,n) = 1$. Proof via Bezout's identity, which states that there exist integers $u,v$ with $au + nv = \gcd(a,n)$. If $\gcd(a,n)=1$ then $au \equiv 1 \pmod n$, so $u$ is the inverse. Conversely, if $au \equiv 1 \pmod n$ then $au - 1 = -vn$ for some $v$, so $au + nv = 1$, and any common divisor of $a$ and $n$ divides 1.

Extended Euclid computes $(u,v)$ by running the division chain and back-substituting. Inverse of 5 mod 19:

```
19 = 3*5 + 4
 5 = 1*4 + 1
 4 = 4*1 + 0     ->  gcd = 1
```

Back-substitute from the last nontrivial remainder:

$$1 = 5 - 1\cdot 4 = 5 - 1\cdot(19 - 3\cdot 5) = 4\cdot 5 - 1\cdot 19$$

So $4 \cdot 5 \equiv 1 \pmod{19}$ and $5^{-1} = 4$ (check: $20 = 19+1$). Cost is $O(\log n)$ divisions, comparable to a handful of modular multiplications, which is why inversion is affordable inside key generation and signature verification.

The invertible elements of $\mathbb{Z}_n$ form a group $\mathbb{Z}_n^*$ under multiplication, of order $\varphi(n)$ (Euler's totient: the count of $a \in [1,n)$ with $\gcd(a,n)=1$). For prime $p$ every nonzero residue is coprime to $p$, so $|\mathbb{Z}_p^*| = p-1$. For $n = pq$ with distinct primes, $\varphi(n) = (p-1)(q-1)$, the identity RSA key generation is built on. Example: $n=15$, $\varphi(15)=8$, $\mathbb{Z}_{15}^* = \{1,2,4,7,8,11,13,14\}$.

**Lagrange's theorem, and why exponents live modulo the group order.** Let $G$ be a finite group with subgroup $H$. For $a \in G$ the coset $aH = \{ah : h \in H\}$ has exactly $|H|$ elements, since $h \mapsto ah$ is injective ($ah_1 = ah_2$ implies $h_1 = h_2$ after multiplying by $a^{-1}$). Two cosets are equal or disjoint: if $x \in aH \cap bH$ then $x = ah_1 = bh_2$, so $a = bh_2h_1^{-1}$ and, by closure of $H$, $aH = bH$. The cosets therefore partition $G$ into blocks of equal size $|H|$, giving $|G| = (\text{number of cosets})\cdot|H|$, so $|H|$ divides $|G|$.

Apply this to $H = \langle a\rangle = \{a^0, a^1, \ldots, a^{d-1}\}$, where $d = \mathrm{ord}(a)$ is the least positive integer with $a^d = e$ (the identity). Lagrange gives $d \mid |G|$, hence $a^{|G|} = (a^d)^{|G|/d} = e$. Three consequences used constantly downstream:

- **Fermat's little theorem.** In $\mathbb{Z}_p^*$, $|G| = p-1$, so $a^{p-1}\equiv 1 \pmod p$ for every $a \not\equiv 0$.
- **Euler's theorem.** In $\mathbb{Z}_n^*$, $a^{\varphi(n)} \equiv 1 \pmod n$ whenever $\gcd(a,n)=1$. RSA's correctness, $m^{ed} \equiv m \pmod n$ when $ed \equiv 1 \pmod{\varphi(n)}$, is this theorem and nothing else.
- **Exponent reduction.** $a^x = a^{x \bmod d}$ with $d = \mathrm{ord}(a)$, and safely $a^x = a^{x \bmod |G|}$ if only $|G|$ is known. Bases reduce mod $n$; exponents reduce mod $\varphi(n)$ or mod the element's order. Two moduli in one formula, which is the bug canon warns about.

A direct proof of Fermat, because it exposes the mechanism rather than quoting a partition argument: for $a \not\equiv 0 \pmod p$ the map $x \mapsto ax \bmod p$ is injective on $\{1,\ldots,p-1\}$ (multiply by $a^{-1}$), and an injective self-map of a finite set is a bijection. So $\{a\cdot1, a\cdot2, \ldots, a(p-1)\}$ is a permutation of $\{1,\ldots,p-1\}$ modulo $p$. Multiplying each list out gives

$$a^{p-1}(p-1)! \equiv (p-1)! \pmod p$$

and $(p-1)!$ is a product of nonzero residues modulo a prime, hence invertible, so cancel it to get $a^{p-1}\equiv 1$. Worked with $p=7$, $a=3$: the list $3,6,2,5,1,4$ is a permutation of $1..6$, and indeed $3^6 = 729 = 104\cdot7 + 1 \equiv 1$.

**Element order in practice, cross-checked against canon's example.** In $\mathbb{Z}_{19}^*$, $|G| = 18$, so by Lagrange $\mathrm{ord}(5) \in \{1,2,3,6,9,18\}$. Compute:

- $5^2 = 25 \equiv 6$
- $5^3 = 5\cdot6 = 30 \equiv 11$
- $5^6 = 11^2 = 121 = 6\cdot19 + 7 \equiv 7$
- $5^9 = 5^6\cdot5^3 = 7\cdot11 = 77 = 4\cdot19 + 1 \equiv 1$

So $\mathrm{ord}(5) = 9$ (it is not 3, since $5^3 = 11 \ne 1$), and 5 is not a generator: its powers reach only 9 of the 18 elements. Now redo canon's $5^{11} \bmod 19$ with no squaring at all: $11 \bmod 9 = 2$, so $5^{11} = 5^9\cdot5^2 \equiv 1\cdot6 = 6$. Same answer as the repeated-squaring trace, reached by a different route, which is a useful way to check an implementation.

Generators: $a$ generates $\mathbb{Z}_p^*$ iff $\mathrm{ord}(a) = p-1$, and you test that without enumerating anything by checking $a^{(p-1)/q} \ne 1$ for each prime $q$ dividing $p-1$. For $p=19$, $p-1 = 18 = 2\cdot3^2$, so test exponents $18/2 = 9$ and $18/3 = 6$. For $a=2$: $2^9 = 512 = 26\cdot19 + 18 \equiv -1 \ne 1$ and $2^6 = 64 = 3\cdot19 + 7 \equiv 7 \ne 1$, so 2 is a generator. The number of generators is $\varphi(p-1) = \varphi(18) = 6$, so a random element generates about a third of the time. Deployed Diffie-Hellman avoids this bookkeeping by working in a subgroup of large prime order $q \mid p-1$, where every non-identity element has order exactly $q$.

**Square-and-multiply: correctness proof, trace, and cost.** Write $e = \sum_{i=0}^{t} e_i 2^i$ with bits $e_i \in \{0,1\}$ and $e_t = 1$. Left-to-right:

```
acc = 1
for i = t down to 0:
    acc = acc * acc mod n              # square
    if e_i == 1: acc = acc * g mod n    # multiply
```

Loop invariant: after the iteration for index $i$, $acc \equiv g^{\lfloor e/2^i\rfloor} \pmod n$. Proof by downward induction. Before the loop (index $t+1$), $\lfloor e/2^{t+1}\rfloor = 0$ and $acc = 1 = g^0$. Assume $acc = g^{\lfloor e/2^{i+1}\rfloor}$ on entering iteration $i$. Squaring gives $g^{2\lfloor e/2^{i+1}\rfloor}$, and the conditional multiply gives $g^{2\lfloor e/2^{i+1}\rfloor + e_i}$. The integer identity $\lfloor e/2^i\rfloor = 2\lfloor e/2^{i+1}\rfloor + e_i$ (dividing by $2^i$ is a shift; $e_i$ is the bit shifted in) makes that exactly $g^{\lfloor e/2^i\rfloor}$. At $i=0$, $\lfloor e/1\rfloor = e$.

Full trace for canon's example, $g=3$, $e = 13 = 1101_2$, $n=17$, computed left-to-right rather than right-to-left:

| $i$ | $e_i$ | after square | after multiply | invariant value |
|---|---|---|---|---|
| 3 | 1 | $1^2 = 1$ | $1\cdot3 = 3$ | $g^{\lfloor 13/8\rfloor} = g^1 = 3$ |
| 2 | 1 | $3^2 = 9$ | $9\cdot3 = 27 \equiv 10$ | $g^{\lfloor 13/4\rfloor} = g^3 = 27 \equiv 10$ |
| 1 | 0 | $10^2 = 100 \equiv 15$ | skipped | $g^{\lfloor 13/2\rfloor} = g^6 = 729 \equiv 15$ |
| 0 | 1 | $15^2 = 225 \equiv 4$ | $4\cdot3 = 12$ | $g^{13} = 12$ |

$3^{13}\equiv 12 \pmod{17}$, matching canon's right-to-left computation. Cost is exactly $t+1$ squarings plus $\mathrm{HW}(e)$ multiplications, where $\mathrm{HW}$ is Hamming weight, so at most $2\lfloor\log_2 e\rfloor + 2$ modular multiplications against $e-1$ for the naive loop.

Bit complexity: schoolbook multiplication of $k$-bit numbers is $O(k^2)$, so modular exponentiation with a $k$-bit exponent is $O(k^3)$ bit operations. In 64-bit words, one 2048-bit modular multiplication is roughly $(2048/64)^2 = 1024$ word multiplications, and a full RSA-2048 private exponentiation is about 3072 modular multiplications (2048 squarings plus roughly 1024 multiplies), so on the order of $3\times10^6$ word multiplications - milliseconds. Compare a symmetric operation at nanoseconds. That ratio is the whole reason asymmetric crypto is used to establish symmetric keys rather than to move data.

Timing leak: the multiply step is conditional on a secret exponent bit, so wall time and cache traffic reveal $\mathrm{HW}(e)$ and, with finer measurement, the bit pattern itself. Constant-time code either always performs the multiply (discarding the result into a dummy) or uses a ladder whose operation sequence is independent of the bits. The rule: a private exponent must never be a branch condition.

**Chinese remainder theorem, stated because unit 5 leans on it.** For coprime $p,q$ the map $x \mapsto (x \bmod p,\; x \bmod q)$ is a ring isomorphism $\mathbb{Z}_{pq} \to \mathbb{Z}_p \times \mathbb{Z}_q$. Tiny instance: $x \equiv 2 \pmod 3$ and $x \equiv 3 \pmod 5$ pin $x = 8$ uniquely in $\mathbb{Z}_{15}$. RSA private operations exploit this by running two exponentiations with half-size moduli and half-size exponents and recombining, costing about $2(k/2)^3 = k^3/4$ for a 4x speedup. It is also why a single injected fault in one of the two branches leaks the factorization.

**How hard discrete log actually is, by algorithm class.** Generic algorithms use only the group operation, with no knowledge of how elements are represented.

*Baby-step giant-step.* To solve $g^x = y$ in a group of order $N$, set $m = \lceil\sqrt N\rceil$ and write $x = im + j$ with $0 \le i,j < m$, which is always possible because $x < N \le m^2$. Rearranging $g^{im+j} = y$ gives $g^j = y\cdot(g^{-m})^i$. Tabulate $g^j$ for $j < m$, then step $i = 0,1,2,\ldots$ computing $y(g^{-m})^i$ and probing the table. Cost: $O(\sqrt N)$ group operations and $O(\sqrt N)$ memory.

Worked, $p=19$, $g=2$, $y=14$, $N=18$, $m=5$:

- Baby steps: $2^0=1$, $2^1=2$, $2^2=4$, $2^3=8$, $2^4=16$.
- $g^m = 2^5 = 32 \equiv 13$, and $13^{-1} \bmod 19 = 3$ since $13\cdot3 = 39 = 2\cdot19+1$.
- $i=0$: value 14, not in the table. $i=1$: $14\cdot3 = 42 \equiv 4 = 2^2$, so $j=2$.
- $x = im+j = 1\cdot5 + 2 = 7$. Check $2^7 = 128 = 6\cdot19 + 14 \equiv 14$. Correct.

Pollard's rho achieves the same $O(\sqrt N)$ time in $O(1)$ memory, and Shoup's lower bound proves $\Omega(\sqrt N)$ is optimal for any generic algorithm. So a group of order $2^{256}$ offers at most 128 bits of security, and "256-bit curve" and "128 bits of security" are the same statement.

Non-generic algorithms are where the three hard problems separate, which is M11 in quantitative form:

- **Finite-field discrete log in $\mathbb{Z}_p^*$** has extra structure available: integers factor, so index calculus applies. Its cost is subexponential, $L_p[1/3,\,1.923] = \exp\big((1.923+o(1))(\ln p)^{1/3}(\ln\ln p)^{2/3}\big)$ - super-polynomial but far below $\sqrt p$. At $\log_2 p = 3072$ this lands near $2^{128}$.
- **Integer factoring** by the general number field sieve has the same $L[1/3,\,1.923]$ shape, which is why RSA-3072 also sits near 128 bits.
- **Elliptic-curve discrete log** has no known index-calculus attack, leaving generic rho at about $0.886\sqrt N$ group operations. A 256-bit curve therefore sits at $2^{128}$.

One security level, four key sizes: 128-bit symmetric, 256-bit elliptic curve, 3072-bit finite-field DH, 3072-bit RSA. No single "key size" transfers, because the exponent in the attack cost depends on the problem's structure rather than on bit width.

## Keyspace, Bits of Security, and the Exponent

**Expected cost of exhaustive search.** With the key uniform over $|\mathcal{K}| = 2^n$ values and candidates tried in a random order without repetition, the trial index $T$ at which you succeed is uniform on $\{1,\ldots,2^n\}$, so

$$E[T] = \frac{1}{2^n}\sum_{i=1}^{2^n} i = \frac{1}{2^n}\cdot\frac{2^n(2^n+1)}{2} = \frac{2^n+1}{2}\approx 2^{n-1}$$

Averaging saves exactly one bit. Reporting $2^n$ or $2^{n-1}$ is the same claim within a factor of 2, which is why the literature is casual about the constant and exacting about the exponent.

**The one formula behind every key-size table.** With an attacker rate of $2^r$ trials per second and $2^{25}$ seconds per year,

$$\text{years} = 2^{\,n - r - 25}$$

Every row is that subtraction. $n=128$, $r=60$: $2^{43}$ years. $n=112$, $r=50$: $2^{37}$. $n=80$, $r=50$: $2^5 = 32$ years, and at $r=60$ it is $2^{-5}$ years, about 11 days - which is why 80-bit security is retired, and why the meaningful quantity is always a difference of exponents rather than a ratio of key lengths.

**Multi-target search, derived.** Let $k_1,\ldots,k_N$ be independent uniform $n$-bit keys and let the attacker want any one of them. A single candidate $k$ can be tested against all $N$ targets at once by hashing the targets into a table and doing one lookup per candidate. The probability a candidate hits something is

$$1 - (1-2^{-n})^N \approx N\,2^{-n} \quad (N \ll 2^n)$$

and the number of candidates until the first hit is geometric with that parameter, so $E[\text{trials}] \approx 2^n/N$. With $N = 2^{32}$ targets and $n=128$, the work is $2^{96}$: a 32-bit loss bought with memory. Two structural defences: per-user salt, which makes the derived keys non-shareable so no work amortises across targets, and a larger $n$. This is the first of the three genuine reasons for 256-bit keys.

**Birthday bound, derived, because it caps more than hash functions.** Draw $q$ values independently and uniformly from a set of size $M$. The probability all are distinct is

$$\prod_{i=0}^{q-1}\left(1-\frac{i}{M}\right) \le \prod_{i=0}^{q-1} e^{-i/M} = \exp\left(-\frac{q(q-1)}{2M}\right)$$

using $1-x \le e^{-x}$. So the collision probability is at least $1 - \exp(-q(q-1)/2M) \approx q^2/(2M)$ while $q^2 \ll M$, and passes $1/2$ near $q \approx 1.18\sqrt M$. With $M = 2^b$, collisions arrive after about $2^{b/2}$ draws. Three downstream facts are this one inequality: hash collision cost is $2^{n/2}$ rather than $2^n$ (unit 2, M7), random 96-bit GCM nonces must be capped far below $2^{48}$ messages per key (unit 3), and the random-IV mode bound derived in the threat-model section.

**Bits of security is a minimum over attacks, not a parameter.** Define $b = \log_2(\text{cost of the cheapest known attack})$, minimised over every attack: key search, cryptanalytic shortcuts, mode-level proof degradation, tag-guessing, side channels. Key length bounds only the key-search term. Concretely, AES-128-GCM has a 128-bit key and 128-bit key search, but its mode bound degrades near $2^{64}$ blocks under one key and its tag bounds forgery at $q_v 2^{-128}$; the deployable claim is the minimum of those, and it is the mode and the tag, not the key, that usually bind first.

**Non-uniform keys: Shannon entropy overstates security, min-entropy does not.** Shannon entropy $H(K) = -\sum_i p_i \log_2 p_i$ is an average code length, not a guessing cost. The guessing-relevant measure is min-entropy

$$H_\infty(K) = -\log_2 \max_i p_i$$

the work factor of the single best first guess. The two can be arbitrarily far apart. Take a distribution over $2^{127}+1$ keys: one specific key with probability $1/2$, and $2^{127}$ other keys each with probability $2^{-128}$ (total mass $1/2$).

- Shannon: $H = \tfrac12\log_2 2 \;+\; 2^{127}\cdot 2^{-128}\cdot 128 = 0.5 + 64 = 64.5$ bits.
- Min-entropy: $H_\infty = -\log_2(1/2) = 1$ bit.
- Actual attacker behaviour: guess the popular key first, succeed half the time on attempt number one.

A 64.5-bit figure attached to a system broken 50% of the time by one guess is not a rounding error, it is the wrong statistic. Quote min-entropy, or quote the guessing curve (success probability after $g$ guesses) directly. The two measures coincide only in the uniform case, $H = H_\infty = \log_2|\mathcal{K}|$, and canon's $H = L\log_2 A$ for a length-$L$ string over an $A$-symbol alphabet is that formula. It is a claim about the generator, not about the string: 12 Diceware words drawn by dice from a 7776-word list give $12\log_2 7776 = 12 \times 12.92 = 155$ bits, while 12 words a person chose give an unknown and much smaller number.

**What a KDF buys, in bits.** Let the passphrase have $H_\infty = h$ bits and let the key derivation function cost $c$ times a bare hash. The attacker's total work is $2^h \cdot c$ hash-equivalents, so

$$h_{\text{eff}} = h + \log_2 c$$

PBKDF2 at $c = 10^6 \approx 2^{20}$ adds 20 bits. The defender pays $c$ once per login (a few hundred milliseconds) while the attacker pays it $2^h$ times, and that asymmetry is the entire construction. It is bounded and small: no iteration count converts 30 bits into 128. Worse, if specialised hardware runs the KDF $A$ times more cost-efficiently than your server, then

$$h_{\text{eff}} = h + \log_2 c - \log_2 A$$

and $\log_2 A$ for PBKDF2-SHA256 on GPUs or ASICs is comfortably 10 bits or more. Memory-hard functions (scrypt, Argon2) exist to drive $A$ toward 1 by making memory bandwidth the bottleneck, since bandwidth does not miniaturise the way arithmetic does. Canon's "maybe 20 bits, no more" is this equation with realistic numbers substituted.

**The thermodynamic floor, and why it does not bind at 128 bits.** Landauer's principle sets the minimum energy to irreversibly erase one bit at $kT\ln 2$, which at $T = 300$ K is $1.38\times10^{-23}\times300\times0.693 \approx 2.9\times10^{-21}$ J. Merely counting to $2^{128}\approx3.4\times10^{38}$ therefore costs at least $3.4\times10^{38}\times2.9\times10^{-21}\approx1\times10^{18}$ J, roughly 15 hours of total world primary energy consumption (about $6\times10^{20}$ J per year). So thermodynamics alone does not forbid a 128-bit search. What forbids it is the real per-trial cost: an aggressive ASIC at $10^{-12}$ J per AES trial needs about $3.4\times10^{26}$ J, on the order of 600,000 years of world energy output, and the work still has to be sequenced in time. Contrast $2^{256}\approx1.2\times10^{77}$: even at the Landauer floor that is $3\times10^{56}$ J, against a total solar output over the Sun's remaining lifetime of order $10^{44}$ J. 128-bit security is infeasible for engineering and economic reasons; 256-bit security is infeasible for physical ones. The distinction matters because engineering and economics improve over decades and physics does not.

**Grover, with the qualification the headline omits.** Grover's algorithm finds a marked item among $M$ candidates with $O(\sqrt M)$ oracle queries, so key search drops from $2^n$ to about $2^{n/2}$: AES-256 to $2^{128}$, AES-128 to $2^{64}$. Two qualifications keep this from being the emergency it sounds like. First, the speedup parallelises poorly: splitting the search across $P$ quantum processors gives each a cost of $2^{n/2}/\sqrt P$, so buying $2^{20}$ machines buys 10 bits rather than 20 - the opposite of classical brute force, which parallelises perfectly. Second, $2^{n/2}$ counts a serial circuit depth of that many coherent operations, a far harder engineering target than the same count of classical operations. Quantum collision finding does even less well ($2^{n/3}$ with large quantum memory), so symmetric primitives need at most a doubling of parameters. The asymmetric side is where the complexity class itself changes: Shor's algorithm solves factoring and discrete log in polynomial time, which is why post-quantum work targets key exchange and signatures rather than AES.

**The bookkeeping convention this all supports.** For any deployed scheme, compute $\log_2$ of the cheapest known attack across every category above - key search adjusted for multi-target and quantum effects, cryptanalytic shortcuts, mode bounds of the $q^2\ell/2^b$ form, tag-forgery bounds of the $q_v 2^{-t}$ form, and the min-entropy of whatever actually produced the key - and compare the minimum to 128. Every "is this enough?" question in the remaining units resolves to that one comparison.