package validation

import (
	"strings"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
)

func validatePayloadCodec(issues *types.ValidationErrors, prefix string, codec *contracts.PayloadCodec) {
	if codec == nil || !codec.Enabled {
		return
	}

	script := strings.TrimSpace(codec.Script)
	if script == "" {
		issues.Add(prefix+".script", "required", "payload codec script is required when the codec is enabled")
		return
	}

	program, err := parser.ParseFile(nil, "payload-codec.js", script, 0)
	if err != nil {
		issues.Add(prefix+".script", "invalid_javascript", "payload codec script contains invalid JavaScript syntax")
		return
	}

	for _, statement := range program.Body {
		var bindings []*ast.Binding
		switch declaration := statement.(type) {
		case *ast.FunctionDeclaration:
			function := declaration.Function
			if function.Name != nil && function.Name.Name == "decodeUplink" && !function.Async && !function.Generator {
				return
			}
		case *ast.VariableStatement:
			bindings = declaration.List
		case *ast.LexicalDeclaration:
			bindings = declaration.List
		}
		for _, binding := range bindings {
			identifier, ok := binding.Target.(*ast.Identifier)
			if !ok || identifier.Name != "decodeUplink" {
				continue
			}
			switch function := binding.Initializer.(type) {
			case *ast.FunctionLiteral:
				if !function.Async && !function.Generator {
					return
				}
			case *ast.ArrowFunctionLiteral:
				if !function.Async {
					return
				}
			}
		}
	}
	issues.Add(prefix+".script", "missing_decoder", "payload codec script must define a synchronous top-level decodeUplink(input) function")
}
