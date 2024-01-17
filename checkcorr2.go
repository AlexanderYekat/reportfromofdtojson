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

const COLFD = "FD"
const COLFN = "FN"
const COLREG = "REG"
const COLFP = "FP"
const COLKASSIR = "kassir"
const COLINNKASSIR = "innkassir"
const COLKASSA_NAME = "kassaName"
const COLDATE = "date"
const COLTYPEOPER = "typeOper"
const COLTYPECHECK = "typeCheck"
const COLNAL = "nal"
const COLBEZ = "bez"
const COLAVANCE = "avance"
const COLCREDIT = "credit"
const COLOBMEN = "obmen"
const COLSUMMNDS20 = "summNDS20"
const COLNAMEGOOD = "name"
const COLQUANTITY = "quantity"
const COLPRICE = "price"
const COLAMOUNT = "amount"
const COLMARK = "mark"
const COLSUMMPREPAYPOS = "summPrepay"
const COLPREDMET = "predmet"
const COLSPOSOB = "sposob"
const COLSTAVKANDS = "stavkaNDS"
const COLEDIN = "edin"

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
const DIRINFILESANDUNION = "./infiles/union/"

const VERSION_OF_PROGRAM = "2024_01_17_06"
const NAME_OF_PROGRAM = "формирование json заданий чеков коррекции на основании отчетов из ОФД (xsl-csv)"

type TClientInfo struct {
	EmailOrPhone string `json:"emailOrPhone"`
}

type TTaxNDS struct {
	Type string `json:"type,omitempty"`
}
type TProductCodes struct {
	Undefined string `json:"undefined,omitempty"` //32 символа только
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
	MeasurementUnit string   `json:"measurementUnit,omitempty"`
	PaymentMethod   string   `json:"paymentMethod,omitempty"`
	PaymentObject   string   `json:"paymentObject,omitempty"`
	Tax             *TTaxNDS `json:"tax,omitempty"`
	//fot type tag1192 //AdditionalAttribute
	Value        string         `json:"value,omitempty"`
	Print        bool           `json:"print,omitempty"`
	ProductCodes *TProductCodes `json:"productCodes,omitempty"`
}

type TOperator struct {
	Name  string `json:"name"`
	Vatin string `json:"vatin,omitempty"`
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

var command = flag.String("command", "getjsons", "команды getjsons - получить json команды, union - объединить файлы чеков")
var numColKassir = flag.Int("col_kassir", 27, "колонка фамилии кассира")
var numColInnKassir = flag.Int("col_Inn_kassir", -1, "колонка ИНН кассира")
var numColKassaNameCh = flag.Int("colKassaNameCh", 1, "колонка названии кассы")
var numColFNCh = flag.Int("colFNCh", 2, "колонка номер ФН")
var numColRegCh = flag.Int("colRegCh", -1, "колонка регномер кассы")
var numColFDCh = flag.Int("colFDCh", 5, "колонка номер ФД")
var numColFPCh = flag.Int("colFPCh", -1, "колонка ФП")
var numColDateCh = flag.Int("colDate", 6, "колонка даты чека")
var numColTypeOper = flag.Int("colTypeOper", 8, "колонка типа чека (приход, возврат и прочее)")
var numColTypeCheck = flag.Int("colTypeCheck", -1, "колонка типа чека (чек, коррекция и прочее)")
var numColNal = flag.Int("colNal", 10, "колонка суммы наличной оплаты")
var numColBez = flag.Int("colBez", 11, "колонка суммы безналичной оплаты")
var numColAv = flag.Int("colAvPay", 13, "колонка сумма зачета аванса")
var numColCr = flag.Int("colCred", 14, "колонка суммы рассрочки")
var numColObm = flag.Int("colObm", 15, "колонка суммы встречным представлением")
var numColSumNDS20 = flag.Int("colSumNDS20", 17, "колонка суммы НДС 20%")
var numColName = flag.Int("colName", 0, "колонка названия товара")
var numColQuant = flag.Int("colQuant", 1, "колонка колчиества")
var numColMark = flag.Int("colMark", -1, "колонка марок")
var numColPrice = flag.Int("colPrice", 2, "колонка цены")
var numColAmountPos = flag.Int("colAmountPos", 3, "колонка суммы позиции")
var numColSummPrePay = flag.Int("colSummPrepay", 8, "колонка суммы предоплаты позиции (бред от ОФД контур, по которой можно определить предмет рачсёта платёж)")
var numColKassaNamePos = flag.Int("colKassaNamePos", 20, "колонка название кассы в таблице товаров")
var numColFDPos = flag.Int("colFDPos", 7, "колонка ФД в таблице товаров")

var numColPred = flag.Int("colPred", -1, "колонка предмета расчета")
var numColSpos = flag.Int("colSpos", -1, "колонка способа расчета")
var numColStavkaNDS = flag.Int("colSatavkaNDS", -1, "колонка ставки НДС")
var numColEdIzm = flag.Int("colEdIzm", -1, "колонка единицы измерения")

//var numCol = flag.Int("col", -1, "колонка ")

//var numCol = flag.Int("col", -1, "колонка ")
//var numCol = flag.Int("col", -1, "колонка ")

var email = flag.String("email", "", "email, на которое будут отсылаться все чеки")
var clearLogsProgramm = flag.Bool("clearlogs", true, "очистить логи программы")
var CollumnsOfHeadCheck map[string]int
var CollumnsOfPositionsCheck map[string]int

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
	//инициализация колонок файлов
	logsmap[LOGINFO].Println("инициализация номеров колонок")
	CollumnsOfHeadCheck, CollumnsOfPositionsCheck, descError, err := inicizlizationsCollimns()
	if err != nil {
		descrError := fmt.Sprintf("ошибка (%v) инициализации номеров колонок)", descError)
		logsmap[LOGERROR].Println(descrError)
		log.Fatal(descrError)
	}
	logsmap[LOGINFO].Println(CollumnsOfHeadCheck)
	logsmap[LOGINFO].Println(CollumnsOfPositionsCheck)
	//инициализация директории результатов
	if foundedLogDir, _ := doesFileExist(JSONRES); !foundedLogDir {
		os.Mkdir(JSONRES, 0777)
	}
	if *command == "union" {
		logsmap[LOGINFO_WITHSTD].Println("объедиение файлов чеков начато")
		unuinToHeaderCheckFiles()
		logsmap[LOGINFO_WITHSTD].Println("объедиение файлов чеков завершено")
		return
		//log.Panic("штатная паника")
	}
	logsmap[LOGINFO_WITHSTD].Println("формирование json заданий начато")
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
			continue //пропускаем настройку названий столбцов
		}
		descrInfo := fmt.Sprintf("обработка строки %v из %v", currLine-1, countAllChecks)
		logsmap[LOGINFO].Println(descrInfo)
		logsmap[LOGINFO].Println(line)
		//SNO := line[31]
		kassir := line[CollumnsOfHeadCheck[COLKASSIR]]
		innkassir := ""
		if CollumnsOfHeadCheck[COLINNKASSIR] != -1 {
			innkassir = line[CollumnsOfHeadCheck[COLINNKASSIR]]
		}
		kassaName := ""
		if CollumnsOfHeadCheck[COLKASSA_NAME] != -1 {
			kassaName = line[CollumnsOfHeadCheck[COLKASSA_NAME]]
		}
		fn := line[CollumnsOfHeadCheck[COLFN]]
		reg := ""
		if CollumnsOfHeadCheck[COLREG] != -1 {
			reg = line[CollumnsOfHeadCheck[COLREG]]
		}
		if kassaName == "" {
			kassaName = reg
		}
		if kassaName == "" {
			logsmap[LOGERROR].Printf("строка %v пропущена, так как в ней не определено название кассы", line)
			continue
		}
		fd := line[CollumnsOfHeadCheck[COLFD]]
		fp := ""
		if CollumnsOfHeadCheck[COLFP] != -1 {
			fp = line[CollumnsOfHeadCheck[COLFP]]
		}
		dateCh := formatMyDate(line[CollumnsOfHeadCheck[COLDATE]])
		typeCheck := line[CollumnsOfHeadCheck[COLTYPEOPER]]
		//sum := line[CollumnsOfHeadCheck[COLAMOUNT]]
		nal := formatMyNumber(line[CollumnsOfHeadCheck[COLNAL]])
		bez := formatMyNumber(line[CollumnsOfHeadCheck[COLBEZ]])
		//sumpplat := strings.ReplaceAll(line[12], ",", ".")
		avance := formatMyNumber(line[CollumnsOfHeadCheck[COLAVANCE]])
		kred := formatMyNumber(line[CollumnsOfHeadCheck[COLCREDIT]])
		obmen := formatMyNumber(line[CollumnsOfHeadCheck[COLOBMEN]])
		strNDS20 := line[CollumnsOfHeadCheck[COLSUMMNDS20]]
		//sumBezNDS := line[32]
		//countOfPos := line[24]
		//if fd == "2039" || fd == "2060" || fd == "2040" || fd == "2041" || fd == "2330" || fd == "2331" || fd == "2332" {
		//	fmt.Println("пропускаем чек, по которому уже был пробит чек коррекции")
		//	continue
		//}
		//ищем позиции в файле позиций чека, которые бы соответсвовали бы текущеё строке чека //по номеру ФН и названию кассы
		checkDescrInfo := fmt.Sprintf("(ФД %v (ФП %v) от %v)", fd, fp, dateCh)
		descrInfo = fmt.Sprintf("для чека ФД %v (ФП %v) от %v ищем позиции", fd, fp, dateCh)
		logsmap[LOGINFO].Println(descrInfo)
		findedPositions := findPositions(fn, kassaName, fd, dateCh, strNDS20, CollumnsOfPositionsCheck)
		countOfPositions := len(findedPositions)
		//countOfPositions = 0
		descrInfo = fmt.Sprintf("для чека ФД %v (ФП %v) от %v найдено %v позиций", fd, fp, dateCh, countOfPositions)
		logsmap[LOGINFO].Println(descrInfo)
		if countOfPositions > 0 { //если для чека были найдены позиции
			logsmap[LOGINFO].Println("генерируем json файл")
			jsonres, descError, err := generateCheckCorrection(kassir, innkassir, dateCh, fd, fp, typeCheck, *email, nal, bez, avance, kred, obmen, findedPositions)
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
		//generateCheckCorrection(kassir, innkassir, dateCh, fd, fp, typeCheck, nal, bez, avance, kred, poss)
	} //перебор чеков
	logsmap[LOGINFO_WITHSTD].Println("формирование json заданий завершено")
	logsmap[LOGINFO_WITHSTD].Printf("обработано %v из %v чеков", countWritedChecks, countAllChecks)
	logsmap[LOGINFO_WITHSTD].Println("проверка завершена")
}

func findPositions(fn, kassaName, fd, dateCh, strNDS20 string, colOfTable map[string]int) map[int]map[string]string {
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
		fdCurr := ""
		regCur := ""
		//fnCurr := ""
		if colOfTable[COLFD] != -1 {
			fdCurr = line[colOfTable[COLFD]]
		}
		currKassaName := ""
		if colOfTable[COLREG] != -1 {
			currKassaName = line[colOfTable[COLKASSA_NAME]]
		}
		if fdCurr == "" {
			descrErr := ""
			regCur, _, fdCurr, descrErr, err = getRegFnFdFromName(currKassaName)
			if err != nil {
				logsmap[LOGERROR].Println(descrErr)
				log.Fatal("не удалось получить регистрационный номер, номер ФД, ФН из имени кассы", err)
			}
			currKassaName = regCur
		}
		if fdCurr == fd && kassaName == currKassaName {
			logsmap[LOGINFO].Println("Найдена строка", line)
			currPos++
			res[currPos] = make(map[string]string)
			res[currPos]["Name"] = line[colOfTable[COLNAMEGOOD]]
			res[currPos]["Quantity"] = formatMyNumber(line[colOfTable[COLQUANTITY]])
			res[currPos]["Price"] = formatMyNumber(line[colOfTable[COLPRICE]])
			res[currPos]["Amount"] = formatMyNumber(line[colOfTable[COLAMOUNT]])
			logsmap[LOGINFO].Println("colOfTable[COLMARK]=", colOfTable[COLMARK])
			if colOfTable[COLMARK] != -1 {
				res[currPos]["mark"] = line[colOfTable[COLMARK]]
				logsmap[LOGINFO].Println("res[currPos][\"mark\"]=", line[colOfTable[COLMARK]])
			}

			res[currPos]["prepayment"] = "false"
			summPrepayment := ""
			if colOfTable[COLSUMMPREPAYPOS] != -1 {
				summPrepayment = line[colOfTable[COLSUMMPREPAYPOS]]
			}
			//summPrepayment := strings.ReplaceAll(line[8], ",", ".")
			//summPrepayment = strings.TrimSpace(strings.ReplaceAll(summPrepayment, "-", ""))
			//fmt.Println("prepayment", summPrepayment)
			if summPrepayment != "" && summPrepayment != "0,00" && summPrepayment != "0,00 ₽" {
				//fmt.Println("true")
				res[currPos]["prepayment"] = "true"
			}
			//nds20str := line[12]
			res[currPos]["taxNDS"] = "none"
			//fmt.Println("nds20str", nds20str)
			if strNDS20 != "" && strNDS20 != "0,00" && strNDS20 != "0,00 ₽" {
				res[currPos]["taxNDS"] = "vat20"
				if summPrepayment != "" && summPrepayment != "0,00" && summPrepayment != "0,00 ₽" {
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

// func generateCheckCorrection(kassir, innkassir, dateCh, fd, fp, typeCheck, email string, nal, bez, avance, kred float64, poss map[int]map[string]string) TCorrectionCheck {
func generateCheckCorrection(kassir, innkassir, dateCh, fd, fp, typeCheck, email, nal, bez, avance,
	kred, obmen string, poss map[int]map[string]string) (TCorrectionCheck, string, error) {
	var checkCorr TCorrectionCheck
	strInfoAboutCheck := fmt.Sprintf("(ФД %v, ФП %v %v)", fd, fp, dateCh)
	checkCorr.Type = ""
	chekcCorrTypeLoc := ""
	typeCheck = strings.ToLower(typeCheck)
	if typeCheck == "приход" {
		chekcCorrTypeLoc = "sellCorrection"
	}
	if typeCheck == "возврат прихода" {
		chekcCorrTypeLoc = "sellReturnCorrection"
	}
	if typeCheck == "расход" {
		chekcCorrTypeLoc = "buyCorrection"
	}
	if typeCheck == "возврат расхода" {
		chekcCorrTypeLoc = "buyReturnCorrection"
	}
	//Кассовый чек. Приход.
	if chekcCorrTypeLoc == "" {
		if strings.Contains(typeCheck, "возврат расхода") {
			chekcCorrTypeLoc = "buyReturnCorrection"
		} else if strings.Contains(typeCheck, "возврат прихода") {
			chekcCorrTypeLoc = "sellReturnCorrection"
		} else if strings.Contains(typeCheck, "приход") {
			chekcCorrTypeLoc = "sellCorrection"
		} else if strings.Contains(typeCheck, "расход") {
			chekcCorrTypeLoc = "buyCorrection"
		}
	}
	checkCorr.Type = chekcCorrTypeLoc
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
	checkCorr.Operator.Vatin = innkassir
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
	//в тег 1192 - записываем ФП (если нет ФП, то записываем ФД)
	if fd != "" {
		newAdditionalAttribute := TPosition{Type: "userAttribute"}
		newAdditionalAttribute.Name = "ФД"
		newAdditionalAttribute.Value = fd
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
		newPos.MeasurementUnit = "piece"
		if pos["prepayment"] == "true" {
			newPos.PaymentMethod = "fullPrepayment"
			newPos.PaymentObject = "payment"
		} else {
			newPos.PaymentMethod = "fullPayment"
			newPos.PaymentObject = "commodity"
		}
		newPos.Tax = new(TTaxNDS)
		newPos.Tax.Type = pos["taxNDS"]
		//logsmap[LOGINFO].Println("pos = ", pos)
		//logsmap[LOGINFO].Println("pos = ", pos)
		if currMark, ok := pos["mark"]; ok {
			byte32onlyCut := min(32, len(currMark))
			logsmap[LOGINFO].Println("mark zap = ", currMark[:byte32onlyCut])
			newPos.ProductCodes = new(TProductCodes)
			newPos.ProductCodes.Undefined = currMark[:byte32onlyCut]
		}
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
	//28.11.2023
	//09.01.2024 15:42
	indOfPoint := strings.Index(dt, ".")
	if indOfPoint == 4 {
		return dt
	}
	y := dt[6:10]
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

func inicizlizationsCollimns() (map[string]int, map[string]int, string, error) {
	resHeaderCollumns := make(map[string]int)
	resPositionsCollumns := make(map[string]int)

	resHeaderCollumns[COLFD] = *numColFDCh
	resHeaderCollumns[COLFN] = *numColFNCh
	resHeaderCollumns[COLREG] = *numColRegCh
	resHeaderCollumns[COLFP] = *numColFPCh

	resHeaderCollumns[COLKASSIR] = *numColKassir
	resHeaderCollumns[COLINNKASSIR] = *numColInnKassir
	resHeaderCollumns[COLKASSA_NAME] = *numColKassaNameCh
	resHeaderCollumns[COLDATE] = *numColDateCh
	resHeaderCollumns[COLTYPEOPER] = *numColTypeOper
	resHeaderCollumns[COLTYPECHECK] = *numColTypeCheck
	resHeaderCollumns[COLNAL] = *numColNal
	resHeaderCollumns[COLBEZ] = *numColBez
	resHeaderCollumns[COLAVANCE] = *numColAv
	resHeaderCollumns[COLCREDIT] = *numColCr
	resHeaderCollumns[COLOBMEN] = *numColObm

	resHeaderCollumns[COLSUMMNDS20] = *numColSumNDS20

	resPositionsCollumns[COLFD] = *numColFDPos
	resPositionsCollumns[COLNAMEGOOD] = *numColName
	resPositionsCollumns[COLQUANTITY] = *numColQuant
	resPositionsCollumns[COLMARK] = *numColMark
	resPositionsCollumns[COLPRICE] = *numColPrice
	resPositionsCollumns[COLAMOUNT] = *numColAmountPos
	resPositionsCollumns[COLSUMMPREPAYPOS] = *numColSummPrePay
	resPositionsCollumns[COLAMOUNT] = *numColAmountPos
	resPositionsCollumns[COLKASSA_NAME] = *numColKassaNamePos

	resPositionsCollumns[COLPREDMET] = *numColPred
	resPositionsCollumns[COLSPOSOB] = *numColSpos
	resPositionsCollumns[COLSTAVKANDS] = *numColStavkaNDS
	resPositionsCollumns[COLEDIN] = *numColEdIzm

	return resHeaderCollumns, resPositionsCollumns, "", nil
}

func unuinToHeaderCheckFiles() {
	fUnion, err := os.Create(DIRINFILESANDUNION + "checks_header_union.csv")
	if err != nil {
		descrError := fmt.Sprintf("не удлаось (%v) открыть файл (checks_header_union.csv) входных данных (шапки чека)", err)
		logsmap[LOGERROR].Println(descrError)
		log.Fatal(descrError)
	}
	defer fUnion.Close()
	csv_union := csv.NewWriter(fUnion)
	csv_union.Comma = ';'
	defer csv_union.Flush()
	csv_union.Write([]string{"FN", "KASSA_NAME", "FD", "FP", "KASSIR", "DATE", "TYPE", "TYPEOPERATION", "NAL", "BEZNAL", "AVANCE", "CREDIT", "OBMEN", "SUMMNDS20"})
	//logsmap[LOGINFO].Println("чтение списка чеков")

	//перебор некорректных чеков
	f, err := os.Open(DIRINFILESANDUNION + "checks_header_no_correction.csv")
	if err != nil {
		descrError := fmt.Sprintf("не удлаось (%v) открыть файл (checks_header_no_correction.csv) входных данных (шапки чека)", err)
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
		descrError := fmt.Sprintf("не удлаось (%v) прочитать файл (checks_header_no_correction.csv) входных данных (шапки чека)", err)
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
		descrInfo := fmt.Sprintf("обработка строки %v из %v", currLine-1, countAllChecks)
		logsmap[LOGINFO].Println(descrInfo)
		logsmap[LOGINFO].Println(line)
		//SNO := line[31]
		//kassir := line[CollumnsOfHeadCheck[COLKASSIR]]
		//kassaName := line[CollumnsOfHeadCheck[COLKASSA_NAME]]
		fn := line[2]
		kassa_name := line[1]
		fd := line[5]
		kassir := line[27]
		dateCheck := formatMyDate(line[6])
		typeCheck := line[3]
		typeCheckOper := line[8]
		nal := strings.ReplaceAll(line[10], ",", ".")
		nal = strings.ReplaceAll(nal, "-", "")
		bez := strings.ReplaceAll(line[11], ",", ".")
		bez = strings.ReplaceAll(bez, "-", "")
		avance := strings.ReplaceAll(line[13], ",", ".")
		avance = strings.ReplaceAll(avance, "-", "")
		cred := strings.ReplaceAll(line[14], ",", ".")
		cred = strings.ReplaceAll(cred, "-", "")
		obmen := strings.ReplaceAll(line[15], ",", ".")
		obmen = strings.ReplaceAll(obmen, "-", "")
		summNDS20 := line[17]
		//"FN;KASSA_NAME;FD;FP;KASSIR;DATE;TYPE;NAL;BEZNAL;AVANCE;CREDIT;OBMEN;SUMMNDS20;"}
		//перебор файла всех чеков
		fieldsValue, derscrError, err := findCheckByRekviz(fd)
		if err != nil {
			descrError := fmt.Sprintf("ошибка (%v): не удлаось найти чек %v в файле всех чеков", derscrError, fd)
			logsmap[LOGERROR].Println(descrError)
			//log.Fatal(descrError)
			continue
		}
		if len(fieldsValue) == 0 {
			descrError := fmt.Sprintf("не удлаось найти чек %v в файле всех чеков", fd)
			logsmap[LOGERROR].Println(descrError)
			continue
		}
		fp := fieldsValue["fp"]
		csv_union.Write([]string{fn, kassa_name, fd, fp, kassir, dateCheck, typeCheck, typeCheckOper, nal, bez, avance, cred, obmen, summNDS20})
		countWritedChecks++
	}
	logsmap[LOGINFO_WITHSTD].Printf("обработано %v из %v чеков", countWritedChecks, countAllChecks)
}

func findCheckByRekviz(fd string) (map[string]string, string, error) {
	var resFieldsValue map[string]string
	//перебор некорректных чеков
	f, err := os.Open(DIRINFILESANDUNION + "checks_header_all_checks.csv")
	if err != nil {
		descrError := fmt.Sprintf("не удлаось (%v) открыть файл (checks_header_no_correction.csv) входных данных (шапки чека)", err)
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
	//countWritedChecks := 0
	countAllChecks := len(lines) - 1
	//logsmap[LOGINFO_WITHSTD].Printf("перебор %v чеков", countAllChecks)
	currLine := 0
	for _, line := range lines {
		currLine++
		if currLine == 1 {
			continue //пропускаем нстроку названий столбцов
		}
		descrInfo := fmt.Sprintf("обработка строки %v из %v", currLine-1, countAllChecks)
		logsmap[LOGINFO].Println(descrInfo)
		logsmap[LOGINFO].Println(line)
		//SNO := line[31]
		//kassir := line[CollumnsOfHeadCheck[COLKASSIR]]
		//kassaName := line[CollumnsOfHeadCheck[COLKASSA_NAME]]
		//fnCurr := line[2]
		//kassa_nameCurr := line[1]
		fdCurr := line[5]
		if fdCurr == fd {
			resFieldsValue = make(map[string]string)
			fp := line[4]
			resFieldsValue["fp"] = fp
			break
		}
	}
	return resFieldsValue, "", nil
}

func getRegFnFdFromName(nameOfKassa string) (reg, fn, fd, desrErr string, err error) {
	var errLoc error
	reg = ""
	fn = ""
	fd = ""
	//0006989495006718_7280440500080718_5045
	indFirstPr := strings.Index(nameOfKassa, "_")
	if indFirstPr == -1 {
		desrErr = fmt.Sprintf("не получилось разобрать имя кассы %v. номер ФН и номер ФД не получены", nameOfKassa)
		errLoc = fmt.Errorf(desrErr)
		logsmap[LOGERROR].Println(desrErr)
		return reg, fn, fd, desrErr, errLoc
	}
	reg = nameOfKassa[:indFirstPr]
	indSecondPr := strings.Index(nameOfKassa[indFirstPr+1:], "_")
	fn = nameOfKassa[indFirstPr+1 : indFirstPr+1+indSecondPr]
	if indSecondPr == -1 {
		desrErr = fmt.Sprintf("не получилось разобрать имя кассы %v. номер ФД не получен", nameOfKassa)
		errLoc = fmt.Errorf(desrErr)
		logsmap[LOGERROR].Println(desrErr)
		return reg, fn, fd, desrErr, errLoc
	}
	fd = nameOfKassa[indFirstPr+1+indSecondPr+1:]
	//logsmap[LOGINFO].Printf("получены номер ФН %v, номер ФД %v и регистрационный номер %v из строки %v", fn, fd, reg, nameOfKassa)
	return reg, fn, fd, "", nil
}

func formatMyNumber(num string) string {
	var res string
	//3 477,00 ₽
	res = strings.ReplaceAll(num, ",", ".")
	res = strings.ReplaceAll(res, "-", "")
	res = strings.ReplaceAll(res, " ", "")
	res = strings.ReplaceAll(res, " ₽", "")
	return res
}
