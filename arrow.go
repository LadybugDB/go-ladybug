package lbug

// #include "lbug.h"
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/cdata"
)

func exportArrowBatches(batches []arrow.RecordBatch) (*C.struct_ArrowSchema, []C.struct_ArrowArray, error) {
	if len(batches) == 0 {
		return nil, nil, fmt.Errorf("at least one Arrow record batch is required")
	}
	schema := batches[0].Schema()
	for i, batch := range batches {
		if !batch.Schema().Equal(schema) {
			return nil, nil, fmt.Errorf("Arrow record batch %d has a different schema", i)
		}
	}

	cSchema := new(C.struct_ArrowSchema)
	cArrays := make([]C.struct_ArrowArray, len(batches))
	cdata.ExportArrowSchema(schema, cdata.SchemaFromPtr(uintptr(unsafe.Pointer(cSchema))))
	for i, batch := range batches {
		cdata.ExportArrowRecordBatch(batch, cdata.ArrayFromPtr(uintptr(unsafe.Pointer(&cArrays[i]))), nil)
	}
	return cSchema, cArrays, nil
}

func releaseExportedArrow(schema *C.struct_ArrowSchema, arrays []C.struct_ArrowArray) {
	if schema != nil {
		cdata.ReleaseCArrowSchema(cdata.SchemaFromPtr(uintptr(unsafe.Pointer(schema))))
	}
	for i := range arrays {
		cdata.ReleaseCArrowArray(cdata.ArrayFromPtr(uintptr(unsafe.Pointer(&arrays[i]))))
	}
}

func lastCAPIError(fallback string) error {
	cErr := C.lbug_get_last_error()
	if cErr == nil {
		return fmt.Errorf("%s", fallback)
	}
	defer C.lbug_destroy_string(cErr)
	return fmt.Errorf("%s", C.GoString(cErr))
}

func queryResultFromArrowCall(conn *Connection, queryResult *QueryResult, status C.lbug_state, fallback string) (*QueryResult, error) {
	queryResult.connection = conn
	if status != C.LbugSuccess {
		queryResult.Close()
		return nil, lastCAPIError(fallback)
	}
	if !C.lbug_query_result_is_success(&queryResult.cQueryResult) {
		cErrMsg := C.lbug_query_result_get_error_message(&queryResult.cQueryResult)
		defer C.lbug_destroy_string(cErrMsg)
		queryResult.Close()
		return nil, fmt.Errorf("%s", C.GoString(cErrMsg))
	}
	return queryResult, nil
}

// CreateArrowTable registers Arrow memory as a node table.
// The first column is used as the table primary key.
// The registered table may outlive this call, so batches should be built with
// memory that is safe to hold through the Arrow C Data Interface.
func (conn *Connection) CreateArrowTable(tableName string, batches []arrow.RecordBatch) (*QueryResult, error) {
	cSchema, cArrays, err := exportArrowBatches(batches)
	if err != nil {
		return nil, err
	}
	cTableName := C.CString(tableName)
	defer C.free(unsafe.Pointer(cTableName))

	queryResult := &QueryResult{}
	status := C.lbug_connection_create_arrow_table(&conn.cConnection, cTableName, cSchema,
		(*C.struct_ArrowArray)(unsafe.Pointer(&cArrays[0])), C.uint64_t(len(cArrays)),
		&queryResult.cQueryResult)
	if status == C.LbugSuccess {
		cSchema = nil
		cArrays = nil
	}
	defer releaseExportedArrow(cSchema, cArrays)
	return queryResultFromArrowCall(conn, queryResult, status, "failed to create Arrow table")
}

// CreateArrowRelTable registers Arrow memory as a relationship table.
// The Arrow schema must include endpoint columns named "from" and "to".
// The registered table may outlive this call, so batches should be built with
// memory that is safe to hold through the Arrow C Data Interface.
func (conn *Connection) CreateArrowRelTable(tableName string, batches []arrow.RecordBatch, srcTableName string, dstTableName string) (*QueryResult, error) {
	cSchema, cArrays, err := exportArrowBatches(batches)
	if err != nil {
		return nil, err
	}
	cTableName := C.CString(tableName)
	cSrcTableName := C.CString(srcTableName)
	cDstTableName := C.CString(dstTableName)
	defer C.free(unsafe.Pointer(cTableName))
	defer C.free(unsafe.Pointer(cSrcTableName))
	defer C.free(unsafe.Pointer(cDstTableName))

	queryResult := &QueryResult{}
	status := C.lbug_connection_create_arrow_rel_table(&conn.cConnection, cTableName,
		cSrcTableName, cDstTableName, cSchema, (*C.struct_ArrowArray)(unsafe.Pointer(&cArrays[0])),
		C.uint64_t(len(cArrays)), &queryResult.cQueryResult)
	if status == C.LbugSuccess {
		cSchema = nil
		cArrays = nil
	}
	defer releaseExportedArrow(cSchema, cArrays)
	return queryResultFromArrowCall(conn, queryResult, status, "failed to create Arrow relationship table")
}

// CreateArrowRelTableCSR registers Arrow memory in CSR form as a relationship table.
// If dstColName is omitted, the destination offset column defaults to "to".
// The registered table may outlive this call, so batches should be built with
// memory that is safe to hold through the Arrow C Data Interface.
func (conn *Connection) CreateArrowRelTableCSR(tableName string, indicesBatches []arrow.RecordBatch, indptrBatches []arrow.RecordBatch, srcTableName string, dstTableName string, dstColName ...string) (*QueryResult, error) {
	cIndicesSchema, cIndicesArrays, err := exportArrowBatches(indicesBatches)
	if err != nil {
		return nil, err
	}
	cIndptrSchema, cIndptrArrays, err := exportArrowBatches(indptrBatches)
	if err != nil {
		releaseExportedArrow(cIndicesSchema, cIndicesArrays)
		return nil, err
	}

	cTableName := C.CString(tableName)
	cSrcTableName := C.CString(srcTableName)
	cDstTableName := C.CString(dstTableName)
	var cDstColName *C.char
	if len(dstColName) > 0 && dstColName[0] != "" {
		cDstColName = C.CString(dstColName[0])
		defer C.free(unsafe.Pointer(cDstColName))
	}
	defer C.free(unsafe.Pointer(cTableName))
	defer C.free(unsafe.Pointer(cSrcTableName))
	defer C.free(unsafe.Pointer(cDstTableName))

	queryResult := &QueryResult{}
	status := C.lbug_connection_create_arrow_rel_table_csr(&conn.cConnection, cTableName,
		cSrcTableName, cDstTableName, cIndicesSchema,
		(*C.struct_ArrowArray)(unsafe.Pointer(&cIndicesArrays[0])),
		C.uint64_t(len(cIndicesArrays)), cIndptrSchema,
		(*C.struct_ArrowArray)(unsafe.Pointer(&cIndptrArrays[0])),
		C.uint64_t(len(cIndptrArrays)), cDstColName, &queryResult.cQueryResult)
	if status == C.LbugSuccess {
		cIndicesSchema = nil
		cIndicesArrays = nil
		cIndptrSchema = nil
		cIndptrArrays = nil
	}
	defer releaseExportedArrow(cIndicesSchema, cIndicesArrays)
	defer releaseExportedArrow(cIndptrSchema, cIndptrArrays)
	return queryResultFromArrowCall(conn, queryResult, status, "failed to create Arrow CSR relationship table")
}

// DropArrowTable drops an Arrow memory-backed table registered on this connection.
func (conn *Connection) DropArrowTable(tableName string) (*QueryResult, error) {
	cTableName := C.CString(tableName)
	defer C.free(unsafe.Pointer(cTableName))
	queryResult := &QueryResult{}
	status := C.lbug_connection_drop_arrow_table(&conn.cConnection, cTableName, &queryResult.cQueryResult)
	return queryResultFromArrowCall(conn, queryResult, status, "failed to drop Arrow table")
}

// GetArrowSchema returns the query result schema as an Arrow schema.
func (queryResult *QueryResult) GetArrowSchema() (*arrow.Schema, error) {
	var cSchema C.struct_ArrowSchema
	status := C.lbug_query_result_get_arrow_schema(&queryResult.cQueryResult, &cSchema)
	if status != C.LbugSuccess {
		return nil, lastCAPIError("failed to get Arrow schema")
	}
	return cdata.ImportCArrowSchema(cdata.SchemaFromPtr(uintptr(unsafe.Pointer(&cSchema))))
}

// GetNextArrowChunk returns the next chunk of the query result as an Arrow record batch.
func (queryResult *QueryResult) GetNextArrowChunk(chunkSize int64) (arrow.RecordBatch, error) {
	var cArray C.struct_ArrowArray
	status := C.lbug_query_result_get_next_arrow_chunk(&queryResult.cQueryResult, C.int64_t(chunkSize), &cArray)
	if status != C.LbugSuccess {
		return nil, lastCAPIError("failed to get next Arrow chunk")
	}
	schema, err := queryResult.GetArrowSchema()
	if err != nil {
		cdata.ReleaseCArrowArray(cdata.ArrayFromPtr(uintptr(unsafe.Pointer(&cArray))))
		return nil, err
	}
	return cdata.ImportCRecordBatchWithSchema(cdata.ArrayFromPtr(uintptr(unsafe.Pointer(&cArray))), schema)
}
