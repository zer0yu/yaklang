package ssa

import (
	"fmt"
	"strings"

	"github.com/yaklang/yaklang/common/log"
)

// value
func SetMemberCall(obj, key, member Value) {
	obj.AddMember(key, member)
	member.SetObject(obj)
	member.SetKey(key)
}

func ReplaceMemberCall(v, to Value) map[string]Value {
	ret := make(map[string]Value)
	builder := v.GetFunc().builder
	recoverScope := builder.SetCurrent(v)
	defer recoverScope()
	createPhi := generalPhi(builder, nil)

	// replace object member-call
	if v.IsObject() {
		for _, member := range v.GetAllMember() {
			// replace this member object to to
			key := member.GetKey()
			v.DeleteMember(key)

			// re-set type
			name, typ := checkCanMemberCall(to, key)
			origin := builder.getOriginMember(name, typ, to, key)

			if member.GetOpcode() != OpUndefined {
				member.SetName(name)
				member.SetType(typ)
				SetMemberCall(to, key, member)
				ret[name] = createPhi(name, []Value{origin, member})
				continue
			}

			ReplaceAllValue(member, origin)
			DeleteInst(member)

			origin.GetUsers().RunOnCall(func(c *Call) {
				c.handleCalleeFunction()
				c.handlerReturnType()
			})

			ret[name] = origin
		}
	}

	if v.IsMember() {
		obj := v.GetObject()
		// obj.AddMember(v.GetKey(), to)
		SetMemberCall(obj, v.GetKey(), v)
	}
	return ret
}

func NewMake(parentI Value, typ Type, low, high, step, Len, Cap Value) *Make {
	i := &Make{
		anValue: NewValue(),
		low:     low,
		high:    high,
		step:    step,
		parentI: parentI,
		Len:     Len,
		Cap:     Cap,
	}
	i.SetType(typ)
	return i
}

func (b *FunctionBuilder) CreateInterfaceWithSlice(vs []Value) *Make {
	return b.InterfaceAddFieldBuild(len(vs),
		func(i int) Value { return NewConst(i) },
		func(i int) Value { return vs[i] },
	)
}
func (b *FunctionBuilder) CreateInterfaceWithMap(keys []Value, vs []Value) *Make {
	return b.InterfaceAddFieldBuild(len(vs),
		func(i int) Value { return keys[i] },
		func(i int) Value { return vs[i] },
	)
}

func (b *FunctionBuilder) InterfaceAddFieldBuild(size int, keys func(int) Value, value func(int) Value) *Make {
	lValueLen := NewConst(size)
	itf := b.EmitMakeWithoutType(lValueLen, lValueLen)
	if b.MarkedVariable != nil {
		itf.SetName(b.MarkedVariable.GetName())
		b.MarkedThisObject = itf

		defer func() {
			b.MarkedThisObject = nil
		}()
	}
	ityp := NewObjectType()
	itf.SetType(ityp)
	for i := 0; i < size; i++ {
		key := keys(i)
		value := value(i)
		v := b.CreateMemberCallVariable(itf, key)
		b.AssignVariable(v, value)
	}
	ityp.Finish()
	// ityp.Len = len(vs)
	return itf
}

func CombineMemberCallVariableName(caller, callee Value) (string, bool) {
	name, _ := checkCanMemberCall(caller, callee)
	return name, name != ""
}

func checkCanMemberCall(value, key Value) (string, Type) {
	type MemberCallKind int
	const (
		None MemberCallKind = iota
		StringKind
		NumberKind
		DynamicKind
	)

	var name string
	kind := None
	if constInst, ok := ToConst(key); ok {
		if constInst.IsNumber() {
			name = fmt.Sprintf("#%d[%d]", value.GetId(), constInst.Number())
			kind = NumberKind
		}
		if constInst.IsString() {
			name = fmt.Sprintf("#%d.%s", value.GetId(), constInst.VarString())
			kind = StringKind
		}
	} else {
		name = fmt.Sprintf("#%d.#%d", value.GetId(), key.GetId())
		kind = DynamicKind
	}

	if kind == DynamicKind {
		//TODO: check type
		switch value.GetType().GetTypeKind() {
		case SliceTypeKind, MapTypeKind:
			typ, _ := ToObjectType(value.GetType())
			return name, typ.FieldType
		case BytesTypeKind, StringTypeKind:
			return name, BasicTypes[NumberTypeKind]
		default:
			return name, BasicTypes[AnyTypeKind]
		}
	}

	// check is method
	if ret := GetMethod(value.GetType(), key.String()); ret != nil {
		return name, ret
	}

	switch value.GetType().GetTypeKind() {
	case ObjectTypeKind:
		typ, ok := ToObjectType(value.GetType())
		if !ok {
			log.Errorf("checkCanMemberCall: %v is structTypeKind but is not a ObjectType", value.GetType())
			break
		}
		if fieldTyp := typ.GetField(key); fieldTyp != nil {
			return name, fieldTyp
		}
		// not this field
		return name, BasicTypes[AnyTypeKind]
	case StructTypeKind: // string
		typ, ok := ToObjectType(value.GetType())
		if !ok {
			log.Errorf("checkCanMemberCall: %v is structTypeKind but is not a ObjectType", value.GetType())
			break
		}
		if TypeCompare(BasicTypes[StringTypeKind], key.GetType()) {
			if fieldTyp := typ.GetField(key); fieldTyp != nil {
				return name, fieldTyp
			} else {
				// not this field
			}
		} else {
			// type check error
		}
	case TupleTypeKind:
		typ, ok := ToObjectType(value.GetType())
		if !ok {
			log.Errorf("checkCanMemberCall: %v is TupleTypeKind but is not a ObjectType", value.GetType())
			break
		}
		if TypeCompare(BasicTypes[NumberTypeKind], key.GetType()) {
			if fieldTyp := typ.GetField(key); fieldTyp != nil {
				return name, fieldTyp
			}
		}
	case MapTypeKind: // string / number
		typ, ok := ToObjectType(value.GetType())
		if !ok {
			log.Errorf("checkCanMemberCall: %v is MapTypeKind but is not a ObjectType", value.GetType())
			break
		}
		if TypeCompare(typ.KeyTyp, key.GetType()) {
			if typ.FieldType.GetTypeKind() == AnyTypeKind {
				if fieldTyp := typ.GetField(key); fieldTyp != nil {
					return name, fieldTyp
				}
			}
			return name, typ.FieldType
		} else {
			// type check error
		}
	case SliceTypeKind:
		typ, ok := ToObjectType(value.GetType())
		if !ok {
			log.Errorf("checkCanMemberCall: %v is SliceTypeKind but is not a ObjectType", value.GetType())
			break
		}
		if TypeCompare(BasicTypes[NumberTypeKind], key.GetType()) {
			if typ.FieldType.GetTypeKind() == AnyTypeKind {
				if fieldTyp := typ.GetField(key); fieldTyp != nil {
					return name, fieldTyp
				}
			}
			return name, typ.FieldType
		} else {
			// type check error
		}
	case BytesTypeKind, StringTypeKind: // number
		if TypeCompare(BasicTypes[NumberTypeKind], key.GetType()) {
			return name, BasicTypes[NumberTypeKind]
		} else {
			// type check error
		}
	case AnyTypeKind:
		return name, BasicTypes[AnyTypeKind]
	default:
	}
	return name, nil
}

func (b *FunctionBuilder) getExternLibMemberCall(value, key Value) string {
	return fmt.Sprintf("%s.%s", value.GetName(), key.String())
}

func (b *FunctionBuilder) ReadMemberCallVariable(value, key Value) Value {
	if extern, ok := ToExternLib(value); ok {
		// write to extern Lib
		name := b.getExternLibMemberCall(value, key)
		// if ret := b.PeekValue(name); ret != nil {
		// 	return ret
		// }
		if ret := ReadVariableFromScope(b.CurrentBlock.ScopeTable, name); ret != nil {
			return ret.Value
		}

		if ret := extern.BuildField(key.String()); ret != nil {
			return ret
		}

		// handler
		// want := b.TryGetSimilarityKey(pa.GetName(), ci.String())
		want := b.TryGetSimilarityKey(extern.GetName(), key.String())
		b.NewErrorWithPos(Error, SSATAG, b.CurrentRange, ExternFieldError("Lib", extern.GetName(), key.String(), want))
		p := NewParam(name, false, b)
		p.SetExtern(true)
		return p
	}

	if para, ok := ToParameter(value); ok {
		if para.IsFreeValue {
			b.appendParam(para)
			delete(b.FreeValues, para.GetName())
		}

		name, typ := checkCanMemberCall(para, key)
		new := b.NewParam(name)
		new.SetType(typ)
		new.IsMemberCall = true
		new.MemberCallObjectIndex = para.FormalParameterIndex
		new.MemberCallKey = key

		SetMemberCall(para, key, new)

		setMemberVerboseName(new)
		return new
	}

	return b.getFieldValue(value, key)
}

func (b *FunctionBuilder) CreateMemberCallVariable(value, key Value) *Variable {
	if _, ok := ToExternLib(value); ok {
		name := b.getExternLibMemberCall(value, key)
		return b.CreateVariable(name)
	}

	if para, ok := ToParameter(value); ok {
		if para.IsFreeValue {
			b.appendParam(para)
			delete(b.FreeValues, para.GetName())
		}
		name, _ := checkCanMemberCall(para, key)
		ret := b.CreateVariable(name)
		ret.object = para
		ret.key = key
		return ret
	}

	name := b.getFieldName(value, key)
	// log.Infof("CreateMemberCallVariable: %v, %v", retValue.GetName(), key)
	ret := b.CreateVariable(name)
	ret.SetMemberCall(value, key)
	return ret
}

func (b *FunctionBuilder) getFieldName(value, key Value) string {
	name, typ := checkCanMemberCall(value, key)
	b.getOriginMember(name, typ, value, key) // create undefine member
	return name
}

func (b *FunctionBuilder) getFieldValue(value, key Value) Value {
	name, typ := checkCanMemberCall(value, key)
	if ret := b.PeekValueInThisFunction(name); ret != nil {
		return ret
	}
	return b.getOriginMember(name, typ, value, key)
}

func (b *FunctionBuilder) getOriginMember(name string, typ Type, value, key Value) Value {
	recoverScope := b.SetCurrent(value)
	origin := b.ReadValueInThisFunction(name)
	recoverScope()
	if undefine, ok := ToUndefined(origin); ok {
		undefine.SetRange(b.CurrentRange)
		// undefine.SetName(b.setMember(key))
		if typ != nil {
			undefine.Kind = UndefinedMemberValid
			undefine.SetType(typ)
		} else {
			undefine.Kind = UndefinedMemberInValid
		}
		SetMemberCall(value, key, undefine)
	}
	setMemberVerboseName(origin)
	return origin
}

func getMemberVerboseName(obj, key Value) string {
	return fmt.Sprintf("%s.%s", obj.GetVerboseName(), GetKeyString(key))
}
func setMemberVerboseName(member Value) {
	if !member.IsMember() {
		return
	}
	text := getMemberVerboseName(member.GetObject(), member.GetKey())
	member.SetVerboseName(text)
}

func GetKeyString(key Value) string {
	text := ""
	if ci, ok := ToConst(key); ok {
		text = ci.String()
	}
	if text == "" {
		text = key.GetVerboseName()
	}

	if text == "" {
		if r := key.GetRange(); r != nil && r.SourceCode != nil {
			list := strings.Split(*r.SourceCode, ".")
			text = list[len(list)-1]
		}
	}
	return text
}
