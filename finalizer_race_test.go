package lbug

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// The race only manifests when running MULTIPLE tests in batch, not in isolation.
// That is, if you run the test individually it won't fail.
// But if you run "go test -v" it will.
//
// Key insight: The issue is accumulated GC pressure across test sessions.
// When test A creates QueryResults and test B runs, the GC may finalize
// QueryResult A while test B's iteration is still in progress.

// The race occurs when:
// 1. QueryResult is finalized while FlatTuple.GetValue() is still accessing C memory
// 2. The GC runs during lbugValueToGoValue() and destroys the parent QueryResult
// 3. The finalizer calls lbug_query_result_destroy() on memory still in use
func TestFinalizerRaceCondition(t *testing.T) {
	// Skip if running with -short (this test can be slow and flaky)
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	db, conn := setupTestDatabase(t)
	defer db.Close()
	defer conn.Close()

	createTestData(t, conn, 100000)

	const numGoroutines = 20
	const queriesPerGoroutine = 30

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*queriesPerGoroutine)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for q := 0; q < queriesPerGoroutine; q++ {
				// Query without storing result in a variable that persists
				// This pattern allows the QueryResult to become "unreachable" quickly
				if err := runLargeQueryAndIterate(conn); err != nil {
					errChan <- err
					return
				}

				// Force GC to increase likelihood of triggering the race
				runtime.GC()
			}
		}(g)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Fatalf("got %d errors during concurrent queries: %v", len(errors), errors[0])
	}
}

// setupTestDatabase creates an in-memory database with test schema.
//
// Returns the database and connection, which the caller must close.
func setupTestDatabase(t *testing.T) (*Database, *Connection) {
	t.Helper()

	db, err := OpenDatabase(":memory:", DefaultSystemConfig())
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	conn, err := OpenConnection(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to open connection: %v", err)
	}

	schemas := []string{
		`CREATE NODE TABLE Node (
			id INT64,
			name STRING,
			fqn STRING,
			category STRING,
			file_path STRING,
			PRIMARY KEY (id)
		)`,
		`CREATE REL TABLE CONNECTS (
			FROM Node TO Node,
			label STRING
		)`,
	}

	for _, schema := range schemas {
		result, err := conn.Query(schema)
		if err != nil {
			conn.Close()
			db.Close()
			t.Fatalf("failed to create schema: %v", err)
		}
		result.Close()
	}

	return db, conn
}

// createTestData populates the database with synthetic test data.
// Creates nodes and relationships to simulate a real codebase.
//
// numNodes: number of DefinitionNode records to create
func createTestData(t *testing.T, conn *Connection, numNodes int) {
	t.Helper()

	const batchSize = 100
	for i := 0; i < numNodes; i += batchSize {
		end := i + batchSize
		if end > numNodes {
			end = numNodes
		}

		for j := i; j < end; j++ {
			query := fmt.Sprintf(`
				CREATE (n:Node {
					id: %d,
					name: 'item_%d',
					fqn: 'src/module%d.item_%d',
					category: 'entity',
					file_path: 'src/module%d.ext'
				})
			`, j, j, j/10, j, j/10)

			result, err := conn.Query(query)
			if err != nil {
				t.Fatalf("failed to insert node %d: %v", j, err)
			}
			result.Close()
		}
	}

	for i := 0; i < numNodes-3; i++ {
		for offset := 1; offset <= 3; offset++ {
			query := fmt.Sprintf(`
				MATCH (from:Node {id: %d})
				MATCH (to:Node {id: %d})
				CREATE (from)-[:CONNECTS {label: 'links'}]->(to)
			`, i, i+offset)

			result, err := conn.Query(query)
			if err != nil {
				continue
			}
			result.Close()
		}
	}
}

// runLargeQueryAndIterate executes a query returning OLAP-scale results (15k+ rows).
// This matches real-world usage patterns where large result sets create GC pressure.
// The query returns all CONNECTS relationships with multiple columns per row.
//
// Returns error if the query or iteration fails.
func runLargeQueryAndIterate(conn *Connection) error {
	result, err := conn.Query(`
		MATCH (source:Node)-[r:CONNECTS]->(target:Node)
		RETURN source.file_path, source.fqn, source.id, 
		       target.file_path, target.fqn, target.id,
		       r.label
		LIMIT 15000
	`)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	rowCount := 0
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return fmt.Errorf("Next() failed at row %d: %w", rowCount, err)
		}

		// Access all 7 columns - each GetValue enters a race
		for col := uint64(0); col < 7; col++ {
			_, err = row.GetValue(col)
			if err != nil {
				return fmt.Errorf("GetValue(%d) failed at row %d: %w", col, rowCount, err)
			}
		}

		rowCount++
	}

	return nil
}