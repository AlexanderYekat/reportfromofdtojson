package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type TTaxNDS struct {
	Type string `json:"type"`
}

type TPayment struct {
	Type string `json:"type"`
	Sum  string `json:"sum"`
}

type TPosition struct {
	Type string `json:"type"`
	Name string `json:"name"`
	//Price           float64 `json:"price"`
	Price string `json:"price"`
	//Quantity        float64 `json:"quantity"`
	Quantity string `json:"quantity"`
	//Amount          float64 `json:"amount"`
	Amount          string  `json:"amount"`
	MeasurementUnit int     `json:"measurementUnit"`
	PaymentMethod   string  `json:"paymentMethod"`
	PaymentObject   string  `json:"paymentObject"`
	Tax             TTaxNDS `json:"tax"`
}

type TOperator struct {
	Name string `json:"name"`
}

type TCorrectionCheck struct {
	Type                 string      `json:"type"`
	Electronically       bool        `json:"electronically"`
	CorrectionType       string      `json:"correctionType"`
	CorrectionBaseDate   string      `json:"correctionBaseDate"`
	CorrectionBaseNumber string      `json:"correctionBaseNumber"`
	Operator             TOperator   `json:"operator"`
	Items                []TPosition `json:"items"`
	Payments             []TPayment  `json:"payments"`
	Total                float64     `json:"total,omitempty"`
}

func formatMyDate(dt string) string {
	y := dt[6:]
	m := dt[3:5]
	d := dt[0:2]
	res := y + "." + m + "." + d
	return res
}

func main() {
	f, err := os.Open("checks_header.csv")
	if err != nil {
		log.Fatal("не удлась открыть файл заголовков чека", err)
	}
	defer f.Close()
	csv_red := csv.NewReader(f)
	csv_red.FieldsPerRecord = -1
	csv_red.LazyQuotes = true
	csv_red.Comma = ';'
	lines, err := csv_red.ReadAll()
	/*for {
		record, err := csv_red.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(record)
	}*/
	if err != nil {
		log.Fatal("не удлась прочитать csv файл шапки чека ", err)
	}
	//log.Fatal("---- ", err)
	currLine := 0
	for _, line := range lines {
		currLine++
		//fmt.Println(line)
		//for i := 0; i < len(line); i++ {
		//	fmt.Println(i, line[i])
		//}
		if currLine == 1 {
			continue
		}
		//for i := 0; i < len(line); i++ {
		//	fmt.Println(i, line[i])
		//}
		//SNO := line[31]
		kassir := line[27]
		kassaName := line[1]
		fn := line[2]
		fd := line[5]
		dateCh := formatMyDate(line[6])
		typeCheck := line[8]
		//sum := line[9]
		nal := strings.ReplaceAll(line[10], ",", ".")
		bez := strings.ReplaceAll(line[11], ",", ".")
		avance := strings.ReplaceAll(line[13], ",", ".")
		kred := strings.ReplaceAll(line[14], ",", ".")
		//nds20 := line[17]
		//sumBezNDS := line[32]
		//countOfPos := line[24]
		if fd == "2039" || fd == "2060" || fd == "2040" || fd == "2041" {
			fmt.Println("пропускаем чек, по которому уже был пробит чек коррекции")
			continue
		}
		poss := findPositions(fn, kassaName, fd, dateCh)
		if len(poss) > 0 {
			jsonres := generateCheckCorrection(kassir, dateCh, fd, typeCheck, nal, bez, avance, kred, poss)
			as_json, err := json.MarshalIndent(jsonres, "", "\t")
			if err != nil {
				fmt.Println("ошибка", err)
			}
			file_name := fmt.Sprintf("%v_%v.json", fn, fd)
			f, err := os.Create(file_name)
			if err != nil {
				fmt.Println("ошибка", err)
			}
			f.Write(as_json)
			f.Close()
			flagsTempOpen := os.O_APPEND | os.O_CREATE | os.O_WRONLY
			file_name_all_json := fmt.Sprintf("%v.json", fn)
			file_all_json, err := os.OpenFile(file_name_all_json, flagsTempOpen, 0644)
			if err != nil {
				fmt.Println("ошибка", err)
			}
			file_all_json.Write(as_json)
			file_all_json.Close()
		} else {
			fmt.Println("для чека", fd, dateCh, "не найдены позиции")
		}
		//generateCheckCorrection(kassir, dateCh, fd, typeCheck, nal, bez, avance, kred, poss)
	}
}

func findPositions(fn, kassaName, fd, dateCh string) map[int]map[string]string {
	res := make(map[int]map[string]string)
	f, err := os.Open("checks_poss.csv")
	if err != nil {
		log.Fatal("не удлась открыть файл позиций чека", err)
	}
	defer f.Close()
	csv_red := csv.NewReader(f)
	csv_red.FieldsPerRecord = -1
	csv_red.LazyQuotes = true
	csv_red.Comma = ';'
	lines, err := csv_red.ReadAll()
	if err != nil {
		log.Fatal("не удлась прочитать csv файл позиций чека", err)
	}
	currPos := 0
	currLine := 0
	for _, line := range lines {
		currLine++
		if currLine == 1 {
			continue
		}
		currKassaName := line[20]
		fdCurr := line[7]
		if fdCurr == fd && kassaName == currKassaName {
			currPos++
			res[currPos] = make(map[string]string)
			res[currPos]["Name"] = line[0]
			res[currPos]["Quantity"] = strings.ReplaceAll(line[1], ",", ".")
			res[currPos]["Price"] = strings.ReplaceAll(line[2], ",", ".")
			res[currPos]["Amount"] = strings.ReplaceAll(line[3], ",", ".")
			nds20str := line[12]
			res[currPos]["taxNDS"] = "none"
			//fmt.Println("nds20str", nds20str)
			if nds20str != "" && nds20str != "0,00" {
				res[currPos]["taxNDS"] = "vat20"
			}
		}
	}
	//if len(res) > 0 {
	//fmt.Println(res)
	//}
	return res
}

// func generateCheckCorrection(kassir, dateCh, fd, typeCheck string, nal, bez, avance, kred float64, poss map[int]map[string]string) TCorrectionCheck {
func generateCheckCorrection(kassir, dateCh, fd, typeCheck, nal, bez, avance, kred string, poss map[int]map[string]string) TCorrectionCheck {
	var checkCorr TCorrectionCheck
	if typeCheck == "приход" {
		checkCorr.Type = "sellCorrection"
	}
	if typeCheck == "возврат прихода" {
		checkCorr.Type = "sellReturnCorrection"
	}
	checkCorr.CorrectionType = "self"
	checkCorr.CorrectionBaseDate = dateCh
	checkCorr.CorrectionBaseNumber = fd
	checkCorr.Operator.Name = kassir
	if nal != "" {
		pay := TPayment{Type: "cash", Sum: nal}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if bez != "" {
		pay := TPayment{Type: "electronically", Sum: bez}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if avance != "" {
		pay := TPayment{Type: "prepaid", Sum: avance}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if kred != "" {
		pay := TPayment{Type: "credit", Sum: kred}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	for _, pos := range poss {
		newPos := TPosition{Type: "postiton"}
		newPos.Name = pos["Name"]
		newPos.Quantity = pos["Quantity"]
		newPos.Price = pos["Price"]
		newPos.Amount = pos["Amount"]
		newPos.MeasurementUnit = 0
		newPos.PaymentMethod = "fullPayment"
		newPos.PaymentObject = "commodity"
		newPos.Tax.Type = pos["taxNDS"]
		checkCorr.Items = append(checkCorr.Items, newPos)
	}
	return checkCorr
}
