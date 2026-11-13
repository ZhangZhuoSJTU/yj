package order

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	goyaml "gopkg.in/yaml.v3"
)

type MapSlice []MapItem

type MapItem struct {
	Key, Val interface{}
}

func (m MapSlice) Merge(in MapSlice) MapSlice {
	if len(m) == 0 {
		return in
	}
	t := make(map[interface{}]struct{}, len(in))
	for _, item := range in {
		t[item.Key] = struct{}{}
	}
	out := m[:0]
	for _, item := range m {
		if _, ok := t[item.Key]; !ok {
			out = append(out, item)
		}
	}
	return append(out, in...)
}

var MapSliceEscapeHTML = false

func encodeJSONValue(v interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(MapSliceEscapeHTML)
	if err := enc.Encode(&v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m MapSlice) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{'{'})
	for i, item := range m {
		val, err := encodeJSONValue(item.Val)
		if err != nil {
			return nil, err
		}
		buf.WriteString(fmt.Sprintf("%q:", fmt.Sprint(item.Key)))
		buf.Write(val)
		if i < len(m)-1 {
			buf.Write([]byte{','})
		}
	}
	buf.Write([]byte{'}'})
	return buf.Bytes(), nil
}

func (m MapSlice) MarshalYAML() (interface{}, error) {
	var node goyaml.Node
	if err := node.Encode(map[string]interface{}{}); err != nil {
		return nil, err
	}
	node.Style = 0
	node.Content = make([]*goyaml.Node, 0, len(m)*2)
	for _, item := range m {
		var knode, vnode goyaml.Node
		if err := knode.Encode(item.Key); err != nil {
			return nil, err
		}
		if err := vnode.Encode(item.Val); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, &knode, &vnode)
	}
	return &node, nil
}

func structFields(m MapSlice) ([]reflect.StructField, error) {
	fields := make([]reflect.StructField, 0, len(m))
	for i, item := range m {
		key, ok := item.Key.(string)
		if !ok {
			return nil, fmt.Errorf("non-string key: %#v", item.Key)
		}
		fields = append(fields, reflect.StructField{
			Name: alphaIndex(uint64(i)),
			Type: itemType(item.Val),
			Tag:  reflect.StructTag("toml:" + strconv.Quote(key)),
		})
	}
	return fields, nil
}

func itemType(v interface{}) reflect.Type {
	typ := reflect.TypeOf(v)
	if typ == nil {
		typ = reflect.TypeOf((*interface{})(nil)).Elem()
	}
	return typ
}

func (m MapSlice) Struct() (interface{}, error) {
	fields, err := structFields(m)
	if err != nil {
		return nil, err
	}
	v := reflect.New(reflect.StructOf(fields)).Elem()
	for i, item := range m {
		if item.Val == nil {
			continue
		}
		v.Field(i).Set(reflect.ValueOf(item.Val))
	}

	return v.Addr().Interface(), nil
}

func alphaIndex(i uint64) string {
	buf := &bytes.Buffer{}
	alphaIndexBuf(i, buf)
	return buf.String()
}

func alphaIndexBuf(i uint64, buf *bytes.Buffer) {
	const base = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	n := uint64(len(base))
	if i/n != 0 {
		alphaIndexBuf(i/n, buf)
	}
	buf.WriteByte(base[i%n])
}
