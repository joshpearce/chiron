#!/bin/bash
# Generate a diceware-style passphrase for the shared client key.
#
# The point is typeability. The key has to be entered by hand on a tablet
# keyboard - it cannot be pasted across from the Mac - and a 48-character random
# string is a transposition waiting to happen. Six words drawn from a ~10k pool
# is around 78 bits, which is far past anything reachable by guessing an HTTPS
# endpoint, and it can be read off a screen and typed without errors.
#
#   scripts/make-passphrase.sh            # one passphrase on stdout
#   scripts/make-passphrase.sh 4          # four candidates, pick one
#
# Pipe it into scripts/sprite-set-auth.sh to install it.
set -euo pipefail

COUNT=${1:-1}
WORDS=6
DICT=/usr/share/dict/words
[ -r "$DICT" ] || { echo "no wordlist at $DICT" >&2; exit 1; }

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
python3 - "$DICT" "$COUNT" "$WORDS" "$ROOT" <<'PYEOF'
import math, re, secrets, sys
from collections import Counter
from pathlib import Path

dict_path, count, nwords, root = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), Path(sys.argv[4])

# web2 has 236k words and no frequency information, so drawing from it
# straight gives "souslik" and "davach" - correct, and useless for reading off
# a screen. Frequency comes from the corpus instead: ~600k words of real
# English prose, which is exactly the vocabulary a person recognises. web2 is
# still used, as a spelling check on what comes out.
#
# Drawing from a known list costs nothing here. Assume the attacker has it:
# what protects the key is the pool size and a uniform random draw, not the
# secrecy of the vocabulary.
corpus_dirs = list(root.glob("corpus*"))

spelled = set()
ok = re.compile(r"^[a-z]{4,7}$")
banned = set("qxzj")
for line in open(dict_path, encoding="utf-8", errors="ignore"):
    w = line.strip().lower()
    if ok.match(w):
        spelled.add(w)

freq = Counter()
for d in corpus_dirs:
    for f in d.rglob("*.md"):
        try:
            text = f.read_text(encoding="utf-8", errors="ignore").lower()
        except OSError:
            continue
        freq.update(re.findall(r"[a-z]+", text))

def typeable(w):
    if not ok.match(w) or banned & set(w):
        return False
    if re.search(r"(.)\1", w):       # doubled letters invite miscounts
        return False
    vowels = sum(c in "aeiou" for c in w)
    return 0 < vowels < len(w)

pool = [w for w, n in freq.most_common() if w in spelled and typeable(w) and n >= 3]
# Drop the most frequent handful: "the", "that", "with" carry no information to
# a reader trying to keep six words straight.
pool = pool[20:2020]

if len(pool) < 1000:
    sys.exit(f"pool too small ({len(pool)}) - refusing to generate a weak key")

bits = nwords * math.log2(len(pool))
for _ in range(count):
    print("-".join(secrets.choice(pool) for _ in range(nwords)))
print(f"\n{len(pool)} common words in the pool, {nwords} per phrase "
      f"= {bits:.0f} bits", file=sys.stderr)
PYEOF
