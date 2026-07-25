package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const serverFixture = `//go:build !js || !wasm

package orders

import "context"

type CreateOrderReq struct {
	SKU string ` + "`json:\"sku\"`" + `
}

type CreateOrderResp struct {
	ID string ` + "`json:\"id\"`" + `
}

//gwc:server
func CreateOrder(ctx context.Context, req CreateOrderReq) (CreateOrderResp, error) {
	return CreateOrderResp{ID: "o1"}, nil
}

//gwc:server
func CancelOrder(ctx context.Context, req CreateOrderReq) (CreateOrderResp, error) {
	return CreateOrderResp{}, nil
}

// NotAServerFunction is a plain function with no directive.
func NotAServerFunction(ctx context.Context, req CreateOrderReq) (CreateOrderResp, error) {
	return CreateOrderResp{}, nil
}
`

func writeServerFixture(parseT *testing.T) string {
	parseT.Helper()
	parseDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseDir, "orders.go"), []byte(serverFixture), 0644); parseErr != nil {
		parseT.Fatalf("write fixture: %v", parseErr)
	}
	return parseDir
}

// firstFunc parses a one-function source snippet and returns its FuncDecl + fset for the
// signature-validation tests.
func firstFunc(parseT *testing.T, parseSrc string) (*token.FileSet, *ast.FuncDecl) {
	parseT.Helper()
	parseFset := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFset, "x.go", "package p\nimport \"context\"\nvar _ = context.Background\n"+parseSrc, parser.ParseComments)
	if parseErr != nil {
		parseT.Fatalf("parse snippet: %v", parseErr)
	}
	for _, parseDecl := range parseFile.Decls {
		if parseFuncDecl, parseOk := parseDecl.(*ast.FuncDecl); parseOk {
			return parseFset, parseFuncDecl
		}
	}
	parseT.Fatal("no function in snippet")
	return nil, nil
}

// TestCollectServerFuncsFindsDirectiveFunctions proves only //gwc:server functions are
// collected, sorted, with their request/response types captured.
func TestCollectServerFuncsFindsDirectiveFunctions(parseT *testing.T) {
	parsePkg, parseFuncs, parseErr := collectServerFuncs(writeServerFixture(parseT))
	if parseErr != nil {
		parseT.Fatalf("collectServerFuncs: %v", parseErr)
	}
	if parsePkg != "orders" {
		parseT.Fatalf("pkg name: got %q", parsePkg)
	}
	parseNames := make([]string, len(parseFuncs))
	for parseI, parseFunc := range parseFuncs {
		parseNames[parseI] = parseFunc.name
	}
	if strings.Join(parseNames, ",") != "CancelOrder,CreateOrder" {
		parseT.Fatalf("expected sorted [CancelOrder CreateOrder], got %v", parseNames)
	}
	if parseFuncs[1].reqType != "CreateOrderReq" || parseFuncs[1].respType != "CreateOrderResp" {
		parseT.Fatalf("unexpected types: %+v", parseFuncs[1])
	}
}

// TestCollectServerFuncsRejectsUnconstrainedFile proves the collision guard: a file with
// server functions that is not constrained off the client build errors.
func TestCollectServerFuncsRejectsUnconstrainedFile(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseSrc := strings.Replace(serverFixture, "//go:build !js || !wasm\n\n", "", 1)
	if parseErr := os.WriteFile(filepath.Join(parseDir, "orders.go"), []byte(parseSrc), 0644); parseErr != nil {
		parseT.Fatalf("write: %v", parseErr)
	}
	if _, _, parseErr := collectServerFuncs(parseDir); parseErr == nil || !strings.Contains(parseErr.Error(), "not constrained to the server") {
		parseT.Fatalf("expected an unconstrained-file error, got %v", parseErr)
	}
}

// TestValidateServerFuncRejectsBadShapes proves the contract is enforced with specific
// errors.
func TestValidateServerFuncRejectsBadShapes(parseT *testing.T) {
	parseCases := map[string]string{
		"must take exactly":            "//gwc:server\nfunc F(ctx context.Context) (R, error) { return R{}, nil }\ntype R struct{}",
		"first parameter must be":      "//gwc:server\nfunc F(s string, req R) (R, error) { return R{}, nil }\ntype R struct{}",
		"second result must be error":  "//gwc:server\nfunc F(ctx context.Context, req R) (R, R) { return R{}, R{} }\ntype R struct{}",
		"must return exactly":          "//gwc:server\nfunc F(ctx context.Context, req R) error { return nil }\ntype R struct{}",
		"defined in this package":      "//gwc:server\nfunc F(ctx context.Context, req R) (time.Time, error) { return time.Time{}, nil }\ntype R struct{}",
		"must be a top-level function": "//gwc:server\nfunc (s S) F(ctx context.Context, req R) (R, error) { return R{}, nil }\ntype R struct{}\ntype S struct{}",
	}
	for parseWant, parseSrc := range parseCases {
		parseFset, parseFuncDecl := firstFunc(parseT, parseSrc)
		_, parseErr := validateServerFunc(parseFset, parseFuncDecl)
		if parseErr == nil || !strings.Contains(parseErr.Error(), parseWant) {
			parseT.Fatalf("source %q: expected error containing %q, got %v", parseSrc, parseWant, parseErr)
		}
	}
}

// TestGenerateServerFilesAreValidAndTyped proves both generated files contain the expected
// typed stubs/registration and parse cleanly (format.Source would have errored otherwise).
func TestGenerateServerFilesAreValidAndTyped(parseT *testing.T) {
	_, parseFuncs, _ := collectServerFuncs(writeServerFixture(parseT))

	parseClient, parseErr := generateServerClientFile("orders", parseFuncs)
	if parseErr != nil {
		parseT.Fatalf("client gen: %v", parseErr)
	}
	for _, parseWant := range []string{
		"//go:build js && wasm",
		"// Code generated by `gwc server gen`; DO NOT EDIT.",
		"package orders",
		`"github.com/monstercameron/GoWebComponents/v5/serverfn"`,
		"func CreateOrder(parseCtx context.Context, parseReq CreateOrderReq) (CreateOrderResp, error) {",
		`return serverfn.Call[CreateOrderReq, CreateOrderResp](parseCtx, "CreateOrder", parseReq)`,
	} {
		if !strings.Contains(parseClient, parseWant) {
			parseT.Fatalf("client missing %q:\n%s", parseWant, parseClient)
		}
	}

	parseServer, parseErr := generateServerRegistrationFile("orders", parseFuncs)
	if parseErr != nil {
		parseT.Fatalf("server gen: %v", parseErr)
	}
	for _, parseWant := range []string{
		"//go:build !js || !wasm",
		"func RegisterServerFunctions(parseMux *http.ServeMux) {",
		`serverfn.Handle(parseMux, "CreateOrder", CreateOrder)`,
		`serverfn.Handle(parseMux, "CancelOrder", CancelOrder)`,
	} {
		if !strings.Contains(parseServer, parseWant) {
			parseT.Fatalf("server missing %q:\n%s", parseWant, parseServer)
		}
	}
}

// TestFileExcludesClientBuild proves the build-constraint detection used by the collision
// guard.
func TestFileExcludesClientBuild(parseT *testing.T) {
	parseCases := map[string]bool{
		"//go:build !js || !wasm\n\npackage p\n": true,
		"//go:build linux\n\npackage p\n":        true,  // js,wasm build excludes it
		"//go:build js && wasm\n\npackage p\n":   false, // included in the client build
		"package p\n":                            false, // no constraint
	}
	for parseSrc, parseWant := range parseCases {
		parseDir := parseT.TempDir()
		parsePath := filepath.Join(parseDir, "f.go")
		if parseErr := os.WriteFile(parsePath, []byte(parseSrc), 0644); parseErr != nil {
			parseT.Fatalf("write: %v", parseErr)
		}
		if parseGot := fileExcludesClientBuild(parsePath); parseGot != parseWant {
			parseT.Fatalf("fileExcludesClientBuild(%q) = %v, want %v", parseSrc, parseGot, parseWant)
		}
	}
}

// TestServerStaleness proves the CI gate detects out-of-date generated files.
func TestServerStaleness(parseT *testing.T) {
	parseDir := writeServerFixture(parseT)
	_, parseFuncs, _ := collectServerFuncs(parseDir)
	parseClientSrc, _ := generateServerClientFile("orders", parseFuncs)

	parseClientPath := filepath.Join(parseDir, serverClientGenFile)
	if routesFileMatches(parseClientPath, parseClientSrc) {
		parseT.Fatal("no generated file should not already match")
	}
	_ = os.WriteFile(parseClientPath, []byte(parseClientSrc), 0644)
	if !routesFileMatches(parseClientPath, parseClientSrc) {
		parseT.Fatal("expected fresh after writing the generated client stubs")
	}
}
