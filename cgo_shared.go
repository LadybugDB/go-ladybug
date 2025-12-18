package lbug

//go:generate sh download_lbug.sh

/*
#cgo darwin LDFLAGS: -lc++ -L${SRCDIR}/lib/dynamic/darwin -llbug -Wl,-rpath,${SRCDIR}/lib/dynamic/darwin
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/lib/dynamic/linux-amd64 -llbug -Wl,-rpath,${SRCDIR}/lib/dynamic/linux-amd64
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/lib/dynamic/linux-arm64 -llbug -Wl,-rpath,${SRCDIR}/lib/dynamic/linux-arm64
#cgo windows LDFLAGS: -L${SRCDIR}/lib/dynamic/windows -llbug_shared
#include "lbug.h"
*/
import "C"
