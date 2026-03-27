package lbug

// #include "lbug.h"
// #include <stdlib.h>
// #include <string.h>
import "C"

import (
	"fmt"
	"reflect"
	"sort"
	"time"
	"unsafe"

	"math/big"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InternalID represents the internal ID of a node or relationship in Lbug.
type InternalID struct {
	TableID uint64
	Offset  uint64
}

// Node represents a node retrieved from Lbug.
// A node has an ID, a label, and properties.
type Node struct {
	ID         InternalID
	Label      string
	Properties map[string]any
}

// Relationship represents a relationship retrieved from Lbug.
// A relationship has a source ID, a destination ID, a label, and properties.
type Relationship struct {
	ID            InternalID
	SourceID      InternalID
	DestinationID InternalID
	Label         string
	Properties    map[string]any
}

// RecursiveRelationship represents a recursive relationship retrieved from a
// path query in Lbug. A recursive relationship has a list of nodes and a list
// of relationships.
type RecursiveRelationship struct {
	Nodes         []Node
	Relationships []Relationship
}

// MapItem represents a key-value pair in a map in Lbug. It is used for both
// the query parameters and the query result.
type MapItem struct {
	Key   any
	Value any
}

// lbugNodeValueToGoValue converts a lbug_value representing a node to a Node
// struct in Go.
func lbugNodeValueToGoValue(lbugValue C.lbug_value) (Node, error) {
	node := Node{}
	node.Properties = make(map[string]any)
	idValue := C.lbug_value{}
	C.lbug_node_val_get_id_val(&lbugValue, &idValue)
	nodeId, _ := lbugValueToGoValue(idValue)
	node.ID = nodeId.(InternalID)
	labelValue := C.lbug_value{}
	C.lbug_node_val_get_label_val(&lbugValue, &labelValue)
	nodeLabel, _ := lbugValueToGoValue(labelValue)
	node.Label = nodeLabel.(string)
	var propertySize C.uint64_t
	C.lbug_node_val_get_property_size(&lbugValue, &propertySize)
	var currentKey *C.char
	var currentVal C.lbug_value
	var errors []error
	for i := C.uint64_t(0); i < propertySize; i++ {
		C.lbug_node_val_get_property_name_at(&lbugValue, i, &currentKey)
		keyString := C.GoString(currentKey)
		C.lbug_destroy_string(currentKey)
		C.lbug_node_val_get_property_value_at(&lbugValue, i, &currentVal)
		value, err := lbugValueToGoValue(currentVal)
		if err != nil {
			errors = append(errors, err)
		}
		node.Properties[keyString] = value
	}
	if len(errors) > 0 {
		return node, fmt.Errorf("failed to get values: %v", errors)
	}
	return node, nil
}

// lbugRelValueToGoValue converts a lbug_value representing a relationship to a
// Relationship struct in Go.
func lbugRelValueToGoValue(lbugValue C.lbug_value) (Relationship, error) {
	relation := Relationship{}
	relation.Properties = make(map[string]any)
	idValue := C.lbug_value{}
	C.lbug_rel_val_get_id_val(&lbugValue, &idValue)
	id, _ := lbugValueToGoValue(idValue)
	relation.ID = id.(InternalID)
	C.lbug_rel_val_get_src_id_val(&lbugValue, &idValue)
	src, _ := lbugValueToGoValue(idValue)
	relation.SourceID = src.(InternalID)
	C.lbug_rel_val_get_dst_id_val(&lbugValue, &idValue)
	dst, _ := lbugValueToGoValue(idValue)
	relation.DestinationID = dst.(InternalID)
	labelValue := C.lbug_value{}
	C.lbug_rel_val_get_label_val(&lbugValue, &labelValue)
	label, _ := lbugValueToGoValue(labelValue)
	relation.Label = label.(string)
	var propertySize C.uint64_t
	C.lbug_rel_val_get_property_size(&lbugValue, &propertySize)
	var currentKey *C.char
	var currentVal C.lbug_value
	var errors []error
	for i := C.uint64_t(0); i < propertySize; i++ {
		C.lbug_rel_val_get_property_name_at(&lbugValue, i, &currentKey)
		keyString := C.GoString(currentKey)
		C.lbug_destroy_string(currentKey)
		C.lbug_rel_val_get_property_value_at(&lbugValue, i, &currentVal)
		value, err := lbugValueToGoValue(currentVal)
		if err != nil {
			errors = append(errors, err)
		}
		relation.Properties[keyString] = value
	}
	if len(errors) > 0 {
		return relation, fmt.Errorf("failed to get values: %v", errors)
	}
	return relation, nil
}

// lbugRecursiveRelValueToGoValue converts a lbug_value representing a recursive
// relationship to a RecursiveRelationship struct in Go.
func lbugRecursiveRelValueToGoValue(lbugValue C.lbug_value) (RecursiveRelationship, error) {
	var nodesVal C.lbug_value
	var relsVal C.lbug_value
	C.lbug_value_get_recursive_rel_node_list(&lbugValue, &nodesVal)
	C.lbug_value_get_recursive_rel_rel_list(&lbugValue, &relsVal)
	nodes, _ := lbugListValueToGoValue(nodesVal)
	rels, _ := lbugListValueToGoValue(relsVal)
	recursiveRel := RecursiveRelationship{}
	recursiveRel.Nodes = make([]Node, len(nodes))
	for i, n := range nodes {
		recursiveRel.Nodes[i] = n.(Node)
	}
	relationships := make([]Relationship, len(rels))
	for i, r := range rels {
		relationships[i] = r.(Relationship)
	}
	recursiveRel.Relationships = relationships
	return recursiveRel, nil
}

// lbugListValueToGoValue converts a lbug_value representing a LIST or ARRAY to
// a slice of any in Go.
func lbugListValueToGoValue(lbugValue C.lbug_value) ([]any, error) {
	var listSize C.uint64_t
	cLogicalType := C.lbug_logical_type{}
	defer C.lbug_data_type_destroy(&cLogicalType)
	C.lbug_value_get_data_type(&lbugValue, &cLogicalType)
	logicalTypeId := C.lbug_data_type_get_id(&cLogicalType)
	if logicalTypeId == C.LBUG_ARRAY {
		C.lbug_data_type_get_num_elements_in_array(&cLogicalType, &listSize)
	} else {
		C.lbug_value_get_list_size(&lbugValue, &listSize)
	}
	list := make([]any, 0, int(listSize))
	var currentVal C.lbug_value
	var errors []error
	for i := C.uint64_t(0); i < listSize; i++ {
		C.lbug_value_get_list_element(&lbugValue, i, &currentVal)
		value, err := lbugValueToGoValue(currentVal)
		if err != nil {
			errors = append(errors, err)
		}
		list = append(list, value)
	}
	if len(errors) > 0 {
		return list, fmt.Errorf("failed to get values: %v", errors)
	}
	return list, nil
}

// lbugStructValueToGoValue converts a lbug_value representing a STRUCT to a
// map of string to any in Go.
func lbugStructValueToGoValue(lbugValue C.lbug_value) (map[string]any, error) {
	structure := make(map[string]any)
	var propertySize C.uint64_t
	C.lbug_value_get_struct_num_fields(&lbugValue, &propertySize)
	var currentKey *C.char
	var currentVal C.lbug_value
	var errors []error
	for i := C.uint64_t(0); i < propertySize; i++ {
		C.lbug_value_get_struct_field_name(&lbugValue, i, &currentKey)
		keyString := C.GoString(currentKey)
		C.lbug_destroy_string(currentKey)
		C.lbug_value_get_struct_field_value(&lbugValue, i, &currentVal)
		value, err := lbugValueToGoValue(currentVal)
		if err != nil {
			errors = append(errors, err)
		}
		structure[keyString] = value
	}
	if len(errors) > 0 {
		return structure, fmt.Errorf("failed to get values: %v", errors)
	}
	return structure, nil
}

// lbugMapValueToGoValue converts a lbug_value representing a MAP to a
// slice of MapItem in Go.
func lbugMapValueToGoValue(lbugValue C.lbug_value) ([]MapItem, error) {
	var mapSize C.uint64_t
	C.lbug_value_get_map_size(&lbugValue, &mapSize)
	mapItems := make([]MapItem, 0, int(mapSize))
	var currentKey C.lbug_value
	var currentValue C.lbug_value
	var errors []error
	for i := C.uint64_t(0); i < mapSize; i++ {
		C.lbug_value_get_map_key(&lbugValue, i, &currentKey)
		C.lbug_value_get_map_value(&lbugValue, i, &currentValue)
		key, err := lbugValueToGoValue(currentKey)
		if err != nil {
			errors = append(errors, err)
		}
		value, err := lbugValueToGoValue(currentValue)
		if err != nil {
			errors = append(errors, err)
		}
		mapItems = append(mapItems, MapItem{Key: key, Value: value})
	}
	if len(errors) > 0 {
		return mapItems, fmt.Errorf("failed to get values: %v", errors)
	}
	return mapItems, nil
}

// lbugValueToGoValue converts a lbug_value to a corresponding Go value.
func lbugValueToGoValue(lbugValue C.lbug_value) (any, error) {
	if C.lbug_value_is_null(&lbugValue) {
		return nil, nil
	}
	var logicalType C.lbug_logical_type
	defer C.lbug_data_type_destroy(&logicalType)
	C.lbug_value_get_data_type(&lbugValue, &logicalType)
	logicalTypeId := C.lbug_data_type_get_id(&logicalType)
	switch logicalTypeId {
	case C.LBUG_BOOL:
		var value C.bool
		status := C.lbug_value_get_bool(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get bool value with status: %d", status)
		}
		return bool(value), nil
	case C.LBUG_INT64, C.LBUG_SERIAL:
		var value C.int64_t
		status := C.lbug_value_get_int64(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get int64 value with status: %d", status)
		}
		return int64(value), nil
	case C.LBUG_INT32:
		var value C.int32_t
		status := C.lbug_value_get_int32(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get int32 value with status: %d", status)
		}
		return int32(value), nil
	case C.LBUG_INT16:
		var value C.int16_t
		status := C.lbug_value_get_int16(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get int16 value with status: %d", status)
		}
		return int16(value), nil
	case C.LBUG_INT128:
		var value C.lbug_int128_t
		status := C.lbug_value_get_int128(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get int128 value with status: %d", status)
		}
		return int128ToBigInt(value)
	case C.LBUG_INT8:
		var value C.int8_t
		status := C.lbug_value_get_int8(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get int8 value with status: %d", status)
		}
		return int8(value), nil
	case C.LBUG_UUID:
		var value *C.char
		status := C.lbug_value_get_uuid(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get uuid value with status: %d", status)
		}
		defer C.lbug_destroy_string(value)
		uuidString := C.GoString(value)
		return uuid.Parse(uuidString)
	case C.LBUG_UINT64:
		var value C.uint64_t
		status := C.lbug_value_get_uint64(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get uint64 value with status: %d", status)
		}
		return uint64(value), nil
	case C.LBUG_UINT32:
		var value C.uint32_t
		status := C.lbug_value_get_uint32(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get uint32 value with status: %d", status)
		}
		return uint32(value), nil
	case C.LBUG_UINT16:
		var value C.uint16_t
		status := C.lbug_value_get_uint16(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get uint16 value with status: %d", status)
		}
		return uint16(value), nil
	case C.LBUG_UINT8:
		var value C.uint8_t
		status := C.lbug_value_get_uint8(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get uint8 value with status: %d", status)
		}
		return uint8(value), nil
	case C.LBUG_DOUBLE:
		var value C.double
		status := C.lbug_value_get_double(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get double value with status: %d", status)
		}
		return float64(value), nil
	case C.LBUG_FLOAT:
		var value C.float
		status := C.lbug_value_get_float(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get float value with status: %d", status)
		}
		return float32(value), nil
	case C.LBUG_STRING:
		var outString *C.char
		status := C.lbug_value_get_string(&lbugValue, &outString)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get string value with status: %d", status)
		}
		defer C.lbug_destroy_string(outString)
		return C.GoString(outString), nil
	case C.LBUG_TIMESTAMP:
		var value C.lbug_timestamp_t
		status := C.lbug_value_get_timestamp(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get timestamp value with status: %d", status)
		}
		return time.Unix(0, int64(value.value)*1000), nil
	case C.LBUG_TIMESTAMP_NS:
		var value C.lbug_timestamp_ns_t
		status := C.lbug_value_get_timestamp_ns(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get timestamp_ns value with status: %d", status)
		}
		return time.Unix(0, int64(value.value)), nil
	case C.LBUG_TIMESTAMP_MS:
		var value C.lbug_timestamp_ms_t
		status := C.lbug_value_get_timestamp_ms(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get timestamp_ms value with status: %d", status)
		}
		return time.Unix(0, int64(value.value)*1000000), nil
	case C.LBUG_TIMESTAMP_SEC:
		var value C.lbug_timestamp_sec_t
		status := C.lbug_value_get_timestamp_sec(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get timestamp_sec value with status: %d", status)
		}
		return time.Unix(int64(value.value), 0), nil
	case C.LBUG_TIMESTAMP_TZ:
		var value C.lbug_timestamp_tz_t
		status := C.lbug_value_get_timestamp_tz(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get timestamp_tz value with status: %d", status)
		}
		return time.Unix(0, int64(value.value)*1000), nil
	case C.LBUG_DATE:
		var value C.lbug_date_t
		status := C.lbug_value_get_date(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get date value with status: %d", status)
		}
		return lbugDateToTime(value), nil
	case C.LBUG_INTERVAL:
		var value C.lbug_interval_t
		status := C.lbug_value_get_interval(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get interval value with status: %d", status)
		}
		return lbugIntervalToDuration(value), nil
	case C.LBUG_INTERNAL_ID:
		var value C.lbug_internal_id_t
		status := C.lbug_value_get_internal_id(&lbugValue, &value)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get internal_id value with status: %d", status)
		}
		return InternalID{TableID: uint64(value.table_id), Offset: uint64(value.offset)}, nil
	case C.LBUG_BLOB:
		var value *C.uint8_t
		var length C.uint64_t
		status := C.lbug_value_get_blob(&lbugValue, &value, &length)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get blob value with status: %d", status)
		}
		defer C.lbug_destroy_blob(value)
		blob := C.GoBytes(unsafe.Pointer(value), C.int(length))
		return blob, nil
	case C.LBUG_NODE:
		return lbugNodeValueToGoValue(lbugValue)
	case C.LBUG_REL:
		return lbugRelValueToGoValue(lbugValue)
	case C.LBUG_RECURSIVE_REL:
		return lbugRecursiveRelValueToGoValue(lbugValue)
	case C.LBUG_LIST, C.LBUG_ARRAY:
		return lbugListValueToGoValue(lbugValue)
	case C.LBUG_STRUCT, C.LBUG_UNION:
		return lbugStructValueToGoValue(lbugValue)
	case C.LBUG_MAP:
		return lbugMapValueToGoValue(lbugValue)
	case C.LBUG_DECIMAL:
		var outString *C.char
		status := C.lbug_value_get_decimal_as_string(&lbugValue, &outString)
		if status != C.LbugSuccess {
			return nil, fmt.Errorf("failed to get string value of decimal type with status: %d", status)
		}
		goString := C.GoString(outString)
		C.lbug_destroy_string(outString)
		goDecimal, casting_error := decimal.NewFromString(goString)
		if casting_error != nil {
			return nil, fmt.Errorf("failed to convert decimal value with error: %w", casting_error)
		}
		return goDecimal, casting_error
	default:
		valueString := C.lbug_value_to_string(&lbugValue)
		defer C.lbug_destroy_string(valueString)
		return C.GoString(valueString), fmt.Errorf("unsupported data type with type id: %d. the value is force-casted to string", logicalTypeId)
	}
}

// int128ToBigInt converts a lbug_int128_t to a big.Int in Go.
func int128ToBigInt(value C.lbug_int128_t) (*big.Int, error) {
	var outString *C.char
	status := C.lbug_int128_t_to_string(value, &outString)
	if status != C.LbugSuccess {
		return nil, fmt.Errorf("failed to convert int128 to string with status: %d", status)
	}
	defer C.lbug_destroy_string(outString)
	valueString := C.GoString(outString)
	bigInt := new(big.Int)
	_, success := bigInt.SetString(valueString, 10)
	if !success {
		return nil, fmt.Errorf("failed to convert string to big.Int")
	}
	return bigInt, nil
}

// goMapToLbugStruct converts a map of string to any to a lbug_value representing
// a STRUCT. It returns an error if the map is empty.
func goMapToLbugStruct(value map[string]any) (*C.lbug_value, error) {
	numFields := C.uint64_t(len(value))
	if numFields == 0 {
		return nil, fmt.Errorf("failed to create STRUCT value because the map is empty")
	}
	fieldNames := make([]*C.char, 0, len(value))
	fieldValues := make([]*C.lbug_value, 0, len(value))
	// Sort the keys to ensure the order is consistent.
	// This is useful for creating a LIST of STRUCTs because in Lbug, all the
	// LIST elements must have the same type (i.e., the same order of fields).
	sortedKeys := make([]string, 0, len(value))
	for k := range value {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		fieldNames = append(fieldNames, C.CString(k))
		lbugValue, error := goValueToLbugValue(value[k])
		if error != nil {
			return nil, fmt.Errorf("failed to convert value in the map with error: %w", error)
		}
		fieldValues = append(fieldValues, lbugValue)
		defer C.lbug_value_destroy(lbugValue)
		defer C.free(unsafe.Pointer(C.CString(k)))
	}

	var lbugValue *C.lbug_value
	status := C.lbug_value_create_struct(numFields, &fieldNames[0], &fieldValues[0], &lbugValue)
	if status != C.LbugSuccess {
		return nil, fmt.Errorf("failed to create STRUCT value with status: %d", status)
	}
	return lbugValue, nil
}

// goSliceOfMapItemsToLbugMap converts a slice of MapItem to a lbug_value
// representing a MAP. It returns an error if the slice is empty or if the keys
// in the slice are of different types or if the values in the slice are of
// different types.
func goSliceOfMapItemsToLbugMap(slice []MapItem) (*C.lbug_value, error) {
	numItems := C.uint64_t(len(slice))
	if numItems == 0 {
		return nil, fmt.Errorf("failed to create MAP value because the slice is empty")
	}
	keys := make([]*C.lbug_value, 0, len(slice))
	values := make([]*C.lbug_value, 0, len(slice))
	for _, item := range slice {
		key, error := goValueToLbugValue(item.Key)
		if error != nil {
			return nil, fmt.Errorf("failed to convert key in the slice with error: %w", error)
		}
		keys = append(keys, key)
		defer C.lbug_value_destroy(key)
		value, error := goValueToLbugValue(item.Value)
		if error != nil {
			return nil, fmt.Errorf("failed to convert value in the slice with error: %w", error)
		}
		values = append(values, value)
		defer C.lbug_value_destroy(value)
	}
	var lbugValue *C.lbug_value
	status := C.lbug_value_create_map(numItems, &keys[0], &values[0], &lbugValue)
	if status != C.LbugSuccess {
		return nil, fmt.Errorf("failed to create MAP value with status: %d. please make sure all the keys are of the same type and all the values are of the same type", status)
	}
	return lbugValue, nil
}

// goSliceToLbugList converts a slice of any to a lbug_value representing a LIST.
// It returns an error if the slice is empty or if the values in the slice are of
// different types.
func goSliceToLbugList(slice []any) (*C.lbug_value, error) {
	numItems := C.uint64_t(len(slice))
	if numItems == 0 {
		return nil, fmt.Errorf("failed to create LIST value because the slice is empty")
	}
	values := make([]*C.lbug_value, 0, len(slice))
	for _, item := range slice {
		value, error := goValueToLbugValue(item)
		if error != nil {
			return nil, fmt.Errorf("failed to convert value in the slice with error: %w", error)
		}
		values = append(values, value)
		defer C.lbug_value_destroy(value)
	}
	var lbugValue *C.lbug_value
	status := C.lbug_value_create_list(numItems, &values[0], &lbugValue)
	if status != C.LbugSuccess {
		return nil, fmt.Errorf("failed to create LIST value with status: %d. please make sure all the values are of the same type", status)
	}
	return lbugValue, nil
}

// lbugValueToGoValue converts a Go value to a lbug_value.
func goValueToLbugValue(value any) (*C.lbug_value, error) {
	if value == nil {
		return C.lbug_value_create_null(), nil
	}
	var lbugValue *C.lbug_value
	switch v := value.(type) {
	case bool:
		lbugValue = C.lbug_value_create_bool(C.bool(v))
	case int:
		lbugValue = C.lbug_value_create_int64(C.int64_t(v))
	case int64:
		lbugValue = C.lbug_value_create_int64(C.int64_t(v))
	case int32:
		lbugValue = C.lbug_value_create_int32(C.int32_t(v))
	case int16:
		lbugValue = C.lbug_value_create_int16(C.int16_t(v))
	case int8:
		lbugValue = C.lbug_value_create_int8(C.int8_t(v))
	case uint:
		lbugValue = C.lbug_value_create_uint64(C.uint64_t(v))
	case uint64:
		lbugValue = C.lbug_value_create_uint64(C.uint64_t(v))
	case uint32:
		lbugValue = C.lbug_value_create_uint32(C.uint32_t(v))
	case uint16:
		lbugValue = C.lbug_value_create_uint16(C.uint16_t(v))
	case uint8:
		lbugValue = C.lbug_value_create_uint8(C.uint8_t(v))
	case float64:
		lbugValue = C.lbug_value_create_double(C.double(v))
	case float32:
		lbugValue = C.lbug_value_create_float(C.float(v))
	case string:
		lbugValue = C.lbug_value_create_string(C.CString(v))
	case time.Time:
		if timeHasNanoseconds(v) {
			lbugValue = C.lbug_value_create_timestamp_ns(timeToLbugTimestampNs(v))
		} else {
			lbugValue = C.lbug_value_create_timestamp(timeToLbugTimestamp(v))
		}
	case time.Duration:
		interval := durationToLbugInterval(v)
		lbugValue = C.lbug_value_create_interval(interval)
	case map[string]any:
		return goMapToLbugStruct(v)
	case []MapItem:
		return goSliceOfMapItemsToLbugMap(v)
	case []any:
		return goSliceToLbugList(v)
	default:
		if reflect.TypeOf(value).Kind() == reflect.Slice {
			sliceValue := reflect.ValueOf(value)
			slice := make([]any, sliceValue.Len())
			for i := 0; i < sliceValue.Len(); i++ {
				slice[i] = sliceValue.Index(i).Interface()
			}
			return goSliceToLbugList(slice)
		}
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
	return lbugValue, nil
}
