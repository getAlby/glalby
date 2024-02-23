package glalby

// #include <glalby.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"runtime"
	"sync/atomic"
	"unsafe"
)

type RustBuffer = C.RustBuffer

type RustBufferI interface {
	AsReader() *bytes.Reader
	Free()
	ToGoBytes() []byte
	Data() unsafe.Pointer
	Len() int
	Capacity() int
}

func RustBufferFromExternal(b RustBufferI) RustBuffer {
	return RustBuffer{
		capacity: C.int(b.Capacity()),
		len:      C.int(b.Len()),
		data:     (*C.uchar)(b.Data()),
	}
}

func (cb RustBuffer) Capacity() int {
	return int(cb.capacity)
}

func (cb RustBuffer) Len() int {
	return int(cb.len)
}

func (cb RustBuffer) Data() unsafe.Pointer {
	return unsafe.Pointer(cb.data)
}

func (cb RustBuffer) AsReader() *bytes.Reader {
	b := unsafe.Slice((*byte)(cb.data), C.int(cb.len))
	return bytes.NewReader(b)
}

func (cb RustBuffer) Free() {
	rustCall(func(status *C.RustCallStatus) bool {
		C.ffi_glalby_bindings_rustbuffer_free(cb, status)
		return false
	})
}

func (cb RustBuffer) ToGoBytes() []byte {
	return C.GoBytes(unsafe.Pointer(cb.data), C.int(cb.len))
}

func stringToRustBuffer(str string) RustBuffer {
	return bytesToRustBuffer([]byte(str))
}

func bytesToRustBuffer(b []byte) RustBuffer {
	if len(b) == 0 {
		return RustBuffer{}
	}
	// We can pass the pointer along here, as it is pinned
	// for the duration of this call
	foreign := C.ForeignBytes{
		len:  C.int(len(b)),
		data: (*C.uchar)(unsafe.Pointer(&b[0])),
	}

	return rustCall(func(status *C.RustCallStatus) RustBuffer {
		return C.ffi_glalby_bindings_rustbuffer_from_bytes(foreign, status)
	})
}

type BufLifter[GoType any] interface {
	Lift(value RustBufferI) GoType
}

type BufLowerer[GoType any] interface {
	Lower(value GoType) RustBuffer
}

type FfiConverter[GoType any, FfiType any] interface {
	Lift(value FfiType) GoType
	Lower(value GoType) FfiType
}

type BufReader[GoType any] interface {
	Read(reader io.Reader) GoType
}

type BufWriter[GoType any] interface {
	Write(writer io.Writer, value GoType)
}

type FfiRustBufConverter[GoType any, FfiType any] interface {
	FfiConverter[GoType, FfiType]
	BufReader[GoType]
}

func LowerIntoRustBuffer[GoType any](bufWriter BufWriter[GoType], value GoType) RustBuffer {
	// This might be not the most efficient way but it does not require knowing allocation size
	// beforehand
	var buffer bytes.Buffer
	bufWriter.Write(&buffer, value)

	bytes, err := io.ReadAll(&buffer)
	if err != nil {
		panic(fmt.Errorf("reading written data: %w", err))
	}
	return bytesToRustBuffer(bytes)
}

func LiftFromRustBuffer[GoType any](bufReader BufReader[GoType], rbuf RustBufferI) GoType {
	defer rbuf.Free()
	reader := rbuf.AsReader()
	item := bufReader.Read(reader)
	if reader.Len() > 0 {
		// TODO: Remove this
		leftover, _ := io.ReadAll(reader)
		panic(fmt.Errorf("Junk remaining in buffer after lifting: %s", string(leftover)))
	}
	return item
}

func rustCallWithError[U any](converter BufLifter[error], callback func(*C.RustCallStatus) U) (U, error) {
	var status C.RustCallStatus
	returnValue := callback(&status)
	err := checkCallStatus(converter, status)

	return returnValue, err
}

func checkCallStatus(converter BufLifter[error], status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		return converter.Lift(status.errorBuf)
	case 2:
		// when the rust code sees a panic, it tries to construct a rustbuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(status.errorBuf)))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func checkCallStatusUnknown(status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		panic(fmt.Errorf("function not returning an error returned an error"))
	case 2:
		// when the rust code sees a panic, it tries to construct a rustbuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(status.errorBuf)))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func rustCall[U any](callback func(*C.RustCallStatus) U) U {
	returnValue, err := rustCallWithError(nil, callback)
	if err != nil {
		panic(err)
	}
	return returnValue
}

func writeInt8(writer io.Writer, value int8) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint8(writer io.Writer, value uint8) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt16(writer io.Writer, value int16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint16(writer io.Writer, value uint16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt32(writer io.Writer, value int32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint32(writer io.Writer, value uint32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt64(writer io.Writer, value int64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint64(writer io.Writer, value uint64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeFloat32(writer io.Writer, value float32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeFloat64(writer io.Writer, value float64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func readInt8(reader io.Reader) int8 {
	var result int8
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint8(reader io.Reader) uint8 {
	var result uint8
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt16(reader io.Reader) int16 {
	var result int16
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint16(reader io.Reader) uint16 {
	var result uint16
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt32(reader io.Reader) int32 {
	var result int32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint32(reader io.Reader) uint32 {
	var result uint32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt64(reader io.Reader) int64 {
	var result int64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint64(reader io.Reader) uint64 {
	var result uint64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readFloat32(reader io.Reader) float32 {
	var result float32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readFloat64(reader io.Reader) float64 {
	var result float64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func init() {

	uniffiCheckChecksums()
}

func uniffiCheckChecksums() {
	// Get the bindings contract version from our ComponentInterface
	bindingsContractVersion := 24
	// Get the scaffolding contract version by calling the into the dylib
	scaffoldingContractVersion := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint32_t {
		return C.ffi_glalby_bindings_uniffi_contract_version(uniffiStatus)
	})
	if bindingsContractVersion != int(scaffoldingContractVersion) {
		// If this happens try cleaning and rebuilding your project
		panic("glalby: UniFFI contract version mismatch")
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_func_new_blocking_greenlight_alby_client(uniffiStatus)
		})
		if checksum != 13984 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_func_new_blocking_greenlight_alby_client: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_func_recover(uniffiStatus)
		})
		if checksum != 3522 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_func_recover: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_connect_peer(uniffiStatus)
		})
		if checksum != 50417 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_connect_peer: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_fund_channel(uniffiStatus)
		})
		if checksum != 52932 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_fund_channel: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_get_info(uniffiStatus)
		})
		if checksum != 49263 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_get_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_key_send(uniffiStatus)
		})
		if checksum != 14883 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_key_send: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_list_funds(uniffiStatus)
		})
		if checksum != 6766 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_list_funds: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_list_invoices(uniffiStatus)
		})
		if checksum != 8342 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_list_invoices: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_list_payments(uniffiStatus)
		})
		if checksum != 56886 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_list_payments: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_make_invoice(uniffiStatus)
		})
		if checksum != 62877 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_make_invoice: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_new_address(uniffiStatus)
		})
		if checksum != 52875 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_new_address: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_pay(uniffiStatus)
		})
		if checksum != 10999 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_pay: UniFFI API checksum mismatch")
		}
	}
}

type FfiConverterUint16 struct{}

var FfiConverterUint16INSTANCE = FfiConverterUint16{}

func (FfiConverterUint16) Lower(value uint16) C.uint16_t {
	return C.uint16_t(value)
}

func (FfiConverterUint16) Write(writer io.Writer, value uint16) {
	writeUint16(writer, value)
}

func (FfiConverterUint16) Lift(value C.uint16_t) uint16 {
	return uint16(value)
}

func (FfiConverterUint16) Read(reader io.Reader) uint16 {
	return readUint16(reader)
}

type FfiDestroyerUint16 struct{}

func (FfiDestroyerUint16) Destroy(_ uint16) {}

type FfiConverterUint32 struct{}

var FfiConverterUint32INSTANCE = FfiConverterUint32{}

func (FfiConverterUint32) Lower(value uint32) C.uint32_t {
	return C.uint32_t(value)
}

func (FfiConverterUint32) Write(writer io.Writer, value uint32) {
	writeUint32(writer, value)
}

func (FfiConverterUint32) Lift(value C.uint32_t) uint32 {
	return uint32(value)
}

func (FfiConverterUint32) Read(reader io.Reader) uint32 {
	return readUint32(reader)
}

type FfiDestroyerUint32 struct{}

func (FfiDestroyerUint32) Destroy(_ uint32) {}

type FfiConverterInt32 struct{}

var FfiConverterInt32INSTANCE = FfiConverterInt32{}

func (FfiConverterInt32) Lower(value int32) C.int32_t {
	return C.int32_t(value)
}

func (FfiConverterInt32) Write(writer io.Writer, value int32) {
	writeInt32(writer, value)
}

func (FfiConverterInt32) Lift(value C.int32_t) int32 {
	return int32(value)
}

func (FfiConverterInt32) Read(reader io.Reader) int32 {
	return readInt32(reader)
}

type FfiDestroyerInt32 struct{}

func (FfiDestroyerInt32) Destroy(_ int32) {}

type FfiConverterUint64 struct{}

var FfiConverterUint64INSTANCE = FfiConverterUint64{}

func (FfiConverterUint64) Lower(value uint64) C.uint64_t {
	return C.uint64_t(value)
}

func (FfiConverterUint64) Write(writer io.Writer, value uint64) {
	writeUint64(writer, value)
}

func (FfiConverterUint64) Lift(value C.uint64_t) uint64 {
	return uint64(value)
}

func (FfiConverterUint64) Read(reader io.Reader) uint64 {
	return readUint64(reader)
}

type FfiDestroyerUint64 struct{}

func (FfiDestroyerUint64) Destroy(_ uint64) {}

type FfiConverterBool struct{}

var FfiConverterBoolINSTANCE = FfiConverterBool{}

func (FfiConverterBool) Lower(value bool) C.int8_t {
	if value {
		return C.int8_t(1)
	}
	return C.int8_t(0)
}

func (FfiConverterBool) Write(writer io.Writer, value bool) {
	if value {
		writeInt8(writer, 1)
	} else {
		writeInt8(writer, 0)
	}
}

func (FfiConverterBool) Lift(value C.int8_t) bool {
	return value != 0
}

func (FfiConverterBool) Read(reader io.Reader) bool {
	return readInt8(reader) != 0
}

type FfiDestroyerBool struct{}

func (FfiDestroyerBool) Destroy(_ bool) {}

type FfiConverterString struct{}

var FfiConverterStringINSTANCE = FfiConverterString{}

func (FfiConverterString) Lift(rb RustBufferI) string {
	defer rb.Free()
	reader := rb.AsReader()
	b, err := io.ReadAll(reader)
	if err != nil {
		panic(fmt.Errorf("reading reader: %w", err))
	}
	return string(b)
}

func (FfiConverterString) Read(reader io.Reader) string {
	length := readInt32(reader)
	buffer := make([]byte, length)
	read_length, err := reader.Read(buffer)
	if err != nil {
		panic(err)
	}
	if read_length != int(length) {
		panic(fmt.Errorf("bad read length when reading string, expected %d, read %d", length, read_length))
	}
	return string(buffer)
}

func (FfiConverterString) Lower(value string) RustBuffer {
	return stringToRustBuffer(value)
}

func (FfiConverterString) Write(writer io.Writer, value string) {
	if len(value) > math.MaxInt32 {
		panic("String is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	write_length, err := io.WriteString(writer, value)
	if err != nil {
		panic(err)
	}
	if write_length != len(value) {
		panic(fmt.Errorf("bad write length when writing string, expected %d, written %d", len(value), write_length))
	}
}

type FfiDestroyerString struct{}

func (FfiDestroyerString) Destroy(_ string) {}

// Below is an implementation of synchronization requirements outlined in the link.
// https://github.com/mozilla/uniffi-rs/blob/0dc031132d9493ca812c3af6e7dd60ad2ea95bf0/uniffi_bindgen/src/bindings/kotlin/templates/ObjectRuntime.kt#L31

type FfiObject struct {
	pointer      unsafe.Pointer
	callCounter  atomic.Int64
	freeFunction func(unsafe.Pointer, *C.RustCallStatus)
	destroyed    atomic.Bool
}

func newFfiObject(pointer unsafe.Pointer, freeFunction func(unsafe.Pointer, *C.RustCallStatus)) FfiObject {
	return FfiObject{
		pointer:      pointer,
		freeFunction: freeFunction,
	}
}

func (ffiObject *FfiObject) incrementPointer(debugName string) unsafe.Pointer {
	for {
		counter := ffiObject.callCounter.Load()
		if counter <= -1 {
			panic(fmt.Errorf("%v object has already been destroyed", debugName))
		}
		if counter == math.MaxInt64 {
			panic(fmt.Errorf("%v object call counter would overflow", debugName))
		}
		if ffiObject.callCounter.CompareAndSwap(counter, counter+1) {
			break
		}
	}

	return ffiObject.pointer
}

func (ffiObject *FfiObject) decrementPointer() {
	if ffiObject.callCounter.Add(-1) == -1 {
		ffiObject.freeRustArcPtr()
	}
}

func (ffiObject *FfiObject) destroy() {
	if ffiObject.destroyed.CompareAndSwap(false, true) {
		if ffiObject.callCounter.Add(-1) == -1 {
			ffiObject.freeRustArcPtr()
		}
	}
}

func (ffiObject *FfiObject) freeRustArcPtr() {
	rustCall(func(status *C.RustCallStatus) int32 {
		ffiObject.freeFunction(ffiObject.pointer, status)
		return 0
	})
}

type BlockingGreenlightAlbyClient struct {
	ffiObject FfiObject
}

func (_self *BlockingGreenlightAlbyClient) ConnectPeer(request ConnectPeerRequest) (ConnectPeerResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_connect_peer(
			_pointer, FfiConverterTypeConnectPeerRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ConnectPeerResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeConnectPeerResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) FundChannel(request FundChannelRequest) (FundChannelResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_fund_channel(
			_pointer, FfiConverterTypeFundChannelRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FundChannelResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeFundChannelResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) GetInfo() (GetInfoResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_get_info(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue GetInfoResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeGetInfoResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) KeySend(request KeySendRequest) (KeySendResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_key_send(
			_pointer, FfiConverterTypeKeySendRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue KeySendResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeKeySendResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) ListFunds(request ListFundsRequest) (ListFundsResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_list_funds(
			_pointer, FfiConverterTypeListFundsRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ListFundsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeListFundsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) ListInvoices(request ListInvoicesRequest) (ListInvoicesResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_list_invoices(
			_pointer, FfiConverterTypeListInvoicesRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ListInvoicesResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeListInvoicesResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) ListPayments(request ListPaymentsRequest) (ListPaymentsResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_list_payments(
			_pointer, FfiConverterTypeListPaymentsRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ListPaymentsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeListPaymentsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) MakeInvoice(request MakeInvoiceRequest) (MakeInvoiceResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_make_invoice(
			_pointer, FfiConverterTypeMakeInvoiceRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue MakeInvoiceResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeMakeInvoiceResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) NewAddress(request NewAddressRequest) (NewAddressResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_new_address(
			_pointer, FfiConverterTypeNewAddressRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue NewAddressResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeNewAddressResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingGreenlightAlbyClient) Pay(request PayRequest) (PayResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_method_blockinggreenlightalbyclient_pay(
			_pointer, FfiConverterTypePayRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PayResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePayResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (object *BlockingGreenlightAlbyClient) Destroy() {
	runtime.SetFinalizer(object, nil)
	object.ffiObject.destroy()
}

type FfiConverterBlockingGreenlightAlbyClient struct{}

var FfiConverterBlockingGreenlightAlbyClientINSTANCE = FfiConverterBlockingGreenlightAlbyClient{}

func (c FfiConverterBlockingGreenlightAlbyClient) Lift(pointer unsafe.Pointer) *BlockingGreenlightAlbyClient {
	result := &BlockingGreenlightAlbyClient{
		newFfiObject(
			pointer,
			func(pointer unsafe.Pointer, status *C.RustCallStatus) {
				C.uniffi_glalby_bindings_fn_free_blockinggreenlightalbyclient(pointer, status)
			}),
	}
	runtime.SetFinalizer(result, (*BlockingGreenlightAlbyClient).Destroy)
	return result
}

func (c FfiConverterBlockingGreenlightAlbyClient) Read(reader io.Reader) *BlockingGreenlightAlbyClient {
	return c.Lift(unsafe.Pointer(uintptr(readUint64(reader))))
}

func (c FfiConverterBlockingGreenlightAlbyClient) Lower(value *BlockingGreenlightAlbyClient) unsafe.Pointer {
	// TODO: this is bad - all synchronization from ObjectRuntime.go is discarded here,
	// because the pointer will be decremented immediately after this function returns,
	// and someone will be left holding onto a non-locked pointer.
	pointer := value.ffiObject.incrementPointer("*BlockingGreenlightAlbyClient")
	defer value.ffiObject.decrementPointer()
	return pointer
}

func (c FfiConverterBlockingGreenlightAlbyClient) Write(writer io.Writer, value *BlockingGreenlightAlbyClient) {
	writeUint64(writer, uint64(uintptr(c.Lower(value))))
}

type FfiDestroyerBlockingGreenlightAlbyClient struct{}

func (_ FfiDestroyerBlockingGreenlightAlbyClient) Destroy(value *BlockingGreenlightAlbyClient) {
	value.Destroy()
}

type ConnectPeerRequest struct {
	Id   string
	Host *string
	Port *uint16
}

func (r *ConnectPeerRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerOptionalString{}.Destroy(r.Host)
	FfiDestroyerOptionalUint16{}.Destroy(r.Port)
}

type FfiConverterTypeConnectPeerRequest struct{}

var FfiConverterTypeConnectPeerRequestINSTANCE = FfiConverterTypeConnectPeerRequest{}

func (c FfiConverterTypeConnectPeerRequest) Lift(rb RustBufferI) ConnectPeerRequest {
	return LiftFromRustBuffer[ConnectPeerRequest](c, rb)
}

func (c FfiConverterTypeConnectPeerRequest) Read(reader io.Reader) ConnectPeerRequest {
	return ConnectPeerRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint16INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConnectPeerRequest) Lower(value ConnectPeerRequest) RustBuffer {
	return LowerIntoRustBuffer[ConnectPeerRequest](c, value)
}

func (c FfiConverterTypeConnectPeerRequest) Write(writer io.Writer, value ConnectPeerRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Host)
	FfiConverterOptionalUint16INSTANCE.Write(writer, value.Port)
}

type FfiDestroyerTypeConnectPeerRequest struct{}

func (_ FfiDestroyerTypeConnectPeerRequest) Destroy(value ConnectPeerRequest) {
	value.Destroy()
}

type ConnectPeerResponse struct {
	Id string
}

func (r *ConnectPeerResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
}

type FfiConverterTypeConnectPeerResponse struct{}

var FfiConverterTypeConnectPeerResponseINSTANCE = FfiConverterTypeConnectPeerResponse{}

func (c FfiConverterTypeConnectPeerResponse) Lift(rb RustBufferI) ConnectPeerResponse {
	return LiftFromRustBuffer[ConnectPeerResponse](c, rb)
}

func (c FfiConverterTypeConnectPeerResponse) Read(reader io.Reader) ConnectPeerResponse {
	return ConnectPeerResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConnectPeerResponse) Lower(value ConnectPeerResponse) RustBuffer {
	return LowerIntoRustBuffer[ConnectPeerResponse](c, value)
}

func (c FfiConverterTypeConnectPeerResponse) Write(writer io.Writer, value ConnectPeerResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
}

type FfiDestroyerTypeConnectPeerResponse struct{}

func (_ FfiDestroyerTypeConnectPeerResponse) Destroy(value ConnectPeerResponse) {
	value.Destroy()
}

type FundChannelRequest struct {
	Id         string
	AmountMsat *uint64
	Announce   *bool
	Minconf    *uint32
}

func (r *FundChannelRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalBool{}.Destroy(r.Announce)
	FfiDestroyerOptionalUint32{}.Destroy(r.Minconf)
}

type FfiConverterTypeFundChannelRequest struct{}

var FfiConverterTypeFundChannelRequestINSTANCE = FfiConverterTypeFundChannelRequest{}

func (c FfiConverterTypeFundChannelRequest) Lift(rb RustBufferI) FundChannelRequest {
	return LiftFromRustBuffer[FundChannelRequest](c, rb)
}

func (c FfiConverterTypeFundChannelRequest) Read(reader io.Reader) FundChannelRequest {
	return FundChannelRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeFundChannelRequest) Lower(value FundChannelRequest) RustBuffer {
	return LowerIntoRustBuffer[FundChannelRequest](c, value)
}

func (c FfiConverterTypeFundChannelRequest) Write(writer io.Writer, value FundChannelRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Announce)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Minconf)
}

type FfiDestroyerTypeFundChannelRequest struct{}

func (_ FfiDestroyerTypeFundChannelRequest) Destroy(value FundChannelRequest) {
	value.Destroy()
}

type FundChannelResponse struct {
	Txid string
}

func (r *FundChannelResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Txid)
}

type FfiConverterTypeFundChannelResponse struct{}

var FfiConverterTypeFundChannelResponseINSTANCE = FfiConverterTypeFundChannelResponse{}

func (c FfiConverterTypeFundChannelResponse) Lift(rb RustBufferI) FundChannelResponse {
	return LiftFromRustBuffer[FundChannelResponse](c, rb)
}

func (c FfiConverterTypeFundChannelResponse) Read(reader io.Reader) FundChannelResponse {
	return FundChannelResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeFundChannelResponse) Lower(value FundChannelResponse) RustBuffer {
	return LowerIntoRustBuffer[FundChannelResponse](c, value)
}

func (c FfiConverterTypeFundChannelResponse) Write(writer io.Writer, value FundChannelResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Txid)
}

type FfiDestroyerTypeFundChannelResponse struct{}

func (_ FfiDestroyerTypeFundChannelResponse) Destroy(value FundChannelResponse) {
	value.Destroy()
}

type GetInfoResponse struct {
	Pubkey      string
	Alias       string
	Color       string
	Network     string
	BlockHeight uint32
}

func (r *GetInfoResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Pubkey)
	FfiDestroyerString{}.Destroy(r.Alias)
	FfiDestroyerString{}.Destroy(r.Color)
	FfiDestroyerString{}.Destroy(r.Network)
	FfiDestroyerUint32{}.Destroy(r.BlockHeight)
}

type FfiConverterTypeGetInfoResponse struct{}

var FfiConverterTypeGetInfoResponseINSTANCE = FfiConverterTypeGetInfoResponse{}

func (c FfiConverterTypeGetInfoResponse) Lift(rb RustBufferI) GetInfoResponse {
	return LiftFromRustBuffer[GetInfoResponse](c, rb)
}

func (c FfiConverterTypeGetInfoResponse) Read(reader io.Reader) GetInfoResponse {
	return GetInfoResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeGetInfoResponse) Lower(value GetInfoResponse) RustBuffer {
	return LowerIntoRustBuffer[GetInfoResponse](c, value)
}

func (c FfiConverterTypeGetInfoResponse) Write(writer io.Writer, value GetInfoResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
	FfiConverterStringINSTANCE.Write(writer, value.Alias)
	FfiConverterStringINSTANCE.Write(writer, value.Color)
	FfiConverterStringINSTANCE.Write(writer, value.Network)
	FfiConverterUint32INSTANCE.Write(writer, value.BlockHeight)
}

type FfiDestroyerTypeGetInfoResponse struct{}

func (_ FfiDestroyerTypeGetInfoResponse) Destroy(value GetInfoResponse) {
	value.Destroy()
}

type GreenlightCredentials struct {
	DeviceKey  string
	DeviceCert string
}

func (r *GreenlightCredentials) Destroy() {
	FfiDestroyerString{}.Destroy(r.DeviceKey)
	FfiDestroyerString{}.Destroy(r.DeviceCert)
}

type FfiConverterTypeGreenlightCredentials struct{}

var FfiConverterTypeGreenlightCredentialsINSTANCE = FfiConverterTypeGreenlightCredentials{}

func (c FfiConverterTypeGreenlightCredentials) Lift(rb RustBufferI) GreenlightCredentials {
	return LiftFromRustBuffer[GreenlightCredentials](c, rb)
}

func (c FfiConverterTypeGreenlightCredentials) Read(reader io.Reader) GreenlightCredentials {
	return GreenlightCredentials{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeGreenlightCredentials) Lower(value GreenlightCredentials) RustBuffer {
	return LowerIntoRustBuffer[GreenlightCredentials](c, value)
}

func (c FfiConverterTypeGreenlightCredentials) Write(writer io.Writer, value GreenlightCredentials) {
	FfiConverterStringINSTANCE.Write(writer, value.DeviceKey)
	FfiConverterStringINSTANCE.Write(writer, value.DeviceCert)
}

type FfiDestroyerTypeGreenlightCredentials struct{}

func (_ FfiDestroyerTypeGreenlightCredentials) Destroy(value GreenlightCredentials) {
	value.Destroy()
}

type KeySendRequest struct {
	Destination string
	AmountMsat  *uint64
	Label       *string
}

func (r *KeySendRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Destination)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
}

type FfiConverterTypeKeySendRequest struct{}

var FfiConverterTypeKeySendRequestINSTANCE = FfiConverterTypeKeySendRequest{}

func (c FfiConverterTypeKeySendRequest) Lift(rb RustBufferI) KeySendRequest {
	return LiftFromRustBuffer[KeySendRequest](c, rb)
}

func (c FfiConverterTypeKeySendRequest) Read(reader io.Reader) KeySendRequest {
	return KeySendRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeKeySendRequest) Lower(value KeySendRequest) RustBuffer {
	return LowerIntoRustBuffer[KeySendRequest](c, value)
}

func (c FfiConverterTypeKeySendRequest) Write(writer io.Writer, value KeySendRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Destination)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
}

type FfiDestroyerTypeKeySendRequest struct{}

func (_ FfiDestroyerTypeKeySendRequest) Destroy(value KeySendRequest) {
	value.Destroy()
}

type KeySendResponse struct {
	PaymentPreimage string
}

func (r *KeySendResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.PaymentPreimage)
}

type FfiConverterTypeKeySendResponse struct{}

var FfiConverterTypeKeySendResponseINSTANCE = FfiConverterTypeKeySendResponse{}

func (c FfiConverterTypeKeySendResponse) Lift(rb RustBufferI) KeySendResponse {
	return LiftFromRustBuffer[KeySendResponse](c, rb)
}

func (c FfiConverterTypeKeySendResponse) Read(reader io.Reader) KeySendResponse {
	return KeySendResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeKeySendResponse) Lower(value KeySendResponse) RustBuffer {
	return LowerIntoRustBuffer[KeySendResponse](c, value)
}

func (c FfiConverterTypeKeySendResponse) Write(writer io.Writer, value KeySendResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentPreimage)
}

type FfiDestroyerTypeKeySendResponse struct{}

func (_ FfiDestroyerTypeKeySendResponse) Destroy(value KeySendResponse) {
	value.Destroy()
}

type ListFundsChannel struct {
	PeerId         string
	OurAmountMsat  *uint64
	AmountMsat     *uint64
	FundingTxid    string
	FundingOutput  uint32
	Connected      bool
	State          int32
	ChannelId      *string
	ShortChannelId *string
}

func (r *ListFundsChannel) Destroy() {
	FfiDestroyerString{}.Destroy(r.PeerId)
	FfiDestroyerOptionalUint64{}.Destroy(r.OurAmountMsat)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerString{}.Destroy(r.FundingTxid)
	FfiDestroyerUint32{}.Destroy(r.FundingOutput)
	FfiDestroyerBool{}.Destroy(r.Connected)
	FfiDestroyerInt32{}.Destroy(r.State)
	FfiDestroyerOptionalString{}.Destroy(r.ChannelId)
	FfiDestroyerOptionalString{}.Destroy(r.ShortChannelId)
}

type FfiConverterTypeListFundsChannel struct{}

var FfiConverterTypeListFundsChannelINSTANCE = FfiConverterTypeListFundsChannel{}

func (c FfiConverterTypeListFundsChannel) Lift(rb RustBufferI) ListFundsChannel {
	return LiftFromRustBuffer[ListFundsChannel](c, rb)
}

func (c FfiConverterTypeListFundsChannel) Read(reader io.Reader) ListFundsChannel {
	return ListFundsChannel{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterInt32INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListFundsChannel) Lower(value ListFundsChannel) RustBuffer {
	return LowerIntoRustBuffer[ListFundsChannel](c, value)
}

func (c FfiConverterTypeListFundsChannel) Write(writer io.Writer, value ListFundsChannel) {
	FfiConverterStringINSTANCE.Write(writer, value.PeerId)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.OurAmountMsat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterStringINSTANCE.Write(writer, value.FundingTxid)
	FfiConverterUint32INSTANCE.Write(writer, value.FundingOutput)
	FfiConverterBoolINSTANCE.Write(writer, value.Connected)
	FfiConverterInt32INSTANCE.Write(writer, value.State)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ChannelId)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ShortChannelId)
}

type FfiDestroyerTypeListFundsChannel struct{}

func (_ FfiDestroyerTypeListFundsChannel) Destroy(value ListFundsChannel) {
	value.Destroy()
}

type ListFundsOutput struct {
	Txid         string
	Output       uint32
	AmountMsat   *uint64
	Scriptpubkey string
	Address      *string
	Redeemscript *string
	Status       int32
	Reserved     bool
	Blockheight  *uint32
}

func (r *ListFundsOutput) Destroy() {
	FfiDestroyerString{}.Destroy(r.Txid)
	FfiDestroyerUint32{}.Destroy(r.Output)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerString{}.Destroy(r.Scriptpubkey)
	FfiDestroyerOptionalString{}.Destroy(r.Address)
	FfiDestroyerOptionalString{}.Destroy(r.Redeemscript)
	FfiDestroyerInt32{}.Destroy(r.Status)
	FfiDestroyerBool{}.Destroy(r.Reserved)
	FfiDestroyerOptionalUint32{}.Destroy(r.Blockheight)
}

type FfiConverterTypeListFundsOutput struct{}

var FfiConverterTypeListFundsOutputINSTANCE = FfiConverterTypeListFundsOutput{}

func (c FfiConverterTypeListFundsOutput) Lift(rb RustBufferI) ListFundsOutput {
	return LiftFromRustBuffer[ListFundsOutput](c, rb)
}

func (c FfiConverterTypeListFundsOutput) Read(reader io.Reader) ListFundsOutput {
	return ListFundsOutput{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterInt32INSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListFundsOutput) Lower(value ListFundsOutput) RustBuffer {
	return LowerIntoRustBuffer[ListFundsOutput](c, value)
}

func (c FfiConverterTypeListFundsOutput) Write(writer io.Writer, value ListFundsOutput) {
	FfiConverterStringINSTANCE.Write(writer, value.Txid)
	FfiConverterUint32INSTANCE.Write(writer, value.Output)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterStringINSTANCE.Write(writer, value.Scriptpubkey)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Address)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Redeemscript)
	FfiConverterInt32INSTANCE.Write(writer, value.Status)
	FfiConverterBoolINSTANCE.Write(writer, value.Reserved)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Blockheight)
}

type FfiDestroyerTypeListFundsOutput struct{}

func (_ FfiDestroyerTypeListFundsOutput) Destroy(value ListFundsOutput) {
	value.Destroy()
}

type ListFundsRequest struct {
	Spent *bool
}

func (r *ListFundsRequest) Destroy() {
	FfiDestroyerOptionalBool{}.Destroy(r.Spent)
}

type FfiConverterTypeListFundsRequest struct{}

var FfiConverterTypeListFundsRequestINSTANCE = FfiConverterTypeListFundsRequest{}

func (c FfiConverterTypeListFundsRequest) Lift(rb RustBufferI) ListFundsRequest {
	return LiftFromRustBuffer[ListFundsRequest](c, rb)
}

func (c FfiConverterTypeListFundsRequest) Read(reader io.Reader) ListFundsRequest {
	return ListFundsRequest{
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListFundsRequest) Lower(value ListFundsRequest) RustBuffer {
	return LowerIntoRustBuffer[ListFundsRequest](c, value)
}

func (c FfiConverterTypeListFundsRequest) Write(writer io.Writer, value ListFundsRequest) {
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Spent)
}

type FfiDestroyerTypeListFundsRequest struct{}

func (_ FfiDestroyerTypeListFundsRequest) Destroy(value ListFundsRequest) {
	value.Destroy()
}

type ListFundsResponse struct {
	Outputs  []ListFundsOutput
	Channels []ListFundsChannel
}

func (r *ListFundsResponse) Destroy() {
	FfiDestroyerSequenceTypeListFundsOutput{}.Destroy(r.Outputs)
	FfiDestroyerSequenceTypeListFundsChannel{}.Destroy(r.Channels)
}

type FfiConverterTypeListFundsResponse struct{}

var FfiConverterTypeListFundsResponseINSTANCE = FfiConverterTypeListFundsResponse{}

func (c FfiConverterTypeListFundsResponse) Lift(rb RustBufferI) ListFundsResponse {
	return LiftFromRustBuffer[ListFundsResponse](c, rb)
}

func (c FfiConverterTypeListFundsResponse) Read(reader io.Reader) ListFundsResponse {
	return ListFundsResponse{
		FfiConverterSequenceTypeListFundsOutputINSTANCE.Read(reader),
		FfiConverterSequenceTypeListFundsChannelINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListFundsResponse) Lower(value ListFundsResponse) RustBuffer {
	return LowerIntoRustBuffer[ListFundsResponse](c, value)
}

func (c FfiConverterTypeListFundsResponse) Write(writer io.Writer, value ListFundsResponse) {
	FfiConverterSequenceTypeListFundsOutputINSTANCE.Write(writer, value.Outputs)
	FfiConverterSequenceTypeListFundsChannelINSTANCE.Write(writer, value.Channels)
}

type FfiDestroyerTypeListFundsResponse struct{}

func (_ FfiDestroyerTypeListFundsResponse) Destroy(value ListFundsResponse) {
	value.Destroy()
}

type ListInvoicesInvoice struct {
	Label              string
	Description        *string
	PaymentHash        string
	Status             int32
	ExpiresAt          uint64
	AmountMsat         *uint64
	Bolt11             *string
	Bolt12             *string
	LocalOfferId       *string
	InvreqPayerNote    *string
	CreatedIndex       *uint64
	UpdatedIndex       *uint64
	PayIndex           *uint64
	AmountReceivedMsat *uint64
	PaidAt             *uint64
	PaidOutpoint       *ListInvoicesInvoicePaidOutpoint
	PaymentPreimage    *string
}

func (r *ListInvoicesInvoice) Destroy() {
	FfiDestroyerString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerInt32{}.Destroy(r.Status)
	FfiDestroyerUint64{}.Destroy(r.ExpiresAt)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalString{}.Destroy(r.Bolt11)
	FfiDestroyerOptionalString{}.Destroy(r.Bolt12)
	FfiDestroyerOptionalString{}.Destroy(r.LocalOfferId)
	FfiDestroyerOptionalString{}.Destroy(r.InvreqPayerNote)
	FfiDestroyerOptionalUint64{}.Destroy(r.CreatedIndex)
	FfiDestroyerOptionalUint64{}.Destroy(r.UpdatedIndex)
	FfiDestroyerOptionalUint64{}.Destroy(r.PayIndex)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountReceivedMsat)
	FfiDestroyerOptionalUint64{}.Destroy(r.PaidAt)
	FfiDestroyerOptionalTypeListInvoicesInvoicePaidOutpoint{}.Destroy(r.PaidOutpoint)
	FfiDestroyerOptionalString{}.Destroy(r.PaymentPreimage)
}

type FfiConverterTypeListInvoicesInvoice struct{}

var FfiConverterTypeListInvoicesInvoiceINSTANCE = FfiConverterTypeListInvoicesInvoice{}

func (c FfiConverterTypeListInvoicesInvoice) Lift(rb RustBufferI) ListInvoicesInvoice {
	return LiftFromRustBuffer[ListInvoicesInvoice](c, rb)
}

func (c FfiConverterTypeListInvoicesInvoice) Read(reader io.Reader) ListInvoicesInvoice {
	return ListInvoicesInvoice{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterInt32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalTypeListInvoicesInvoicePaidOutpointINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListInvoicesInvoice) Lower(value ListInvoicesInvoice) RustBuffer {
	return LowerIntoRustBuffer[ListInvoicesInvoice](c, value)
}

func (c FfiConverterTypeListInvoicesInvoice) Write(writer io.Writer, value ListInvoicesInvoice) {
	FfiConverterStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterInt32INSTANCE.Write(writer, value.Status)
	FfiConverterUint64INSTANCE.Write(writer, value.ExpiresAt)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Bolt12)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LocalOfferId)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.InvreqPayerNote)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.CreatedIndex)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.UpdatedIndex)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.PayIndex)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountReceivedMsat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.PaidAt)
	FfiConverterOptionalTypeListInvoicesInvoicePaidOutpointINSTANCE.Write(writer, value.PaidOutpoint)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.PaymentPreimage)
}

type FfiDestroyerTypeListInvoicesInvoice struct{}

func (_ FfiDestroyerTypeListInvoicesInvoice) Destroy(value ListInvoicesInvoice) {
	value.Destroy()
}

type ListInvoicesInvoicePaidOutpoint struct {
	Txid   *string
	Outnum *uint32
}

func (r *ListInvoicesInvoicePaidOutpoint) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.Txid)
	FfiDestroyerOptionalUint32{}.Destroy(r.Outnum)
}

type FfiConverterTypeListInvoicesInvoicePaidOutpoint struct{}

var FfiConverterTypeListInvoicesInvoicePaidOutpointINSTANCE = FfiConverterTypeListInvoicesInvoicePaidOutpoint{}

func (c FfiConverterTypeListInvoicesInvoicePaidOutpoint) Lift(rb RustBufferI) ListInvoicesInvoicePaidOutpoint {
	return LiftFromRustBuffer[ListInvoicesInvoicePaidOutpoint](c, rb)
}

func (c FfiConverterTypeListInvoicesInvoicePaidOutpoint) Read(reader io.Reader) ListInvoicesInvoicePaidOutpoint {
	return ListInvoicesInvoicePaidOutpoint{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListInvoicesInvoicePaidOutpoint) Lower(value ListInvoicesInvoicePaidOutpoint) RustBuffer {
	return LowerIntoRustBuffer[ListInvoicesInvoicePaidOutpoint](c, value)
}

func (c FfiConverterTypeListInvoicesInvoicePaidOutpoint) Write(writer io.Writer, value ListInvoicesInvoicePaidOutpoint) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Txid)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Outnum)
}

type FfiDestroyerTypeListInvoicesInvoicePaidOutpoint struct{}

func (_ FfiDestroyerTypeListInvoicesInvoicePaidOutpoint) Destroy(value ListInvoicesInvoicePaidOutpoint) {
	value.Destroy()
}

type ListInvoicesRequest struct {
	Label       *string
	Invstring   *string
	PaymentHash *string
	OfferId     *string
	Index       *ListInvoicesIndex
	Start       *uint64
	Limit       *uint32
}

func (r *ListInvoicesRequest) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Invstring)
	FfiDestroyerOptionalString{}.Destroy(r.PaymentHash)
	FfiDestroyerOptionalString{}.Destroy(r.OfferId)
	FfiDestroyerOptionalTypeListInvoicesIndex{}.Destroy(r.Index)
	FfiDestroyerOptionalUint64{}.Destroy(r.Start)
	FfiDestroyerOptionalUint32{}.Destroy(r.Limit)
}

type FfiConverterTypeListInvoicesRequest struct{}

var FfiConverterTypeListInvoicesRequestINSTANCE = FfiConverterTypeListInvoicesRequest{}

func (c FfiConverterTypeListInvoicesRequest) Lift(rb RustBufferI) ListInvoicesRequest {
	return LiftFromRustBuffer[ListInvoicesRequest](c, rb)
}

func (c FfiConverterTypeListInvoicesRequest) Read(reader io.Reader) ListInvoicesRequest {
	return ListInvoicesRequest{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeListInvoicesIndexINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListInvoicesRequest) Lower(value ListInvoicesRequest) RustBuffer {
	return LowerIntoRustBuffer[ListInvoicesRequest](c, value)
}

func (c FfiConverterTypeListInvoicesRequest) Write(writer io.Writer, value ListInvoicesRequest) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Invstring)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.OfferId)
	FfiConverterOptionalTypeListInvoicesIndexINSTANCE.Write(writer, value.Index)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.Start)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Limit)
}

type FfiDestroyerTypeListInvoicesRequest struct{}

func (_ FfiDestroyerTypeListInvoicesRequest) Destroy(value ListInvoicesRequest) {
	value.Destroy()
}

type ListInvoicesResponse struct {
	Invoices []ListInvoicesInvoice
}

func (r *ListInvoicesResponse) Destroy() {
	FfiDestroyerSequenceTypeListInvoicesInvoice{}.Destroy(r.Invoices)
}

type FfiConverterTypeListInvoicesResponse struct{}

var FfiConverterTypeListInvoicesResponseINSTANCE = FfiConverterTypeListInvoicesResponse{}

func (c FfiConverterTypeListInvoicesResponse) Lift(rb RustBufferI) ListInvoicesResponse {
	return LiftFromRustBuffer[ListInvoicesResponse](c, rb)
}

func (c FfiConverterTypeListInvoicesResponse) Read(reader io.Reader) ListInvoicesResponse {
	return ListInvoicesResponse{
		FfiConverterSequenceTypeListInvoicesInvoiceINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListInvoicesResponse) Lower(value ListInvoicesResponse) RustBuffer {
	return LowerIntoRustBuffer[ListInvoicesResponse](c, value)
}

func (c FfiConverterTypeListInvoicesResponse) Write(writer io.Writer, value ListInvoicesResponse) {
	FfiConverterSequenceTypeListInvoicesInvoiceINSTANCE.Write(writer, value.Invoices)
}

type FfiDestroyerTypeListInvoicesResponse struct{}

func (_ FfiDestroyerTypeListInvoicesResponse) Destroy(value ListInvoicesResponse) {
	value.Destroy()
}

type ListPaymentsPayment struct {
	PaymentHash    string
	Status         int32
	Destination    *string
	CreatedAt      uint64
	CompletedAt    *uint64
	Label          *string
	Bolt11         *string
	Description    *string
	Bolt12         *string
	AmountMsat     *uint64
	AmountSentMsat *uint64
	Preimage       *string
	NumberOfParts  *uint64
	Erroronion     *string
}

func (r *ListPaymentsPayment) Destroy() {
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerInt32{}.Destroy(r.Status)
	FfiDestroyerOptionalString{}.Destroy(r.Destination)
	FfiDestroyerUint64{}.Destroy(r.CreatedAt)
	FfiDestroyerOptionalUint64{}.Destroy(r.CompletedAt)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Bolt11)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalString{}.Destroy(r.Bolt12)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountSentMsat)
	FfiDestroyerOptionalString{}.Destroy(r.Preimage)
	FfiDestroyerOptionalUint64{}.Destroy(r.NumberOfParts)
	FfiDestroyerOptionalString{}.Destroy(r.Erroronion)
}

type FfiConverterTypeListPaymentsPayment struct{}

var FfiConverterTypeListPaymentsPaymentINSTANCE = FfiConverterTypeListPaymentsPayment{}

func (c FfiConverterTypeListPaymentsPayment) Lift(rb RustBufferI) ListPaymentsPayment {
	return LiftFromRustBuffer[ListPaymentsPayment](c, rb)
}

func (c FfiConverterTypeListPaymentsPayment) Read(reader io.Reader) ListPaymentsPayment {
	return ListPaymentsPayment{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterInt32INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListPaymentsPayment) Lower(value ListPaymentsPayment) RustBuffer {
	return LowerIntoRustBuffer[ListPaymentsPayment](c, value)
}

func (c FfiConverterTypeListPaymentsPayment) Write(writer io.Writer, value ListPaymentsPayment) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterInt32INSTANCE.Write(writer, value.Status)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Destination)
	FfiConverterUint64INSTANCE.Write(writer, value.CreatedAt)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.CompletedAt)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Bolt12)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountSentMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Preimage)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.NumberOfParts)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Erroronion)
}

type FfiDestroyerTypeListPaymentsPayment struct{}

func (_ FfiDestroyerTypeListPaymentsPayment) Destroy(value ListPaymentsPayment) {
	value.Destroy()
}

type ListPaymentsRequest struct {
	Bolt11      *string
	PaymentHash *string
	Status      *ListPaymentsStatus
}

func (r *ListPaymentsRequest) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.Bolt11)
	FfiDestroyerOptionalString{}.Destroy(r.PaymentHash)
	FfiDestroyerOptionalTypeListPaymentsStatus{}.Destroy(r.Status)
}

type FfiConverterTypeListPaymentsRequest struct{}

var FfiConverterTypeListPaymentsRequestINSTANCE = FfiConverterTypeListPaymentsRequest{}

func (c FfiConverterTypeListPaymentsRequest) Lift(rb RustBufferI) ListPaymentsRequest {
	return LiftFromRustBuffer[ListPaymentsRequest](c, rb)
}

func (c FfiConverterTypeListPaymentsRequest) Read(reader io.Reader) ListPaymentsRequest {
	return ListPaymentsRequest{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeListPaymentsStatusINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListPaymentsRequest) Lower(value ListPaymentsRequest) RustBuffer {
	return LowerIntoRustBuffer[ListPaymentsRequest](c, value)
}

func (c FfiConverterTypeListPaymentsRequest) Write(writer io.Writer, value ListPaymentsRequest) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterOptionalTypeListPaymentsStatusINSTANCE.Write(writer, value.Status)
}

type FfiDestroyerTypeListPaymentsRequest struct{}

func (_ FfiDestroyerTypeListPaymentsRequest) Destroy(value ListPaymentsRequest) {
	value.Destroy()
}

type ListPaymentsResponse struct {
	Payments []ListPaymentsPayment
}

func (r *ListPaymentsResponse) Destroy() {
	FfiDestroyerSequenceTypeListPaymentsPayment{}.Destroy(r.Payments)
}

type FfiConverterTypeListPaymentsResponse struct{}

var FfiConverterTypeListPaymentsResponseINSTANCE = FfiConverterTypeListPaymentsResponse{}

func (c FfiConverterTypeListPaymentsResponse) Lift(rb RustBufferI) ListPaymentsResponse {
	return LiftFromRustBuffer[ListPaymentsResponse](c, rb)
}

func (c FfiConverterTypeListPaymentsResponse) Read(reader io.Reader) ListPaymentsResponse {
	return ListPaymentsResponse{
		FfiConverterSequenceTypeListPaymentsPaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListPaymentsResponse) Lower(value ListPaymentsResponse) RustBuffer {
	return LowerIntoRustBuffer[ListPaymentsResponse](c, value)
}

func (c FfiConverterTypeListPaymentsResponse) Write(writer io.Writer, value ListPaymentsResponse) {
	FfiConverterSequenceTypeListPaymentsPaymentINSTANCE.Write(writer, value.Payments)
}

type FfiDestroyerTypeListPaymentsResponse struct{}

func (_ FfiDestroyerTypeListPaymentsResponse) Destroy(value ListPaymentsResponse) {
	value.Destroy()
}

type MakeInvoiceRequest struct {
	AmountMsat  uint64
	Description string
	Label       string
}

func (r *MakeInvoiceRequest) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerString{}.Destroy(r.Description)
	FfiDestroyerString{}.Destroy(r.Label)
}

type FfiConverterTypeMakeInvoiceRequest struct{}

var FfiConverterTypeMakeInvoiceRequestINSTANCE = FfiConverterTypeMakeInvoiceRequest{}

func (c FfiConverterTypeMakeInvoiceRequest) Lift(rb RustBufferI) MakeInvoiceRequest {
	return LiftFromRustBuffer[MakeInvoiceRequest](c, rb)
}

func (c FfiConverterTypeMakeInvoiceRequest) Read(reader io.Reader) MakeInvoiceRequest {
	return MakeInvoiceRequest{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeMakeInvoiceRequest) Lower(value MakeInvoiceRequest) RustBuffer {
	return LowerIntoRustBuffer[MakeInvoiceRequest](c, value)
}

func (c FfiConverterTypeMakeInvoiceRequest) Write(writer io.Writer, value MakeInvoiceRequest) {
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Label)
}

type FfiDestroyerTypeMakeInvoiceRequest struct{}

func (_ FfiDestroyerTypeMakeInvoiceRequest) Destroy(value MakeInvoiceRequest) {
	value.Destroy()
}

type MakeInvoiceResponse struct {
	Bolt11 string
}

func (r *MakeInvoiceResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Bolt11)
}

type FfiConverterTypeMakeInvoiceResponse struct{}

var FfiConverterTypeMakeInvoiceResponseINSTANCE = FfiConverterTypeMakeInvoiceResponse{}

func (c FfiConverterTypeMakeInvoiceResponse) Lift(rb RustBufferI) MakeInvoiceResponse {
	return LiftFromRustBuffer[MakeInvoiceResponse](c, rb)
}

func (c FfiConverterTypeMakeInvoiceResponse) Read(reader io.Reader) MakeInvoiceResponse {
	return MakeInvoiceResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeMakeInvoiceResponse) Lower(value MakeInvoiceResponse) RustBuffer {
	return LowerIntoRustBuffer[MakeInvoiceResponse](c, value)
}

func (c FfiConverterTypeMakeInvoiceResponse) Write(writer io.Writer, value MakeInvoiceResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
}

type FfiDestroyerTypeMakeInvoiceResponse struct{}

func (_ FfiDestroyerTypeMakeInvoiceResponse) Destroy(value MakeInvoiceResponse) {
	value.Destroy()
}

type NewAddressRequest struct {
	AddressType *NewAddressType
}

func (r *NewAddressRequest) Destroy() {
	FfiDestroyerOptionalTypeNewAddressType{}.Destroy(r.AddressType)
}

type FfiConverterTypeNewAddressRequest struct{}

var FfiConverterTypeNewAddressRequestINSTANCE = FfiConverterTypeNewAddressRequest{}

func (c FfiConverterTypeNewAddressRequest) Lift(rb RustBufferI) NewAddressRequest {
	return LiftFromRustBuffer[NewAddressRequest](c, rb)
}

func (c FfiConverterTypeNewAddressRequest) Read(reader io.Reader) NewAddressRequest {
	return NewAddressRequest{
		FfiConverterOptionalTypeNewAddressTypeINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeNewAddressRequest) Lower(value NewAddressRequest) RustBuffer {
	return LowerIntoRustBuffer[NewAddressRequest](c, value)
}

func (c FfiConverterTypeNewAddressRequest) Write(writer io.Writer, value NewAddressRequest) {
	FfiConverterOptionalTypeNewAddressTypeINSTANCE.Write(writer, value.AddressType)
}

type FfiDestroyerTypeNewAddressRequest struct{}

func (_ FfiDestroyerTypeNewAddressRequest) Destroy(value NewAddressRequest) {
	value.Destroy()
}

type NewAddressResponse struct {
	P2tr       *string
	Bech32     *string
	P2shSegwit *string
}

func (r *NewAddressResponse) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.P2tr)
	FfiDestroyerOptionalString{}.Destroy(r.Bech32)
	FfiDestroyerOptionalString{}.Destroy(r.P2shSegwit)
}

type FfiConverterTypeNewAddressResponse struct{}

var FfiConverterTypeNewAddressResponseINSTANCE = FfiConverterTypeNewAddressResponse{}

func (c FfiConverterTypeNewAddressResponse) Lift(rb RustBufferI) NewAddressResponse {
	return LiftFromRustBuffer[NewAddressResponse](c, rb)
}

func (c FfiConverterTypeNewAddressResponse) Read(reader io.Reader) NewAddressResponse {
	return NewAddressResponse{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeNewAddressResponse) Lower(value NewAddressResponse) RustBuffer {
	return LowerIntoRustBuffer[NewAddressResponse](c, value)
}

func (c FfiConverterTypeNewAddressResponse) Write(writer io.Writer, value NewAddressResponse) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.P2tr)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Bech32)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.P2shSegwit)
}

type FfiDestroyerTypeNewAddressResponse struct{}

func (_ FfiDestroyerTypeNewAddressResponse) Destroy(value NewAddressResponse) {
	value.Destroy()
}

type PayRequest struct {
	Bolt11 string
}

func (r *PayRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Bolt11)
}

type FfiConverterTypePayRequest struct{}

var FfiConverterTypePayRequestINSTANCE = FfiConverterTypePayRequest{}

func (c FfiConverterTypePayRequest) Lift(rb RustBufferI) PayRequest {
	return LiftFromRustBuffer[PayRequest](c, rb)
}

func (c FfiConverterTypePayRequest) Read(reader io.Reader) PayRequest {
	return PayRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePayRequest) Lower(value PayRequest) RustBuffer {
	return LowerIntoRustBuffer[PayRequest](c, value)
}

func (c FfiConverterTypePayRequest) Write(writer io.Writer, value PayRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
}

type FfiDestroyerTypePayRequest struct{}

func (_ FfiDestroyerTypePayRequest) Destroy(value PayRequest) {
	value.Destroy()
}

type PayResponse struct {
	Preimage string
}

func (r *PayResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Preimage)
}

type FfiConverterTypePayResponse struct{}

var FfiConverterTypePayResponseINSTANCE = FfiConverterTypePayResponse{}

func (c FfiConverterTypePayResponse) Lift(rb RustBufferI) PayResponse {
	return LiftFromRustBuffer[PayResponse](c, rb)
}

func (c FfiConverterTypePayResponse) Read(reader io.Reader) PayResponse {
	return PayResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePayResponse) Lower(value PayResponse) RustBuffer {
	return LowerIntoRustBuffer[PayResponse](c, value)
}

func (c FfiConverterTypePayResponse) Write(writer io.Writer, value PayResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Preimage)
}

type FfiDestroyerTypePayResponse struct{}

func (_ FfiDestroyerTypePayResponse) Destroy(value PayResponse) {
	value.Destroy()
}

type ListInvoicesIndex uint

const (
	ListInvoicesIndexCreated ListInvoicesIndex = 1
	ListInvoicesIndexUpdated ListInvoicesIndex = 2
)

type FfiConverterTypeListInvoicesIndex struct{}

var FfiConverterTypeListInvoicesIndexINSTANCE = FfiConverterTypeListInvoicesIndex{}

func (c FfiConverterTypeListInvoicesIndex) Lift(rb RustBufferI) ListInvoicesIndex {
	return LiftFromRustBuffer[ListInvoicesIndex](c, rb)
}

func (c FfiConverterTypeListInvoicesIndex) Lower(value ListInvoicesIndex) RustBuffer {
	return LowerIntoRustBuffer[ListInvoicesIndex](c, value)
}
func (FfiConverterTypeListInvoicesIndex) Read(reader io.Reader) ListInvoicesIndex {
	id := readInt32(reader)
	return ListInvoicesIndex(id)
}

func (FfiConverterTypeListInvoicesIndex) Write(writer io.Writer, value ListInvoicesIndex) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeListInvoicesIndex struct{}

func (_ FfiDestroyerTypeListInvoicesIndex) Destroy(value ListInvoicesIndex) {
}

type ListPaymentsStatus uint

const (
	ListPaymentsStatusPending  ListPaymentsStatus = 1
	ListPaymentsStatusComplete ListPaymentsStatus = 2
	ListPaymentsStatusFailed   ListPaymentsStatus = 3
)

type FfiConverterTypeListPaymentsStatus struct{}

var FfiConverterTypeListPaymentsStatusINSTANCE = FfiConverterTypeListPaymentsStatus{}

func (c FfiConverterTypeListPaymentsStatus) Lift(rb RustBufferI) ListPaymentsStatus {
	return LiftFromRustBuffer[ListPaymentsStatus](c, rb)
}

func (c FfiConverterTypeListPaymentsStatus) Lower(value ListPaymentsStatus) RustBuffer {
	return LowerIntoRustBuffer[ListPaymentsStatus](c, value)
}
func (FfiConverterTypeListPaymentsStatus) Read(reader io.Reader) ListPaymentsStatus {
	id := readInt32(reader)
	return ListPaymentsStatus(id)
}

func (FfiConverterTypeListPaymentsStatus) Write(writer io.Writer, value ListPaymentsStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeListPaymentsStatus struct{}

func (_ FfiDestroyerTypeListPaymentsStatus) Destroy(value ListPaymentsStatus) {
}

type NewAddressType uint

const (
	NewAddressTypeBech32 NewAddressType = 1
	NewAddressTypeP2tr   NewAddressType = 2
	NewAddressTypeAll    NewAddressType = 3
)

type FfiConverterTypeNewAddressType struct{}

var FfiConverterTypeNewAddressTypeINSTANCE = FfiConverterTypeNewAddressType{}

func (c FfiConverterTypeNewAddressType) Lift(rb RustBufferI) NewAddressType {
	return LiftFromRustBuffer[NewAddressType](c, rb)
}

func (c FfiConverterTypeNewAddressType) Lower(value NewAddressType) RustBuffer {
	return LowerIntoRustBuffer[NewAddressType](c, value)
}
func (FfiConverterTypeNewAddressType) Read(reader io.Reader) NewAddressType {
	id := readInt32(reader)
	return NewAddressType(id)
}

func (FfiConverterTypeNewAddressType) Write(writer io.Writer, value NewAddressType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeNewAddressType struct{}

func (_ FfiDestroyerTypeNewAddressType) Destroy(value NewAddressType) {
}

type SdkError struct {
	err error
}

func (err SdkError) Error() string {
	return fmt.Sprintf("SdkError: %s", err.err.Error())
}

func (err SdkError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrSdkErrorGreenlightApi = fmt.Errorf("SdkErrorGreenlightApi")
var ErrSdkErrorInvalidArgument = fmt.Errorf("SdkErrorInvalidArgument")

// Variant structs
type SdkErrorGreenlightApi struct {
	message string
}

func NewSdkErrorGreenlightApi() *SdkError {
	return &SdkError{
		err: &SdkErrorGreenlightApi{},
	}
}

func (err SdkErrorGreenlightApi) Error() string {
	return fmt.Sprintf("GreenlightApi: %s", err.message)
}

func (self SdkErrorGreenlightApi) Is(target error) bool {
	return target == ErrSdkErrorGreenlightApi
}

type SdkErrorInvalidArgument struct {
	message string
}

func NewSdkErrorInvalidArgument() *SdkError {
	return &SdkError{
		err: &SdkErrorInvalidArgument{},
	}
}

func (err SdkErrorInvalidArgument) Error() string {
	return fmt.Sprintf("InvalidArgument: %s", err.message)
}

func (self SdkErrorInvalidArgument) Is(target error) bool {
	return target == ErrSdkErrorInvalidArgument
}

type FfiConverterTypeSdkError struct{}

var FfiConverterTypeSdkErrorINSTANCE = FfiConverterTypeSdkError{}

func (c FfiConverterTypeSdkError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeSdkError) Lower(value *SdkError) RustBuffer {
	return LowerIntoRustBuffer[*SdkError](c, value)
}

func (c FfiConverterTypeSdkError) Read(reader io.Reader) error {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &SdkError{&SdkErrorGreenlightApi{message}}
	case 2:
		return &SdkError{&SdkErrorInvalidArgument{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeSdkError.Read()", errorID))
	}

}

func (c FfiConverterTypeSdkError) Write(writer io.Writer, value *SdkError) {
	switch variantValue := value.err.(type) {
	case *SdkErrorGreenlightApi:
		writeInt32(writer, 1)
	case *SdkErrorInvalidArgument:
		writeInt32(writer, 2)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeSdkError.Write", value))
	}
}

type FfiConverterOptionalUint16 struct{}

var FfiConverterOptionalUint16INSTANCE = FfiConverterOptionalUint16{}

func (c FfiConverterOptionalUint16) Lift(rb RustBufferI) *uint16 {
	return LiftFromRustBuffer[*uint16](c, rb)
}

func (_ FfiConverterOptionalUint16) Read(reader io.Reader) *uint16 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterUint16INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalUint16) Lower(value *uint16) RustBuffer {
	return LowerIntoRustBuffer[*uint16](c, value)
}

func (_ FfiConverterOptionalUint16) Write(writer io.Writer, value *uint16) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterUint16INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalUint16 struct{}

func (_ FfiDestroyerOptionalUint16) Destroy(value *uint16) {
	if value != nil {
		FfiDestroyerUint16{}.Destroy(*value)
	}
}

type FfiConverterOptionalUint32 struct{}

var FfiConverterOptionalUint32INSTANCE = FfiConverterOptionalUint32{}

func (c FfiConverterOptionalUint32) Lift(rb RustBufferI) *uint32 {
	return LiftFromRustBuffer[*uint32](c, rb)
}

func (_ FfiConverterOptionalUint32) Read(reader io.Reader) *uint32 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterUint32INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalUint32) Lower(value *uint32) RustBuffer {
	return LowerIntoRustBuffer[*uint32](c, value)
}

func (_ FfiConverterOptionalUint32) Write(writer io.Writer, value *uint32) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterUint32INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalUint32 struct{}

func (_ FfiDestroyerOptionalUint32) Destroy(value *uint32) {
	if value != nil {
		FfiDestroyerUint32{}.Destroy(*value)
	}
}

type FfiConverterOptionalUint64 struct{}

var FfiConverterOptionalUint64INSTANCE = FfiConverterOptionalUint64{}

func (c FfiConverterOptionalUint64) Lift(rb RustBufferI) *uint64 {
	return LiftFromRustBuffer[*uint64](c, rb)
}

func (_ FfiConverterOptionalUint64) Read(reader io.Reader) *uint64 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterUint64INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalUint64) Lower(value *uint64) RustBuffer {
	return LowerIntoRustBuffer[*uint64](c, value)
}

func (_ FfiConverterOptionalUint64) Write(writer io.Writer, value *uint64) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterUint64INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalUint64 struct{}

func (_ FfiDestroyerOptionalUint64) Destroy(value *uint64) {
	if value != nil {
		FfiDestroyerUint64{}.Destroy(*value)
	}
}

type FfiConverterOptionalBool struct{}

var FfiConverterOptionalBoolINSTANCE = FfiConverterOptionalBool{}

func (c FfiConverterOptionalBool) Lift(rb RustBufferI) *bool {
	return LiftFromRustBuffer[*bool](c, rb)
}

func (_ FfiConverterOptionalBool) Read(reader io.Reader) *bool {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterBoolINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalBool) Lower(value *bool) RustBuffer {
	return LowerIntoRustBuffer[*bool](c, value)
}

func (_ FfiConverterOptionalBool) Write(writer io.Writer, value *bool) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterBoolINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalBool struct{}

func (_ FfiDestroyerOptionalBool) Destroy(value *bool) {
	if value != nil {
		FfiDestroyerBool{}.Destroy(*value)
	}
}

type FfiConverterOptionalString struct{}

var FfiConverterOptionalStringINSTANCE = FfiConverterOptionalString{}

func (c FfiConverterOptionalString) Lift(rb RustBufferI) *string {
	return LiftFromRustBuffer[*string](c, rb)
}

func (_ FfiConverterOptionalString) Read(reader io.Reader) *string {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterStringINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalString) Lower(value *string) RustBuffer {
	return LowerIntoRustBuffer[*string](c, value)
}

func (_ FfiConverterOptionalString) Write(writer io.Writer, value *string) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalString struct{}

func (_ FfiDestroyerOptionalString) Destroy(value *string) {
	if value != nil {
		FfiDestroyerString{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeListInvoicesInvoicePaidOutpoint struct{}

var FfiConverterOptionalTypeListInvoicesInvoicePaidOutpointINSTANCE = FfiConverterOptionalTypeListInvoicesInvoicePaidOutpoint{}

func (c FfiConverterOptionalTypeListInvoicesInvoicePaidOutpoint) Lift(rb RustBufferI) *ListInvoicesInvoicePaidOutpoint {
	return LiftFromRustBuffer[*ListInvoicesInvoicePaidOutpoint](c, rb)
}

func (_ FfiConverterOptionalTypeListInvoicesInvoicePaidOutpoint) Read(reader io.Reader) *ListInvoicesInvoicePaidOutpoint {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeListInvoicesInvoicePaidOutpointINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeListInvoicesInvoicePaidOutpoint) Lower(value *ListInvoicesInvoicePaidOutpoint) RustBuffer {
	return LowerIntoRustBuffer[*ListInvoicesInvoicePaidOutpoint](c, value)
}

func (_ FfiConverterOptionalTypeListInvoicesInvoicePaidOutpoint) Write(writer io.Writer, value *ListInvoicesInvoicePaidOutpoint) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeListInvoicesInvoicePaidOutpointINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeListInvoicesInvoicePaidOutpoint struct{}

func (_ FfiDestroyerOptionalTypeListInvoicesInvoicePaidOutpoint) Destroy(value *ListInvoicesInvoicePaidOutpoint) {
	if value != nil {
		FfiDestroyerTypeListInvoicesInvoicePaidOutpoint{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeListInvoicesIndex struct{}

var FfiConverterOptionalTypeListInvoicesIndexINSTANCE = FfiConverterOptionalTypeListInvoicesIndex{}

func (c FfiConverterOptionalTypeListInvoicesIndex) Lift(rb RustBufferI) *ListInvoicesIndex {
	return LiftFromRustBuffer[*ListInvoicesIndex](c, rb)
}

func (_ FfiConverterOptionalTypeListInvoicesIndex) Read(reader io.Reader) *ListInvoicesIndex {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeListInvoicesIndexINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeListInvoicesIndex) Lower(value *ListInvoicesIndex) RustBuffer {
	return LowerIntoRustBuffer[*ListInvoicesIndex](c, value)
}

func (_ FfiConverterOptionalTypeListInvoicesIndex) Write(writer io.Writer, value *ListInvoicesIndex) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeListInvoicesIndexINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeListInvoicesIndex struct{}

func (_ FfiDestroyerOptionalTypeListInvoicesIndex) Destroy(value *ListInvoicesIndex) {
	if value != nil {
		FfiDestroyerTypeListInvoicesIndex{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeListPaymentsStatus struct{}

var FfiConverterOptionalTypeListPaymentsStatusINSTANCE = FfiConverterOptionalTypeListPaymentsStatus{}

func (c FfiConverterOptionalTypeListPaymentsStatus) Lift(rb RustBufferI) *ListPaymentsStatus {
	return LiftFromRustBuffer[*ListPaymentsStatus](c, rb)
}

func (_ FfiConverterOptionalTypeListPaymentsStatus) Read(reader io.Reader) *ListPaymentsStatus {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeListPaymentsStatusINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeListPaymentsStatus) Lower(value *ListPaymentsStatus) RustBuffer {
	return LowerIntoRustBuffer[*ListPaymentsStatus](c, value)
}

func (_ FfiConverterOptionalTypeListPaymentsStatus) Write(writer io.Writer, value *ListPaymentsStatus) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeListPaymentsStatusINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeListPaymentsStatus struct{}

func (_ FfiDestroyerOptionalTypeListPaymentsStatus) Destroy(value *ListPaymentsStatus) {
	if value != nil {
		FfiDestroyerTypeListPaymentsStatus{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeNewAddressType struct{}

var FfiConverterOptionalTypeNewAddressTypeINSTANCE = FfiConverterOptionalTypeNewAddressType{}

func (c FfiConverterOptionalTypeNewAddressType) Lift(rb RustBufferI) *NewAddressType {
	return LiftFromRustBuffer[*NewAddressType](c, rb)
}

func (_ FfiConverterOptionalTypeNewAddressType) Read(reader io.Reader) *NewAddressType {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeNewAddressTypeINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeNewAddressType) Lower(value *NewAddressType) RustBuffer {
	return LowerIntoRustBuffer[*NewAddressType](c, value)
}

func (_ FfiConverterOptionalTypeNewAddressType) Write(writer io.Writer, value *NewAddressType) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeNewAddressTypeINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeNewAddressType struct{}

func (_ FfiDestroyerOptionalTypeNewAddressType) Destroy(value *NewAddressType) {
	if value != nil {
		FfiDestroyerTypeNewAddressType{}.Destroy(*value)
	}
}

type FfiConverterSequenceTypeListFundsChannel struct{}

var FfiConverterSequenceTypeListFundsChannelINSTANCE = FfiConverterSequenceTypeListFundsChannel{}

func (c FfiConverterSequenceTypeListFundsChannel) Lift(rb RustBufferI) []ListFundsChannel {
	return LiftFromRustBuffer[[]ListFundsChannel](c, rb)
}

func (c FfiConverterSequenceTypeListFundsChannel) Read(reader io.Reader) []ListFundsChannel {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]ListFundsChannel, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeListFundsChannelINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeListFundsChannel) Lower(value []ListFundsChannel) RustBuffer {
	return LowerIntoRustBuffer[[]ListFundsChannel](c, value)
}

func (c FfiConverterSequenceTypeListFundsChannel) Write(writer io.Writer, value []ListFundsChannel) {
	if len(value) > math.MaxInt32 {
		panic("[]ListFundsChannel is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeListFundsChannelINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeListFundsChannel struct{}

func (FfiDestroyerSequenceTypeListFundsChannel) Destroy(sequence []ListFundsChannel) {
	for _, value := range sequence {
		FfiDestroyerTypeListFundsChannel{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeListFundsOutput struct{}

var FfiConverterSequenceTypeListFundsOutputINSTANCE = FfiConverterSequenceTypeListFundsOutput{}

func (c FfiConverterSequenceTypeListFundsOutput) Lift(rb RustBufferI) []ListFundsOutput {
	return LiftFromRustBuffer[[]ListFundsOutput](c, rb)
}

func (c FfiConverterSequenceTypeListFundsOutput) Read(reader io.Reader) []ListFundsOutput {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]ListFundsOutput, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeListFundsOutputINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeListFundsOutput) Lower(value []ListFundsOutput) RustBuffer {
	return LowerIntoRustBuffer[[]ListFundsOutput](c, value)
}

func (c FfiConverterSequenceTypeListFundsOutput) Write(writer io.Writer, value []ListFundsOutput) {
	if len(value) > math.MaxInt32 {
		panic("[]ListFundsOutput is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeListFundsOutputINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeListFundsOutput struct{}

func (FfiDestroyerSequenceTypeListFundsOutput) Destroy(sequence []ListFundsOutput) {
	for _, value := range sequence {
		FfiDestroyerTypeListFundsOutput{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeListInvoicesInvoice struct{}

var FfiConverterSequenceTypeListInvoicesInvoiceINSTANCE = FfiConverterSequenceTypeListInvoicesInvoice{}

func (c FfiConverterSequenceTypeListInvoicesInvoice) Lift(rb RustBufferI) []ListInvoicesInvoice {
	return LiftFromRustBuffer[[]ListInvoicesInvoice](c, rb)
}

func (c FfiConverterSequenceTypeListInvoicesInvoice) Read(reader io.Reader) []ListInvoicesInvoice {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]ListInvoicesInvoice, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeListInvoicesInvoiceINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeListInvoicesInvoice) Lower(value []ListInvoicesInvoice) RustBuffer {
	return LowerIntoRustBuffer[[]ListInvoicesInvoice](c, value)
}

func (c FfiConverterSequenceTypeListInvoicesInvoice) Write(writer io.Writer, value []ListInvoicesInvoice) {
	if len(value) > math.MaxInt32 {
		panic("[]ListInvoicesInvoice is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeListInvoicesInvoiceINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeListInvoicesInvoice struct{}

func (FfiDestroyerSequenceTypeListInvoicesInvoice) Destroy(sequence []ListInvoicesInvoice) {
	for _, value := range sequence {
		FfiDestroyerTypeListInvoicesInvoice{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeListPaymentsPayment struct{}

var FfiConverterSequenceTypeListPaymentsPaymentINSTANCE = FfiConverterSequenceTypeListPaymentsPayment{}

func (c FfiConverterSequenceTypeListPaymentsPayment) Lift(rb RustBufferI) []ListPaymentsPayment {
	return LiftFromRustBuffer[[]ListPaymentsPayment](c, rb)
}

func (c FfiConverterSequenceTypeListPaymentsPayment) Read(reader io.Reader) []ListPaymentsPayment {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]ListPaymentsPayment, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeListPaymentsPaymentINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeListPaymentsPayment) Lower(value []ListPaymentsPayment) RustBuffer {
	return LowerIntoRustBuffer[[]ListPaymentsPayment](c, value)
}

func (c FfiConverterSequenceTypeListPaymentsPayment) Write(writer io.Writer, value []ListPaymentsPayment) {
	if len(value) > math.MaxInt32 {
		panic("[]ListPaymentsPayment is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeListPaymentsPaymentINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeListPaymentsPayment struct{}

func (FfiDestroyerSequenceTypeListPaymentsPayment) Destroy(sequence []ListPaymentsPayment) {
	for _, value := range sequence {
		FfiDestroyerTypeListPaymentsPayment{}.Destroy(value)
	}
}

func NewBlockingGreenlightAlbyClient(mnemonic string, credentials GreenlightCredentials) (*BlockingGreenlightAlbyClient, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_glalby_bindings_fn_func_new_blocking_greenlight_alby_client(FfiConverterStringINSTANCE.Lower(mnemonic), FfiConverterTypeGreenlightCredentialsINSTANCE.Lower(credentials), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *BlockingGreenlightAlbyClient
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBlockingGreenlightAlbyClientINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func Recover(mnemonic string) (GreenlightCredentials, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_glalby_bindings_fn_func_recover(FfiConverterStringINSTANCE.Lower(mnemonic), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue GreenlightCredentials
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeGreenlightCredentialsINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}
