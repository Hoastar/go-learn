# JSON
JSON（JavaScript 对象表示，JavaScript Object Notation）作为一种轻量级的数据交换格式，在今天几乎占据了绝大多数的比例。虽然更紧凑的数据交换格式相比，它的序列化和反序列化性能不足，但是它也提供了良好的可读性与易用性，在不极致机制性能的情况下，JSON 是一种非常好的选择。

## 设计原理
几乎所有的现代编程语言都会将处理JSON的函数直接纳入标准库，Go语言也不例外，它通过encoding/json 对外提高标准的 JSON 序列化和反序列化方法，即 encoding/json.Marshal 和 encoding/json.Unmarshal，他们也是包中最常用的两个方法。

[图-序列化和反序列化](https://img.draveness.me/2020-04-25-15878293719232-json-marshal-and-unmarshal.png)

序列化和反序列化的开销完全不同，JSON 反序列化的开销是序列化开销的好几倍。Go 语言中的 JSON 序列化过程不需要被序列化的对象预先实现任何接口，它会通过反射获取结构体或者数组中的值并以树形的结构递归地进行编码，标准库也会根据 encoding/json.Unmarshal 中传入的值对 JSON 进行解码。

Go 语言 JSON 标准库编码和解码的过程大量地运用了反射这一特性。我们在这里会简单学习 JSON 标准库中的接口和标签，这是它为开发者提供的为数不多的影响编解码过程的接口。

### 接口
JSON 标准库中提供了 encoding/json.Marshal 和 encoding/json.Unmarshal 两个接口分别可以影响 JSON的序列化和反序列化结果：

```
type Marshaler interface {
    MarshalJSON() ([]byte, error)
}

type Unmarshaler interface {
    UnmarshalJSON([]byte, error)
}
```

在 JSON 序列化和反序列化的过程中，它们会使用反射判断结构体类型是否实现了上述接口，如果实现了上述接口就会优先使用对应的方法进行编码和解码操作，除了这两个方法之外，Go 语言其实还提供了另外两个用于控制编解码结果的方法，即 encoding.TextMarshaler 和 encoding.TextUnmarshaler：

```
type TextMarshaler interface {
	MarshalText() (text []byte, err error)
}

type TextUnmarshaler interface {
	UnmarshalText(text []byte) error
}
```
一旦发现 JSON 相关的序列化方法没有被实现，上述两个方法会作为候选方法被 JSON 标准库调用，参与编解码的过程。总得来说，我们可以在任意类型上实现上述这四个方法自定义最终的结果，后面的两个方法的适用范围更广，但是不会被 JSON 标准库优先调用。

### 标签

Go 语言的结构体也是一个比较有趣的功能，在默认情况下，当我们在序列化和反序列化结构体时，标准库都会认为字段名和JSON中具有一一对应的关系，然而Go语言的字段一般都是驼峰命名法，JSON 中下划线的命名方式相对比较常见，所以使用标签这一特性直接建立键与字段之间的映射关系是一个非常方便的设计。

[结构体与JSON的映射](https://img.draveness.me/2020-04-25-15878293719272-struct-and-json.png)

```
type Author struct {
    // omitempty 忽略零值 
    Name string `json:"name, omitempty"`
    Age string `json:"age, string, omitempty"`
}
```
常见的两个标签是 string 和 omitempty，前者表示当前的整数或者浮点数是由 JSON 中的字符串表示的，而另一个字段 omitempty 会在字段为零值时，直接在生成的 JSON 中忽略对应的键值对，例如："age": 0、"author": "" 等。标准库会使用 encoding/json.parseTag 函数来解析标签：

```
func parseTag(tag string) (string, tagOptions) {
    if idx := strings.Index(tag, ","); idx != -1 {
        return tag[:idx], tagOptions(tag[idx+1:])
    }
    return tag, tagOptions("")
}
```

从该方法的实现中，我们能分析出 JSON 标准库中的合法标签是什么形式的 — 标签名和标签选项都以 , 连接，最前面的字符串为标签名，后面的都是标签选项。

## 序列化
encoding/json.Marshal 是 JSON 标准库中提供的最简单的序列化函数，它会接受一个 interface{} 类型的值作为参考，这也意味这几乎全部的 Go语言变量都可以被JSON 标准库序列化，为了提供如此复杂和通用的功能，在静态语言中是常见的选项，我们来学习一下该方法的实现：
```
func Marshaler(v interface{}) ([]byte, error) {
    e := NewEncodeState()
    err := e.marshal(v, encOpts{escapeHTML: true})
    if err != nil {
        return nil, err
    }

    buf := append([]byte(nil), e.Bytes()...)
    encodeStatePool.Put(e)
    return buf, nil
}
```
上述方法会调用 encoding/json.newEncodeState 从全局的编码状态池中获取 encoding/json.encodeState，随后的序列化过程都会使用这个编码状态，该结构体也会在编码结束后被重新放回池中以便重复利用。

[图-序列化调用栈](https://img.draveness.me/2020-04-25-15878293719272-struct-and-json.png)

按照如上所示的复杂调用栈，一系列的序列化方法在最后获取了对象的反射类型并调用了 encoding/json.newTypeEncoder 这个核心的编码方法，该方法会递归地为所有的类型找到对应的编码方法，不过它的执行过程可以分为以下两个步骤：

1. 获取用户自定义的 encoding /json.Marshaler 或者 encoding.TextMarshaler 编码器
2. 获取标准库中为基本类型内置的 JSON 编码器

在该方法的第一部分，我们会检查当前值的类型是否可以使用自定义的编码器，这里有两种不同的判断方法：

```
func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {

    // t.Kind() != reflect.Ptr 如果不是指针
    if t.Kind() != reflect.Ptr && allowAddr && reflect.PtrTo(t).Implements(marshalerType) {
        return newCondAddrEncoder(addrMarshalerEncoder, newTypeEncoder(t, false))
    }

    if t.Implements(marshalerType) {
        return marshalerEncoder
    }

    if t.Kind() != reflect.Ptr && allowAddr && reflect.PtrTo(t).Implements(textMarshalerType) {
        return newCondAddrEncoder(addrTextMarshalerEncoder, newTypeEncoder(t, false))
    }

    if t.Implements(textMarshalerType) {
        return textMarshalerEncoder
    }
    ...
}
```
1. 如果当前值时值类型、可以取地址并且值类型对应的指针实现了 encoding/json.Marshaler接口，调用 encoding.newCondAddrEncoder 获取一个条件编码器，条件编码器会在 encoding/json.addrMarshalerEncoder 失败时重新选择新的编码器
2. 如果当前类型实现了 encoding/json.Marshaler 接口，可以直接使用 encoding/json.marshalerEncoder 对值进行序列化

在这段代码中，标准库对 encoding/TextMarshaler 的处理也几乎完全相同，只是它会先预判 encoding/json.Marshaler 接口。

encoding/json.newTypeEncoder方法随后会根据传入值的反射类型获取对应的编码器，其中包括bool、int、float等基本类型编码器等和数组、结构体、切片等复杂类型的编码器：
```
func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {
    ...
    switch t.Kind() {
	case reflect.Bool:
		return boolEncoder
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intEncoder
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintEncoder
	case reflect.Float32:
		return float32Encoder
	case reflect.Float64:
		return float64Encoder
	case reflect.String:
		return stringEncoder
	case reflect.Interface:
		return interfaceEncoder
	case reflect.Struct:
		return newStructEncoder(t)
	case reflect.Map:
		return newMapEncoder(t)
	case reflect.Slice:
		return newSliceEncoder(t)
	case reflect.Array:
		return newArrayEncoder(t)
	case reflect.Ptr:
		return newPtrEncoder(t)
	default:
		return unsupportedTypeEncoder
	}
}
```

我们在这里就不一一介绍学习的内置类型编码器了，只挑选其中几个进行学习。首先我们先来看看布尔值的 JSON 编码器，它的实现很简单。

```
func boolEncoder(e *encodeState, v reflect.Value, opts encOpts) {
    if opts.quoted {
        e.WriteByte('"')
    }
    if v.Bool() {
        e.WriteString("true")
    } else {
        e.WriteString("false")
    }
    if opts.quoted {
        e.WriteByte('"')
    }
}
```
它会根据当前值向编码状态写入不同的字符串，也就是 true 或者 false，除此之外还会根据编码配置决定是否要在布尔值周围写入双引号"，而其他的编码类型也都大同小异。

复杂类型的编码器有着相对复杂的控制结构，我们在这里以结构体的编码器 encoding/json.structEncoder为例学习它们的原理，encoding/json.typeEncoder 会为当前结构体所有的字段调用 encoding/json.typeEncoder 获取类型编码器并返回 encoding/json.structEncoder.encode 方法：

```
func newStructEncoder(t reflect.Type) encoderFunc {
    se := structEncoder{fields: cachedTypeFidlds(t)}
    return se.encode
}
```

从 encoding/json.structEncoder.encode 的实现我们能看出结构体序列的结果，该方法会遍历结构体中的全部字段，在写入字段名之后，它会调用字段对应类型的编码方法将该字段对应的 JSON 写入缓冲区：
```
func (se StructEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
    next := byte('{')
FieldLoop:
    for i := range se.fields.list {
        f := &se.fields.list[i]

        fv := v
        for _, i := range f.index {
            if fv.Kind() == reflect.Ptr {
                if fv.IsNil() {
                    continue FieldLoop
                }
                fv = fv.Elem()
            }
            fv = fv.Field(i)
        }
        if f.omitEmpty && isEmptyValue(fv) {
            continue
        }

        e.WriteByte(next)
        next = ','
        e.WriteString(f.nameNonESC)
        opts.quoted = f.quoted
        f.encoder(e, fv, opts)
    }
    if next == '{' {
        e.WriteString("{}")
    } else {
        e.WriteByte('}')
    }
}
```

数组以及指针等编码器的实现原理与该方法也没有太大的区别，它们都会使用类似的策略递归地调用持有字段的编码方法，这也就能形成一个如下图所示的树形结构：

[图-序列化与树形结构体](https://img.draveness.me/2020-04-25-15878293719308-struct-encoder.png)

树形结构的所有叶节点都是基础类型编码器或者开发者自定义的编码器，得到了整棵树的编码器之后会调用 encoding/json.encodeState.reflectValue 从根节点依次调用整棵树的序列化函数，整个 JSON 序列化的过程其实是查找类型和子类型的编码方法并调用的过程，它利用了大量反射的特性做到了足够的通用。