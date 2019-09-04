package postgres

import (
	"strings"
)

type table struct {
	Name    string
	Columns []column
	Indexes []index
}

type column struct {
	Name       string
	IsNullable postgresBool
	DataType   string
}

type index struct {
	Name         string
	Type         string
	Columns      []string
	IsUnique     postgresBool
	IsPrimary    postgresBool
	IsFunctional postgresBool
	IsPartial    postgresBool
}

func (t *table) hasIndex(x index) bool {
	for _, i := range t.Indexes {
		if toSnake(i.Name) == toSnake(x.Name) &&
			i.Type == x.Type &&
			equalStringSlice(stringSliceToSnake(i.Columns), stringSliceToSnake(x.Columns)) &&
			i.IsUnique == x.IsUnique &&
			i.IsPrimary == x.IsPrimary &&
			i.IsFunctional == x.IsFunctional &&
			i.IsPartial == x.IsPartial {
			return true
		}
	}
	return false
}

func (t *table) hasUniqueIndexByColumns(columnNames []string) bool {
	for i := 0; i < len(columnNames); i++ {
		columnNames[i] = toSnake(columnNames[i])
	}

	for _, i := range t.Indexes {
		if i.IsUnique {
			if equalStringSliceIgnoreOrder(i.Columns, columnNames) {
				return true
			}
		}
	}
	return false
}

func (t *table) hasColumnByName(name string) bool {
	n := toSnake(name)
	for _, c := range t.Columns {
		if strings.EqualFold(c.Name, n) {
			return true
		}
	}
	return false
}
