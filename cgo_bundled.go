//go:build !system_ladybug

package lbug

//go:generate sh download_lbug.sh

/*
#cgo CFLAGS: -I${SRCDIR}/lib
#cgo darwin LDFLAGS: -lc++ -L${SRCDIR}/lib -llbug -Wl,-rpath,${SRCDIR}/lib
#cgo linux LDFLAGS: -L${SRCDIR}/lib -llbug -Wl,-rpath,${SRCDIR}/lib
#cgo windows LDFLAGS: -L${SRCDIR}/lib -llbug_shared
#include "lbug.h"
*/
import "C"
