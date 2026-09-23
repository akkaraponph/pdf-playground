# 1. Unpack streams and normalize indirect objects
qpdf --qdf --object-streams=disable mock-signing-document-signed.pdf unpacked.pdf

# 2. Search for the Perms dictionary and its sub-entries
grep -E "(/Perms|/DocMDP|/UR3)" unpacked.pdf