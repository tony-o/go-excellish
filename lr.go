package main

import (
	"fmt"

	lr "example.com/lr/src"
	"github.com/gofrs/uuid"
)

// TYPES
type Status int8

const (
	Purchased Status = iota
	Abandoned
	Browsing
)

type Product struct {
	ID    uuid.UUID
	Name  string
	Price float64
}

type Cart struct {
	ID       uuid.UUID
	Products []Product
	Status   Status
}

// METHODS
func main() {
	prod := &Product{
		ID:    uuid.Must(uuid.NewV4()),
		Name:  "product 1",
		Price: 65.25,
	}
	prods := []interface{}{
		*(&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 1", Price: 1.11}),
		*(&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 2", Price: 2.22}),
		*(&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 3", Price: 3.33}),
	}
	OK := map[string]string{
		"[name] + '++'":                          "",
		"[name] + ': ' + [price]":                "",
		"sum([price])":                           "",
		"sum([price], [amount])":                 "",
		"([price] + [amount]) * 1":               "",
		"(((((((((([price]))))))))))":            "",
		"SUM([loans], [xyzs])":                   "SUM([loans], [xyzs])",
		"    5 +     6.1235566777":               "     5+    6.1235566777",
		"\"escaped \\\" q\"":                     "\"escaped \\\" q\"",
		"\"hello: \" + concat(\"world\", \"!\")": "\"hello: \" + concat(\"world\", \"!\")",
		"sumif([price], [name] = 'Prod 1')":      "",
		"sumif([price], [price] > 2)":            "",
	}
	for k, v := range OK {
		l := lr.NewParser()
		var e string = v
		if v == "" {
			e = k
		}
		err := l.Parse(e)
		if err == nil {
			fmt.Printf("[CALCULATED FIELD]")
		} else {
			fmt.Printf("[INCALCULABLE]")
		}
		fmt.Printf(" %s", e)
		if err != nil {
			fmt.Printf("%v", err)
		}
		fmt.Printf("\n[AST]\n%s\n", l.AST())
		applies, err := l.AppliesTo(prod)
		if applies && err == nil {
			r, e := l.Run(prod)
			fmt.Printf("[APPLYING] &Product{}\n  result=%v\n  error=%v\n", r, e)
		} else {
			fmt.Printf("[SKIPPING] &Product{}: %v\n", err)
		}
		applies, err = l.AppliesTo(prods...)
		if applies && err == nil {
			r, e := l.Run(prods...)
			fmt.Printf("[APPLYING] []Product\n  result=%v\n  error=%v\n", r, e)
		} else {
			fmt.Printf("[SKIPPING] []Product: %v\n", err)
		}
		fmt.Printf("\n\n\n")
	}
}
