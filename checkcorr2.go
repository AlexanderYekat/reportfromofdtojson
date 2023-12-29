package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const JSONRES = "./json/"
const DIRINFILES = "./infiles/"

type TClientInfo struct {
	EmailOrPhone string `json:"emailOrPhone"`
}

type TTaxNDS struct {
	Type string `json:"type"`
}

type TPayment struct {
	Type string  `json:"type"`
	Sum  float64 `json:"sum"`
}

type TPosition struct {
	Type string `json:"type"`
	Name string `json:"name"`
	//Price           float64 `json:"price"`
	Price float64 `json:"price"`
	//Quantity        float64 `json:"quantity"`
	Quantity float64 `json:"quantity"`
	//Amount          float64 `json:"amount"`
	Amount          float64 `json:"amount"`
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
	ClientInfo           TClientInfo `json:"clientInfo"`
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
	if foundedLogDir, _ := doesFileExist(JSONRES); !foundedLogDir {
		os.Mkdir(JSONRES, 0777)
		f, err := os.Create(JSONRES + "printed.txt")
		if err == nil {
			f.Close()
		}
		f, err = os.Create(JSONRES + "connection.txt")
		if err == nil {
			f.Close()
		}
	}
	f, err := os.Open(DIRINFILES + "checks_header.csv")
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
		nal = strings.ReplaceAll(nal, "-", "")
		bez := strings.ReplaceAll(line[11], ",", ".")
		bez = strings.ReplaceAll(bez, "-", "")
		//sumpplat := strings.ReplaceAll(line[12], ",", ".")
		avance := strings.ReplaceAll(line[13], ",", ".")
		avance = strings.ReplaceAll(avance, "-", "")
		kred := strings.ReplaceAll(line[14], ",", ".")
		kred = strings.ReplaceAll(kred, "-", "")
		obmen := strings.ReplaceAll(line[15], ",", ".")
		obmen = strings.ReplaceAll(obmen, "-", "")
		strNDS20 := line[17]
		//sumBezNDS := line[32]
		//countOfPos := line[24]
		//if fd == "2039" || fd == "2060" || fd == "2040" || fd == "2041" || fd == "2330" || fd == "2331" || fd == "2332" {
		//	fmt.Println("пропускаем чек, по которому уже был пробит чек коррекции")
		//	continue
		//}
		poss := findPositions(fn, kassaName, fd, dateCh, strNDS20)
		if len(poss) > 0 {
			jsonres := generateCheckCorrection(kassir, dateCh, fd, typeCheck, nal, bez, avance, kred, obmen, poss)
			as_json, err := json.MarshalIndent(jsonres, "", "\t")
			if err != nil {
				fmt.Println("ошибка", err)
			}
			dir_file_name := fmt.Sprintf("%v%v/", JSONRES, fn)
			if foundedLogDir, _ := doesFileExist(dir_file_name); !foundedLogDir {
				os.Mkdir(dir_file_name, 0777)
				f, err := os.Create(dir_file_name + "printed.txt")
				if err == nil {
					f.Close()
				}
				f, err = os.Create(dir_file_name + "connection.txt")
				if err == nil {
					f.Close()
				}
			}
			file_name := fmt.Sprintf("%v%v/%v_%v.json", JSONRES, fn, fn, fd)
			f, err := os.Create(file_name)
			if err != nil {
				fmt.Println("ошибка", err)
			}
			f.Write(as_json)
			f.Close()
			//flagsTempOpen := os.O_APPEND | os.O_CREATE | os.O_WRONLY
			//file_name_all_json := fmt.Sprintf("%v.json", fn)
			//file_all_json, err := os.OpenFile(file_name_all_json, flagsTempOpen, 0644)
			//if err != nil {
			//	fmt.Println("ошибка", err)
			//}
			//file_all_json.Write(as_json)
			//file_all_json.Close()
		} else {
			fmt.Println("для чека", fd, dateCh, "не найдены позиции")
		}
		//generateCheckCorrection(kassir, dateCh, fd, typeCheck, nal, bez, avance, kred, poss)
	}
}

func findPositions(fn, kassaName, fd, dateCh, strNDS20 string) map[int]map[string]string {
	res := make(map[int]map[string]string)
	f, err := os.Open(DIRINFILES + "checks_poss.csv")
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
			currPrice := strings.ReplaceAll(line[2], ",", ".")
			res[currPos]["Price"] = strings.ReplaceAll(currPrice, "-", "")
			currAmount := strings.ReplaceAll(line[3], ",", ".")
			res[currPos]["Amount"] = strings.ReplaceAll(currAmount, "-", "")
			res[currPos]["prepaymeny"] = "false"
			summPrepayment := line[8]
			//summPrepayment := strings.ReplaceAll(line[8], ",", ".")
			//summPrepayment = strings.TrimSpace(strings.ReplaceAll(summPrepayment, "-", ""))
			if summPrepayment != "" {
				res[currPos]["prepaymeny"] = "true"
			}
			//nds20str := line[12]
			res[currPos]["taxNDS"] = "none"
			//fmt.Println("nds20str", nds20str)
			if strNDS20 != "" && strNDS20 != "0,00" {
				res[currPos]["taxNDS"] = "vat20"
				if summPrepayment != "" {
					res[currPos]["taxNDS"] = "vat120"
				}
		}
	}
	//if len(res) > 0 {
	//fmt.Println(res)
	//}
	return res
}

// func generateCheckCorrection(kassir, dateCh, fd, typeCheck string, nal, bez, avance, kred float64, poss map[int]map[string]string) TCorrectionCheck {
func generateCheckCorrection(kassir, dateCh, fd, typeCheck, nal, bez, avance, kred, obmen string, poss map[int]map[string]string) TCorrectionCheck {
	var checkCorr TCorrectionCheck
	checkCorr.Type = ""
	if typeCheck == "приход" {
		checkCorr.Type = "sellCorrection"
	}
	if typeCheck == "возврат прихода" {
		checkCorr.Type = "sellReturnCorrection"
	}
	if typeCheck == "расход" {
		checkCorr.Type = "buyCorrection"
	}
	if checkCorr.Type == "" {
		descError := fmt.Sprintf("ошибка тип чека коррекциии для типа %v - не определён", typeCheck)
		fmt.Println(descError)
		log.Fatal(descError)
	}

	checkCorr.Electronically = true
	checkCorr.CorrectionType = "self"
	checkCorr.CorrectionBaseDate = dateCh
	checkCorr.CorrectionBaseNumber = fd
	checkCorr.ClientInfo.EmailOrPhone = "t.halemina@glazurit.ru"
	checkCorr.Operator.Name = kassir
	if nal != "" && nal != "0.00" {
		nalch, err := strconv.ParseFloat(nal, 64)
		if err != nil {
			fmt.Println("ошибка", err)
		}
		pay := TPayment{Type: "cash", Sum: nalch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if bez != "" && bez != "0.00" {
		bezch, err := strconv.ParseFloat(bez, 64)
		if err != nil {
			fmt.Println("ошибка", err)
		}
		pay := TPayment{Type: "electronically", Sum: bezch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if avance != "" && avance != "0.00" {
		avancech, err := strconv.ParseFloat(avance, 64)
		if err != nil {
			fmt.Println("ошибка", err)
		}
		pay := TPayment{Type: "prepaid", Sum: avancech}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if kred != "" && kred != "0.00" {
		kredch, err := strconv.ParseFloat(kred, 64)
		if err != nil {
			fmt.Println("ошибка", err)
		}
		pay := TPayment{Type: "credit", Sum: kredch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if obmen != "" && obmen != "0.00" {
		obmench, err := strconv.ParseFloat(obmen, 64)
		if err != nil {
			fmt.Println("ошибка", err)
		}
		pay := TPayment{Type: "other", Sum: obmench}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	for _, pos := range poss {
		newPos := TPosition{Type: "position"}
		newPos.Name = pos["Name"]
		qch, err := strconv.ParseFloat(pos["Quantity"], 64)
		if err != nil {
			fmt.Println("ошибка", err)
		}
		newPos.Quantity = qch
		prch, err := strconv.ParseFloat(pos["Price"], 64)
		if err != nil {
			fmt.Println("ошибка", err)
		}
		newPos.Price = prch
		sch, err := strconv.ParseFloat(pos["Amount"], 64)
		if err != nil {
			fmt.Println("ошибка", err)
		}
		newPos.Amount = sch
		newPos.MeasurementUnit = 0
		if pos["prepaymeny"] == "true" {
			newPos.PaymentMethod = "fullPrepayment"
			newPos.PaymentObject = "payment"
		} else {
			newPos.PaymentMethod = "fullPayment"
			newPos.PaymentObject = "commodity"
		}
		newPos.Tax.Type = pos["taxNDS"]
		checkCorr.Items = append(checkCorr.Items, newPos)
	}
	return checkCorr
}
func doesFileExist(fullFileName string) (found bool, err error) {
	found = false
	if _, err = os.Stat(fullFileName); err == nil {
		// path/to/whatever exists
		found = true
	}
	return
}
