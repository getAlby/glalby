package glalby

/*
#cgo LDFLAGS: -lglalby_bindings
#cgo linux,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR}/x86_64-unknown-linux-gnu -L${SRCDIR}/x86_64-unknown-linux-gnu
#cgo darwin,arm64 LDFLAGS: -Wl,-rpath,${SRCDIR}/aarch64-apple-darwin -L${SRCDIR}/aarch64-apple-darwin
#cgo windows,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR}/x86_64-pc-windows-gnu -L${SRCDIR}/x86_64-pc-windows-gnu
*/
import "C"
