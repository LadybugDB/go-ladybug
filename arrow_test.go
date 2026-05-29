package lbug

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupArrowTestConnection(t *testing.T) (*Database, *Connection) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "testdb")
	dbPath = strings.ReplaceAll(dbPath, "\\", "/")
	db, err := OpenDatabase(dbPath, DefaultSystemConfig())
	require.NoError(t, err)
	conn, err := OpenConnection(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
		db.Close()
	})
	return db, conn
}

func newPersonBatch(t *testing.T, alloc memory.Allocator) arrow.RecordBatch {
	t.Helper()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int64},
		{Name: "name", Type: arrow.BinaryTypes.String},
	}, nil)
	builder := array.NewRecordBuilder(alloc, schema)
	defer builder.Release()
	builder.Field(0).(*array.Int64Builder).AppendValues([]int64{1, 2}, nil)
	builder.Field(1).(*array.StringBuilder).AppendValues([]string{"Alice", "Bob"}, nil)
	batch := builder.NewRecordBatch()
	t.Cleanup(batch.Release)
	return batch
}

func TestCreateArrowTableAndDrop(t *testing.T) {
	_, conn := setupArrowTestConnection(t)
	alloc := memory.DefaultAllocator
	batch := newPersonBatch(t, alloc)

	result, err := conn.CreateArrowTable("Person", []arrow.RecordBatch{batch})
	require.NoError(t, err)
	result.Close()

	result, err = conn.Query("MATCH (p:Person) RETURN p.name ORDER BY p.id;")
	require.NoError(t, err)
	assert.Equal(t, "p.name\nAlice\nBob\n", result.ToString())
	result.Close()

	result, err = conn.DropArrowTable("Person")
	require.NoError(t, err)
	result.Close()

	_, err = conn.Query("MATCH (p:Person) RETURN p.name;")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Table Person does not exist")
}

func TestCreateArrowRelTableCSR(t *testing.T) {
	_, conn := setupArrowTestConnection(t)
	alloc := memory.DefaultAllocator
	nodes := newPersonBatch(t, alloc)
	result, err := conn.CreateArrowTable("Person", []arrow.RecordBatch{nodes})
	require.NoError(t, err)
	result.Close()

	indicesSchema := arrow.NewSchema([]arrow.Field{
		{Name: "to", Type: arrow.PrimitiveTypes.Uint64},
		{Name: "weight", Type: arrow.PrimitiveTypes.Int64},
	}, nil)
	indicesBuilder := array.NewRecordBuilder(alloc, indicesSchema)
	defer indicesBuilder.Release()
	indicesBuilder.Field(0).(*array.Uint64Builder).AppendValues([]uint64{1, 0}, nil)
	indicesBuilder.Field(1).(*array.Int64Builder).AppendValues([]int64{7, 9}, nil)
	indices := indicesBuilder.NewRecordBatch()
	defer indices.Release()

	indptrSchema := arrow.NewSchema([]arrow.Field{
		{Name: "indptr", Type: arrow.PrimitiveTypes.Uint64},
	}, nil)
	indptrBuilder := array.NewRecordBuilder(alloc, indptrSchema)
	defer indptrBuilder.Release()
	indptrBuilder.Field(0).(*array.Uint64Builder).AppendValues([]uint64{0, 1, 2}, nil)
	indptr := indptrBuilder.NewRecordBatch()
	defer indptr.Release()

	result, err = conn.CreateArrowRelTableCSR("Knows", []arrow.RecordBatch{indices},
		[]arrow.RecordBatch{indptr}, "Person", "Person")
	require.NoError(t, err)
	result.Close()

	result, err = conn.Query("MATCH (a:Person)-[r:Knows]->(b:Person) RETURN a.id, r.weight, b.id ORDER BY a.id, b.id;")
	require.NoError(t, err)
	assert.Equal(t, "a.id|r.weight|b.id\n1|7|2\n2|9|1\n", result.ToString())
	result.Close()
}

func TestQueryResultArrowChunk(t *testing.T) {
	_, conn := setupArrowTestConnection(t)
	result, err := conn.Query("RETURN CAST(1, \"INT64\") AS one;")
	require.NoError(t, err)
	defer result.Close()

	schema, err := result.GetArrowSchema()
	require.NoError(t, err)
	require.Len(t, schema.Fields(), 1)
	assert.Equal(t, "one", schema.Field(0).Name)

	batch, err := result.GetNextArrowChunk(8)
	require.NoError(t, err)
	defer batch.Release()
	require.Equal(t, int64(1), batch.NumRows())
	values := batch.Column(0).(*array.Int64)
	assert.Equal(t, int64(1), values.Value(0))
}
