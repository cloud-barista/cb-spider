package main

import (
        "fmt"
	"log"

        th "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/table-handler")


func main() {

        srv, err := th.GetHandler()
        if err != nil {
                log.Fatalf("Unable to retrieve Sheets client: %v", err)
        }
	// read cell
	fmt.Println( th.ReadCell(srv, &th.Cell{Sheet:"Status", X:"B", Y:"5"}) )
	fmt.Println( th.ReadCell(srv, &th.Cell{Sheet:"Status", X:"C", Y:"5"}) )
	fmt.Println( th.ReadCell(srv, &th.Cell{Sheet:"Status", X:"D", Y:"5"}) )
	fmt.Println( th.ReadCell(srv, &th.Cell{Sheet:"Status", X:"E", Y:"5"}) )

	// read range
	fmt.Println( th.ReadRange(srv, &th.CellRange{Sheet:"Status", X:"B", Y:"5", X2:"E"}) )
	fmt.Println( th.ReadRange(srv, &th.CellRange{Sheet:"Status", X:"B", Y:"6", X2:"E"}) )

	fmt.Println( th.ReadRange2(srv, &th.CellRange2{Sheet:"Status", X:"B", Y:"5", X2:"E", Y2:"50"}) )
/*
	// write cell
	fmt.Println( th.WriteCell(srv, &th.Cell{Sheet:"Status", X:"C", Y:"5"}, "13.124.44.241:4096-2020.11.16.04:06:50") )
	fmt.Println( th.WriteCell(srv, &th.Cell{Sheet:"Status", X:"D", Y:"5"}, "L") )
	fmt.Println( th.WriteCell(srv, &th.Cell{Sheet:"Status", X:"E", Y:"5"}, "2020.11.16 04:06:53 Mon") )
	// write range
	fmt.Println( th.WriteRange(srv, &th.CellRange{Sheet:"Status", X:"C", Y:"6", X2:"E"}, []string{"3.131.82.23:4096-2020.11.16.03:59:15", "L", "2020.11.16 04:02:04 Mon"}) )
*/
}
