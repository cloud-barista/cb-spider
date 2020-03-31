package main

import (
	// "github.com/metakeule/fmtdate"

	"fmt"
	"time"
)

func main() {

	t := time.Now()

	// to 2019-10-24T16:52:44+09:0
	fmt.Println(t.Format(time.RFC3339)) // 2019-01-12T01:02:03Z

	s := time.Now().Format("2006-01-02 15:04:05")
	// to 2019-10-24 16:52:44
	fmt.Println(s) // 2019-01-12 10:20:30

	s1 := time.Now().Format("2006-01-02T15:04Z")
	// to 2019-10-24 16:52:44
	fmt.Println(s1) // 2019-01-12 10:20:30

	s2 := "2019-01-12 12:30:00"
	t2, _ := time.Parse("2006-01-02 15:04:05", s2)
	fmt.Println(t2) // 2019-01-12 12:30:00 +0000 UTC

	// s3 := "2019-10-24T16:55Z"

	// test, err := fmtdate.Parse("MM/DD/YYYY", "10/15/1983")
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(test)
}
