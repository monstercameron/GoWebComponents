package projection

import (
	"context"
	"errors"
	"fmt"
)

// Typed commands — P3.7 criterion (d): a misspelled command does not compile.
//
// The usual shape for a worker RPC is client.Call("createItem", args), and it
// fails at runtime for "creatItem", in the worker, after the user clicked
// something. A typo becomes a support ticket.
//
// Here a command is DECLARED once as a package-level variable:
//
//	var CreateItem = projection.Define[CreateItemArgs, CreateItemResult]("createItem")
//
// and invoked through that variable:
//
//	result, err := CreateItem.Invoke(ctx, client, CreateItemArgs{Name: "x"})
//
// Misspelling it at a call site is an undefined identifier, and passing the
// wrong argument type or assigning the wrong result type is a type error. The
// name string appears exactly once in the program, where it is checked against
// the worker's registry at startup rather than per call.

// CommandClient carries an encoded command to the domain worker.
//
// An interface so commands are testable without a worker, and so the same
// declarations work over any transport.
type CommandClient interface {
	// Send delivers a command and returns its encoded result.
	Send(parseCtx context.Context, parseName string, parseRequest []byte) ([]byte, error)
}

// Codec encodes a command's arguments and decodes its result.
//
// Separate from the command declaration so the same command can be carried by
// different encodings — JSON in development, a compact form in production —
// without touching the call sites.
type Codec interface {
	Encode(parseValue any) ([]byte, error)
	Decode(parseData []byte, parseTarget any) error
}

// Command is a typed command declaration.
//
// The type parameters are the contract: A is what callers pass, R is what they
// get back, and both are checked at every call site.
type Command[A any, R any] struct {
	name string
}

// Define declares a command with its argument and result types.
//
// Call it once, at package scope, and use the returned value everywhere. The
// name is the only string in the arrangement.
func Define[A any, R any](parseName string) Command[A, R] {
	return Command[A, R]{name: parseName}
}

// Name reports the wire name, for registry checks and diagnostics.
func (parseCommand Command[A, R]) Name() string {
	return parseCommand.name
}

// Invoke sends the command and decodes its result.
func (parseCommand Command[A, R]) Invoke(parseCtx context.Context, parseClient CommandClient, parseCodec Codec, parseArgs A) (R, error) {
	var parseZero R
	if parseCommand.name == "" {
		return parseZero, errors.New("projection: command was not declared with a name")
	}
	if parseClient == nil {
		return parseZero, fmt.Errorf("projection: command %q needs a client", parseCommand.name)
	}
	if parseCodec == nil {
		return parseZero, fmt.Errorf("projection: command %q needs a codec", parseCommand.name)
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}

	parseRequest, parseEncodeErr := parseCodec.Encode(parseArgs)
	if parseEncodeErr != nil {
		return parseZero, fmt.Errorf("projection: encoding %q: %w", parseCommand.name, parseEncodeErr)
	}

	parseResponse, parseSendErr := parseClient.Send(parseCtx, parseCommand.name, parseRequest)
	if parseSendErr != nil {
		return parseZero, fmt.Errorf("projection: sending %q: %w", parseCommand.name, parseSendErr)
	}

	var parseResult R
	if parseDecodeErr := parseCodec.Decode(parseResponse, &parseResult); parseDecodeErr != nil {
		return parseZero, fmt.Errorf("projection: decoding the result of %q: %w", parseCommand.name, parseDecodeErr)
	}
	return parseResult, nil
}

// Registry is the worker-side set of command names that exist.
//
// It closes the one gap the type system cannot: the compiler checks that a
// caller spelled the VARIABLE correctly, but not that the name string inside
// the declaration matches a handler the worker actually has. Verifying the whole
// set once at startup turns that into a boot-time failure with a list, rather
// than a runtime failure on the first click of a rarely-used feature.
type Registry struct {
	knownNames map[string]bool
}

// NewRegistry creates a registry from the names the worker handles.
func NewRegistry(parseNames []string) *Registry {
	parseKnown := make(map[string]bool, len(parseNames))
	for _, parseName := range parseNames {
		parseKnown[parseName] = true
	}
	return &Registry{knownNames: parseKnown}
}

// Declared is a command declaration erased to its name, so a heterogeneous set
// of commands can be verified together.
type Declared interface {
	Name() string
}

// Verify reports every declared command the worker cannot handle.
//
// It returns all of them rather than the first: a version skew between app and
// worker usually breaks several commands at once, and fixing them one boot at a
// time is needless.
func (parseRegistry *Registry) Verify(parseCommands ...Declared) error {
	if parseRegistry == nil {
		return errors.New("projection: registry is nil")
	}

	var parseMissing []string
	for _, parseCommand := range parseCommands {
		if parseCommand == nil {
			return errors.New("projection: a nil command was declared")
		}
		if parseCommand.Name() == "" {
			return errors.New("projection: a command was declared without a name")
		}
		if !parseRegistry.knownNames[parseCommand.Name()] {
			parseMissing = append(parseMissing, parseCommand.Name())
		}
	}
	if len(parseMissing) > 0 {
		return fmt.Errorf("projection: the domain worker does not handle %d declared command(s): %v",
			len(parseMissing), parseMissing)
	}
	return nil
}
