package glalby

/*
#cgo LDFLAGS: -lglalby_bindings
#cgo linux,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR} -L${SRCDIR}
#cgo darwin,arm64 LDFLAGS: -Wl,-rpath,${SRCDIR} -L${SRCDIR}
#cgo darwin,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR} -L${SRCDIR}
*/
import "C"
