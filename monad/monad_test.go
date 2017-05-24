package monad

import "fmt"


var hallo_ Monad = print("hallo ")
var welt_  Monad = print("welt\n")

func ExampleThen_() {
	var m Monad = hallo.Then_(welt)
	m.(IO)()
	// Output: hallo welt
}

func ExampleThenReturn_() {
	var m Monad = hallo.ThenReturn_("wurst")
	var x interface{} = m.(IO)()
	fmt.Println(x)
	// Output: hallo wurst
}

func ExampleBind_1() {
	var print_ = func(x interface{}) Monad {
		return print(x.(string))
	}

	var m Monad = hallo.Bind_(print_).Then_(welt)
	m.(IO)()
	// Output: hallo hallo welt
}

func ExampleBind_2() {
	var m Monad = hallo.Bind_(func(x interface{}) Monad {
		return print(x.(string)).Then_(print("welt "))
	}).Then_(print("yeah\n"))
	m.(IO)()
	// Output: hallo hallo welt yeah
}
