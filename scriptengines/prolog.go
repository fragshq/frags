package scriptengines

import (
	"fmt"

	"github.com/ichiban/prolog"
	"github.com/ichiban/prolog/engine"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/util"
)

type PrologEngine struct {
}

func (e *PrologEngine) Run(ctx *util.FragsContext, code string, query string, runner frags.ExportableRunner) (any, error) {
	interpreter := e.init()
	e.initializeConsult(interpreter, runner.Components())
	if err := interpreter.Compile(ctx, code); err != nil {
		return nil, err
	}
	solutions, err := interpreter.Query(query)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0)
	defer func() {
		_ = solutions.Close()
	}()
	for solutions.Next() {
		s := make(map[string]any)
		if err = solutions.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
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
