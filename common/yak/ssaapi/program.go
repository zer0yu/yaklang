package ssaapi

import (
	"github.com/samber/lo"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

type Program struct {
	Program *ssa.Program
}

func NewProgram(prog *ssa.Program) *Program {
	return &Program{
		Program: prog,
	}
}

func (p *Program) IsNil() bool {
	return utils.IsNil(p) || utils.IsNil(p.Program)
}

func (p *Program) GetErrors() ssa.SSAErrors {
	return p.Program.GetErrors()
}

func (p *Program) GetValueById(id int) *Value {
	return NewValue(p.Program.GetInstructionById(id).(ssa.InstructionNode))
}

func (p *Program) Ref(name string) Values {
	return lo.FilterMap(
		p.Program.GetInstructionsByName(name),
		func(i ssa.Instruction, _ int) (*Value, bool) {
			if v, ok := i.(ssa.InstructionNode); ok {
				return NewValue(v), true
			} else {
				return nil, false
			}
		},
	)
}

func (p *Program) GetAllSymbols() map[string]Values {
	ret := make(map[string]Values, 0)
	p.Program.NameToInstructions.ForEach(func(name string, insts []ssa.Instruction) bool {
		ret[name] = lo.FilterMap(
			insts,
			func(i ssa.Instruction, _ int) (*Value, bool) {
				if v, ok := i.(ssa.InstructionNode); ok {
					return NewValue(v), true
				} else {
					return nil, false
				}
			},
		)
		return true
	})
	return ret
}

func getValuesWithUpdateSingle(v *Value) Values {
	ret := make(Values, 0)
	ret = append(ret, v)
	// check if: a[0] = value.Name; also append a[0]
	v.GetUsers().ForEach(func(user *Value) {
		if user.IsUpdate() && v.Compare(user.GetOperand(1)) {
			ret = append(ret, getValuesWithUpdateSingle(user.GetOperand(0))...)
		}
	})
	return ret
}

func getValuesWithUpdate(vs Values) Values {
	ret := make(Values, 0, len(vs))
	// copy(ret, vs)

	vs.ForEach(func(v *Value) {
		ret = append(ret, getValuesWithUpdateSingle(v)...)
	})

	return ret
}
