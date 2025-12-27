//go:build system_ladybug

package lbug

/*
#cgo darwin LDFLAGS: -lc++ -llbug
#cgo linux LDFLAGS: -llbug
#cgo windows LDFLAGS: -llbug_shared
#include "lbug.h"
*/
import "C"
