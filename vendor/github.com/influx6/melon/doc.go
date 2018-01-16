// Package melon exposes a series of reader and writers for different types, it is expected to be a repositoty for
// given series of types following the io.Reader and io.WriteCloser approach.
//
// @templater(id => Model, gen => Partial.Go, file => _model.gml)
//
// @templaterTypesFor(id => Model, filename => conn_melon.go, Name => Conn, Type => net.Conn, asJSON, {
//  {
//    "imports": []string{"net"}
//  }
// })
// @templaterTypesFor(id => Model, filename => int_melon.go, Name => Int, Type => int)
// @templatertypesfor(id => model, filename => uint_melon.go, name => uint, type => uint)
// @templaterTypesFor(id => Model, filename => bool_melon.go, Name => Bool, Type => bool)
// @templaterTypesFor(id => Model, filename => byte_melon.go, Name => Byte, Type => byte)
// @templaterTypesFor(id => Model, filename => int8_melon.go, Name => Int8, Type => int8)
// @templaterTypesFor(id => Model, filename => error_melon.go, Name => Error, Type => error)
// @templaterTypesFor(id => Model, filename => int16_melon.go, Name => Int16, Type => int16)
// @templaterTypesFor(id => Model, filename => int32_melon.go, Name => Int32, Type => int32)
// @templaterTypesFor(id => Model, filename => int64_melon.go, Name => Int64, Type => int64)
// @templaterTypesFor(id => Model, filename => uint8_melon.go, Name => UInt8, Type => uint8)
// @templaterTypesFor(id => Model, filename => uint32_melon.go, Name => UInt32, Type => uint32)
// @templaterTypesFor(id => Model, filename => uint64_melon.go, Name => UInt64, Type => uint64)
// @templaterTypesFor(id => Model, filename => string_melon.go, Name => String, Type => string)
// @templaterTypesFor(id => Model, filename => float64_melon.go, Name => Float64, Type => float64)
// @templaterTypesFor(id => Model, filename => float32_melon.go, Name => Float32, Type => float32)
// @templaterTypesFor(id => Model, filename => map_melon.go, Name => Map, Type => map[string]interface{})
// @templaterTypesFor(id => Model, filename => complex64_melon.go, Name => Complex64, Type => complex64)
// @templaterTypesFor(id => Model, filename => interface_melon.go, Name => Interface, Type => interface{})
// @templaterTypesFor(id => Model, filename => complex128_melon.go, Name => Complex128, Type => complex128)
// @templaterTypesFor(id => Model, filename => map_of_string_melon.go, Name => MapOfString, Type => map[string]string)
// @templaterTypesFor(id => Model, filename => map_of_any_melon.go, Name => MapOfAny, Type => map[interface{}]interface{})
//
package melon
