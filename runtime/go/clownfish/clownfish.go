/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package clownfish

/*

#include <limits.h>

#include "charmony.h"

#include "cfish_parcel.h"
#include "testcfish_parcel.h"

#include "Clownfish/Obj.h"
#include "Clownfish/Err.h"
#include "Clownfish/Class.h"
#include "Clownfish/String.h"
#include "Clownfish/Blob.h"
#include "Clownfish/Hash.h"
#include "Clownfish/HashIterator.h"
#include "Clownfish/Vector.h"
#include "Clownfish/Num.h"
#include "Clownfish/Boolean.h"
#include "Clownfish/Util/Memory.h"
#include "Clownfish/Method.h"

extern void
GoCfish_PanicErr_internal(cfish_Err *error);
typedef void
(*cfish_Err_do_throw_t)(cfish_Err *error);
extern cfish_Err_do_throw_t GoCfish_PanicErr;

extern cfish_Err*
GoCfish_TrapErr_internal(CFISH_Err_Attempt_t routine, void *context);
typedef cfish_Err*
(*cfish_Err_trap_t)(CFISH_Err_Attempt_t routine, void *context);
extern cfish_Err_trap_t GoCfish_TrapErr;

// C symbols linked into a Go-built package archive are not visible to
// external C code -- but internal code *can* see symbols from outside.
// This allows us to fake up symbol export by assigning values only known
// interally to external symbols during Go package initialization.
static CHY_INLINE void
GoCfish_glue_exported_symbols() {
	GoCfish_PanicErr = GoCfish_PanicErr_internal;
	GoCfish_TrapErr  = GoCfish_TrapErr_internal;
}

static CHY_INLINE void
GoCfish_RunRoutine(CFISH_Err_Attempt_t routine, void *context) {
	routine(context);
}

*/
import "C"
import "runtime"
import "unsafe"
import "reflect"
import "fmt"
import "math"
import "sync"

const (
	maxUint = ^uint(0)
	minUint = 0
	maxInt  = int(^uint(0) >> 1)
	minInt  = -(maxInt - 1)
)

var classRegMutex sync.Mutex
var classReg *map[unsafe.Pointer]ObjCLASS

func init() {
	C.GoCfish_glue_exported_symbols()
	C.cfish_bootstrap_parcel()
	C.testcfish_bootstrap_parcel()
	registerClasses()
}

func RegisterClasses(classes []ObjCLASS) {
	classRegMutex.Lock()
	newSize := len(classes)
	if classReg != nil {
		newSize += len(*classReg)
	}
	newReg := make(map[unsafe.Pointer]ObjCLASS, newSize)
	if classReg != nil {
		for k, v := range *classReg {
			newReg[k] = v
		}
	}
	for _, v := range classes {
		newReg[unsafe.Pointer(v.TOPTR())] = v
	}
	classReg = &newReg
	classRegMutex.Unlock()
}

func fetchClass(ptr unsafe.Pointer) ObjCLASS {
	classPtr := C.cfish_Obj_get_class((*C.cfish_Obj)(ptr))
	class := (*classReg)[unsafe.Pointer(classPtr)]
	if class == nil {
		className := CFStringToGo(unsafe.Pointer(C.CFISH_Class_Get_Name(classPtr)))
		panic(fmt.Sprintf("Failed to find implementation for %s", className))
	}
	return class
}

func WRAPAny(ptr unsafe.Pointer) Obj {
	if ptr == nil {
		return nil
	}
	return fetchClass(ptr).CF_WRAP_PTR(ptr)
}

type ObjCLASSIMP struct {
	ClassIMP
}

type ObjIMP struct {
	ref uintptr
}

func GetClass(o Obj) Class {
	objCF := (*C.cfish_Obj)(Unwrap(o, "o"))
	classCF := C.cfish_Obj_get_class(objCF)
	return WRAPClass(unsafe.Pointer(classCF))
}

func FetchClass(className string) Class {
	nameCF := (*C.cfish_String)(GoToString(className))
	defer C.cfish_decref(unsafe.Pointer(nameCF))
	class := C.cfish_Class_fetch_class(nameCF)
	return WRAPClass(unsafe.Pointer(class))
}

func (c *ClassIMP) GetMethods() []Method {
	self := (*C.cfish_Class)(Unwrap(c, "c"))
	methsVec := C.CFISH_Class_Get_Methods(self)
	size := C.CFISH_Vec_Get_Size(methsVec)
	meths := make([]Method, 0, int(size))
	for i := C.size_t(0); i < size; i++ {
		meths = append(meths, WRAPMethod(unsafe.Pointer(C.CFISH_Vec_Fetch(methsVec, i))))
	}
	C.cfish_decref(unsafe.Pointer(methsVec))
	return meths
}

func (c *ClassIMP) MakeObj() Obj {
	self := (*C.cfish_Class)(Unwrap(c, "c"))
	retvalCF := C.CFISH_Class_Make_Obj(self)
	return WRAPAny(unsafe.Pointer(retvalCF))
}

func NewMethod(name string, callbackFunc unsafe.Pointer, offset uint32) Method {
	nameCF := (*C.cfish_String)(GoToString(name))
	defer C.cfish_decref(unsafe.Pointer(nameCF))
	methCF := C.cfish_Method_new(nameCF, C.cfish_method_t(callbackFunc),
		C.uint32_t(offset));
	return WRAPMethod(unsafe.Pointer(methCF))
}

func NewString(goString string) String {
	return WRAPString(GoToString(goString))
}

func NewStringIterator(str String, offset uintptr) StringIterator {
	strCF := (*C.cfish_String)(Unwrap(str, "str"))
	iter := C.cfish_StrIter_new(strCF, C.size_t(offset))
	return WRAPStringIterator(unsafe.Pointer(iter))
}

func NewVector(size int) Vector {
	if (size < 0 || uint64(size) > ^uint64(0)) {
		panic(NewErr(fmt.Sprintf("Param 'size' out of range: %d", size)))
	}
	cfObj := C.cfish_Vec_new(C.size_t(size))
	return WRAPVector(unsafe.Pointer(cfObj))
}

func NewHash(size int) Hash {
	if (size < 0 || uint64(size) > ^uint64(0)) {
		panic(NewErr(fmt.Sprintf("Param 'size' out of range: %d", size)))
	}
	cfObj := C.cfish_Hash_new(C.size_t(size))
	return WRAPHash(unsafe.Pointer(cfObj))
}

func NewHashIterator(hash Hash) HashIterator {
	hashCF := (*C.cfish_Hash)(Unwrap(hash, "hash"))
	cfObj := C.cfish_HashIter_new(hashCF)
	return WRAPHashIterator(unsafe.Pointer(cfObj))
}

func (h *HashIMP) Keys() []string {
	self := (*C.cfish_Hash)(Unwrap(h, "h"))
	keysCF := C.CFISH_Hash_Keys(self)
	numKeys := C.CFISH_Vec_Get_Size(keysCF)
	keys := make([]string, 0, int(numKeys))
	for i := C.size_t(0); i < numKeys; i++ {
		keys = append(keys, CFStringToGo(unsafe.Pointer(C.CFISH_Vec_Fetch(keysCF, i))))
	}
	C.cfish_decref(unsafe.Pointer(keysCF))
	return keys
}

func (o *ObjIMP) INITOBJ(ptr unsafe.Pointer) {
	o.ref = uintptr(ptr)
	runtime.SetFinalizer(o, ClearRef)
}

func ClearRef (o *ObjIMP) {
	C.cfish_dec_refcount(unsafe.Pointer(o.ref))
	o.ref = 0
}

func (o *ObjIMP) TOPTR() uintptr {
	return o.ref
}

func (o *ObjIMP)Clone() Obj {
	self := (*C.cfish_Obj)(Unwrap(o, "o"))
	dupe := C.CFISH_Obj_Clone(self)
	return WRAPAny(unsafe.Pointer(dupe)).(Obj)
}

// Convert a Go type into an incremented Clownfish object.  If the supplied
// object is a Clownfish object wrapped in a Go struct, extract the Clownfish
// object and incref it before returning its address.
func GoToClownfish(value interface{}, nullable bool) unsafe.Pointer {
	if obj, ok := value.(Obj); ok {
		return unsafe.Pointer(C.cfish_incref(unsafe.Pointer(obj.TOPTR())))
	}

	v := reflect.ValueOf(value)
	k := v.Kind()
	if k == reflect.Ptr {
		v = v.Elem()
		k = v.Kind()
	}

	// Convert the value according to its type if possible.
	switch (k) {
	case reflect.Invalid:
		if !nullable {
			panic(NewErr("Can't convert nil to non-nullable type"))
		}
		return nil
	case reflect.String:
		return GoToString(v.String())
	case reflect.Int,
	     reflect.Int64,
	     reflect.Int32,
	     reflect.Int16,
	     reflect.Int8:
		return GoToInteger(v.Int())
	case reflect.Uint,
	     reflect.Uintptr,
	     reflect.Uint64,
	     reflect.Uint32,
	     reflect.Uint16,
	     reflect.Uint8:
		u := v.Uint()
		if u > math.MaxInt64 {
			mess := fmt.Sprintf("uint value too large: %v", v)
			panic(NewErr(mess))
		}
		return GoToInteger(int64(u))
	case reflect.Float32,
	     reflect.Float64:
		return GoToFloat(v.Float())
	case reflect.Bool:
		return GoToBoolean(v.Bool())
	case reflect.Array,
	     reflect.Slice:
		if a, ok := value.([]byte); ok {
			return GoToBlob(a)
		} else if a, ok := value.([]interface{}); ok {
			return GoToVector(a)
		}
	case reflect.Map:
		if m, ok := value.(map[string]interface{}); ok {
			return GoToHash(m)
		}
	}

	// Report a conversion error.
	panic(NewErr(fmt.Sprintf("Can't convert a %T to Clownfish", value)))
}

func GoToString(v string) unsafe.Pointer {
	size := len(v)
	str := C.CString(v)
	return unsafe.Pointer(C.cfish_Str_new_steal_utf8(str, C.size_t(size)))
}

func GoToBlob(v []byte) unsafe.Pointer {
	size := C.size_t(len(v))
	var buf unsafe.Pointer = nil
	if size > 0 {
		buf = unsafe.Pointer(&v[0])
	}
	return unsafe.Pointer(C.cfish_Blob_new(buf, size))
}

func GoToInteger(v int64) unsafe.Pointer {
	return unsafe.Pointer(C.cfish_Int_new(C.int64_t(v)))
}

func GoToFloat(v float64) unsafe.Pointer {
	return unsafe.Pointer(C.cfish_Float_new(C.double(v)))
}

func GoToBoolean(v bool) unsafe.Pointer {
	return unsafe.Pointer(C.cfish_Bool_singleton(C.bool(v)))
}

func GoToVector(v []interface{}) unsafe.Pointer {
	size := len(v)
	vec := C.cfish_Vec_new(C.size_t(size))
	for i := 0; i < size; i++ {
		elem := GoToClownfish(v[i], true)
		C.CFISH_Vec_Store(vec, C.size_t(i), (*C.cfish_Obj)(elem))
	}
	return unsafe.Pointer(vec)
}

func GoToHash(v map[string]interface{}) unsafe.Pointer {
	size := len(v)
	hash := C.cfish_Hash_new(C.size_t(size))
	for key, val := range v {
		newVal := GoToClownfish(val, true)
		keySize := len(key)
		keyStr := C.CString(key)
		cfKey := C.cfish_Str_new_steal_utf8(keyStr, C.size_t(keySize))
		defer C.cfish_dec_refcount(unsafe.Pointer(cfKey))
		C.CFISH_Hash_Store(hash, cfKey, (*C.cfish_Obj)(newVal))
	}
	return unsafe.Pointer(hash)
}

func UnwrapNullable(value Obj) unsafe.Pointer {
	if value == nil {
		return nil
	}
	return unsafe.Pointer(value.TOPTR())
}

func Unwrap(value Obj, name string) unsafe.Pointer {
	if value == nil {
		panic(NewErr(fmt.Sprintf("%s cannot be nil", name)))
	}
	return unsafe.Pointer(value.TOPTR())
}

func ToGo(ptr unsafe.Pointer) interface{} {
	if ptr == nil {
		return nil
	}
	class := fetchClass(ptr)
	return class.CF_NEW_FROM_PTR(class, ptr)
}

func (*ObjCLASSIMP) CF_NEW_FROM_PTR(c ObjCLASS, ptr unsafe.Pointer) interface{} {
	return c.CF_WRAP_PTR(ptr)
}

func (*StringCLASSIMP) CF_NEW_FROM_PTR(c ObjCLASS, ptr unsafe.Pointer) interface{} {
	return CFStringToGo(ptr)
}

func CFStringToGo(ptr unsafe.Pointer) string {
	return StringToGo(ptr)
}

func StringToGo(ptr unsafe.Pointer) string {
	cfString := (*C.cfish_String)(ptr)
	if cfString == nil {
		return ""
	}
	if !C.cfish_Str_is_a(cfString, C.CFISH_STRING) {
		cfString := C.CFISH_Str_To_String(cfString)
		defer C.cfish_dec_refcount(unsafe.Pointer(cfString))
	}
	data := C.CFISH_Str_Get_Ptr8(cfString)
	size := C.CFISH_Str_Get_Size(cfString)
	if size > C.size_t(C.INT_MAX) {
		panic(fmt.Sprintf("Overflow: %d > %d", size, C.INT_MAX))
	}
	return C.GoStringN(data, C.int(size))
}

func (*BlobCLASSIMP) CF_NEW_FROM_PTR(c ObjCLASS, ptr unsafe.Pointer) interface{} {
	return BlobToGo(ptr)
}

func BlobToGo(ptr unsafe.Pointer) []byte {
	blob := (*C.cfish_Blob)(ptr)
	if blob == nil {
		return nil
	}
	class := C.cfish_Obj_get_class((*C.cfish_Obj)(ptr))
	if class != C.CFISH_BLOB {
		mess := "Not a Blob: " + StringToGo(unsafe.Pointer(C.CFISH_Class_Get_Name(class)))
		panic(NewErr(mess))
	}
	data := C.CFISH_Blob_Get_Buf(blob)
	size := C.CFISH_Blob_Get_Size(blob)
	if size > C.size_t(C.INT_MAX) {
		panic(fmt.Sprintf("Overflow: %d > %d", size, C.INT_MAX))
	}
	return C.GoBytes(unsafe.Pointer(data), C.int(size))
}

func (*VectorCLASSIMP) CF_NEW_FROM_PTR(c ObjCLASS, ptr unsafe.Pointer) interface{} {
	return VectorToGo(ptr)
}

func VectorToGo(ptr unsafe.Pointer) []interface{} {
	vec := (*C.cfish_Vector)(ptr)
	if vec == nil {
		return nil
	}
	class := C.cfish_Obj_get_class((*C.cfish_Obj)(ptr))
	if class != C.CFISH_VECTOR {
		mess := "Not a Vector: " + StringToGo(unsafe.Pointer(C.CFISH_Class_Get_Name(class)))
		panic(NewErr(mess))
	}
	size := C.CFISH_Vec_Get_Size(vec)
	if size > C.size_t(maxInt) {
		panic(fmt.Sprintf("Overflow: %d > %d", size, maxInt))
	}
	slice := make([]interface{}, int(size))
	for i := 0; i < int(size); i++ {
		slice[i] = ToGo(unsafe.Pointer(C.CFISH_Vec_Fetch(vec, C.size_t(i))))
	}
	return slice
}

func (*HashCLASSIMP) CF_NEW_FROM_PTR(c ObjCLASS, ptr unsafe.Pointer) interface{} {
	return HashToGo(ptr)
}

func HashToGo(ptr unsafe.Pointer) map[string]interface{} {
	hash := (*C.cfish_Hash)(ptr)
	if hash == nil {
		return nil
	}
	class := C.cfish_Obj_get_class((*C.cfish_Obj)(ptr))
	if class != C.CFISH_HASH {
		mess := "Not a Hash: " + StringToGo(unsafe.Pointer(C.CFISH_Class_Get_Name(class)))
		panic(NewErr(mess))
	}
	size := C.CFISH_Hash_Get_Size(hash)
	m := make(map[string]interface{}, int(size))
	iter := C.cfish_HashIter_new(hash)
	defer C.cfish_dec_refcount(unsafe.Pointer(iter))
	for C.CFISH_HashIter_Next(iter) {
		key := C.CFISH_HashIter_Get_Key(iter)
		val := C.CFISH_HashIter_Get_Value(iter)
		m[StringToGo(unsafe.Pointer(key))] = ToGo(unsafe.Pointer(val))
	}
	return m
}

func (*BooleanCLASSIMP) CF_NEW_FROM_PTR(c ObjCLASS, ptr unsafe.Pointer) interface{} {
	return BooleanToGo(ptr)
}

func BooleanToGo(ptr unsafe.Pointer) bool {
	return bool(C.CFISH_Bool_Get_Value((*C.cfish_Boolean)(ptr)))
}

func (*IntegerCLASSIMP) CF_NEW_FROM_PTR(c ObjCLASS, ptr unsafe.Pointer) interface{} {
	return IntegerToGo(ptr)
}

func IntegerToGo(ptr unsafe.Pointer) int64 {
	val := C.CFISH_Int_Get_Value((*C.cfish_Integer)(ptr))
	return int64(val)
}

func (*FloatCLASSIMP) CF_NEW_FROM_PTR(c ObjCLASS, ptr unsafe.Pointer) interface{} {
	return FloatToGo(ptr)
}

func FloatToGo(ptr unsafe.Pointer) float64 {
	val := C.CFISH_Float_Get_Value((*C.cfish_Float)(ptr))
	return float64(val)
}

func (e *ErrIMP) Error() string {
	mess := C.CFISH_Err_Get_Mess((*C.cfish_Err)(unsafe.Pointer(e.ref)))
	return StringToGo(unsafe.Pointer(mess))
}

//export GoCfish_PanicErr_internal
func GoCfish_PanicErr_internal(cfErr *C.cfish_Err) {
	goErr := WRAPAny(unsafe.Pointer(cfErr)).(Err)
	panic(goErr)
}

//export GoCfish_TrapErr_internal
func GoCfish_TrapErr_internal(routine C.CFISH_Err_Attempt_t,
	context unsafe.Pointer) *C.cfish_Err {
	err := TrapErr(func() { C.GoCfish_RunRoutine(routine, context) })
	if err != nil {
		ptr := (err.(Err)).TOPTR()
		return ((*C.cfish_Err)(unsafe.Pointer(C.cfish_incref(unsafe.Pointer(ptr)))))
	}
	return nil
}

// Run the supplied routine, and if it panics with a clownfish.Err, trap and
// return it.
func TrapErr(routine func()) (trapped error) {
	defer func() {
		if r := recover(); r != nil {
			// TODO: pass whitelist of Err types to trap.
			myErr, ok := r.(Err)
			if ok {
				trapped = myErr
			} else {
				// re-panic
				panic(r)
			}
		}
	}()
	routine()
	return trapped
}

func (s *StringIMP) CodePointAt(tick uintptr) rune {
	self := ((*C.cfish_String)(Unwrap(s, "s")))
	retvalCF := C.CFISH_Str_Code_Point_At(self, C.size_t(tick))
	return rune(retvalCF)
}

func (s *StringIMP) CodePointFrom(tick uintptr) rune {
	self := ((*C.cfish_String)(Unwrap(s, "s")))
	retvalCF := C.CFISH_Str_Code_Point_From(self, C.size_t(tick))
	return rune(retvalCF)
}

func NewBoolean(val bool) Boolean {
	return WRAPBoolean(GoToBoolean(val))
}

func NewBlob(content []byte) Blob {
	return WRAPBlob(GoToBlob(content))
}

func (b *BlobIMP) GetBuf() uintptr {
	self := (*C.cfish_Blob)(Unwrap(b, "b"))
	return uintptr(unsafe.Pointer(C.CFISH_Blob_Get_Buf(self)))
}
