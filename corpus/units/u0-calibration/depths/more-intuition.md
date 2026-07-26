# More intuition: u0

No new symbols appear in this file. Every idea here is a picture for something
canon states algebraically.

## How this unit works

Think of the pretest as a hearing test, not an exam. The technician plays tones
at different frequencies to find the edges of your range. Nobody hears every
tone. The tones you miss are the useful data - they are what determines how the
rest of the session is tuned. Missing several is a normal, informative result,
and rushing through is fine.

## The shape contract

Picture a vector as an arrow from the origin. A two-dimensional vector is an
arrow on a sheet of paper; a three-dimensional one is an arrow in the room you
are sitting in. A 4096-dimensional one is the same idea with more directions
available, and the honest thing to say is that nobody visualizes it - you
visualize three dimensions and trust the algebra to behave analogously, which it
mostly does.

A matrix, in this picture, is a machine that takes arrows and hands back arrows,
possibly in a different space. Feed it an arrow in the room, get back an arrow
on the paper. The shape of the matrix is a label on the machine listing what it
accepts and what it emits, exactly like the type signature on a function.

The shape rule then reads as: you can only feed a machine's output into another
machine if the second machine accepts what the first one emits. A machine that
emits arrows-on-paper cannot feed one that expects arrows-in-a-room. That is all
"inner dimensions must match" says. And the composite machine takes what the
first one takes and emits what the last one emits, which is why the outer
dimensions survive.

The row-vs-column convention is a pure question of which side you write the
machine on. Some traditions write the machine on the left of what it eats, some
on the right. The machine is the same machine; write it on the wrong side and
the label reads backwards. It is the difference between `f(x)` and `x |> f` -
same computation, and you have to know which pipeline direction the document
uses before you can read any of its shapes.

The picture to hold for the whole book: a token starts life as an arrow, and
every layer of the model is a machine that nudges that arrow. Generation is
following where the arrow ends up.

## Dot products: alignment, not distance

Sunlight and a shadow. Hold a stick (call it $v$) and shine light straight down
onto a line drawn on the ground pointing along $u$. The stick's shadow on that
line is the projection of $v$ onto $u$'s direction. The dot product is the
length of that shadow, times the length of $u$.

Everything about dot products falls out of the shadow picture:

- **Perpendicular gives zero.** A stick held at right angles to the line casts
  no shadow along it. Zero dot product is not "unrelated" in any deep sense -
  it means the two directions carry no component of each other.
- **Opposite gives negative.** Point the stick backwards along the line and the
  shadow lands on the negative side.
- **Longer stick, longer shadow.** Double the stick's length without rotating
  it and the shadow doubles. The angle did not change. This is the entire
  content of the refutation in canon: shadow length depends on both the angle
  and the size of the object, so a big object at a bad angle can out-shadow a
  small one at a perfect angle. Cosine similarity throws away the size and keeps
  only the angle. The dot product keeps both, deliberately.

The size-matters property is not a defect. Inside a transformer, magnitude is
how a component says "this is important" and direction is how it says "this is
what it is about". The dot product listens to both at once, in one number. That
is what makes it the right primitive for attention, and also what makes
attention distributions sensitive to a single unusually large vector - one very
long stick can dominate the shadows of everything else on the line.

## Matrix multiply: composition, not a loop

The right picture is the darkroom, not the spreadsheet.

A matrix is a transformation of space itself. Not a table of numbers that gets
looped over - a warping. Draw a grid on a rubber sheet, then stretch, rotate,
shear, or flatten the sheet while keeping the origin pinned and all grid lines
straight and evenly spaced. That constraint - straight stays straight, evenly
spaced stays evenly spaced, origin stays put - is exactly what "linear" means,
and every matrix is one such warp.

Multiplying two matrices means doing one warp, then the other, and asking what
single warp does the same job. Rotate 90 degrees, then stretch horizontally.
The combined effect is one warp, and the product matrix is its recipe. Once you
see it that way, three properties stop needing memorization:

- **Order matters.** Rotate-then-stretch is visibly not stretch-then-rotate.
  Rotate a square 45 degrees then stretch horizontally and you get a slanted
  diamond; stretch first then rotate and you get a tilted rectangle. Different
  final shapes, same two operations. $AB \neq BA$.
- **Grouping does not matter.** Doing three warps in a fixed order gives the
  same result no matter which pair you mentally combine first. That is
  associativity, and it is why the same computation can be reassociated for
  speed without changing the answer.
- **Flattening is permanent.** A warp that squashes the sheet onto a line has
  destroyed a dimension, and no later warp recovers it. That is what a
  non-invertible matrix is, and it is the geometric reason a model's projection
  down to a narrower space is a lossy commitment.

The "grid of dot products" formula is the recipe for computing the warp, not the
meaning of it. When you read $XW$ in a later unit, the useful thought is not
"loop over the rows"; it is "every token's arrow gets warped into the space $W$
maps to". A transformer is a long chain of such warps with a nonlinear kink
between each pair, and the kink is there precisely because chained warps with no
kink collapse into a single warp.

## Gradients and expectations

Fog on a hillside.

You are standing on a landscape in thick fog and want to get downhill. You
cannot see the valley - you cannot see ten feet. What you can do is feel the
ground under your feet: which way does it tilt, and how steeply? That local
tilt is the gradient. Step against it, then feel again.

The picture repairs three misreadings at once.

**Why the gradient is not one number.** A hillside does not have "a slope" - it
has a slope in each direction you might face. Facing north it may drop sharply,
facing east it may be flat. The full answer is a reading for every direction you
can face, and with a weight matrix of a million entries there are a million
directions to face. The gradient is that whole set of readings, which is why it
is the same size and arrangement as the thing it describes: one reading per
weight, laid out in the same grid the weights live in.

**Why the shapes must match.** The step you take is "move each weight a little
against its own reading". Weights and readings pair up one-to-one, so they must
be arranged the same way. Trying to subtract a single number from a matrix of
weights would be like responding to a topographic survey by taking one step in
one direction - it throws away every reading but one.

**Why the fog never lifts.** You only ever learn the ground where you stand. You
never learn whether there is a deeper valley two ridges over. Training is
millions of local feel-and-step moves with no global view at any point, which is
why u2's claim that gradient descent does not find "the" minimum should already
feel obvious rather than surprising.

Now the expectation, which is where the fog comes from. The true landscape is
defined over all the text that could ever exist, and you cannot survey it. Each
training batch is a handful of soil samples from where you stand. Average them
and you get an estimate of the local tilt - roughly right, individually noisy.
More samples per step, steadier reading, slower going. Fewer samples, jumpier
path. That trade is the whole of batch-size tuning, and the surprise waiting in
u2 is that the jumpiness turns out to help: a walker who staggers a bit does not
get stuck in every small dip on the way down.
