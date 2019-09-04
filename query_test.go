package postgres

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryStr(t *testing.T) {
	assert.Equal(t, `WHERE foo = $1 and bar = $2`, Query("foo = $1 and bar = $2").queryStr())
}

func TestOrderStr(t *testing.T) {
	assert.Equal(t, `ORDER BY "foo" ASC, "bar" DESC`, Query("").Asc("foo").Desc("bar").orderStr())
}

type TestValidate_Struct struct {
	Foo string
	Bar string
}

func TestValidate(t *testing.T) {
	f, _ := newMetaStruct(&TestValidate_Struct{})

	assert.Error(t, Query("Foo = $1 and Bar = $2", 1, 2).Whitelist("Foo").validate(f))
	assert.NoError(t, Query("Foo = $1 and Bar = $2", 1, 2).Whitelist("Foo", "Bar").validate(f))
	assert.Error(t, Query("Foo = $1 and Bar = $2", 1, 2).Whitelist().validate(f))
	assert.NoError(t, Query("Foo = $1 and Bar = $2", 1, 2).validate(f))
	assert.Error(t, Query("").validate(f))
	assert.Error(t, Query("Foo = $1", 1, 2).validate(f))

	assert.Error(t, UntrustedQuery("Foo = $1 and Bar = $2", 1, 2).Whitelist("Foo").validate(f))
	assert.NoError(t, UntrustedQuery("Foo = $1 and Bar = $2", 1, 2).Whitelist("Foo", "Bar").validate(f))
	assert.Error(t, UntrustedQuery("Foo = $1 and Bar = $2", 1, 2).Whitelist().validate(f))
	assert.Error(t, UntrustedQuery("Foo = $1 and Bar = $2", 1, 2).validate(f))
	assert.Error(t, UntrustedQuery("").validate(f))
	assert.Error(t, UntrustedQuery("Foo = $1", 1, 2).validate(f))
}

type TestVerifyWhitelist_Struct struct {
	Foo string
	Bar string
}

func TestVerifyWhitelist(t *testing.T) {
	f := mustNewFields(&TestVerifyWhitelist_Struct{}, false)

	assert.False(t, Query("Foo = $1 or Bar = $2").Whitelist("Foo", "Bar", "abc").validateWhitelist(f))
	assert.True(t, Query("Foo = $1 or Bar = $2").Whitelist("Foo", "Bar").validateWhitelist(f))
	assert.False(t, Query("Foo = $1 or Bar = $2").Whitelist("Bar").validateWhitelist(f))
	assert.False(t, Query("Foo = $1 or Bar = $2").Whitelist().validateWhitelist(f))

	assert.False(t, UntrustedQuery("Foo = $1 or Bar = $2").Whitelist("Foo", "Bar", "abc").validateWhitelist(f))
	assert.True(t, UntrustedQuery("Foo = $1 or Bar = $2").Whitelist("Foo", "Bar").validateWhitelist(f))
	assert.False(t, UntrustedQuery("Foo = $1 or Bar = $2").Whitelist("Bar").validateWhitelist(f))
	assert.False(t, UntrustedQuery("Foo = $1 or Bar = $2").Whitelist().validateWhitelist(f))
}

func TestVerifyArgsCount(t *testing.T) {
	assert.True(t, Query("Foo = $1 and Bar = $2", 1, 2).validateArgsCount())
	assert.False(t, Query("Foo = $1 and Bar = $2", 1).validateArgsCount())
	assert.False(t, Query("Foo = $1 and Bar = $2").validateArgsCount())
	assert.True(t, Query("Foo = $1 and Bar = $1", 1).validateArgsCount())
	assert.False(t, Query("Foo = $1 and Bar = $3", 1).validateArgsCount())
	assert.False(t, Query("Foo = $1 and Bar = $3", 1, 2).validateArgsCount())
	assert.False(t, Query("Foo = $1 and Bar = $3", 1, 2, 3).validateArgsCount())
	assert.False(t, Query("Foo = $10 and Bar = $11", 1, 2).validateArgsCount())
	assert.False(t, Query("Foo = $10", 1).validateArgsCount())
}

type TestQuoteIdentifiers_Struct struct {
	Col1   string
	Col2   string
	FooBar string
}

func mustQuoteIdentifiers(q *QueryStmt, r fields) *QueryStmt {
	if err := q.quoteIdentifiers(r); err != nil {
		panic(err)
	}
	return q
}

func TestQuoteIdentifiers(t *testing.T) {
	f := mustNewFields(&TestQuoteIdentifiers_Struct{}, false)

	assert.Equal(t, `WHERE "col1" = $1`, mustQuoteIdentifiers(Query("Col1 = $1"), f).queryStr())
	assert.Equal(t, `WHERE "col1" = $1`, mustQuoteIdentifiers(Query("col1 = $1"), f).queryStr())
	assert.Equal(t, `WHERE  "col1"  = $1`, mustQuoteIdentifiers(Query(" Col1  = $1"), f).queryStr())
	assert.Equal(t, `WHERE  "col1"  = $1`, mustQuoteIdentifiers(Query(" col1  = $1"), f).queryStr())
	assert.Equal(t, `WHERE "Col_1" = $1`, mustQuoteIdentifiers(Query("Col_1 = $1"), f).queryStr())
	assert.Equal(t, `WHERE "col_1" = $1`, mustQuoteIdentifiers(Query("col_1 = $1"), f).queryStr())
	assert.Equal(t, `WHERE "foo_bar" = $1`, mustQuoteIdentifiers(Query("FooBar = $1"), f).queryStr())
	assert.Equal(t, `WHERE "Foobar" = $1`, mustQuoteIdentifiers(Query("Foobar = $1"), f).queryStr())
	assert.Equal(t, `WHERE "Foobar" = $1`, mustQuoteIdentifiers(Query(`"Foobar" = $1`), f).queryStr())
	assert.Equal(t, `WHERE "col1" = $1 and "col2" = $2 or "foo_bar" = $3`, mustQuoteIdentifiers(Query(`Col1 = $1 and Col2 = $2 or FooBar = $3`), f).queryStr())
}

func TestIsParenthesesBalanced(t *testing.T) {
	fn := isParenthesesBalanced

	assert.True(t, fn(`()`))
	assert.True(t, fn(`(())`))
	assert.True(t, fn(`()()`))
	assert.True(t, fn(`( () () )`))

	assert.False(t, fn(`(`))
	assert.False(t, fn(`((`))
	assert.False(t, fn(`)`))
	assert.False(t, fn(`))`))
	assert.False(t, fn(`())`))
	assert.False(t, fn(`()(`))
	assert.False(t, fn(`(()`))
	assert.False(t, fn(`)()`))

	assert.False(t, fn(`(((`))
	assert.False(t, fn(`)))`))
	assert.False(t, fn(`(()`))
	assert.False(t, fn(`())`))

	assert.False(t, fn(`)(`))

	assert.False(t, fn(`(((()))`))
	assert.False(t, fn(`(())())`))
}

func TestIsValidUntrustedQuery(t *testing.T) {
	fn := isValidUntrustedQuery

	assert.False(t, fn(``))
	assert.True(t, fn(`(foo = $1 and bar = $2) or (abc = $3 and def = $4) or ghi != $5 or jkl like $6`))
	assert.False(t, fn(`(foo = $1 and bar = $2) or (abc() = $3 and def = $4) or ghi != $5 or jkl like $6`))
	assert.True(t, fn(`foo = $1 or bar = $1`)) // double assign

	assert.False(t, fn(`foo = $0`))
	assert.True(t, fn(`foo = $1`))
	assert.True(t, fn(`foo = $2`))
	assert.True(t, fn(`foo = $999`))
	assert.False(t, fn(`foo = $1000`))

	assert.False(t, fn(`foo = 1`))
	assert.False(t, fn(`foo = '1'`))
	assert.False(t, fn(`foo = "1"`))
	assert.False(t, fn(`foo = abc`))
	assert.False(t, fn(`foo = 'abc'`))
	assert.False(t, fn(`foo = "abc"`))
	assert.False(t, fn(`foo = true`))
	assert.False(t, fn(`foo = TRUE`))
	assert.False(t, fn(`foo = YES`))
	assert.False(t, fn(`foo = yes`))
	assert.False(t, fn(`foo = ON`))
	assert.False(t, fn(`foo = on`))
	assert.False(t, fn(`foo = false`))
	assert.False(t, fn(`foo = FALSE`))
	assert.False(t, fn(`foo = NO`))
	assert.False(t, fn(`foo = no`))
	assert.False(t, fn(`foo = OFF`))
	assert.False(t, fn(`foo = off`))
	assert.False(t, fn(`foo = t`))
	assert.False(t, fn(`foo = f`))
	assert.False(t, fn(`foo = 'yes' :: boolean`))
	assert.False(t, fn(`foo = 0`))
	assert.False(t, fn(`foo = 1999-01-08 04:05:06`))

	assert.False(t, fn(`foo = $`))
	assert.False(t, fn(`foo = *`))
	assert.False(t, fn(`foo = '*'`))
	assert.False(t, fn(`foo = "*"`))
	assert.False(t, fn(`foo = %`))
	assert.False(t, fn(`foo = '%'`))
	assert.False(t, fn(`foo = "%"`))
	assert.False(t, fn(`foo = @1`))
	assert.False(t, fn(`foo = ?`))
	assert.False(t, fn(`foo = ?1`))
	assert.False(t, fn(`? = $1`))
	assert.False(t, fn(` = $1`))
	assert.False(t, fn(`= $1`))
	assert.False(t, fn(`= 1`))
	assert.False(t, fn(`=`))
	assert.False(t, fn(` = `))

	assert.True(t, fn(`(foo = $1)`))
	assert.True(t, fn(`(Foo = $1)`))
	assert.True(t, fn(`(foo = $1) (bar = $2)`))
	assert.False(t, fn(`foo = $1; bar = $2`))
	assert.False(t, fn(`foo = $1, bar = $2`))
	assert.False(t, fn(`(1foo = $1)`))
	assert.False(t, fn(`(_foo = $1)`))
	assert.False(t, fn(`($foo = $1)`))

	assert.True(t, fn(`foo = $1 and bar = $2`))
	assert.True(t, fn(`foo = $1 AND bar = $2`))
	assert.True(t, fn(`foo = $1 or bar = $2`))
	assert.True(t, fn(`foo = $1 OR bar = $2`))
	assert.True(t, fn(`foo = $1 or (bar = $2 and abc = $3)`))

	assert.True(t, fn(`foo < $1`))
	assert.True(t, fn(`foo > $1`))
	assert.True(t, fn(`foo <= $1`))
	assert.True(t, fn(`foo >= $1`))
	assert.True(t, fn(`foo != $1`))
	assert.True(t, fn(`foo  <  $1`))
	assert.True(t, fn(`foo  >  $1`))
	assert.True(t, fn(`foo  <=  $1`))
	assert.True(t, fn(`foo  >=  $1`))
	assert.True(t, fn(`foo  !=  $1`))

	assert.True(t, fn(`foo LIKE $1`))
	assert.True(t, fn(`foo like $1`))
	assert.True(t, fn(`foo ILIKE $1`))
	assert.True(t, fn(`foo ilike $1`))

	// we don't allow regular expressions
	assert.False(t, fn(`foo ~ $1`))
	assert.False(t, fn(`foo ~* $1`))
	assert.False(t, fn(`foo !~ $1`))
	assert.False(t, fn(`foo !~* $1`))

	assert.False(t, fn(`foo IS $1`))
	assert.False(t, fn(`foo is $1`))
	assert.False(t, fn(`foo IS NULL`))
	assert.False(t, fn(`foo is NULL`))

	assert.False(t, fn(`foo <> $1`)) // we don't allow it, use != instead
	assert.False(t, fn(`foo =< $1`))
	assert.False(t, fn(`foo => $1`))
	assert.False(t, fn(`foo =! $1`))
	assert.False(t, fn(`foo ! $1`))
	assert.False(t, fn(`foo == $1`))
	assert.False(t, fn(`foo === $1`))
	assert.False(t, fn(`foo !== $1`))

	assert.True(t, fn(`"foo" = $1`))
	assert.False(t, fn(`'foo' = $1`))
	assert.False(t, fn(`foo = "$1"`))
	assert.False(t, fn(`foo = '$1'`))
	assert.False(t, fn(`'foo = $1'`))
	assert.False(t, fn(`"foo = $1"`))
	assert.False(t, fn(`""foo = $1""`))
	assert.False(t, fn(`""foo"" = $1`))
	assert.False(t, fn(`foo' = $1`))
	assert.False(t, fn(`foo" = $1`))

	// we generally don't allow to specify schemas or table identifiers
	assert.False(t, fn(`public.foo = $1`))
	assert.False(t, fn(`"public"."foo" = $1`))
	assert.False(t, fn(`"public.foo" = $1`))
	assert.False(t, fn(`".foo" = $1`))
	assert.False(t, fn(`.foo = $1`))

	// wrong field names
	assert.True(t, fn(`f_o_o = $1`))
	assert.False(t, fn(`_f_o_o_ = $1`))
	assert.False(t, fn(`1foo = $1`))
	assert.False(t, fn(`_1foo = $1`))
	assert.True(t, fn(`f1_bar = $1`))
	assert.False(t, fn(`f,oo = $1`))
	assert.False(t, fn(`f;oo = $1`))
	assert.False(t, fn(`f$oo = $1`))
	assert.False(t, fn(`f(oo) = $1`))
	assert.False(t, fn(`f!oo = $1`))
	assert.False(t, fn(`f@oo = $1`))
	assert.False(t, fn(`f%oo = $1`))
	assert.False(t, fn(`f^oo = $1`))
	assert.False(t, fn(`f*oo = $1`))
	assert.False(t, fn(`* = $1`))
	assert.False(t, fn(`* = *`))
	assert.False(t, fn(`*`))
	assert.False(t, fn(`%`))
	assert.False(t, fn(`foo`))
	assert.False(t, fn(`foo = `))
	assert.False(t, fn(`_ = $1`))
	assert.False(t, fn(`1 = $1`))
	assert.False(t, fn(`1foo = $1`))
	assert.True(t, fn(`f = $1`))

	// try to escape things
	assert.False(t, fn(`\foo = $1`))
	assert.False(t, fn(`f\oo = $1`))
	assert.False(t, fn(`foo\ = $1`))
	assert.False(t, fn(`foo \= $1`))
	assert.False(t, fn(`foo =\ $1`))
	assert.False(t, fn(`foo = \$1`))
	assert.False(t, fn(`foo = $\1`))
	assert.False(t, fn(`foo = $1\`))
	assert.False(t, fn(`\\foo = $1`))
	assert.False(t, fn(`f\\oo = $1`))
	assert.False(t, fn(`foo\\ = $1`))
	assert.False(t, fn(`foo \\= $1`))
	assert.False(t, fn(`foo =\\ $1`))
	assert.False(t, fn(`foo = \\$1`))
	assert.False(t, fn(`foo = $\\1`))
	assert.False(t, fn(`foo = $1\\`))
	assert.False(t, fn(`\\\foo = $1`))
	assert.False(t, fn(`f\\\oo = $1`))
	assert.False(t, fn(`foo\\\ = $1`))
	assert.False(t, fn(`foo \\\= $1`))
	assert.False(t, fn(`foo =\\\ $1`))
	assert.False(t, fn(`foo = \\\$1`))
	assert.False(t, fn(`foo = $\\\1`))
	assert.False(t, fn(`foo = $1\\\`))
	assert.False(t, fn(`\\\\foo = $1`))
	assert.False(t, fn(`f\\\\oo = $1`))
	assert.False(t, fn(`foo\\\\ = $1`))
	assert.False(t, fn(`foo \\\\= $1`))
	assert.False(t, fn(`foo =\\\\ $1`))
	assert.False(t, fn(`foo = \\\\$1`))
	assert.False(t, fn(`foo = $\\\\1`))
	assert.False(t, fn(`foo = $1\\\\`))

	assert.False(t, fn(`foo = \'$1\'`))
	assert.False(t, fn(`\"foo\" = $1`))
	assert.False(t, fn(`\\"foo\\" = $1`))
	assert.False(t, fn(`\\\"foo\\\" = $1`))
	assert.False(t, fn(`\\\\"foo\\\\" = $1`))

	// quoted identifiers that start with U&
	//  "data" = 'foo'
	assert.False(t, fn(`U&"d\0061t\+000061" = $1`))
	assert.False(t, fn(`U&"d!0061t!+000061" UESCAPE '!' = $1`))

	// "slon" = 'foo'
	assert.False(t, fn(`U&"\0441\043B\043E\043D" = $1`))

	// we don't support functions
	assert.False(t, fn(`foo = upper($1)`))
	assert.False(t, fn(`foo = UPPER($1)`))
	assert.False(t, fn(`foo = UPPER $1`)) // invalid syntax anyway
	assert.False(t, fn(`foo = upper $1`)) // see above
	assert.False(t, fn(`foo = current_database()`))
	assert.False(t, fn(`foo = version()`))
	assert.False(t, fn(`foo = foobar()`))
	assert.False(t, fn(`current_database() = $1`))
	assert.False(t, fn(`current_database(1) = $1`))
	assert.False(t, fn(`current_database("abc") = $1`))
	assert.False(t, fn(`current_database(abc) = $1`))
	assert.False(t, fn(`current_database($1) = $1`))
	assert.True(t, fn(`current_database = $1`))
	assert.False(t, fn(`version() = $1`))
	assert.False(t, fn(`ver sion() = $1`))
	assert.False(t, fn(`ver sion () = $1`))
	assert.False(t, fn(`ver  sion  () = $1`))
	assert.False(t, fn(`version(1) = $1`))
	assert.False(t, fn(`version("abc") = $1`))
	assert.False(t, fn(`version($1) = $1`))
	assert.False(t, fn(`version () = $1`))
	assert.False(t, fn(`version (1) = $1`))
	assert.False(t, fn(`version ("abc") = $1`))
	assert.False(t, fn(`version ($1) = $1`))
	assert.False(t, fn(`version  () = $1`))
	assert.False(t, fn(`version  (1) = $1`))
	assert.False(t, fn(`version  ("abc") = $1`))
	assert.False(t, fn(`version  ($1) = $1`))
	assert.True(t, fn(`version = $1`))
	assert.False(t, fn(`foo     () = $1`)) // postgres executes `func         ()`, lol
	assert.False(t, fn(`foo     (1) = $1`))
	assert.False(t, fn(`(foo ()) = $1`))
	assert.False(t, fn(`(foo () = $1)`))
	assert.False(t, fn(`(foo() = $1)`))

	// no comments are allowed
	assert.False(t, fn(`-- foo = $1`))
	assert.False(t, fn(`foo = $1 --`))
	assert.False(t, fn(`foo -- = $1`))
	assert.False(t, fn(`foo --= $1`))

	// there is only one way to express "not", that's !=
	assert.False(t, fn(`foo = $1 and not bar = $2`))
	assert.False(t, fn(`foo = $1 or not bar = $2`))

	// new line
	assert.True(t, fn("foo = $1\n"))
	assert.True(t, fn("\nfoo = $1\n"))
	assert.False(t, fn("foo =\n$1"))
	assert.False(t, fn("foo\n = $1"))
	assert.False(t, fn("foo = $\n1"))

	// tabs
	assert.True(t, fn("foo = $1\t"))
	assert.True(t, fn("\tfoo = $1\t"))
	assert.False(t, fn("foo =\t$1"))
	assert.False(t, fn("foo\t = $1"))
	assert.False(t, fn("foo = $\t1"))

	// fun with null bytes
	assert.False(t, fn("foo\000 = $1"))
	assert.False(t, fn("foo\x00 = $1"))
	assert.False(t, fn("foo\u0000 = $1"))
	assert.False(t, fn(`foo\000 = $1`))
	assert.False(t, fn(`foo\x00 = $1`))
	assert.False(t, fn(`foo\u0000 = $1`))
	assert.False(t, fn("foo = $\000"))
	assert.False(t, fn("foo = $\x00"))
	assert.False(t, fn("foo = $\u0000"))
	assert.False(t, fn(`foo = $\000`))
	assert.False(t, fn(`foo = $\x00`))
	assert.False(t, fn(`foo = $\u0000`))
	assert.False(t, fn("foo = \000"))
	assert.False(t, fn("foo = \x00"))
	assert.False(t, fn("foo = \u0000"))
	assert.False(t, fn(`foo = \000`))
	assert.False(t, fn(`foo = \x00`))
	assert.False(t, fn(`foo = \u0000`))
	assert.False(t, fn("\000 = $1"))
	assert.False(t, fn("\x00 = $1"))
	assert.False(t, fn("\u0000 = $1"))
	assert.False(t, fn(`\000 = $1`))
	assert.False(t, fn(`\x00 = $1`))
	assert.False(t, fn(`\u0000 = $1`))
	assert.False(t, fn(`"\000" = $1`))
	assert.False(t, fn(`"\x00" = $1`))
	assert.False(t, fn(`"\u0000" = $1`))
	assert.False(t, fn("\000"))
	assert.False(t, fn("\x00"))
	assert.False(t, fn("\u0000"))
	assert.False(t, fn(`\000`))
	assert.False(t, fn(`\x00`))
	assert.False(t, fn(`\u0000`))

	// test some sql injections
	assert.False(t, fn(`foo = $1 or 1 = 1`))
	assert.False(t, fn(`foo = $1 or 1=1`))
	assert.False(t, fn(`foo = $1 or1=1`))
	assert.False(t, fn(`foo = $1; or1=1`))
	assert.False(t, fn(`foo = $1;or1=1`))
	assert.False(t, fn(`foo = $1 or x = x`))
	assert.False(t, fn(`foo = $1; or x = x`))
	assert.False(t, fn(`foo = $1, or x = x`))
	assert.False(t, fn(`foo = $1 ; or x = x`))
	assert.False(t, fn(`foo = $1 , or x = x`))
	assert.False(t, fn(`foo = $1 or 'x' = 'x'`))

	assert.False(t, fn(`foo = "" or "" = "" and bar = "" or "" = ""`))

	assert.False(t, fn(`foo = $1; DROP TABLE bar`))
	assert.False(t, fn(`foo = $1, DROP TABLE bar`))
	assert.False(t, fn(`DROP TABLE bar; foo = $1`))
	assert.False(t, fn(`DROP TABLE bar, foo = $1`))
	assert.False(t, fn(`DROP TABLE bar`))

	assert.False(t, fn(`x' and foo = $1 and bar is null; --'`))

	assert.False(t, fn(`foo = $1; INSERT INTO table (bar) VALUES (1)`))
	assert.False(t, fn(`foo = $1; UNION SELECT bar FROM table`))
	assert.False(t, fn(`UNION SELECT bar FROM table`))
	assert.False(t, fn(`SELECT bar FROM table`))
	assert.False(t, fn(`UPDATE table SET bar = $1`))

	// brute force
	for i, b := range naughtyBytes {
		l := fmt.Sprintf("naughty byte line no %v", i)
		assert.False(t, fn(`foo = `+string(b)), l)
		assert.False(t, fn(`foo(`+string(b)+`) = $1`), l)
		assert.False(t, fn(`foo `+string(b)+` = $1`), l)
		assert.False(t, fn(`foo `+string(b)), l)
		assert.False(t, fn(`foo = '`+string(b)+`'`), l)
		assert.False(t, fn(`foo = "`+string(b)+`"`), l)
		assert.False(t, fn(`foo = $1; `+string(b)), l)
		assert.False(t, fn(`foo = $1; -- `+string(b)), l)
		assert.False(t, fn(`foo = $1 -- `+string(b)), l)
	}
}

func BenchmarkIsValidUntrustedQuery(b *testing.B) {
	for n := 0; n < b.N; n++ {
		isValidUntrustedQuery(`(foo = $1 and bar = $2) or (abc = $3 and def = $4) or ghi != $5 or jkl like $6`)
	}
}
