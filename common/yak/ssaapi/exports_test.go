package ssaapi

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/yaklang/yaklang/common/log"
	"reflect"
	"strings"
	"testing"
)

func TestYakChanExplor_Phi_For_Negative(t *testing.T) {
	prog := Parse(`
originValue = 1
var f = outter()
for i := 3; i < f; i++ {
	var d = originValue + i
	d += f
}
g = d 
`)
	// g not phi
	prog.Ref("g").ForEach(func(value *Value) {
		log.Infof("g value: %v", value.String()) // phi? why
		// g value: phi(d)[d,add(add(1, phi(i-2)[3,add(i-2, 1)]), outter())]

		defs := value.GetTopDefs()
		spew.Dump(defs)
	})
}

func TestYakChanExplor_Phi_For(t *testing.T) {
	prog := Parse(`
originValue = 1
var f = outter()
var d = 2
for i := 3; i < f; i++ {
	d = originValue + i
	d += f
}
g = d // g deps-> 1 / 2 / 3
`)
	c1 := false
	c2 := false
	c3 := false
	prog.Ref("g").ForEach(func(value *Value) {
		defs := value.GetTopDefs()
		for _, i := range defs {
			if i.GetConstValue() == 1 {
				c1 = true
			}
			if i.GetConstValue() == 2 {
				c2 = true
			}
			if i.GetConstValue() == 3 {
				c3 = true
			}
		}
	})

	if !c1 {
		t.Error("c1 check failed")
	}

	if !c2 {
		t.Error("c2 check failed")
	}

	if !c3 {
		t.Error("c3 check failed")
	}
}

func TestYakChanExplor_4(t *testing.T) {
	prog := Parse(`
originValue = 1
var f = outter()
a = e => {
	return e
}
if (f) {
	d = 3
} else {
	d = a(originValue)
}
g = d
`)
	lenCheck := false
	valCheck := false
	valCheck_eq3 := false
	prog.Ref("g").ForEach(func(value *Value) {
		defs := value.GetTopDefs()
		if len(defs) == 2 {
			lenCheck = true
		}
		if len(defs) > 0 {
			for _, def := range defs {
				log.Infof("found def: %v", def.String())
				if def.GetConstValue() == 1 {
					valCheck = true
				}
				if def.GetConstValue() == 3 {
					valCheck_eq3 = true
				}
			}
		}
	})
	if !lenCheck {
		t.Error("len check failed")
	}
	if !valCheck {
		t.Error("val check failed")
	}
	if !valCheck_eq3 {
		t.Error("val eq 3 check failed")
	}
}

func TestYakChanExplor_3(t *testing.T) {
	prog := Parse(`
originValue = 1
var f = outter()
a = e => {
	return e
}
if (f) {
	d = 3
} else {
	d = a(originValue)
}
g = d
`)
	valCheck := false
	valCheck_eq3 := false
	var topDefsCount = 0
	prog.Ref("d").ForEach(func(value *Value) {
		defs := value.GetTopDefs()
		spew.Dump(len(defs))
		topDefsCount += len(defs)
		if len(defs) > 0 {
			for _, def := range defs {
				log.Infof("found def: %v", def.String())
				if def.GetConstValue() == 1 {
					valCheck = true
				}
				if def.GetConstValue() == 3 {
					valCheck_eq3 = true
				}
			}
		}
	})
	if topDefsCount != 4 {
		t.Error("len check failed")
	}
	if !valCheck {
		t.Error("val check failed")
	}
	if !valCheck_eq3 {
		t.Error("val eq 3 check failed")
	}
}

func TestYakChanExplor_2(t *testing.T) {
	prog := Parse(`
originValue = 1
a = e => {
	return e
}
d = a(originValue)
`)
	lenCheck := false
	valCheck := false
	prog.Ref("d").ForEach(func(value *Value) {
		defs := value.GetTopDefs()
		if len(defs) == 1 {
			lenCheck = true
		}
		if defs[0].GetConstValue() == 1 {
			valCheck = true
		}
	})
	if !lenCheck {
		t.Error("len check failed")
	}
	if !valCheck {
		t.Error("val check failed")
	}
}

func TestYakChanExplore(t *testing.T) {
	prog := Parse(`
a = () => {
	var c = 1
	return c
}

d = a()
`)
	lenCheck := false
	valCheck := false
	prog.Ref("d").ForEach(func(value *Value) {
		defs := value.GetTopDefs()
		if len(defs) == 1 {
			lenCheck = true
		}
		if defs[0].GetConstValue() == 1 {
			valCheck = true
		}
	})
	if !lenCheck {
		t.Error("len check failed")
	}
	if !valCheck {
		t.Error("val check failed")
	}
}

func TestA(t *testing.T) {

	prog := Parse(
		`
() => {
	window.location.href = "11"
}
a = 1 
if (a > 8){
	setTimeout(() => {
		window.location.href = "22"
	})
}
if (checkFunc(a)) {
	setTimeout(() => {
		window.location.href = "33"
	})
}
setTimeout(() => {
	window.location.href = "44"
})

a = {}
a["b"] = window.location
b = window.location
a.b.href = "5555"
window.location.href = "6666"
b.href = "7777"

a["c"] = window 
a.c.location.href = "8888"
a["d"] = a.c
a.d.location.href = "9999"
a["e"] = a.c.location
a.e.href = "1010"


var b = ()=>{return window.location.hostname + "/app/"}()
window.location.href = b + "/login.html?ts=";
window.location.href = "www"
	`,
		WithLanguage(JS),
	)

	// prog.Show()
	// the `Ref` just a filter
	// window := prog.Ref("window")
	// window.Show()
	// fmt.Println("windows : ")
	// window.ForEach(func(v *Value) {
	// 	v.ShowUseDefChain()
	// })

	win := prog.Ref("window").Ref("location").Ref("href")
	// win.Show()
	// Values: 1
	//       0: Field: window.location.href

	result := make([]string, 0)

	// win.Get(0) // get windows.location.href
	checkValueReachable := func(v *Value) bool {
		reach := v.IsReachable()
		if reach == -1 {
			return false
		}
		if reach == 0 {
			fmt.Printf("in condition %s, this value %s can reachable\n", v.GetReachable(), v)
		}
		return true
	}

	checkFunctionReachable := func(fun *Value) bool {
		if !fun.HasUsers() {
			return false
		}
		// fun.ShowUseDefChain()
		ret := false
		fun.GetUsers().ForEach(func(v *Value) {
			// v.ShowUseDefChain()
			if !checkValueReachable(v) {
				return
			}
			if v.IsCall() {
				callee := v.GetOperand(0)
				if callee == fun {
					ret = true
				}
				if callee.String() == "setTimeout" {
					ret = true
				}
			}
			// fmt.Println(v)
		})
		return ret
	}

	win.Show()
	win.ShowWithSource()
	win.ForEach(func(window *Value) {
		window.ShowWithSource()
		if !window.InMainFunction() {
			fun := window.GetFunction()
			if !checkFunctionReachable(fun) {
				fmt.Println("this value in unreachable sub-function,skip")
				return
			}
		} else {
			if !checkValueReachable(window) {
				fmt.Println("this value is unreachable,skip")
				return
			}
		}

		// show use-def-chain
		// window.ShowUseDefChain()
		// use-def:  |Type   |index  |Opcode |Value
		// 			Operand 0       Field   window.location
		// 			Operand 1       Const   href
		// 			Self            Field   window.location.href
		// 			User    0       Update  update(window.location.href, "6666")
		// 			User    1       Update  update(window.location.href, "7777")
		// 			User    2       Update  update(window.location.href, add(yak-main$5(, binding(window)), "/login.html?ts="))
		// 			User    3       Update  update(window.location.href, "www")

		// this `GetOperands` return Values, use foreach
		// window.GetOperands().ForEach(func(v *Value) {
		// })
		// window.GetOperand(0) // get href
		// window.GetUsers()    // get all users, return a Values
		// window.GetUser(0)    // get update(window.location.href, add(yak-main$1(), "/login.html?ts="))

		// get this function :
		window.GetUsers().ForEach(func(v *Value) {
			// v.ShowUseDefChain()
			if !v.IsUpdate() {
				return
			}
			// check this value reachable

			target := v.GetOperand(1)
			// target.ShowUseDefChain()
			if target.IsBinOp() {
				// target.ShowUseDefChain()
				call := target.GetOperand(0)
				// call.ShowUseDefChain()
				fun := call.GetOperand(0)
				// fun.ShowUseDefChain()
				_ = fun
				if fun.IsFunction() {
					ret := fun.GetReturn()
					// ret.Show()
					ret.ForEach(func(v *Value) {
						// v.ShowUseDefChain()
						v1 := v.GetOperand(0)
						// v1.ShowUseDefChain()
						str := strings.Replace(target.String(), target.GetOperand(0).String(), v1.String(), -1)
						// fmt.Println("windows.location.href set by:", str)
						result = append(result, str)
					})
				}

				// how do next ??
				// fun.GetReturn()

			}
			if target.IsConstInst() {
				// fmt.Println("windows.location.href set by: ", target)
				result = append(result, target.String())
			}
		})
	})

	for _, res := range result {
		fmt.Println("window.location.href = ", res)
	}
}

func TestB(t *testing.T) {
	prog := Parse(`
	$(document).ready(function(){
		$("button").click(function(){
		  $.get("/example/jquery/demo_test.asp",function(data,status){
			alert("数据：" + data + "\n状态：" + status);
		  });
		});
	  });
	`, WithLanguage(JS))

	values := prog.Ref("$")
	values.Show()
	u := values.GetUsers()
	u.Show()
	u2 := u.Filter(func(v *Value) bool { return v.IsCall() })
	fmt.Println(reflect.TypeOf(u2))
}
