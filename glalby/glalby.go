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
			return C.uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_get_info(uniffiStatus)
		})
		if checksum != 49263 {
			// If this happens try cleaning and rebuilding your project
			panic("glalby: uniffi_glalby_bindings_checksum_method_blockinggreenlightalbyclient_get_info: UniFFI API checksum mismatch")
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
}

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
