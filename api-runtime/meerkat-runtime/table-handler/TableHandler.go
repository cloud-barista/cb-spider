package tablehandler

import (
        "encoding/json"
        "fmt"
        "io/ioutil"
        "log"
        "net/http"
        "os"

        "golang.org/x/net/context"
        "golang.org/x/oauth2"
        "golang.org/x/oauth2/google"
        "google.golang.org/api/sheets/v4"
)

// now, const. @todo from config
const spreadsheetId string = "123pwye5DiSuUxK6uK-2LBsccabfXHA1ofWCcJ4vfsPw"

type Cell struct {
	Sheet string
	X string
	Y string
}

type CellRange struct {
        Sheet string
        X string
        Y string
        X2 string
}


func GetHandler() (*sheets.Service, error) {
	b, err := ioutil.ReadFile("credentials.json")
        if err != nil {
                log.Fatalf("Unable to read client secret file: %v", err)
        }

        // powerkim config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
        config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
        if err != nil {
                log.Fatalf("Unable to parse client secret file to config: %v", err)
        }
        client := getClient(config)

        srv, err := sheets.New(client)
        if err != nil {
                log.Fatalf("Unable to retrieve Sheets client: %v", err)
        }
	return srv, nil
}

func ReadCell(handler *sheets.Service, cell *Cell) (string, error) {
	// ex) range format: "SheetName!C4:C"
	readRange := cell.Sheet + "!" + cell.X +cell.Y + ":" + cell.X
	resp, err := handler.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
        if err != nil {
                log.Fatalf("Unable to retrieve data from sheet: %v", err)
		return "", err
        }
	if resp.Values == nil {
		return "", nil
	}
	// [][]interface[] ==> string
	strResults := make([]string, len(resp.Values[0]))
        for i, v := range resp.Values[0] {
                strResults[i] = v.(string)
        }
	if (strResults == nil) || (len(strResults) == 0) {
		return "", nil
	}

	return strResults[0], nil
}

func ReadRange(handler *sheets.Service, cellRange *CellRange) ([]string, error) {
        // ex) range format: "SheetName!C4:E"
        readRange := cellRange.Sheet + "!" + cellRange.X +cellRange.Y + ":" + cellRange.X2
        resp, err := handler.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
        if err != nil {
                log.Fatalf("Unable to retrieve data from sheet: %v", err)
                return []string{}, err
        }
        if resp.Values == nil {
                return []string{}, nil
        }

	// [][]interface[] ==> []string
	strResults := make([]string, len(resp.Values[0]))
	for i, v := range resp.Values[0] {
		strResults[i] = v.(string)
	}
	if (strResults == nil) || (len(strResults) == 0) {
                return []string{}, nil
        }

        return strResults, nil
}

func WriteCell(handler *sheets.Service, cell *Cell, value string) error {

	// ex) range format: "SheetName!C4:C"
        writeRange := cell.Sheet + "!" + cell.X +cell.Y + ":" + cell.X

        var vr sheets.ValueRange
        cellVal := []interface{}{value}
        vr.Values = append(vr.Values, cellVal)

        _, err := handler.Spreadsheets.Values.Update(spreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()
        if err != nil {
                log.Fatalf("Unable to write data into sheet: %v", err)
                return err
        }

        return nil
}

func WriteRange(handler *sheets.Service, cellRange *CellRange, valueList []string) error {
	// ex) range format: "SheetName!C4:#"
        writeRange := cellRange.Sheet + "!" + cellRange.X +cellRange.Y + ":" + cellRange.X2

	ifList := make([]interface{}, len(valueList))
	for i, v := range valueList {
		ifList[i] = v
	}
        var vr sheets.ValueRange
	vr.Values = append(vr.Values, ifList)

        _, err := handler.Spreadsheets.Values.Update(spreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()
        if err != nil {
                log.Fatalf("Unable to write data into sheet: %v", err)
                return err
        }

        return nil
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
        // The file token.json stores the user's access and refresh tokens, and is
        // created automatically when the authorization flow completes for the first
        // time.
        tokFile := "token.json"
        tok, err := tokenFromFile(tokFile)
        if err != nil {
                tok = getTokenFromWeb(config)
                saveToken(tokFile, tok)
        }
        return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
        authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
        fmt.Printf("Go to the following link in your browser then type the "+
                "authorization code: \n%v\n", authURL)

        var authCode string
        if _, err := fmt.Scan(&authCode); err != nil {
                log.Fatalf("Unable to read authorization code: %v", err)
        }

        tok, err := config.Exchange(context.TODO(), authCode)
        if err != nil {
                log.Fatalf("Unable to retrieve token from web: %v", err)
        }
        return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
        f, err := os.Open(file)
        if err != nil {
                return nil, err
        }
        defer f.Close()
        tok := &oauth2.Token{}
        err = json.NewDecoder(f).Decode(tok)
        return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
        fmt.Printf("Saving credential file to: %s\n", path)
        f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
        if err != nil {
                log.Fatalf("Unable to cache oauth token: %v", err)
        }
        defer f.Close()
        json.NewEncoder(f).Encode(token)
}
