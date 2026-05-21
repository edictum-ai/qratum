Forward design reference.
This document does not expand the current implementation scope.
Current executable scope is defined only in SPEC.md.

# Architecture Overview

Qratum observes AI coding sessions locally, normalizes them into a native
session model, redacts sensitive data, extracts compact evidence, writes review
cards, renders static HTML, and exports ADP strict JSONL.

Milestone A is local-only and filesystem-only.
