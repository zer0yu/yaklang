package java2ssa

import (
	javaparser "github.com/yaklang/yaklang/common/yak/java/parser"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

func (y *builder) VisitArrayInitializer(raw javaparser.IArrayInitializerContext) ssa.Value {
	if y == nil || raw == nil {
		return nil
	}
	i, _ := raw.(*javaparser.ArrayInitializerContext)
	if i == nil {
		return nil
	}

	allVariableInitializer := i.AllVariableInitializer()
	if len(allVariableInitializer) == 0 {
		return y.EmitMakeBuildWithType(
			ssa.NewSliceType(ssa.BasicTypes[ssa.AnyTypeKind]),
			y.EmitConstInst(0), y.EmitConstInst(0),
		)
	}
	obj := y.InterfaceAddFieldBuild(len(allVariableInitializer),
		func(i int) ssa.Value { return y.EmitConstInst(i) },
		func(i int) ssa.Value {
			return y.VisitVariableInitializer(allVariableInitializer[i])
		})
	obj.GetType().(*ssa.ObjectType).Kind = ssa.SliceTypeKind
	return obj

}
