package lbug

import (
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBorrowedValueOwnershipRecursiveRelationship(t *testing.T) {
	runBorrowedValueOwnershipScenario(t, "recursive_relationship")
}

func TestBorrowedValueOwnershipNestedContainers(t *testing.T) {
	runBorrowedValueOwnershipScenario(t, "nested_containers")
}

func TestBorrowedValueOwnershipSubprocess(t *testing.T) {
	scenario := os.Getenv("LBUG_BORROWED_VALUE_SCENARIO")
	if scenario == "" {
		t.Skip("subprocess helper")
	}

	prevGCPercent := debug.SetGCPercent(1)
	defer debug.SetGCPercent(prevGCPercent)

	switch scenario {
	case "recursive_relationship":
		runRecursiveRelationshipOwnershipStress(t)
	case "nested_containers":
		runNestedContainerOwnershipStress(t)
	default:
		t.Fatalf("unknown ownership scenario %q", scenario)
	}
}

func runBorrowedValueOwnershipScenario(t *testing.T, scenario string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestBorrowedValueOwnershipSubprocess$")
	cmd.Env = append(os.Environ(), "LBUG_BORROWED_VALUE_SCENARIO="+scenario)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "scenario %s crashed:\n%s", scenario, string(output))
}

func runRecursiveRelationshipOwnershipStress(t *testing.T) {
	db, conn := setupTestDatabase(t)
	defer db.Close()
	defer conn.Close()

	createTestData(t, conn, 250)

	runOwnershipStress(t, 8, 12, func() error {
		return queryRecursiveRelationships(conn)
	})
}

func runNestedContainerOwnershipStress(t *testing.T) {
	_, conn := SetupTestDatabase(t)

	runOwnershipStress(t, 8, 20, func() error {
		return queryNestedContainers(conn)
	})
}

func runOwnershipStress(t *testing.T, numGoroutines int, queriesPerGoroutine int, queryFn func() error) {
	t.Helper()

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*queriesPerGoroutine)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range queriesPerGoroutine {
				if err := queryFn(); err != nil {
					errChan <- err
					return
				}
				runtime.GC()
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		require.NoError(t, err)
	}
}

func queryRecursiveRelationships(conn *Connection) error {
	result, err := conn.Query(`
		MATCH (source:Node {id: 0})-[r:CONNECTS* ALL SHORTEST 1..3]->(target:Node {id: 4})
		RETURN r, source.fqn, target.fqn
	`)
	if err != nil {
		return err
	}
	defer result.Close()

	rowCount := 0
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return err
		}

		value, err := row.GetValue(0)
		if err != nil {
			row.Close()
			return err
		}

		recursiveRel := value.(RecursiveRelationship)
		if len(recursiveRel.Relationships) != 2 {
			row.Close()
			return requireLengthError("recursive relationships", 2, len(recursiveRel.Relationships))
		}

		row.Close()
		rowCount++
	}

	if rowCount == 0 {
		return requireLengthError("recursive relationship rows", 1, 0)
	}

	return nil
}

func queryNestedContainers(conn *Connection) error {
	result, err := conn.Query(`
		MATCH (p:person)-[r:workAt]->(o:organisation)
		WHERE p.ID = 5
		RETURN
			p,
			r,
			p.courseScoresPerTerm,
			{
				name: p.fName,
				scores: p.courseScoresPerTerm,
				usedNames: p.usedNames
			},
			o
	`)
	if err != nil {
		return err
	}
	defer result.Close()

	if !result.HasNext() {
		return requireLengthError("nested container rows", 1, 0)
	}

	row, err := result.Next()
	if err != nil {
		return err
	}
	defer row.Close()

	values, err := row.GetAsSlice()
	if err != nil {
		return err
	}
	if len(values) != 5 {
		return requireLengthError("nested container values", 5, len(values))
	}

	person := values[0].(Node)
	if person.Label != "person" || person.Properties["ID"] != int64(5) {
		return errUnexpectedValue("person payload")
	}

	rel := values[1].(Relationship)
	if rel.Label != "workAt" || rel.Properties["year"] != int64(2010) {
		return errUnexpectedValue("relationship payload")
	}

	scores := values[2].([]any)
	if len(scores) == 0 {
		return errUnexpectedValue("scores payload")
	}

	profile := values[3].(map[string]any)
	if profile["name"] != person.Properties["fName"] {
		return errUnexpectedValue("profile payload")
	}

	org := values[4].(Node)
	if org.Label != "organisation" {
		return errUnexpectedValue("organisation payload")
	}

	return nil
}

func requireLengthError(name string, want int, got int) error {
	return &ownershipError{name: name, message: "unexpected length", want: want, got: got}
}

func errUnexpectedValue(name string) error {
	return &ownershipError{name: name, message: "unexpected value"}
}

type ownershipError struct {
	name    string
	message string
	want    int
	got     int
}

func (e *ownershipError) Error() string {
	if e.want == 0 && e.got == 0 {
		return e.name + ": " + e.message
	}
	return e.name + ": " + e.message
}
