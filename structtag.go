package postgres

import (
	"fmt"

	"github.com/alecthomas/participle"
)

var structTagParser *participle.Parser

func init() {
	structTagParser = participle.MustBuild(&structTag{})
}

type structTag struct {
	Functions []stFunction `@@ ( ","? @@ )*`
}

type stFunction struct {
	Name string  `@Ident`
	Args []stArg `("(" ( @@ ( ","? @@ )* )? ")")?`
}

type stArg struct {
	Name  string  `@Ident`
	Value stValue `"=" @@`
}

func (a *stArg) String() string {
	if a.Value.String != nil {
		return *a.Value.String
	}
	return ""
}

func (a *stArg) List() []string {
	return a.Value.List
}

func (a *stArg) GoString() string {
	if a.Value.String != nil {
		return fmt.Sprintf("%v = %v", a.Name, *a.Value.String)

	} else if a.Value.Float != nil {
		return fmt.Sprintf("%v = %f", a.Name, *a.Value.Float)

	} else if a.Value.Int != nil {
		return fmt.Sprintf("%v = %v", a.Name, *a.Value.Int)

	} else if a.Value.List != nil {
		return fmt.Sprintf("%v = %v", a.Name, a.Value.List)

	} else {
		return fmt.Sprintf("%v = <nil>", a.Name)
	}
}

type stValue struct {
	String *string  `  (@String|@Ident)`
	Float  *float64 `| @Float`
	Int    *int     `| @Int`
	List   []string `| "{" ( (@String|@Ident) ( ","? (@String|@Ident) )* )? "}"`
}

func parseStructTag(tag string) (*structTag, error) {
	if tag == "" {
		return &structTag{}, nil
	}

	st := &structTag{}
	if err := structTagParser.ParseString(tag, st); err != nil {
		return nil, fmt.Errorf(`StructTag "%v": %v`, tag, err)
	}
	return st, nil
}

type indexStructTag struct {
	name      string
	unique    bool
	method    string
	order     string
	composite []string // composite is guaranteed to not include parent name
}

type primaryKeyStructTag struct {
	name      string
	method    string
	order     string
	composite []string // composite is guaranteed to not include parent name
}

type foreignKeyStructTag struct {
	structName string
	fieldNames []string
}

type partitionByRangeStructTag struct{}

func (f *field) parseStructTag(tag string) error {
	s, err := parseStructTag(tag)
	if err != nil {
		return err
	}

	for _, function := range s.Functions {
		switch function.Name {

		case "pk":
			pkSt := &primaryKeyStructTag{}
			for _, arg := range function.Args {
				switch arg.Name {
				case "method":
					pkSt.method = arg.String()

				case "name":
					pkSt.name = arg.String()

				case "order":
					pkSt.order = arg.String()

				case "composite":
					pkSt.composite = arg.List()

				default:
					return fmt.Errorf("pk: unknown argument %v", arg.Name)
				}
			}

			// make sure composite does not contain name
			pkSt.composite = removeFromStringSlice(pkSt.composite, f.name)

			f.primaryKey = pkSt

		case "index":
			fallthrough

		case "unique":
			indexSt := indexStructTag{}
			indexSt.unique = function.Name == "unique"

			for _, arg := range function.Args {
				switch arg.Name {

				case "method":
					indexSt.method = arg.String()

				case "name":
					indexSt.name = arg.String()

				case "order":
					indexSt.order = arg.String()

				case "composite":
					indexSt.composite = arg.List()

				default:
					if function.Name == "unique" {
						return fmt.Errorf("unique: unknown argument %v", arg.Name)
					} else {
						return fmt.Errorf("index: unknown argument %v", arg.Name)
					}
				}
			}

			// make sure composite does not contain name
			indexSt.composite = removeFromStringSlice(indexSt.composite, f.name)

			f.indexes = append(f.indexes, indexSt)

		case "references":
			fkSt := foreignKeyStructTag{}
			for _, arg := range function.Args {
				switch arg.Name {

				case "struct":
					fkSt.structName = arg.String()

				case "fields":
					fkSt.fieldNames = arg.List()

				case "field":
					fkSt.fieldNames = []string{arg.String()}

				default:
					return fmt.Errorf("references: unknown argument %v", arg.Name)
				}
			}
			f.foreignKeys = append(f.foreignKeys, fkSt)

		case "partitionByRange":
			f.partitionByRange = &partitionByRangeStructTag{}

		// if unknown function name...
		default:
			return fmt.Errorf("unknown: %v", function.Name)
		}
	}

	return nil
}
