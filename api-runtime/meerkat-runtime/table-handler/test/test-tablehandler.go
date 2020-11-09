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
	fmt.Println( th.ReadCell(srv, &th.Cell{Sheet:"Status", X:"C", Y:"4"}) )
	fmt.Println( th.ReadCell(srv, &th.Cell{Sheet:"Status", X:"D", Y:"4"}) )
	fmt.Println( th.ReadCell(srv, &th.Cell{Sheet:"Status", X:"E", Y:"4"}) )
	// read range
	fmt.Println( th.ReadRange(srv, &th.CellRange{Sheet:"Status", X:"C", Y:"4", X2:"E"}) )

	// write cell
	fmt.Println( th.WriteCell(srv, &th.Cell{Sheet:"Status", X:"C", Y:"5"}, "108") )
	fmt.Println( th.WriteCell(srv, &th.Cell{Sheet:"Status", X:"D", Y:"5"}, "109") )
	fmt.Println( th.WriteCell(srv, &th.Cell{Sheet:"Status", X:"E", Y:"5"}, "110") )
	// write range
	fmt.Println( th.WriteRange(srv, &th.CellRange{Sheet:"Status", X:"C", Y:"6", X2:"E"}, []string{"208", "209", "210"}) )
}
