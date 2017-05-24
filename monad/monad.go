package monad

// the methods of the Monad interface carry a postfix underscore to signify
// that they mention the generic type Monad where we actually mean the concrete
// instance type. by conventions, they are expected to be wrappers around a
// type-specific name of the plain name, e.g.:
//
//     (IO) Bind(func(interface{}) IO) IO
//
// this idiom allows the direct use of the type-specific methods in
// appropriate contexts, possibly avoiding unnecessary type assertions.
//
// so for a Monad instance T, we conceptually expect:
//
//     Bind(func(interface{}) T) T
//     Then(T) T
//     ThenReturn(interface{}) T
//
// then, the interface methods are implemented by generic boilerplate:
//
//     func (a T) Then_(b_ Monad) Monad {
//         return a.Then(b_.(T))
//     }
//
//     func (a T) ThenReturn_(x interface{}) Monad {
//         return a.ThenReturn(x)
//     }
//
//     func (a T) Bind_(f_ func(interface{}) Monad) Monad {
//         f := func(x interface{}) T {return f_(x).(T)}
//         return a.Bind(f)
//     }
//
// conceptually, we would also expect a type-specific Return function which
// cannot be represented by an interface method due to not taking a monad
// argument. such a method could be provided at a package level.
//
//     Return(interface{}) T
//
// together, these methods should satisfy the monad laws:
//
//  - left identity:    Return(x).Bind(f) ~ f(x)
//  - right identity:   a.Bind(Return) ~ a
//  - associativity:    a.Bind(f).Bind(g) ~ a.Bind(func(x) {return f(x).Bind(g)})
//
//  - def. Then:        a.Then(b) ~ a.Bind({return b})
//  - def. ThenReturn:  a.ThenReturn(x) ~ a.Then(Return(x))
//
type Monad interface {
	Bind_(func(interface{}) Monad) Monad	// (>>=)
	Then_(Monad) Monad						// (>>)
	ThenReturn_(interface{}) Monad			// \a x -> a >> return x
}
