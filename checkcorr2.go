package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

var LOGSDIR = "./logs/"
var filelogmap map[string]*os.File
var logsmap map[string]*log.Logger

const LOGINFO = "info"
const LOGINFO_WITHSTD = "info_std"
const LOGERROR = "error"
const LOGSKIP_LINES = "lines_skip"

// const LOGSKIP_LINES = "skip_line"
const LOGOTHER = "other"
const LOG_PREFIX = "XLSTOJSON"

const JSONRES = "./json/"
const DIRINFILES = "./infiles/"

const VERSION_OF_PROGRAM = "2023_01_01_01"
const NAME_OF_PROGRAM = "формирование json заданий чеков коррекции на основании отчетов из ОФД (xsl-csv)"

type TClientInfo struct {
	EmailOrPhone string `json:"emailOrPhone"`
}

type TTaxNDS struct {
	Type string `json:"type,omitempty"`
}

type TPayment struct {
	Type string  `json:"type"`
	Sum  float64 `json:"sum"`
}

type TPosition struct {
	Type            string   `json:"type"`
	Name            string   `json:"name,omitempty"`
	Price           float64  `json:"price,omitempty"`
	Quantity        float64  `json:"quantity,omitempty"`
	Amount          float64  `json:"amount,omitempty"`
	MeasurementUnit int      `json:"measurementUnit,omitempty"`
	PaymentMethod   string   `json:"paymentMethod,omitempty"`
	PaymentObject   string   `json:"paymentObject,omitempty"`
	Tax             *TTaxNDS `json:"tax,omitempty"`
	//fot type tag1192 //AdditionalAttribute
	Value string `json:"value,omitempty"`
	Print bool   `json:"print,omitempty"`
}

type TOperator struct {
	Name string `json:"name"`
}

// При работе по ФФД ≥ 1.1 чеки коррекции имеют вид, аналогичный обычным чекам, но с
// добавлением информации о коррекции: тип, описание, дата документа основания и
// номер документа основания.
type TCorrectionCheck struct {
	Type string `json:"type"` //sellCorrection - чек коррекции прихода
	//buyCorrection - чек коррекции расхода
	//sellReturnCorrection - чек коррекции возврата прихода (ФФД ≥ 1.1)
	//buyReturnCorrection - чек коррекции возврата расхода
	Electronically       bool        `json:"electronically"`
	ClientInfo           TClientInfo `json:"clientInfo"`
	CorrectionType       string      `json:"correctionType"` //
	CorrectionBaseDate   string      `json:"correctionBaseDate"`
	CorrectionBaseNumber string      `json:"correctionBaseNumber"`
	Operator             TOperator   `json:"operator"`
	Items                []TPosition `json:"items"`
	Payments             []TPayment  `json:"payments"`
	Total                float64     `json:"total,omitempty"`
}

var email = flag.String("email", "", "email, на которое будут отсылаться все чеки")
var clearLogsProgramm = flag.Bool("clearlogs", true, "очистить логи программы")

//var emulation = flag.Bool("emul", false, "эмуляция")

func main() {
	runDescription := fmt.Sprintf("программа %v версии %v", NAME_OF_PROGRAM, VERSION_OF_PROGRAM)
	fmt.Println(runDescription, "запущена")
	defer fmt.Println(runDescription, "звершена")
	fmt.Println("парсинг параметров запуска программы")
	flag.Parse()
	//инициализация лог файлов
	descrError, err := InitializationLogsFiles()
	defer func() {
		fmt.Println("закрытие дескрипторов лог файлов программы")
		for _, v := range filelogmap {
			if v != nil {
				v.Close()
			}
		}
	}()
	if err != nil {
		log.Panic(descrError)
	}
	logsmap[LOGINFO].Println(runDescription)
	//инициализация директории результатов
	if foundedLogDir, _ := doesFileExist(JSONRES); !foundedLogDir {
		os.Mkdir(JSONRES, 0777)
	}
	//инициализация входных данных
	logsmap[LOGINFO].Println("открытие файла списка чеков")
	f, err := os.Open(DIRINFILES + "checks_header.csv")
	if err != nil {
		descrError := fmt.Sprintf("не удлаось (%v) открыть файл (checks_header.csv) входных данных (шапки чека)", err)
		logsmap[LOGERROR].Println(descrError)
		log.Fatal(descrError)
	}
	defer f.Close()
	csv_red := csv.NewReader(f)
	csv_red.FieldsPerRecord = -1
	csv_red.LazyQuotes = true
	csv_red.Comma = ';'
	logsmap[LOGINFO].Println("чтение списка чеков")
	lines, err := csv_red.ReadAll()
	if err != nil {
		descrError := fmt.Sprintf("не удлаось (%v) прочитать файл (checks_header.csv) входных данных (шапки чека)", err)
		logsmap[LOGERROR].Println(descrError)
		log.Fatal(descrError)
	}
	//перебор всех строчек файла с шапкоми чеков
	countWritedChecks := 0
	countAllChecks := len(lines) - 1
	logsmap[LOGINFO_WITHSTD].Printf("перебор %v чеков", countAllChecks)
	currLine := 0
	for _, line := range lines {
		currLine++
		if currLine == 1 {
			continue //пропускаем нстроку названий столбцов
		}
		//SNO := line[31]
		kassir := line[27]
		kassaName := line[1]
		fn := line[2]
		fd := line[5]
		fp := ""
		//fp := strings.TrimSpace(line[100])
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
		//ищем позиции в файле позиций чека, которые бы соответсвовали бы текущеё строке чека //по номеру ФН и названию кассы
		checkDescrInfo := fmt.Sprintf("(ФД %v (ФП %v) от %v)", fd, fp, dateCh)
		descrInfo := fmt.Sprintf("для чека ФД %v (ФП %v) от %v ищем позиции", fd, fp, dateCh)
		logsmap[LOGINFO].Println(descrInfo)
		findedPositions := findPositions(fn, kassaName, fd, dateCh, strNDS20)
		countOfPositions := len(findedPositions)
		descrInfo = fmt.Sprintf("для чека ФД %v (ФП %v) от %v найдено %v позиций", fd, fp, dateCh, countOfPositions)
		logsmap[LOGINFO].Println(descrInfo)
		if countOfPositions > 0 { //если для чека были найдены позиции
			logsmap[LOGINFO].Println("генерируем json файл")
			jsonres, descError, err := generateCheckCorrection(kassir, dateCh, fd, fp, typeCheck, *email, nal, bez, avance, kred, obmen, findedPositions)
			if err != nil {
				descrError := fmt.Sprintf("ошибка (%v) полчуение json чека коррекции (%v)", descError, checkDescrInfo)
				logsmap[LOGERROR].Println(descrError)
				continue //пропускаем чек
			}
			logsmap[LOGINFO].Println(jsonres)
			as_json, err := json.MarshalIndent(jsonres, "", "\t")
			if err != nil {
				descrError := fmt.Sprintf("ошибка (%v) преобразвания объекта в json для чека %v", err, checkDescrInfo)
				logsmap[LOGERROR].Println(descrError)
				continue //пропускаем чек
			}
			//logsmap[LOGINFO].Println(as_json)
			dir_file_name := fmt.Sprintf("%v%v/", JSONRES, fn)
			if foundedLogDir, _ := doesFileExist(dir_file_name); !foundedLogDir {
				logsmap[LOGINFO].Println("генерируем папку результатов, если раньше она не была сгенерирована")
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
				descrError := fmt.Sprintf("ошибка (%v) создания файла json чека (%v)", err, checkDescrInfo)
				logsmap[LOGERROR].Println(descrError)
				continue //пропускаем чек
			}
			_, err = f.Write(as_json)
			if err != nil {
				descrError := fmt.Sprintf("ошибка (%v) записи json задания в файл (%v)", err, checkDescrInfo)
				logsmap[LOGERROR].Println(descrError)
				f.Close()
				continue //пропускаем чек
			}
			f.Close()
			countWritedChecks++
			//flagsTempOpen := os.O_APPEND | os.O_CREATE | os.O_WRONLY
			//file_name_all_json := fmt.Sprintf("%v.json", fn)
			//file_all_json, err := os.OpenFile(file_name_all_json, flagsTempOpen, 0644)
			//if err != nil {
			//	fmt.Println("ошибка", err)
			//}
			//file_all_json.Write(as_json)
			//file_all_json.Close()
		} else {
			descrError := fmt.Sprintf("для чека ФД %v (ФП %v) от %v не найдены позиции", fd, fp, dateCh)
			logsmap[LOGERROR].Println(descrError)
		} //если для чека были найдены позиции
		//generateCheckCorrection(kassir, dateCh, fd, fp, typeCheck, nal, bez, avance, kred, poss)
	} //перебор чеков
	logsmap[LOGINFO_WITHSTD].Printf("обработано %v из %v чеков", countWritedChecks, countAllChecks)
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
	for _, line := range lines { //перебор всех строк в файле позиций чека
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
			res[currPos]["prepayment"] = "false"
			summPrepayment := line[8]
			//summPrepayment := strings.ReplaceAll(line[8], ",", ".")
			//summPrepayment = strings.TrimSpace(strings.ReplaceAll(summPrepayment, "-", ""))
			//fmt.Println("prepayment", summPrepayment)
			if summPrepayment != "" && summPrepayment != "0,00" {
				//fmt.Println("true")
				res[currPos]["prepayment"] = "true"
			}
			//nds20str := line[12]
			res[currPos]["taxNDS"] = "none"
			//fmt.Println("nds20str", nds20str)
			if strNDS20 != "" && strNDS20 != "0,00" {
				res[currPos]["taxNDS"] = "vat20"
				if summPrepayment != "" && summPrepayment != "0,00" {
					res[currPos]["taxNDS"] = "vat120"
				}
			}
		}
	} //перебор всех строк в файле позиций чека
	//if len(res) > 0 {
	//fmt.Println(res)
	//}
	return res
} //findPositions

// func generateCheckCorrection(kassir, dateCh, fd, fp, typeCheck, email string, nal, bez, avance, kred float64, poss map[int]map[string]string) TCorrectionCheck {
func generateCheckCorrection(kassir, dateCh, fd, fp, typeCheck, email, nal, bez, avance,
	kred, obmen string, poss map[int]map[string]string) (TCorrectionCheck, string, error) {
	var checkCorr TCorrectionCheck
	strInfoAboutCheck := fmt.Sprintf("(ФД %v, ФП %v %v)", fd, fp, dateCh)
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
		//descError := fmt.Sprintf("ошибка тип чека коррекциии для типа %v - не определён", typeCheck)
		descError := fmt.Sprintf("ошибка (для типа %v не определён тип чека коррекциии) %v", typeCheck, strInfoAboutCheck)
		logsmap[LOGERROR].Println(descError)
		return checkCorr, descError, errors.New("ошибка определения типа чека коррекции")
	}
	if email == "" {
		checkCorr.Electronically = false
	} else {
		checkCorr.Electronically = true
	}
	checkCorr.CorrectionType = "self"
	checkCorr.CorrectionBaseDate = dateCh
	checkCorr.CorrectionBaseNumber = ""
	checkCorr.ClientInfo.EmailOrPhone = email
	checkCorr.Operator.Name = kassir
	if nal != "" && nal != "0.00" {
		nalch, err := strconv.ParseFloat(nal, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для налчиного расчёта %v", err, nal, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "cash", Sum: nalch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if bez != "" && bez != "0.00" {
		bezch, err := strconv.ParseFloat(bez, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для безналичного расчёта %v", err, bez, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "electronically", Sum: bezch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if avance != "" && avance != "0.00" {
		avancech, err := strconv.ParseFloat(avance, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для суммы зачета аванса %v", err, avance, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "prepaid", Sum: avancech}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if kred != "" && kred != "0.00" {
		kredch, err := strconv.ParseFloat(kred, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для суммы оплаты в рассрочку %v", err, kred, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "credit", Sum: kredch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	if obmen != "" && obmen != "0.00" {
		obmench, err := strconv.ParseFloat(obmen, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для суммы оплаты встречным представлением %v", err, obmen, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "other", Sum: obmench}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	currFP := fp
	if currFP == "" {
		currFP = fd
	}
	//в тег 1192 - записываем ФП (если нет ФП, то записываем ФД)
	if currFP != "" {
		newAdditionalAttribute := TPosition{Type: "additionalAttribute"}
		newAdditionalAttribute.Value = currFP
		newAdditionalAttribute.Print = true
		checkCorr.Items = append(checkCorr.Items, newAdditionalAttribute)
	}
	for _, pos := range poss {
		newPos := TPosition{Type: "position"}
		newPos.Name = pos["Name"]
		qch, err := strconv.ParseFloat(pos["Quantity"], 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v количества %v", err, pos["Quantity"], strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		newPos.Quantity = qch
		prch, err := strconv.ParseFloat(pos["Price"], 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v цены %v", err, pos["Price"], strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		newPos.Price = prch
		sch, err := strconv.ParseFloat(pos["Amount"], 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v суммы %v", err, pos["Amount"], strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		newPos.Amount = sch
		newPos.MeasurementUnit = 0
		if pos["prepayment"] == "true" {
			newPos.PaymentMethod = "fullPrepayment"
			newPos.PaymentObject = "payment"
		} else {
			newPos.PaymentMethod = "fullPayment"
			newPos.PaymentObject = "commodity"
		}
		newPos.Tax = new(TTaxNDS)
		newPos.Tax.Type = pos["taxNDS"]
		checkCorr.Items = append(checkCorr.Items, newPos)
	} //запись всех позиций чека
	return checkCorr, "", nil
}
func doesFileExist(fullFileName string) (found bool, err error) {
	found = false
	if _, err = os.Stat(fullFileName); err == nil {
		// path/to/whatever exists
		found = true
	}
	return
}
func formatMyDate(dt string) string {
	y := dt[6:]
	m := dt[3:5]
	d := dt[0:2]
	res := y + "." + m + "." + d
	return res
}

func InitializationLogsFiles() (string, error) {
	var err error
	var descrError string
	if foundedLogDir, _ := doesFileExist(LOGSDIR); !foundedLogDir {
		os.Mkdir(LOGSDIR, 0777)
	}
	clearLogsDescr := fmt.Sprintf("Очистить логи программы %v", *clearLogsProgramm)
	fmt.Println(clearLogsDescr)
	fmt.Println("инициализация лог файлов программы")
	if foundedLogDir, _ := doesFileExist(LOGSDIR); !foundedLogDir {
		os.Mkdir(LOGSDIR, 0777)
	}
	//if foundedLogDir, _ := doesFileExist(RESULTSDIR); !foundedLogDir {
	//	os.Mkdir(RESULTSDIR, 0777)
	//}
	filelogmap, logsmap, descrError, err = initializationLogs(*clearLogsProgramm, LOGINFO, LOGERROR, LOGSKIP_LINES, LOGOTHER)
	if err != nil {
		descrMistake := fmt.Sprintf("ошибка инициализации лог файлов %v", descrError)
		fmt.Fprint(os.Stderr, descrMistake)
		return descrMistake, err
		//log.Panic(descrMistake)
	}
	fmt.Println("лог файлы инициализированы в папке " + LOGSDIR)
	multwriterLocLoc := io.MultiWriter(logsmap[LOGINFO].Writer(), os.Stdout)
	logsmap[LOGINFO_WITHSTD] = log.New(multwriterLocLoc, LOG_PREFIX+"_"+strings.ToUpper(LOGINFO)+" ", log.LstdFlags)
	logsmap[LOGINFO].Println(clearLogsDescr)
	return "", nil
}

func initializationLogs(clearLogs bool, logstrs ...string) (map[string]*os.File, map[string]*log.Logger, string, error) {
	var reserr, err error
	reserr = nil
	filelogmapLoc := make(map[string]*os.File)
	logsmapLoc := make(map[string]*log.Logger)
	descrError := ""
	for _, logstr := range logstrs {
		filenamelogfile := logstr + "logs.txt"
		preflog := LOG_PREFIX + "_" + strings.ToUpper(logstr)
		fullnamelogfile := LOGSDIR + filenamelogfile
		filelogmapLoc[logstr], logsmapLoc[logstr], err = intitLog(fullnamelogfile, preflog, clearLogs)
		if err != nil {
			descrError = fmt.Sprintf("ошибка инициализации лог файла %v с ошибкой %v", fullnamelogfile, err)
			fmt.Fprintln(os.Stderr, descrError)
			reserr = err
			break
		}
	}
	return filelogmapLoc, logsmapLoc, descrError, reserr
}

func intitLog(logFile string, pref string, clearLogs bool) (*os.File, *log.Logger, error) {
	flagsTempOpen := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if clearLogs {
		flagsTempOpen = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	}
	f, err := os.OpenFile(logFile, flagsTempOpen, 0644)
	multwr := io.MultiWriter(f)
	//if pref == LOG_PREFIX+"_INFO" {
	//	multwr = io.MultiWriter(f, os.Stdout)
	//}
	flagsLogs := log.LstdFlags
	if pref == LOG_PREFIX+"_ERROR" {
		multwr = io.MultiWriter(f, os.Stderr)
		flagsLogs = log.LstdFlags | log.Lshortfile
	}
	if err != nil {
		fmt.Println("Не удалось создать лог файл ", logFile, err)
		return nil, nil, err
	}
	loger := log.New(multwr, pref+" ", flagsLogs)
	return f, loger, nil
}
