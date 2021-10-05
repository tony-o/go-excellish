package fieldcalculator_test

import (
	"math"
	"testing"

	fieldCalculator "example.com/lr/pkg/field-calculator"
	"github.com/gofrs/uuid"
)

type Product struct {
	ID    uuid.UUID
	Name  string
	Price float64
}

type Receipt struct {
	Lines *[]Product
}

// METHODS
func TestEvaluator_Slices(t *testing.T) {
	rcpt := &Receipt{
		Lines: (&[]Product{
			*&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 1", Price: 1.11},
			*&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 2", Price: 2.22},
			*&Product{ID: uuid.Must(uuid.NewV4()), Name: "Prod 3", Price: 3.33},
		}),
	}
	rcptField := fieldCalculator.NewParser()
	err := rcptField.Parse("sum([lines.price]) * 0.2")
	if err != nil {
		t.Logf("err=%v\n", err)
		t.FailNow()
		return
	}
	if ok, err := rcptField.AppliesTo(rcpt); err != nil {
		t.Logf("err=%v\n", err)
		t.FailNow()
		return
	} else if !ok {
		t.Logf("this field should apply to receipts")
		t.FailNow()
		return
	}
	if ok, err := rcptField.AppliesTo(&Receipt{}); err != nil {
		t.Logf("err=%v\n", err)
		t.Fail()
	} else if !ok {
		t.Logf("this field should apply to &Receipt{}")
		t.Fail()
	}
	if ok, err := rcptField.AppliesTo(&Product{}); err != nil {
		t.Logf("err=%v\n", err)
		t.Fail()
	} else if !ok {
		t.Logf("this field should not apply to &Product{}")
		t.Fail()
	}

	t.Run("sum([lines.price])*0.2", func(t *testing.T) {
		if rcptField.AST() != "{\n  \"type\": \"func\",\n  \"name\": \"*\",\n  \"args\": [\n    {\n      \"type\": \"func\",\n      \"name\": \"sum\",\n      \"args\": [\n        { \"type\": \"field\", \"name\": \"lines.price\" }\n      ]\n    },\n    { \"type\": \"literal\", \"value\": \"%!s(float64=0.2)\" }\n  ]\n}" {
			t.Logf("AST is not as expected")
			t.Fail()
		}
		r, e := rcptField.Run(rcpt)
		if e != nil {
			t.Logf("err=%v\n", e)
			t.FailNow()
		} else if len(r) != 1 {
			t.Logf("no result or too many results from calculation, expected=1,got=%d", len(r))
			t.FailNow()
		} else {
			switch f := r[0].(type) {
			case float64:
				if f != (1.11+2.22+3.33)*0.2 {
					t.Logf("expected=%0.3f,got=%0.3f", (1.11+2.22+3.33)*0.2, f)
					t.FailNow()
				}
			default:
				t.Logf("calculator returned wrong type, expected=float64,got=%T", f)
				t.FailNow()
			}
		}
	})
}

func TestEvaluator_Calculators(t *testing.T) {
	prod := &Product{
		ID:    uuid.Must(uuid.NewV4()),
		Name:  "product 1",
		Price: 65.25,
	}
	OK := map[string]interface{}{
		"[name] + '++'":                          "product 1++",
		"[name] + ': ' + [price]":                "product 1: 65.25",
		"sum([price])":                           65.25,
		"(((((((((([price]))))))))))":            65.25,
		"    5 +     6.1235566777":               5 + 6.1235566777,
		"\"escaped \\\" q\"":                     "escaped \\\" q",
		"\"hello: \" + concat(\"world\", \"!\")": "hello: world!",
		"sumif([price], [name] = 'Prod 1')":      0.00,
		"sumif([price], [price] > 2)":            65.25,
		"1 + 2 / 3":                              (2.0 / 3.0) + 1.0,
		"1 + 2 / 3 = (2 / 3) + 1":                true,
		"1 + 2 / 3 = 1":                          false,
		" 1 + 2 = 5 + 'hello'":                   false,
		"1 + 1 + 1 + 1 / 4 / 1":                  3.25,
	}
	//		"sum([price], [amount])":                 "",
	//		"([price] + [amount]) * 1":               "",
	//		"SUM([loans], [xyzs])":                   "",
	//		"sum([name])":                            "",
	for k, expect := range OK {
		t.Run(k, func(t *testing.T) {
			l := fieldCalculator.NewParser()
			err := l.Parse(k)
			if err != nil {
				t.Logf("error compiling:%v", err)
				t.FailNow()
				return
			}
			if applies, err := l.AppliesTo(prod); err != nil {
				t.Logf("error, should apply:%v", err)
				t.FailNow()
				return
			} else if !applies {
				t.Logf("should apply to prod")
				t.FailNow()
				return
			}
			r, e := l.Run(prod)
			if e != nil {
				t.Logf("error in calc:%v", err)
				t.FailNow()
				return
			}
			switch r[0].(type) {
			case float64:
				diff := math.Abs(r[0].(float64) - expect.(float64))
				if diff > .0000000001 {
					t.Logf("expected=%0.8f,got=%0.8f", expect, r[0])
					t.FailNow()
					return

				}
			default:
				if r[0] != expect {
					t.Logf("expected=%v,got=%v", expect, r[0])
					t.FailNow()
					return
				}
			}
		})
	}
}
