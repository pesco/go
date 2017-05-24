package monad

import "fmt"


func print(s string) IO {
	return IO(func() interface{} {
		fmt.Print(s)
		return s
	})
}

var hallo IO = print("hallo ")
var welt  IO = print("welt\n")

func ExampleThen() {
	var m IO = hallo.Then(welt)
	m()
	// Output: hallo welt
}

func ExampleThenReturn() {
	var m IO = hallo.ThenReturn("wurst")
	var x interface{} = m()
	fmt.Println(x)
	// Output: hallo wurst
}

func ExampleBind1() {
	var print_ = func(x interface{}) IO {
		return print(x.(string))
	}

	var m IO = hallo.Bind(print_).Then(welt)
	m()
	// Output: hallo hallo welt
}

func ExampleBind2() {
	var m IO = hallo.Bind(func(x interface{}) IO {
		return print(x.(string)).Then(print("welt "))
	}).Then(print("yeah\n"))
	m()
	// Output: hallo hallo welt yeah
}
