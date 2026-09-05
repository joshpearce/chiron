package roles

// Rules every role that writes for the reader shares.

// acronymRule: the reader is an expert, but not in everything the material
// touches, and a summary is where an unexplained initialism hurts most.
const acronymRule = `
- Expand every acronym or initialism the first time it appears, as "retrieval-augmented generation (RAG)", unless the material you were given defines it already; after that the short form is fine.`
