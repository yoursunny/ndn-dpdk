package cptr

/*
#include "../../csrc/core/common.h"
#include <spdk/env.h>
#include <spdk/json.h>

int go_spdkJSONWrite(void* ctx, void* data, size_t size);
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"errors"
	"unsafe"
)

func init() {
	// As of SPDK 21.04, explicitly calling a function in libspdk_env_dpdk.so is needed to prevent a linker error.
	C.spdk_env_get_core_count()
}

// CaptureSpdkJSON invokes a function that writes to *C.struct_spdk_json_write_ctx, and unmarshals what's been written.
func CaptureSpdkJSON(f func(w unsafe.Pointer), ptr interface{}) (e error) {
	buf := new(bytes.Buffer)
	ctx := CtxPut(buf)
	defer CtxClear(ctx)

	w := C.spdk_json_write_begin(C.spdk_json_write_cb(C.go_spdkJSONWrite), ctx, 0)
	f(unsafe.Pointer(w))
	if res := C.spdk_json_write_end(w); res != 0 {
		return errors.New("spdk_json_write_end failed")
	}
	return json.Unmarshal(buf.Bytes(), ptr)
}

// SpdkJSONObject can be used with CaptureSpdkJSON to wrap the output in a JSON object.
func SpdkJSONObject(f func(w unsafe.Pointer)) func(w unsafe.Pointer) {
	return func(w unsafe.Pointer) {
		jw := (*C.struct_spdk_json_write_ctx)(w)
		C.spdk_json_write_object_begin(jw)
		f(w)
		C.spdk_json_write_object_end(jw)
	}
}

//export go_spdkJSONWrite
func go_spdkJSONWrite(ctx, data unsafe.Pointer, size C.size_t) C.int {
	buf := CtxGet(ctx).(*bytes.Buffer)
	buf.Write(C.GoBytes(data, C.int(size)))
	return 0
}
