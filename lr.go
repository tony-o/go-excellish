package main

import (
	"fmt"

	lr "example.com/lr/src"
	"github.com/gofrs/uuid"
	"github.com/loov/hrtime"
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
		"sum([name])":                            "",
	}
	for k, v := range OK {
		l := lr.NewParser()
		var e string = v
		if v == "" {
			e = k
		}
		bench := hrtime.NewBenchmark(1000)
		runn1 := hrtime.NewBenchmark(1000)
		runn2 := hrtime.NewBenchmark(1000)
		for bench.Next() {
			l.Parse(e)
		}
		err := l.Parse(e)
		if err == nil {
			fmt.Printf("[CALCULATED FIELD]")
			fmt.Printf("\n[COMPILATION]\n")
			fmt.Println(bench.Histogram(10))
		} else {
			fmt.Printf("[INCALCULABLE]")
		}
		fmt.Printf(" %s", e)
		if err != nil {
			fmt.Printf("%v", err)
			continue
		}
		fmt.Printf("\n[AST]\n%s\n", l.AST())
		applies, err := l.AppliesTo(prod)
		if applies && err == nil {
			r, e := l.Run(prod)
			fmt.Printf("[APPLYING] &Product{}\n  result=%v\n  error=%v\n", r, e)
			for runn1.Next() {
				l.Run(prods...)
			}
			fmt.Printf("[RUN SINGLE INPUT]\n")
			fmt.Println(runn1.Histogram(10))
		} else {
			fmt.Printf("[SKIPPING] &Product{}: %v\n", err)
		}
		applies, err = l.AppliesTo(prods...)
		if applies && err == nil {
			r, e := l.Run(prods...)
			fmt.Printf("[APPLYING] []Product\n  result=%v\n  error=%v\n", r, e)
			for runn2.Next() {
				l.Run(prods...)
			}
			fmt.Printf("[RUN MULTI INPUT]\n")
			fmt.Println(runn2.Histogram(10))
		} else {
			fmt.Printf("[SKIPPING] []Product: %v\n", err)
		}
		fmt.Printf("\n\n\n")
	}
}
