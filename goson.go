package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type JsonNode interface {
	isGoson() bool
}

type MapNode struct {
	Object map[string]interface{}
}

type ArrayNode struct {
	Objects []interface{}
}

type ValueNode struct {
	Val interface{}
}

func (o *MapNode) isGoson() bool {
	return true
}

func (v *ArrayNode) isGoson() bool {
	return true
}

func (v *ValueNode) isGoson() bool {
	return true
}

func (o *MapNode) GetMap(keys ...string) *MapNode {
	return asMapNode(o.Get(keys...))
}

func (o *MapNode) GetValue(keys ...string) *ValueNode {
	return asValueNode(o.Get(keys...))
}

func (o *MapNode) GetString(keys ...string) string {
	node := asValueNode(o.Get(keys...))
	if node == nil {
		return ""
	}
	return node.String()
}

func (o *MapNode) GetToString(keys ...string) string {
	node := asValueNode(o.Get(keys...))
	if node == nil {
		return ""
	}
	return node.ToString()
}

func (o *MapNode) DeleteKeys(keys ...string) {
	for _, key := range keys {
		delete(o.Object, key)
	}
}

func (o *MapNode) Set(key string, value interface{}) {
	o.Object[key] = value
}

func (o *MapNode) GetArray(keys ...string) *ArrayNode {
	return asArrayNode(o.Get(keys...))
}

func (v *ValueNode) Value() interface{} {
	return v.Val
}

func (v *ValueNode) String() string {
	s, ok := v.Val.(string)
	if ok {
		return s
	} else {
		return ""
	}
}

func (v *ValueNode) ToString() string {
	return fmt.Sprintf("%v", v.Val)
}

func (v *ArrayNode) Items() []JsonNode {
	if len(v.Objects) == 0 {
		return nil
	}
	var items []JsonNode
	for _, object := range v.Objects {
		node, err := wrap(object)
		if err != nil {
			continue
		}
		items = append(items, node)
	}
	return items
}

func IsMap(value JsonNode) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case *MapNode:
		return true
	}
	return false
}
func IsArray(value JsonNode) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case *ArrayNode:
		return true
	}
	return false
}

func IsValue(value JsonNode) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case *ValueNode:
		return true
	}
	return false
}

func asMapNode(value JsonNode) *MapNode {
	if value == nil {
		return nil
	}
	switch value.(type) {
	case *MapNode:
		return value.(*MapNode)
	}
	return nil
}

func asArrayNode(value JsonNode) *ArrayNode {
	if value == nil {
		return nil
	}
	switch value.(type) {
	case *ArrayNode:
		return value.(*ArrayNode)
	}
	return nil
}

func asValueNode(value JsonNode) *ValueNode {
	if value == nil {
		return nil
	}
	switch value.(type) {
	case *ValueNode:
		return value.(*ValueNode)
	}
	return nil
}

func (o *MapNode) Get(keys ...string) JsonNode {
	respMap := o.Object
	for i, key := range keys {
		value, exists := respMap[key]
		if !exists {
			return nil
		}
		if i == len(keys)-1 {
			node, err := wrap(value)
			if err != nil {
				return nil
			}
			return node
		}
		respMap = value.(map[string]interface{})
	}
	return nil
}

func ParseFileToMap(fp string) (*MapNode, error) {
	jsonNode, err := ParseFile(fp)
	if err != nil {
		return nil, err
	}
	switch jsonNode.(type) {
	case *MapNode:
		return jsonNode.(*MapNode), nil

	}
	return nil, fmt.Errorf("The type is not a map %T", jsonNode)
}

func ParseFile(fp string) (JsonNode, error) {
	file, err := os.Open(fp)
	if err != nil {
		return nil, err //wrap
	}
	defer file.Close()
	return ParseReader(file)
}

func ParseReader(r io.Reader) (JsonNode, error) {
	jsonBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err //wrap
	}
	return ParseBytes(jsonBytes)
}

func ParseBytes(b []byte) (JsonNode, error) {
	var in interface{}
	err := json.Unmarshal(b, &in)
	if err != nil {
		return nil, err //wrap
	}
	return wrap(in)
}

func ParseObject(in interface{}) (JsonNode, error) {
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}
	switch in.(type) {
	case []interface{}:
		return toArrayNode(in.([]interface{})), nil
	case map[string]interface{}:
		return toObjectNode(in.(map[string]interface{})), nil
	}
	return nil, fmt.Errorf("unexpected input. only `[]interface` or `map[string]interface{}` is expected")
}

func wrap(in interface{}) (JsonNode, error) {
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}
	switch in.(type) {
	case []interface{}:
		return toArrayNode(in.([]interface{})), nil
	case map[string]interface{}:
		return toObjectNode(in.(map[string]interface{})), nil
	}
	return toValueNode(in), nil
}

func toValueNode(in interface{}) *ValueNode {
	return &ValueNode{Val: in}
}

func toArrayNode(items []interface{}) *ArrayNode {
	return &ArrayNode{
		Objects: items,
	}
}

func toObjectNode(in map[string]interface{}) *MapNode {
	return &MapNode{
		Object: in,
	}
}

func AsValue(n JsonNode) *ValueNode {
	return asValueNode(n)
}
func AsMap(n JsonNode) *MapNode {
	return asMapNode(n)
}
func AsArray(n JsonNode) *ArrayNode {
	return asArrayNode(n)
}
func FindAll(n JsonNode, keys ...string) []JsonNode {
	return find(n, true, keys...)
}
func Find(n JsonNode, keys ...string) JsonNode {
	nodes := find(n, false, keys...)
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

func find(n JsonNode, all bool, keys ...string) []JsonNode {
	if IsArray(n) {
		var items []JsonNode
		for _, node := range AsArray(n).Items() {
			item := find(node, all, keys...)
			if item == nil {
				continue
			}
			items = append(items, item...)
			if !all {
				return items
			}
		}
		return items
	} else if IsMap(n) {
		mn := AsMap(n)
		object := mn.Object
		if item := object[keys[0]]; item != nil {
			w, _ := wrap(item)
			if len(keys) == 1 {
				return []JsonNode{w}
			}
			return find(w, all, keys[1:]...)
		}
		var items []JsonNode
		for _, v := range object {
			w, _ := wrap(v)
			item := find(w, all, keys...)
			if item == nil {
				continue
			}
			items = append(items, item...)
			if !all {
				return items
			}
		}
		return items
	} else {
		return nil
	}
}
