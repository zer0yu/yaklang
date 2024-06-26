package ssa

import (
	"fmt"

	"github.com/yaklang/yaklang/common/utils/omap"
)

func (p *Package) NewFunction(name string) *Function {
	return p.NewFunctionWithParent(name, nil)
}
func (p *Package) NewFunctionWithParent(name string, parent *Function) *Function {
	index := len(p.Funcs)
	if index == 0 {
		name = "main"
	}
	if name == "" {
		if parent != nil {
			name = fmt.Sprintf("%s$%d", parent.GetName(), index)
		} else {
			name = fmt.Sprintf("AnonymousFunc-%d", index)
		}
	}
	f := &Function{
		anValue:             NewValue(),
		Package:             p,
		Param:               make([]*Parameter, 0),
		paramMap:            make(map[Value]int),
		hasEllipsis:         false,
		Blocks:              make([]*BasicBlock, 0),
		EnterBlock:          nil,
		ExitBlock:           nil,
		ChildFuncs:          make([]*Function, 0),
		parent:              nil,
		FreeValues:          make(map[string]*Parameter),
		SideEffects:         make([]*FunctionSideEffect, 0),
		cacheExternInstance: make(map[string]Value),
		externType:          make(map[string]Type),
		builder:             nil,
		referenceFiles:      omap.NewOrderedMap(map[string]string{}),
	}
	f.SetName(name)
	if parent != nil {
		parent.addAnonymous(f)
		// Pos: parent.CurrentPos,
		f.SetRange(parent.builder.CurrentRange)
	} else {
		p.Funcs[name] = f
	}
	f.EnterBlock = f.NewBasicBlock("entry")
	return f
}

func (f *Function) GetProgram() *Program {
	return f.Package.Prog
}

func (f *Function) GetFunc() *Function {
	return f
}

func (f *Function) GetReferenceFiles() []string {
	return f.referenceFiles.Keys()
}

func (f *Function) addAnonymous(anon *Function) {
	f.ChildFuncs = append(f.ChildFuncs, anon)
	anon.parent = f
}

func (f *FunctionBuilder) NewParam(name string) *Parameter {
	p := NewParam(name, false, f)
	f.appendParam(p)
	return p
}

func (f *FunctionBuilder) appendParam(p *Parameter) {
	f.Param = append(f.Param, p)
	f.paramMap[p] = len(f.Param) - 1
	p.FormalParameterIndex = len(f.Param) - 1
	p.IsFreeValue = false
	variable := f.CreateVariable(p.GetName())
	f.AssignVariable(variable, p)
}

func (f *Function) ReturnValue() []Value {
	ret := f.ExitBlock.LastInst().(*Return)
	return ret.Results
}

func (f *Function) IsMain() bool {
	return f.GetName() == "main"
}

func (f *Function) GetParent() *Function {
	return f.parent
}

// just create a function define, only function parameter type \ return type \ ellipsis
func NewFunctionWithType(name string, typ *FunctionType) *Function {
	f := &Function{
		anValue:        NewValue(),
		referenceFiles: omap.NewOrderedMap(map[string]string{}),
	}
	f.SetType(typ)
	f.SetName(name)
	return f
}
