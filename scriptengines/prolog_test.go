package scriptengines

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/util"
)

func TestPrologEngine_Query(t *testing.T) {
	e := NewPrologEngine()
	res, err := e.Run(util.WithFragsContext(context.Background(), 1*time.Minute), `
			% Facts: parent(Parent, Child)
			parent(albert, bob).
			parent(albert, betsy).
			parent(betsy, charlie).

			% Rule: X is a grandparent of Y if X is a parent of Z, and Z is a parent of Y.
			grandparent(X, Y) :- parent(X, Z), parent(Z, Y).
	`, "grandparent(CHARLIES_GRANDPARENT, charlie).", &frags.Runner{})
	require.NoError(t, err)
	fmt.Println(res)
}

func TestPrologLoad(t *testing.T) {
	e := NewPrologEngine()
	runner := frags.NewRunner(frags.SessionManager{
		Components: frags.Components{
			Scripts: map[string]frags.ScriptComponent{
				"the_script": {
					Type: "kbs",
					Script: `% Facts: parent(Parent, Child)
			parent(albert, bob).
			parent(albert, betsy).
			parent(betsy, charlie).

			% Rule: X is a grandparent of Y if X is a parent of Z, and Z is a parent of Y.
			grandparent(X, Y) :- parent(X, Z), parent(Z, Y).`,
				},
			},
		},
	}, nil, nil)
	res, err := e.Run(util.WithFragsContext(context.Background(), 1*time.Minute),
		":- consult('the_script').",
		"grandparent(CHARLIES_GRANDPARENT, charlie).", &runner)
	fmt.Println(res)
	fmt.Println(err)
}
