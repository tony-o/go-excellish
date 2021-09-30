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
	Product  *Product
	Products *[]Product
	Amount   int
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

	cart := &Cart{
		Product: &Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 1", Price: 1.11},
		Products: (&[]Product{
			*&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 1", Price: 1.11},
			*&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 2", Price: 2.22},
			*&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 3", Price: 3.33},
		}),
		Amount: 5,
	}
	cartParser := lr.NewParser()
	err := cartParser.Parse("[product.price] * 10")
	if err != nil {
		fmt.Printf("cartParser err=%v\n", err)
	}
	okcart, err := cartParser.AppliesTo(cart)
	if err != nil {
		fmt.Printf("cartParser applies err=%v\n", err)
	}
	fmt.Printf("%s\n", cartParser.AST())
	if okcart {
		r, e := cartParser.Run(cart)
		if e != nil {
			fmt.Printf("cart run err=%v\n", e)
		} else {
			fmt.Printf("product.price * 10 = %f * 10 = %v\n", cart.Product.Price, r)
		}
	} else {
		fmt.Printf("doesn't apply\n")
	}
	err = cartParser.Parse("sum([products.price]) * 100")
	if err != nil {
		fmt.Printf("cartParser err=%v\n", err)
	}
	okcart, err = cartParser.AppliesTo(cart)
	if err != nil {
		fmt.Printf("cartParser applies err=%v\n", err)
	}
	fmt.Printf("%s\n", cartParser.AST())
	if okcart {
		r, e := cartParser.Run(cart)
		if e != nil {
			fmt.Printf("cart run err=%v\n", e)
		} else {
			fmt.Printf("products.price * 100 = %f * 100 = %v\n", 6.66, r)
		}
	} else {
		fmt.Printf("[products.price] doesn't apply\n")
	}

	OK := map[string]string{
		"[name] + '++'":                          "",
		"[name] + ': ' + [price]":                "",
		"sum([price])":                           "",
		"sum([price], [amount])":                 "",
		"([price] + [amount]) * 1":               "",
		"(((((((((([price]))))))))))":            "",
		"SUM([loans], [xyzs])":                   "",
		"    5 +     6.1235566777":               "",
		"\"escaped \\\" q\"":                     "",
		"\"hello: \" + concat(\"world\", \"!\")": "",
		"sumif([price], [name] = 'Prod 1')":      "",
		"sumif([price], [price] > 2)":            "",
		"sum([name])":                            "",
		"1 + 2 / 3":                              "",
		"1 + 2 / 3 = (2 / 3) + 1":                "",
		"1 + 2 / 3 = 1":                          "",
		" 1 + 2 = 5 + 'hello'":                   "",
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
			fmt.Printf("\n%v\n", err)
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
