package scriptengines

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/ichiban/prolog"
	"github.com/ichiban/prolog/engine"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/util"
)

type PrologEngine struct {
}

func (e *PrologEngine) Run(ctx *util.FragsContext, code string, query string, runner frags.ExportableRunner) (any, error) {
	localCtx := ctx.Child(30 * time.Second)
	defer localCtx.Cancel(nil)
	interpreter := e.init()
	if runner != nil {
		e.initializeConsult(interpreter, runner.Components())
	}
	if err := interpreter.Compile(ctx, code); err != nil {
		return nil, err
	}
	parser := engine.NewParser(&interpreter.VM, strings.NewReader(query))
	_, err := parser.Term()

	solutions, err := interpreter.QueryContext(localCtx, query)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0)
	defer func() {
		_ = solutions.Close()
	}()

	for solutions.Next() {
		solMap := ExtractBindings(solutions)
		out = append(out, solMap)
	}
	if solutions.Err() != nil {
		return nil, solutions.Err()
	}
	return out, nil
}

func (e *PrologEngine) initializeConsult(i *prolog.Interpreter, components frags.Components) {
	i.Register1(engine.NewAtom("consult"), func(vm *engine.VM, pathTerm engine.Term, k engine.Cont, env *engine.Env) *engine.Promise {
		// 1. De-reference the Prolog term using the current environment
		resolved := env.Resolve(pathTerm)

		// 2. Cast it to an Atom so you can extract the string path
		pathAtom, ok := resolved.(engine.Atom)
		if !ok {
			return engine.Error(fmt.Errorf("type_error: consult/1 requires an atom path"))
		}

		virtualPath := pathAtom.String()

		// 3. Pull the content from your sandboxed storage
		component, exists := components.Scripts[virtualPath]
		if !exists {
			return engine.Error(fmt.Errorf("existence_error: sandboxed file '%s' not found", virtualPath))
		}

		scriptContent := component.Script

		// 4. Compile it directly into the interpreter instance
		if err := i.Exec(scriptContent); err != nil {
			return engine.Error(fmt.Errorf("syntax_error: failed to compile '%s': %w", virtualPath, err))
		}

		// 5. Pass execution to the next continuation block (signals success)
		return k(env)
	})
}

func ExtractBindings(sols *prolog.Solutions) map[string]string {
	res := make(map[string]string)

	v := reflect.ValueOf(sols)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var env *engine.Env
	var varsVal reflect.Value

	// 1. Walk nested structs to find the real `env` and `vars`
	var findFields func(val reflect.Value)
	findFields = func(val reflect.Value) {
		if !val.IsValid() {
			return
		}
		if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
			if val.IsNil() {
				return
			}
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			return
		}

		t := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldType := t.Field(i)
			if !field.CanInterface() {
				field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
			}

			if fieldType.Name == "env" {
				if e, ok := field.Interface().(*engine.Env); ok {
					env = e
				}
			} else if fieldType.Name == "vars" {
				varsVal = field
			} else if field.Kind() == reflect.Struct || field.Kind() == reflect.Ptr || field.Kind() == reflect.Interface {
				findFields(field)
			}
		}
	}

	findFields(v)

	if env == nil || !varsVal.IsValid() || varsVal.Kind() != reflect.Slice {
		return res
	}
	for i := 0; i < varsVal.Len(); i++ {
		pv := varsVal.Index(i)
		nameField := pv.FieldByName("Name")
		var varName string
		if atom, ok := nameField.Interface().(engine.Atom); ok {
			varName = atom.String()
		} else {
			atomPtr := (*engine.Atom)(unsafe.Pointer(nameField.UnsafeAddr()))
			varName = atomPtr.String()
		}

		if varName == "" || varName == "_" {
			continue
		}
		varField := pv.FieldByName("Variable")
		var variable engine.Variable
		if v, ok := varField.Interface().(engine.Variable); ok {
			variable = v
		} else {
			variable = *(*engine.Variable)(unsafe.Pointer(varField.UnsafeAddr()))
		}
		resolved := env.Resolve(variable)
		if resolved == nil {
			continue
		}
		res[varName] = formatTerm(env, resolved)
	}

	return res
}

// formatTerm recursively resolves and formats engine.Term into clean Prolog text
func formatTerm(env *engine.Env, term engine.Term) string {
	if term == nil {
		return ""
	}

	// Always resolve variables through the environment
	term = env.Resolve(term)

	switch t := term.(type) {
	case engine.Atom:
		return t.String()

	case engine.Integer:
		return fmt.Sprintf("%d", t)

	case engine.Float:
		return fmt.Sprintf("%g", t)

	case engine.Variable:
		// Unbound/free variable
		return fmt.Sprintf("_%d", t)

	case engine.Compound:
		functorStr := t.Functor().String()
		arity := t.Arity()

		// Special handling for Prolog lists: '.'(Head, Tail)
		if functorStr == "." && arity == 2 {
			var elements []string
			curr := term
			for {
				curr = env.Resolve(curr)
				list, ok := curr.(engine.Compound)
				if !ok || list.Functor().String() != "." || list.Arity() != 2 {
					break
				}
				elements = append(elements, formatTerm(env, list.Arg(0)))
				curr = list.Arg(1)
			}

			// Check if terminated nicely with empty list '[]'
			if nilAtom, ok := env.Resolve(curr).(engine.Atom); ok && nilAtom.String() == "[]" {
				return "[" + strings.Join(elements, ", ") + "]"
			}

			// Tail-improper list fallback (e.g., [1, 2 | Rest])
			return "[" + strings.Join(elements, ", ") + " | " + formatTerm(env, curr) + "]"
		}

		// Standard compound term (e.g., schedule(A, B))
		args := make([]string, arity)
		for i := 0; i < arity; i++ {
			args[i] = formatTerm(env, t.Arg(i))
		}
		return fmt.Sprintf("%s(%s)", functorStr, strings.Join(args, ", "))

	default:
		return fmt.Sprintf("%v", t)
	}
}
func (e *PrologEngine) init() *prolog.Interpreter {
	p := new(prolog.Interpreter)

	// 1. Register the raw op/3 predicate first
	p.Register3(engine.NewAtom("op"), engine.Op)

	// Register your other built-ins
	p.Register2(engine.NewAtom("is"), engine.Is)
	p.Register2(engine.NewAtom("=:="), engine.Equal)
	p.Register2(engine.NewAtom("=\\="), engine.NotEqual)
	p.Register2(engine.NewAtom("<"), engine.LessThan)
	p.Register2(engine.NewAtom("=<"), engine.LessThanOrEqual)
	p.Register2(engine.NewAtom(">"), engine.GreaterThan)
	p.Register2(engine.NewAtom(">="), engine.GreaterThanOrEqual)
	p.Register2(engine.NewAtom("="), engine.Unify)
	p.Register2(engine.NewAtom("unify_with_occurs_check"), engine.UnifyWithOccursCheck)
	p.Register2(engine.NewAtom("keysort"), engine.KeySort)
	p.Register3(engine.NewAtom("compare"), engine.Compare)
	p.Register1(engine.NewAtom("var"), engine.TypeVar)
	p.Register1(engine.NewAtom("atom"), engine.TypeAtom)
	p.Register1(engine.NewAtom("integer"), engine.TypeInteger)
	p.Register1(engine.NewAtom("float"), engine.TypeFloat)
	p.Register1(engine.NewAtom("compound"), engine.TypeCompound)
	p.Register1(engine.NewAtom("current_predicate"), engine.CurrentPredicate)
	// 2. Define the essential operator rules
	// We run these as separate, single-term queries so the parser doesn't need ',' to read them.
	operators := []string{
		"op(1200, xfx, ':-').", // rule operator (infix)
		"op(1200, fx, ':-').",  // directive operator (prefix)
		"op(1000, xfy, ',').",  // conjunction / AND (infix)
		"op(700, xfx, '=').",   // unification (infix)
		"op(700, xfx, 'is').",  // arithmetic evaluation (infix)

		// Arithmetic comparison operators
		"op(700, xfx, '<').",
		"op(700, xfx, '>').",
		"op(700, xfx, '=<').",
		"op(700, xfx, '>=').",
		"op(700, xfx, '=\\=').",
		"op(700, xfx, '=:=').",

		// Basic math operators (so `X is A + B` parses nicely)
		"op(500, yfx, '+').",
		"op(500, yfx, '-').",
		"op(400, yfx, '*').",
		"op(400, yfx, '/').",
	}

	for _, opQuery := range operators {
		sols, err := p.Query(opQuery)
		if err == nil {
			// In ichiban/prolog, we must advance the iterator and close it
			// to ensure the query actually executes and releases resources.
			if sols.Next() {
				_ = sols.Close()
			}
		}
	}
	return p
}

func NewPrologEngine() *PrologEngine {
	return &PrologEngine{}
}
