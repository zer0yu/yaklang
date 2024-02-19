package test

import (
	"testing"

	"github.com/yaklang/yaklang/common/yak/ssa"
)

func TestYaklangBasic_Foreach(t *testing.T) {
	t.Run("for each with chan", func(t *testing.T) {
		CheckType(t, `
		ch = make(chan int)

		for i in ch { 
			_ = i 
			target = i
		}
		`,
			ssa.NumberTypeKind)
	})

	t.Run("for each with list", func(t *testing.T) {
		CheckType(t, `
		ch = make([]int, 3)

		for i in ch { 
			_ = i 
			target = i
		}
		`,
			ssa.NumberTypeKind)
	})
}

func TestYaklangType_Loop(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		CheckType(t, `
		num = make([]int, 3)
		for i=0; i < 3; i++ {
			target = num[i]
		}
		`,
			ssa.NumberTypeKind)
	})
}

func TestYaklangType_BuildInMethod(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		CheckType(t, `
		s = make([]int, 3)
		target = s.Len()
		`,
			ssa.NumberTypeKind)
	})

	t.Run("loop", func(t *testing.T) {
		CheckType(t, `
		a = make([]int, 3)
		for i=0; i<3; i++ {
			target = a.Len()
		}`,
			ssa.NumberTypeKind)
	})
}

func TestYaklangType_FreeValue(t *testing.T) {
	t.Run("in closure", func(t *testing.T) {
		CheckType(t, `
		m = make(map[string]string)
		() => {
			target = m.Len()
		}
		`, ssa.NumberTypeKind)
	})

	t.Run("in sub closure", func(t *testing.T) {
		CheckType(t, `
		m = make(map[string]string)
		() => {
			() => {
				target = m.Len()
			}
		}
		`, ssa.NumberTypeKind)
	})

	t.Run("in loop", func(t *testing.T) {
		CheckType(t, `
		m = make(map[string]string)
		() => {
			for i=0; i<10; i++ {
				target = m.Len()
			}
		}
		`, ssa.NumberTypeKind)
	})
}
