# Technical Debt ML Seed Dataset

This directory contains a small metric-only calibration dataset for the
Technical Debt Analyzer.

The seed uses the same kind of file/process metrics found in public software
defect prediction datasets such as SQuaD, PROMISE, and NASA MDP: churn,
change frequency, contributor count, complexity, size, TODO count, bugfix
touches, and test presence.

Render constraint: do not vendor large raw datasets or train heavy Python
models at runtime. The application embeds this compact JSON file and trains a
tiny logistic regression model in Go memory. No repository source code or code
snippets are stored in this dataset.
