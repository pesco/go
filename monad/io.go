package monad


type IO func() interface{}

func (a IO) Then(b IO) IO {
	return func() interface{} {
		a(); return b()
	}
}

func (a IO) ThenReturn(x interface{}) IO {
	return func() interface{} {
		a(); return x
	}
}
func (a IO) Bind(f func(interface{}) IO) IO {
	return func() interface{} {
		x := a()
		b := f(x)
		return b()
	}
}

func (a IO) Then_(b_ Monad) Monad {
	return a.Then(b_.(IO))
}

func (a IO) ThenReturn_(x interface{}) Monad {
	return a.ThenReturn(x)
}

func (a IO) Bind_(f_ func(interface{}) Monad) Monad {
	f := func(x interface{}) IO {return f_(x).(IO)}
	return a.Bind(f)
}
