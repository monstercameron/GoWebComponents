Documentation Ingestion Plan
Concept-Based, AI-Assisted, Search-First Model

Goal

Build a single searchable concepts table for the GoWebComponents docs corpus.

The ingestion target is the repository docs/ corpus, using repo-relative paths such as docs/ACCESSIBILITY.md rather than machine-local paths. This corpus is the canonical source set for the docs concepts database.

For each source document, the pipeline should:
- parse deterministically
- segment into stable source blocks
- extract typed concepts with AI
- shape rows for retrieval usefulness
- dedupe redundant rows
- fact-check against source text only
- upsert a new active document version safely
- preserve history without exposing unverified rows by default

Default search should only return verified, active rows.

Corpus Scope

The initial corpus is the full docs/ set currently identified for GoWebComponents, including onboarding, reference, architecture, integration, troubleshooting, performance, TODO, and roadmap-oriented markdown files.

Scope rules:
- ingest by discovered repo-relative path only
- store doc_path as a repo-relative path
- treat listed docs/*.md files as eligible by default
- allow future docs/ additions to be ingested automatically if they match ingestion rules
- manage exclusions through explicit allow/deny rules, not ad hoc skipping

What Is Worth Ingesting

For this repo, the highest-value rows are the ones that help a user understand the current supported surface, choose the right package or workflow, and act on real implementation or production guidance.

Prefer rows that capture:
- public API behavior, package purpose, and supported authoring surfaces
- workflows, setup steps, commands, migrations, and troubleshooting procedures
- production caveats, browser/runtime limits, correctness constraints, and deployment notes
- decision rules about when to use a feature
- example-backed guidance tied to actual usage patterns
- concrete facts about routing, hydration, SSR, workers, state, fetch, forms, diagnostics, and integrations
- navigation rows that help a user find the right doc, package, example, or workflow quickly

In this repo, docs such as START_HERE.md, REFERENCE_MAP.md, WORKFLOWS.md, WALKTHROUGHS.md, TROUBLESHOOTING.md, integration docs, runtime docs, and production/correctness docs should usually yield a high density of useful rows.

Usually low-value or trash

Do not treat every paragraph as worth indexing.

Usually low-value rows include:
- vague framing text
- generic motivation or marketing prose
- repeated summaries with no new fact, step, caveat, or decision rule
- stale roadmap language and speculative aspirations
- internal implementation discussion that is not part of the supported public surface
- headings turned into fake facts
- oversized summaries that collapse many claims into one weak row
- boilerplate link lists with no real semantic value
- maintainer-only notes that are not useful for future retrieval

Repo-specific bias

Bias extraction toward questions a real GoWebComponents adopter would ask:
- What package should I use?
- How do I do this feature?
- What is the supported path?
- What are the caveats?
- What is shipped now versus planned?
- What doc or example should I read next?

Bias against rows that mainly explain internal organization, philosophy without operational consequence, or speculative design without current user value.

Special handling for TODO and roadmap docs

Files like TODO.md, DOCUMENTATION_PUSH_TODO.md, MULTITHREADED_RUNTIME_TODO.md, and ADOPTION_MATURITY.md are higher risk.

Keep:
- concrete gaps
- explicit non-shipped status
- clearly stated planned capabilities useful for evaluation or planning

Downgrade or drop:
- vague wish-list bullets
- repeated aspirations
- broad roadmap summaries with no concrete retrieval value

Future-facing statements should usually be stored as note, warning, or concept, not as current fact, unless the source clearly says the capability exists today.

A row is worth keeping if it helps answer:
- What is the supported surface?
- How do I do X?
- What should I start with?
- What are the caveats?
- Is this shipped, experimental, or future?
- What should I read next?

If it does not materially help answer one of those, it is probably noise.

Canonical Table

Use one canonical table:

docs_concepts

Recommended columns:
- concept_id
- doc_path
- doc_title
- parent_concept_id
- concept_type
- concept_title
- concept_text
- concept_json
- source_span_start
- source_span_end
- source_excerpt
- search_text
- concept_confidence
- factcheck_status
- factcheck_notes
- factcheck_evidence_span_start
- factcheck_evidence_span_end
- factcheck_evidence_excerpt
- adjusted_concept_text
- active
- soft_deleted
- ingest_batch_id
- source_file_hash
- content_hash
- schema_version
- prompt_version
- ai_model
- created_at

This is the canonical searchable and reviewable row store.

Notes on schema shape

This is no longer truly minimal-column; it is a compact audit-friendly schema. That is acceptable because the added fact-check evidence fields materially improve trust, reviewability, and debugging.

Generate concept_id in code, not by AI.

content_hash must be computed over the normalized retrieval tuple, not only concept_text. Recommended basis:
- concept_type
- concept_title
- normalized concept_text
- normalized concept_json
- source span

This avoids collapsing distinct rows that share similar prose but differ in type, structure, metadata, or anchoring.

Span semantics

All source_span_* and factcheck_evidence_span_* fields must use one canonical unit:
- byte offsets into UTF-8 normalized document content

Do not mix line numbers in one implementation and byte offsets in another. Display layers may derive line numbers separately for UI, but storage, dedupe, evidence validation, and comparisons must use the same canonical offset scale.

Suggested DDL

CREATE TABLE docs_concepts (
  concept_id TEXT PRIMARY KEY,
  doc_path TEXT NOT NULL,
  doc_title TEXT NOT NULL,
  parent_concept_id TEXT,
  concept_type TEXT NOT NULL,
  concept_title TEXT NOT NULL,
  concept_text TEXT NOT NULL,
  concept_json TEXT,
  source_span_start INTEGER,
  source_span_end INTEGER,
  source_excerpt TEXT,
  search_text TEXT NOT NULL,
  concept_confidence REAL,
  factcheck_status TEXT NOT NULL DEFAULT 'needs_review' CHECK (factcheck_status IN ('pass','needs_review','fail')),
  factcheck_notes TEXT,
  factcheck_evidence_span_start INTEGER,
  factcheck_evidence_span_end INTEGER,
  factcheck_evidence_excerpt TEXT,
  adjusted_concept_text TEXT,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  soft_deleted INTEGER NOT NULL DEFAULT 0 CHECK (soft_deleted IN (0,1)),
  ingest_batch_id TEXT NOT NULL,
  source_file_hash TEXT,
  content_hash TEXT NOT NULL,
  schema_version INTEGER NOT NULL DEFAULT 1,
  prompt_version TEXT,
  ai_model TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX idx_docs_concepts_lookup
ON docs_concepts (doc_path, active, soft_deleted, factcheck_status, concept_type);

CREATE INDEX idx_docs_concepts_search
ON docs_concepts (concept_type, doc_path, created_at);

If desired later, file-size and mtime metadata can live in a batch/file manifest table instead of this row table. Add source_file_size and source_file_mtime only if you plan to use them for stale-detection or incremental ingest decisions.

One-active-version rule

The schema alone does not guarantee that only one version per doc_path is active. Enforce this in application logic during upsert, including concurrent ingests. Never rely on the DDL above alone to preserve the active-version invariant.

Row Contract

Each row is one extracted concept or snippet from one document version.

Required behavior:
- concept_id is generated in code
- source spans must anchor the row to the original source
- search_text is normalized in code from title, text, and useful metadata
- content_hash supports dedupe and change detection
- factcheck_status must be one of:
  - pass
  - needs_review
  - fail

Recommended taxonomy:
- title
- heading
- paragraph
- concept
- fact
- api
- code
- link
- checklist_item
- procedure_step
- note
- warning

Row quality rules

Do not keep low-information rows just because they passed parsing and fact-check.

Required shaping rules:
- do not emit generic concept_title values such as Paragraph, Section, or Item when a more specific short title can be generated safely from the source
- if a row cannot get a meaningful title and does not add semantic retrieval value, drop it
- prefer concise semantic rows over large raw paragraph copies when the source supports a more useful typed concept
- a fact-check pass is necessary but not sufficient for retention; the row must also pass the usefulness filter

Retention preference

When multiple possible row shapes could be emitted from the same source block, prefer them in roughly this order:
- procedure_step
- api
- fact
- warning
- concept
- note
- paragraph
- heading

This is a retrieval preference, not a truth ranking. Heading and paragraph rows are still allowed, but they should not crowd out stronger semantic rows.

Paragraph handling rules

Paragraph rows are allowed, but they are lower-value fallback rows.

Keep paragraph rows only when they preserve useful prose that would be hard to represent as a smaller fact or concept row.

Drop or downgrade paragraph rows when:
- they are mostly rhetorical or introductory
- they repeat nearby text without adding retrieval value
- they carry a generic title
- they are weaker than a clear semantic row that can be extracted from the same source

If a paragraph contains multiple strong claims, split it into multiple semantic rows where possible.

If a paragraph contains one strong central claim, prefer a concept or fact row and optionally omit the paragraph row.

Heading and hierarchy rules

Always ingest headings as structural rows.

Assign parent_concept_id for non-heading rows using the nearest containing heading or section anchor.

Structural rows may be retained for navigation even when not ideal for semantic search.

Pipeline

1. Discover and parse

Read one file at a time from the scoped docs/ corpus and produce:
- doc_path
- doc_title
- normalized text
- canonical byte-offset map
- source_file_hash

Rules:
- use repo-relative paths like docs/README.md
- normalize line endings before span calculation
- derive titles deterministically from markdown first
- do not use AI in this stage

2. Segment

Split each source into stable, non-overlapping blocks using headings, paragraphs, lists, and code fences.

3. Extract

Run AI on one block at a time with strict JSON output.

Each concept must include:
- concept_type
- concept_title
- concept_text
- concept_json
- source_span_start
- source_span_end
- concept_confidence

The model must not generate concept_id.

Extraction preference rules:
- prefer semantic rows over literal block copies
- if a paragraph contains a concrete product claim, limitation, workflow rule, or package behavior, emit a semantic row rather than only a raw paragraph row
- shipped capability claims should usually become fact rows
- scope or positioning statements should usually become concept rows
- step-like prose should usually become procedure_step rows

4. Normalize

In code:
- trim and normalize strings
- map taxonomy aliases
- generate concept_id
- generate search_text
- generate content_hash
- reject malformed spans
- cap excessive text length

Normalization must also:
- reject generic titles when a row fails the usefulness filter
- assign parent_concept_id using the nearest structural ancestor
- preserve enough source_excerpt for debugging and review

5. Dedupe

Remove duplicate or redundant candidates before fact-check and upsert.

6. Fact-check

Validate each kept candidate against source text only.

7. Upsert

Insert the new document version, then deactivate prior active rows for that doc_path only after successful insert and successful validation of the full replacement set.

Dedupe Rules

Apply dedupe within the same document version before persistence.

Exact duplicates

If two candidates have the same:
- doc_path
- concept_type
- normalized concept_text
- source span

keep one.

Same-span collision handling

Before confidence-based tie-breaks, treat rows with the same doc_path and the same source span as a dedicated collision class. Only collapse them if they are materially the same row after considering:
- concept_type
- concept_title
- normalized concept_text
- normalized concept_json

This avoids dropping distinct but adjacent or normalization-shifted rows too aggressively.

Same-content duplicates

If two candidates have the same content_hash and heavily overlapping spans, keep one.

Near-duplicates

If text similarity is very high and spans overlap significantly, keep the better candidate in this order:
1. more precise concept_type
2. tighter source span
3. higher concept_confidence
4. richer useful metadata

Examples:
- keep api over generic concept
- keep procedure_step over paragraph

Deterministic tie-break for remaining ties:
1. higher concept_confidence
2. shorter source span
3. lexicographically smaller concept_text
4. lower content_hash

Parent-child redundancy

Structural and semantic rows may coexist, but semantic rows should win for retrieval value. Keep structural rows only when they help navigation.

Dedupe caution

Do not dedupe across document versions. Dedupe applies only within one candidate set for one document version. Small wording changes, metadata changes, or span shifts in a new version should normally produce new rows in the new batch rather than being merged into prior versions.

Staleness and Collision Rules

One active version per document

For a given doc_path, only one promoted batch version may have active = 1.

Failed ingest does not retire good rows

If a new ingest fails for a document:
- do not deactivate prior active rows
- keep prior searchable rows live

Soft delete only for actual removal

Use soft_deleted = 1 only when a source document is intentionally removed or retired from the corpus. Do not use it for normal replacement.

Newer file hash wins

If retries or repeated ingests collide, the successful ingest tied to the intended latest source_file_hash wins.

New version, new rows

If wording, spans, or extracted structure changes, treat the result as new rows in the new batch. Do not mutate old rows in place.

Contradictory old vs new facts

When a new verified version is promoted, prior active rows become inactive. Default search should see only the current active version.

Fact-Checking Rules

Fact-check must be source-only.

The verifier may not use:
- outside knowledge
- other documents
- likely true reasoning

Must be fact-checked

Mandatory for:
- fact
- api
- procedure_step
- warning
- checklist_item
- behavior claims in general

Lighter checking is acceptable for:
- title
- heading
- paragraph
- note
- code when only preserving obvious source meaning

Outcomes

pass
Supported by source as written, or with only a small wording adjustment.

needs_review
Partially supported, ambiguous, overstated, or too broad.

fail
Unsupported, materially wrong, contradictory, or missing grounding.

Rules
- no evidence, no pass for factual or operational claims
- overstated wording must be downgraded
- bundled multi-claim concepts should be split or marked needs_review
- API, config, and command descriptions must stay close to source behavior
- headings do not count as proof of semantic claims beneath them
- code-derived explanations are high-risk unless directly stated or trivially obvious

Minimal fact-check output:
- factcheck_status
- factcheck_notes
- adjusted_concept_text
- factcheck_evidence_span_start
- factcheck_evidence_span_end
- factcheck_evidence_excerpt

For factual and operational claims, fact-check evidence spans must point to the exact supporting byte range in the normalized source document. If the source does not support the claim, mark needs_review or fail.

Search Rules

Default search must use only rows where:
- active = 1
- soft_deleted = 0
- factcheck_status = 'pass'

Rows marked needs_review or fail are persisted but excluded by default.

Search implementation note

A plain search_text column is enough for a first pass, but it sets a search-quality ceiling. If the system remains single-table, define one of these paths explicitly:
- SQLite FTS over the same corpus
- an indexed search_text strategy with ranking rules

Do not assume raw LIKE over search_text is sufficient for long-term retrieval quality.

Example safe query:

SELECT concept_id, doc_path, concept_title, concept_text
FROM docs_concepts
WHERE active = 1
  AND soft_deleted = 0
  AND factcheck_status = 'pass'
  AND concept_type IN ('api', 'procedure_step', 'warning', 'fact', 'note')
ORDER BY created_at DESC
LIMIT 50;

Upsert Rules

Use per-document atomic replacement.

Required flow:
1. begin transaction
2. parse, extract, normalize, dedupe, and fact-check the full document candidate set
3. insert the new batch rows with active = 0 or in a non-promoted state
4. validate that all required inserts succeeded and no uniqueness or integrity constraint failed
5. deactivate prior active rows for the same doc_path
6. promote the new rows for that doc_path to active = 1
7. commit

If any step fails:
- roll back the transaction
- keep the prior active version unchanged

Promotion rules

Staged rows may be inserted with active = 0.

On successful per-document completion, promote the replacement set by setting its rows to active = 1.

During the same transaction, deactivate the prior active rows for that doc_path.

The final committed state must leave exactly one promoted document version for that doc_path.

Never hand-edit docs_concepts rows directly. Always reingest the full document after source edits.

Suggested Defaults
- FACTCHECK_THRESHOLD = 0.82
- MAX_CONCEPT_TEXT_CHARS = 2000
- MAX_SOURCE_EXCERPT_CHARS = 1000
- NEAR_DUPLICATE_SIMILARITY = 0.97
- SOURCE_SPAN_OVERLAP_DUPLICATE = 0.70
- DB_TRANSACTION_SCOPE = per_document
- BATCH_HISTORY_MIN = 3
- PARSER_FAIL_MODE = fail_closed_per_document

Prompt Guidance

Extraction prompt

Input:
- doc metadata
- one source block
- taxonomy

Output:
- strict JSON concepts containing:
  - concept_type
  - concept_title
  - concept_text
  - concept_json
  - source_span_start
  - source_span_end
  - concept_confidence

Do not emit concept_id.

Additional extraction instructions:
- prefer semantic rows over literal prose copies
- do not emit generic concept_title values when a specific short title can be derived safely
- do not emit low-value paragraph rows just because they are easy to extract
- emit rows that maximize future retrieval value while remaining faithful to source

Fact-check prompt

Input:
- source block
- one concept candidate

Output:
- factcheck_status
- factcheck_notes
- adjusted_concept_text
- evidence span fields

Keep it source-only.

Non-document noise

Ignore shell, profile, or runtime noise emitted by the execution environment if it is not part of the source document itself. Incidental terminal errors or PowerShell profile load messages are not document content and must not be ingested.

Completion Checklist

A run is complete when:
- a unique ingest_batch_id is created
- all scoped docs/ files are processed one at a time
- all kept candidates are normalized and deduped
- parent hierarchy is assigned for non-heading rows
- low-value generic rows are dropped or downgraded
- fact-check is recorded for every kept row
- only verified rows are promoted to default search
- prior active rows are preserved when a new ingest fails
- searchable row count and review row count are reported