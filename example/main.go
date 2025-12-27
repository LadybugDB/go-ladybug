package main

import (
	"fmt"

	lbug "github.com/LadybugDB/go-ladybug"
)

// To run this example with Option 2 (local library download), use:
//
// 1. Download libraries:
//    go generate ./...
//
// 2. Build/Run with the system_ladybug tag:
//    go run -tags system_ladybug main.go

//go:generate sh -c "curl -sL https://raw.githubusercontent.com/LadybugDB/go-ladybug/refs/heads/master/download_lbug.sh | bash -s -- -out lib-ladybug"

/*
#cgo darwin LDFLAGS: -L${SRCDIR}/lib-ladybug -Wl,-rpath,${SRCDIR}/lib-ladybug
#cgo linux LDFLAGS: -L${SRCDIR}/lib-ladybug -Wl,-rpath,${SRCDIR}/lib-ladybug
#cgo windows LDFLAGS: -L${SRCDIR}/lib-ladybug
*/
import "C"

func main() {
	// Use an in-memory database for demonstration.
	dbPath := ":memory:"

	// Open a database with default system configuration.
	systemConfig := lbug.DefaultSystemConfig()
	systemConfig.BufferPoolSize = 1024 * 1024 * 1024
	db, err := lbug.OpenDatabase(dbPath, systemConfig)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Open a connection to the database.
	conn, err := lbug.OpenConnection(db)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Set up the schema and load data.
	queries := []string{
		"CREATE NODE TABLE User(name STRING, age INT64, PRIMARY KEY (name))",
		"CREATE NODE TABLE City(name STRING, population INT64, PRIMARY KEY (name))",
		"CREATE REL TABLE Follows(FROM User TO User, since INT64)",
		"CREATE REL TABLE LivesIn(FROM User TO City)",
		"COPY User FROM \"../dataset/demo-db/user.csv\"",
		"COPY City FROM \"../dataset/demo-db/city.csv\"",
		"COPY Follows FROM \"../dataset/demo-db/follows.csv\"",
		"COPY LivesIn FROM \"../dataset/demo-db/lives-in.csv\"",
	}
	for _, query := range queries {
		fmt.Println("Executing query:", query)
		queryResult, err := conn.Query(query)
		if err != nil {
			panic(err)
		}
		defer queryResult.Close()
	}

	query := "MATCH (a:User)-[e:Follows]->(b:User) RETURN a.name, e.since, b.name"
	println("Executing query:", query)
	// Execute a query and print the result.
	result, err := conn.Query(query)
	if err != nil {
		panic(err)
	}
	defer result.Close()
	for result.HasNext() {
		tuple, err := result.Next()
		if err != nil {
			panic(err)
		}
		defer tuple.Close()
		// The result is a tuple, which can be converted to a slice or a map.
		slice, err := tuple.GetAsSlice()
		if err != nil {
			panic(err)
		}
		fmt.Println(slice)
		m, err := tuple.GetAsMap()
		if err != nil {
			panic(err)
		}
		fmt.Println(m)
	}

	// Execute a query with parameters.
	query = "MATCH (a:User) WHERE a.name = $name RETURN a.age"
	println("Executing query:", query)
	preparedStatement, err := conn.Prepare(query)
	if err != nil {
		panic(err)
	}
	defer preparedStatement.Close()
	args := map[string]interface{}{"name": "Adam"}
	result, err = conn.Execute(preparedStatement, args)
	if err != nil {
		panic(err)
	}
	defer result.Close()
	for result.HasNext() {
		tuple, err := result.Next()
		if err != nil {
			panic(err)
		}
		defer tuple.Close()
		// The tuple can also be converted to a string.
		fmt.Print(tuple.GetAsString())
	}

	fmt.Println("All queries executed successfully.")
}
