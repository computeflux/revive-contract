package main

// jsonutil.go — 通用 JSON 转换工具
//
// 负责两类转换：
//  1. toJSON: 把链上返回的 Go 类型（H160/U128/Option/Result/结构体/枚举...）
//     转换为可 JSON 序列化的值，供前端展示
//  2. convertArg / fillStruct: 把前端提交的 JSON 参数转换为 Go 绑定要求的类型

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

// ──────────────────────────────────────────────
// 1. toJSON — Go 值 → JSON 友好值
// ──────────────────────────────────────────────

func toJSON(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	return toJSONValue(rv)
}

func toJSONValue(rv reflect.Value) any {
	if !rv.IsValid() {
		return nil
	}

	// 指针：解引用（保持 nil）
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		return toJSONValue(rv.Elem())
	}

	// error 接口（枚举错误类型，如 subnet.Error）
	if rv.CanInterface() {
		if err, ok := rv.Interface().(error); ok {
			return err.Error()
		}
	}

	// Option[T]: IsSome() 方法 + V 字段
	if isOptionType(rv.Type()) {
		if rv.MethodByName("IsSome").Call(nil)[0].Bool() {
			return toJSONValue(rv.FieldByName("V"))
		}
		return nil
	}

	// Result[T, E]: IsErr / V / E 字段
	if rv.Type().Name() == "Result" {
		if rv.FieldByName("IsErr").Bool() {
			return map[string]any{
				"ok":    false,
				"error": toJSONValue(rv.FieldByName("E")),
			}
		}
		return map[string]any{
			"ok":    true,
			"value": toJSONValue(rv.FieldByName("V")),
		}
	}

	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool()
	case reflect.String:
		return rv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint()
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			// []byte → 0x hex
			b := make([]byte, rv.Len())
			reflect.Copy(reflect.ValueOf(b), rv)
			return "0x" + hex.EncodeToString(b)
		}
		out := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out = append(out, toJSONValue(rv.Index(i)))
		}
		return out
	case reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			// [N]byte（H160/H256/AccountID）→ 0x hex
			b := make([]byte, rv.Len())
			reflect.Copy(reflect.ValueOf(b), rv)
			return "0x" + hex.EncodeToString(b)
		}
		out := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out = append(out, toJSONValue(rv.Index(i)))
		}
		return out
	case reflect.Struct:
		// 大整数（U128/U256: 含 Int *big.Int 字段）→ 十进制字符串
		if f := rv.FieldByName("Int"); f.IsValid() && f.Kind() == reflect.Ptr && !f.IsNil() {
			if bi, ok := f.Interface().(*big.Int); ok {
				return bi.String()
			}
		}
		// big.Int 值类型 / 定义类型（UCompact = big.Int、Weight.RefTime 等）→ 十进制字符串
		bigIntType := reflect.TypeOf(big.Int{})
		if rv.Type().ConvertibleTo(bigIntType) {
			tmp := reflect.New(bigIntType).Elem()
			tmp.Set(rv.Convert(bigIntType))
			return tmp.Addr().Interface().(fmt.Stringer).String()
		}
		// 其他 Stringer
		if rv.CanInterface() {
			if s, ok := rv.Interface().(fmt.Stringer); ok {
				return s.String()
			}
		}
		return structToJSON(rv)
	case reflect.Map:
		out := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			out[fmt.Sprintf("%v", iter.Key().Interface())] = toJSONValue(iter.Value())
		}
		return out
	default:
		if rv.CanInterface() {
			return fmt.Sprintf("%v", rv.Interface())
		}
		return nil
	}
}

// structToJSON — 结构体 → map[string]any（含枚举指针字段等）
func structToJSON(rv reflect.Value) map[string]any {
	out := map[string]any{}

	t := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		ft := t.Field(i)
		fv := rv.Field(i)
		// 枚举指针字段：nil 则跳过
		if fv.Kind() == reflect.Ptr && fv.IsNil() {
			continue
		}
		out[ft.Name] = toJSONValue(fv)
	}
	return out
}

// isOptionType — 判断是否为 util.Option[T]（有 IsSome 方法 + V 字段）
func isOptionType(t reflect.Type) bool {
	_, hasIsSome := t.MethodByName("IsSome")
	if !hasIsSome {
		return false
	}
	_, hasV := t.FieldByName("V")
	return hasV
}

// ──────────────────────────────────────────────
// 2. JSON 参数 → Go 绑定类型
// ──────────────────────────────────────────────

// convertArg 把单个 JSON 参数转换为目标类型
func convertArg(raw any, t reflect.Type) (reflect.Value, error) {
	if raw == nil {
		return reflect.Zero(t), errors.New("参数为空")
	}
	switch t.Kind() {
	case reflect.String:
		return reflect.ValueOf(fmt.Sprintf("%v", raw)), nil
	case reflect.Bool:
		b, err := toBool(raw)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(b), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := toInt64(raw)
		if err != nil {
			return reflect.Value{}, err
		}
		v := reflect.New(t).Elem()
		v.SetInt(n)
		return v, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := toUint64(raw)
		if err != nil {
			return reflect.Value{}, err
		}
		v := reflect.New(t).Elem()
		v.SetUint(n)
		return v, nil
	case reflect.Slice:
		return convertSlice(raw, t)
	case reflect.Array:
		return convertArray(raw, t)
	case reflect.Struct:
		// U128/U256 等大整数结构体（含 Int *big.Int 字段）
		if f, ok := t.FieldByName("Int"); ok && f.Type == reflect.TypeOf((*big.Int)(nil)) {
			n, err := toBigInt(raw)
			if err != nil {
				return reflect.Value{}, err
			}
			v := reflect.New(t).Elem()
			v.FieldByName("Int").Set(reflect.ValueOf(n))
			return v, nil
		}
		// Option[T]
		if isOptionType(t) {
			if isNoneValue(raw) {
				return reflect.Zero(t), nil
			}
			sf, _ := t.FieldByName("V")
			inner := sf.Type
			converted, err := convertArg(raw, inner)
			if err != nil {
				return reflect.Value{}, err
			}
			opt := reflect.New(t).Elem()
			opt.MethodByName("Set").Call([]reflect.Value{converted})
			return opt, nil
		}
		// 通用结构体（Ip/RunPrice/AssetInfo/...）
		return convertStruct(raw, t)
	default:
		return reflect.Value{}, fmt.Errorf("不支持的类型: %v", t)
	}
}

// isNoneValue — Option 的 None 表示：null / "" / "none" / "null"
func isNoneValue(raw any) bool {
	switch v := raw.(type) {
	case nil:
		return true
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		return s == "" || s == "none" || s == "null"
	default:
		return false
	}
}

// convertStruct — JSON 对象 → 结构体（按字段名大小写不敏感匹配；枚举用非空指针字段表示）
func convertStruct(raw any, t reflect.Type) (reflect.Value, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return reflect.Value{}, fmt.Errorf("需要 JSON 对象，收到 %T", raw)
	}
	v := reflect.New(t).Elem()
	if err := fillStruct(v, m); err != nil {
		return reflect.Value{}, err
	}
	return v, nil
}

func fillStruct(v reflect.Value, m map[string]any) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		raw, ok := lookupJSONKey(m, ft.Name)
		if !ok {
			continue
		}
		fv := v.Field(i)

		// 指针字段（枚举变体）：分配并递归填充
		if fv.Kind() == reflect.Ptr {
			if raw == nil {
				continue // 保持 nil
			}
			elem := reflect.New(fv.Type().Elem()).Elem()
			if err := fillStructValue(elem, raw); err != nil {
				return fmt.Errorf("%s: %w", ft.Name, err)
			}
			fv.Set(elem.Addr())
			continue
		}

		// Option 字段：none 保持零值；some 用 Set 设置
		if isOptionType(fv.Type()) {
			if isNoneValue(raw) {
				continue
			}
			sf, _ := fv.Type().FieldByName("V")
			inner := sf.Type
			converted, err := convertArg(raw, inner)
			if err != nil {
				return fmt.Errorf("%s: %w", ft.Name, err)
			}
			fv.Addr().MethodByName("Set").Call([]reflect.Value{converted})
			continue
		}

		converted, err := convertArg(raw, fv.Type())
		if err != nil {
			return fmt.Errorf("%s: %w", ft.Name, err)
		}
		fv.Set(converted)
	}
	return nil
}

func fillStructValue(v reflect.Value, raw any) error {
	if v.Kind() == reflect.Struct && isOptionType(v.Type()) {
		if isNoneValue(raw) {
			return nil
		}
		sf, _ := v.Type().FieldByName("V")
		inner := sf.Type
		converted, err := convertArg(raw, inner)
		if err != nil {
			return err
		}
		v.Addr().MethodByName("Set").Call([]reflect.Value{converted})
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		converted, err := convertArg(raw, v.Type())
		if err != nil {
			return err
		}
		v.Set(converted)
		return nil
	}
	return fillStruct(v, m)
}

// convertSlice — JSON 数组 → 切片
func convertSlice(raw any, t reflect.Type) (reflect.Value, error) {
	// []byte 特例：接受 hex 字符串或数字数组
	if t.Elem().Kind() == reflect.Uint8 {
		if s, ok := raw.(string); ok {
			b, err := hexStringToBytes(s)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(b), nil
		}
	}
	arr, ok := raw.([]any)
	if !ok {
		return reflect.Value{}, fmt.Errorf("需要 JSON 数组，收到 %T", raw)
	}
	out := reflect.MakeSlice(t, 0, len(arr))
	for i, item := range arr {
		cv, err := convertArg(item, t.Elem())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("元素 %d: %w", i, err)
		}
		out = reflect.Append(out, cv)
	}
	return out, nil
}

// convertArray — JSON 数组 → 定长数组（如 [32]byte）
func convertArray(raw any, t reflect.Type) (reflect.Value, error) {
	// 定长字节数组：hex 字符串
	if t.Elem().Kind() == reflect.Uint8 {
		var b []byte
		if s, ok := raw.(string); ok {
			var err error
			b, err = hexStringToBytes(s)
			if err != nil {
				return reflect.Value{}, err
			}
		} else if arr, ok := raw.([]any); ok {
			for _, item := range arr {
				n, err := toUint64(item)
				if err != nil {
					return reflect.Value{}, err
				}
				b = append(b, byte(n))
			}
		} else {
			return reflect.Value{}, fmt.Errorf("需要 hex 字符串或数组，收到 %T", raw)
		}
		if len(b) != t.Len() {
			return reflect.Value{}, fmt.Errorf("hex 长度 %d 不等于 %d", len(b), t.Len())
		}
		out := reflect.New(t).Elem()
		reflect.Copy(out, reflect.ValueOf(b))
		return out, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return reflect.Value{}, fmt.Errorf("需要 JSON 数组，收到 %T", raw)
	}
	if len(arr) != t.Len() {
		return reflect.Value{}, fmt.Errorf("数组长度 %d 不等于 %d", len(arr), t.Len())
	}
	out := reflect.New(t).Elem()
	for i := 0; i < t.Len(); i++ {
		cv, err := convertArg(arr[i], t.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		out.Index(i).Set(cv)
	}
	return out, nil
}

// ──────────────────────────────────────────────
// 3. 基础类型解析
// ──────────────────────────────────────────────

func lookupJSONKey(m map[string]any, fieldName string) (any, bool) {
	if v, ok := m[fieldName]; ok {
		return v, true
	}
	if v, ok := m[strings.ToLower(fieldName)]; ok {
		return v, true
	}
	// F0 → f0
	if len(fieldName) == 2 && fieldName[0] == 'F' {
		if v, ok := m[strings.ToLower(fieldName)]; ok {
			return v, true
		}
	}
	return nil, false
}

func toBool(raw any) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off", "":
			return false, nil
		}
	}
	return false, fmt.Errorf("无法解析为 bool: %v", raw)
}

func toInt64(raw any) (int64, error) {
	switch v := raw.(type) {
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	case json.Number:
		return v.Int64()
	}
	return 0, fmt.Errorf("无法解析为整数: %v", raw)
}

func toUint64(raw any) (uint64, error) {
	switch v := raw.(type) {
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("负数: %v", v)
		}
		return uint64(v), nil
	case string:
		s := strings.TrimSpace(v)
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			return strconv.ParseUint(s[2:], 16, 64)
		}
		return strconv.ParseUint(s, 10, 64)
	case json.Number:
		return strconv.ParseUint(v.String(), 10, 64)
	}
	return 0, fmt.Errorf("无法解析为无符号整数: %v", raw)
}

// toBigInt — 解析大整数（十进制字符串 / 0x 十六进制 / 数字）
func toBigInt(raw any) (*big.Int, error) {
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, errors.New("空字符串")
		}
		base := 10
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			base = 16
			s = s[2:]
		}
		n, ok := new(big.Int).SetString(s, base)
		if !ok {
			return nil, fmt.Errorf("无法解析大整数: %v", raw)
		}
		return n, nil
	case float64:
		return big.NewInt(int64(v)), nil
	case json.Number:
		n, ok := new(big.Int).SetString(v.String(), 10)
		if !ok {
			return nil, fmt.Errorf("无法解析大整数: %v", raw)
		}
		return n, nil
	}
	return nil, fmt.Errorf("无法解析大整数: %v", raw)
}

func hexStringToBytes(s string) ([]byte, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	s = strings.TrimPrefix(s, "0X")
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("hex 长度必须为偶数: %s", s)
	}
	return hex.DecodeString(s)
}
