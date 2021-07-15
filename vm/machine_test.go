package vm

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	ledger "github.com/numary/ledger/core"
	"github.com/numary/machine/core"
	"github.com/numary/machine/script/compiler"
)

type CaseResult struct {
	Printed  []core.Value
	Postings []ledger.Posting
	ExitCode byte
	Error    string
}

type TestCase struct {
	Code      string
	Variables map[string]core.Value
	Expected  CaseResult
}

type TestCaseJSON struct {
	Code      string
	Variables map[string]core.Value
	Expected  CaseResult
}

func test(t *testing.T, code string, variables map[string]core.Value, balances map[string]map[string]uint64, expected CaseResult) {
	testimpl(t, code, expected, func(m *Machine) (byte, error) {
		return m.Execute(variables, balances)
	})
}

func testJSON(t *testing.T, code string, variables string, balances map[string]map[string]uint64, expected CaseResult) {
	testimpl(t, code, expected, func(m *Machine) (byte, error) {
		var v map[string]json.RawMessage
		err := json.Unmarshal([]byte(variables), &v)
		if err != nil {
			return 0, err
		}
		return m.ExecuteFromJSON(v, balances)
	})
}

func testimpl(t *testing.T, code string, expected CaseResult, exec func(*Machine) (byte, error)) {
	p, err := compiler.Compile(code)

	if err != nil {
		t.Error(fmt.Errorf("compile error: %v", err))
		return
	}

	printed := []core.Value{}

	var wg sync.WaitGroup
	wg.Add(1)

	machine := NewMachine(p)
	machine.Printer = func(c chan core.Value) {
		for v := range c {
			printed = append(printed, v)
		}
		wg.Done()
	}
	exit_code, err := exec(machine)

	if err != nil && expected.Error != "" {
		if !strings.Contains(err.Error(), expected.Error) {
			t.Error(fmt.Errorf("unexpected execution error: %v", err))
			return
		}
	} else if err != nil {
		t.Error(fmt.Errorf("did not expect an execution error: %v", err))
		return
	} else if expected.Error != "" {
		t.Error(fmt.Errorf("expected an execution error"))
		return
	}

	wg.Wait()

	if exit_code != expected.ExitCode {
		t.Error(fmt.Errorf("unexpected exit code: %v", exit_code))
		return
	}

	if len(machine.Postings) != len(expected.Postings) {
		t.Error(fmt.Errorf("unexpected postings output: %v", machine.Postings))
		return
	} else {
		for i := range machine.Postings {
			if machine.Postings[i] != expected.Postings[i] {
				t.Error(fmt.Errorf("unexpected postings output: %v", machine.Postings[i]))
				return
			}
		}
	}

	if len(printed) != len(expected.Printed) {
		t.Error(fmt.Errorf("unexpected print output: %v", printed))
		return
	} else {
		for i := range printed {
			if printed[i] != expected.Printed[i] {
				t.Error(fmt.Errorf("unexpected print output: %v", printed[i]))
				return
			}
		}
	}
}

func TestFail(t *testing.T) {
	test(t,
		"fail",
		map[string]core.Value{},
		map[string]map[string]uint64{},
		CaseResult{
			Printed:  []core.Value{},
			Postings: []ledger.Posting{},
			ExitCode: EXIT_FAIL,
		},
	)
}

func TestPrint(t *testing.T) {
	test(t,
		"print 29 + 15 - 2",
		map[string]core.Value{},
		map[string]map[string]uint64{},
		CaseResult{
			Printed:  []core.Value{core.Number(42)},
			Postings: []ledger.Posting{},
			ExitCode: EXIT_OK,
		},
	)
}

func TestSend(t *testing.T) {
	test(t,
		`send [EUR/2 100] (
			source=@alice
			destination=@bob
		)`,
		map[string]core.Value{},
		map[string]map[string]uint64{
			"alice": {
				"EUR/2": 100,
			},
		},
		CaseResult{
			Printed: []core.Value{},
			Postings: []ledger.Posting{
				{
					Asset:       "EUR/2",
					Amount:      100,
					Source:      "alice",
					Destination: "bob",
				},
			},
			ExitCode: EXIT_OK,
		},
	)
}

func TestVariables(t *testing.T) {
	test(t,
		`vars {
			account $rider
			account $driver
		}
		send [EUR/2 999] (
			source=$rider
			destination=$driver
		)`,
		map[string]core.Value{
			"rider":  core.Account("users:001"),
			"driver": core.Account("users:002"),
		},
		map[string]map[string]uint64{
			"users:001": {
				"EUR/2": 1000,
			},
		},
		CaseResult{
			Printed: []core.Value{},
			Postings: []ledger.Posting{
				{
					Asset:       "EUR/2",
					Amount:      999,
					Source:      "users:001",
					Destination: "users:002",
				},
			},
			ExitCode: EXIT_OK,
		},
	)
}

func TestVariablesJSON(t *testing.T) {
	testJSON(t,
		`vars {
			account $rider
			account $driver
		}
		send [EUR/2 999] (
			source=$rider
			destination=$driver
		)`,
		`{
			"rider": "users:001",
			"driver": "users:002"
		}`,
		map[string]map[string]uint64{
			"users:001": {
				"EUR/2": 1000,
			},
		},
		CaseResult{
			Printed: []core.Value{},
			Postings: []ledger.Posting{
				{
					Asset:       "EUR/2",
					Amount:      999,
					Source:      "users:001",
					Destination: "users:002",
				},
			},
			ExitCode: EXIT_OK,
		},
	)
}

func TestSource(t *testing.T) {
	testJSON(t,
		`vars {
	account $balance
	account $payment
	account $seller
}
send [GEM 15] (
	source = {
		$balance
		$payment
	}
	destination = $seller
)`,
		`{
			"balance": "users:001",
			"payment": "payments:001",
			"seller": "users:002"
		}`,
		map[string]map[string]uint64{
			"users:001": {
				"GEM": 3,
			},
			"payments:001": {
				"GEM": 12,
			},
		},
		CaseResult{
			Printed: []core.Value{},
			Postings: []ledger.Posting{
				{
					Asset:       "GEM",
					Amount:      12,
					Source:      "payments:001",
					Destination: "users:002",
				},
				{
					Asset:       "GEM",
					Amount:      3,
					Source:      "users:001",
					Destination: "users:002",
				},
			},
			ExitCode: EXIT_OK,
		},
	)
}

func TestAllocation(t *testing.T) {
	testJSON(t,
		`vars {
	account $rider
	account $driver
}
send [GEM 15] (
	source = $rider
	destination = {
		80% to $driver
		8% to @a
		12% to @b
	}
)`,
		`{
			"rider": "users:001",
			"driver": "users:002"
		}`,
		map[string]map[string]uint64{
			"users:001": {
				"GEM": 15,
			},
		},
		CaseResult{
			Printed: []core.Value{},
			Postings: []ledger.Posting{
				{
					Asset:       "GEM",
					Amount:      1,
					Source:      "users:001",
					Destination: "b",
				},
				{
					Asset:       "GEM",
					Amount:      1,
					Source:      "users:001",
					Destination: "a",
				},
				{
					Asset:       "GEM",
					Amount:      13,
					Source:      "users:001",
					Destination: "users:002",
				},
			},
			ExitCode: EXIT_OK,
		},
	)
}

func TestInsufficientFunds(t *testing.T) {
	testJSON(t,
		`vars {
	account $balance
	account $payment
	account $seller
}
send [GEM 16] (
	source = {
		$balance
		$payment
	}
	destination = $seller
)`,
		`{
			"balance": "users:001",
			"payment": "payments:001",
			"seller": "users:002"
		}`,
		map[string]map[string]uint64{
			"users:001": {
				"GEM": 3,
			},
			"payments:001": {
				"GEM": 12,
			},
		},
		CaseResult{
			Printed:  []core.Value{},
			Postings: []ledger.Posting{},
			ExitCode: EXIT_FAIL,
		},
	)
}

func TestMissingBalance(t *testing.T) {
	testJSON(t,
		`send [GEM 15] (
			source = @a
			destination = @a
		)`,
		`{}`,
		map[string]map[string]uint64{
			"users:001": {
				"GEM": 3,
			},
			"payments:001": {
				"USD/2": 564,
			},
		},
		CaseResult{
			Printed:  []core.Value{},
			Postings: []ledger.Posting{},
			ExitCode: 0,
			Error:    "missing balance",
		},
	)
}
