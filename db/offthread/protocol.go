// Package offthread is v5's client for a SQLite database living in another
// thread (plan item P3.5).
//
// This is the plan's OpenOffThread. It lives in its own package rather than
// beside sqlite.Open for one concrete reason: criterion (b) requires an app
// using the off-thread model to contain no wazero symbol at all, and any package
// importing db/sqlite links the whole engine — a ~1MB wasm interpreter — into
// app.wasm. Nothing in this package imports db/sqlite, ncruces, or wazero, and
// TestOffThreadClientDoesNotLinkTheEngine fails the build if that ever changes.
//
// The engine-side half is db/offthread/server, which does import db/sqlite and
// is meant to be compiled into domain.wasm.
//
// The two halves speak this protocol and nothing else.
package offthread

import (
	"fmt"
	"math"
)

// Op names one database operation.
type Op string

const (
	// OpExec runs a statement that returns no rows.
	OpExec Op = "exec"
	// OpQuery runs a statement that returns rows.
	OpQuery Op = "query"
	// OpBegin opens a transaction and returns its id.
	OpBegin Op = "begin"
	// OpCommit commits an open transaction.
	OpCommit Op = "commit"
	// OpRollback discards an open transaction.
	OpRollback Op = "rollback"
	// OpFlush forces the database image to its durable backend.
	OpFlush Op = "flush"
	// OpClose closes the database.
	OpClose Op = "close"
)

// IsKnownOp reports whether an op is one the server understands.
//
// Unknown ops are refused rather than ignored: a request that silently did
// nothing would look identical to one that succeeded and wrote no rows.
func IsKnownOp(parseOp Op) bool {
	switch parseOp {
	case OpExec, OpQuery, OpBegin, OpCommit, OpRollback, OpFlush, OpClose:
		return true
	default:
		return false
	}
}

// ValueKind names one SQLite storage class.
//
// SQLite has exactly five, and modelling them explicitly rather than shipping
// `any` across the boundary is what keeps the encoding total: every value has
// exactly one representation, and a decoder never has to guess whether a number
// was an integer or a float. That guess is how JSON round-trips silently turn
// an int64 id into a float64 and lose precision above 2^53.
type ValueKind uint8

const (
	// ValueNull is SQL NULL.
	ValueNull ValueKind = iota
	// ValueInt is INTEGER.
	ValueInt
	// ValueFloat is REAL.
	ValueFloat
	// ValueText is TEXT.
	ValueText
	// ValueBlob is BLOB.
	ValueBlob
)

func (parseKind ValueKind) String() string {
	switch parseKind {
	case ValueNull:
		return "null"
	case ValueInt:
		return "int"
	case ValueFloat:
		return "float"
	case ValueText:
		return "text"
	case ValueBlob:
		return "blob"
	default:
		return fmt.Sprintf("unknown-kind(%d)", uint8(parseKind))
	}
}

// Value is one SQLite value in transit.
//
// A struct with a kind rather than an interface: the wire form has to survive a
// structured clone or a JSON encode without the receiver inferring types from
// what the numbers happen to look like.
type Value struct {
	Kind  ValueKind `json:"kind"`
	Int   int64     `json:"int,omitempty"`
	Float float64   `json:"float,omitempty"`
	Text  string    `json:"text,omitempty"`
	Blob  []byte    `json:"blob,omitempty"`
}

// NewValue converts a Go value to its wire form.
//
// The accepted set is deliberately narrow — what SQLite actually stores, plus
// the Go integer and float widths that convert without loss. Anything else is an
// error at the call site rather than a surprise at the far end, where the only
// available response is a string.
func NewValue(parseGoValue any) (Value, error) {
	switch parseTyped := parseGoValue.(type) {
	case nil:
		return Value{Kind: ValueNull}, nil
	case bool:
		// SQLite has no boolean type; it stores 0 and 1. Converting here rather
		// than refusing keeps callers from writing that conversion everywhere.
		if parseTyped {
			return Value{Kind: ValueInt, Int: 1}, nil
		}
		return Value{Kind: ValueInt, Int: 0}, nil
	case int:
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case int8:
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case int16:
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case int32:
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case int64:
		return Value{Kind: ValueInt, Int: parseTyped}, nil
	case uint8:
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case uint16:
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case uint32:
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case uint:
		if uint64(parseTyped) > math.MaxInt64 {
			return Value{}, fmt.Errorf("offthread: uint %d exceeds int64, which is the widest integer SQLite stores", parseTyped)
		}
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case uint64:
		if parseTyped > math.MaxInt64 {
			return Value{}, fmt.Errorf("offthread: uint64 %d exceeds int64, which is the widest integer SQLite stores", parseTyped)
		}
		return Value{Kind: ValueInt, Int: int64(parseTyped)}, nil
	case float32:
		return Value{Kind: ValueFloat, Float: float64(parseTyped)}, nil
	case float64:
		return Value{Kind: ValueFloat, Float: parseTyped}, nil
	case string:
		return Value{Kind: ValueText, Text: parseTyped}, nil
	case []byte:
		return Value{Kind: ValueBlob, Blob: parseTyped}, nil
	default:
		return Value{}, fmt.Errorf("offthread: %T cannot be sent to an off-thread database; convert it to a SQLite storage class first", parseGoValue)
	}
}

// NewValues converts a Go argument list to its wire form.
func NewValues(parseGoValues []any) ([]Value, error) {
	if len(parseGoValues) == 0 {
		return nil, nil
	}
	parseValues := make([]Value, 0, len(parseGoValues))
	for parseIndex, parseGoValue := range parseGoValues {
		parseValue, parseErr := NewValue(parseGoValue)
		if parseErr != nil {
			return nil, fmt.Errorf("argument %d: %w", parseIndex, parseErr)
		}
		parseValues = append(parseValues, parseValue)
	}
	return parseValues, nil
}

// Go converts a wire value back to a Go value.
func (parseValue Value) Go() any {
	switch parseValue.Kind {
	case ValueInt:
		return parseValue.Int
	case ValueFloat:
		return parseValue.Float
	case ValueText:
		return parseValue.Text
	case ValueBlob:
		return parseValue.Blob
	default:
		return nil
	}
}

// Request is one operation sent to the database thread.
type Request struct {
	Op  Op     `json:"op"`
	SQL string `json:"sql,omitempty"`
	// Args are bind parameters. They are never interpolated into SQL, which is
	// what keeps the boundary from becoming an injection surface.
	Args []Value `json:"args,omitempty"`
	// TxID scopes the operation to an open transaction. Empty means autocommit.
	TxID string `json:"txId,omitempty"`
	// MaxRows caps a query's result set. Zero means the server's default.
	MaxRows int `json:"maxRows,omitempty"`
}

// Response is one operation's result.
type Response struct {
	// Err is the failure message, empty on success. A string rather than an
	// error because the failure has to cross a thread boundary; RemoteError
	// wraps it back into an error on arrival.
	Err          string    `json:"err,omitempty"`
	RowsAffected int64     `json:"rowsAffected,omitempty"`
	LastInsertID int64     `json:"lastInsertId,omitempty"`
	Columns      []string  `json:"columns,omitempty"`
	Rows         [][]Value `json:"rows,omitempty"`
	TxID         string    `json:"txId,omitempty"`
	// Truncated reports that the result set hit MaxRows and is incomplete.
	//
	// Reported rather than silently cut: a caller that paginates on a short
	// result would stop early and believe it had everything.
	Truncated bool `json:"truncated,omitempty"`
}

// RemoteError is a failure that happened on the database thread.
//
// A distinct type so a caller can tell "the query was rejected" from "the worker
// never answered" — the first is a bug in the query, the second is a transport
// or liveness problem, and they want different handling.
type RemoteError struct {
	Op      Op
	Message string
}

func (parseErr *RemoteError) Error() string {
	return fmt.Sprintf("offthread: %s failed on the database thread: %s", parseErr.Op, parseErr.Message)
}
