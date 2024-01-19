package main

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const VERSION_OF_PROGRAM = "2024_01_17_06"
const NAME_OF_PROGRAM = "формирование json заданий чеков коррекции на основании отчетов из ОФД (xsl-csv)"

const EMAILFIELD = "email"
const NOPRINTFIELD = "electronically"
const NAMETYPEOFMARK = "TYPEMARK"

const COLREGNUMKKT = "regnumkkt"
const COLFNKKT = "fnkkt"
const COLNAMEOFKKT = "nameofkkt"

const COLFD = "fd"
const COLFP = "fp"
const COLNAL = "nal"
const COLBEZ = "bez"
const COLCREDIT = "credit"
const COLAVANCE = "avance"
const COLVSTRECHPREDST = "vstrechpredst"
const COLKASSIR = "kassir"
const COLINNKASSIR = "innkassir"
const COLDATE = "date"
const COLOSN = "osn"
const COLTAG1054 = "tag1054"
const COLTYPECHECK = "typeCheck"
const COLLINK = "link"
const COLBINDHEADFIELDKASSA = "bindheadfieldkassa"
const COLBINDHEADDIELDCHECK = "bindheadfieldcheck"

const COLNAME = "name"
const COLQUANTITY = "quantity"
const COLPRICE = "price"
const COLAMOUNTPOS = "amountpos"
const COLPREDMET = "predmet"
const COLSPOSOB = "sposob"

// const COLSTAVKANDS = "stavkaNDS"
const COLSTAVKANDS0 = "stavkaNDS"
const COLSTAVKANDS10 = "stavkaNDS"
const COLSTAVKANDS20 = "stavkaNDS"
const COLSTAVKANDS110 = "stavkaNDS"
const COLSTAVKANDS120 = "stavkaNDS"
const COLMARK = "mark"
const COLBINDPOSFIELDKASSA = "bindposfieldkassa"
const COLBINDPOSFIELDCHECK = "bindposfieldcheck"

const STAVKANDSNONE = "none"
const STAVKANDS0 = "vat0"
const STAVKANDS10 = "vat10"
const STAVKANDS20 = "vat20"
const STAVKANDS110 = "vat110"
const STAVKANDS120 = "vat120"

//const COLSUMMNDS20 = "summNDS20"
//const COLSUMMPREPAYPOS = "summPrepay"

//const COLEDIN = "edin"

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
const DIROFREQUEST = "./request/"

type TClientInfo struct {
	EmailOrPhone string `json:"emailOrPhone"`
}

type TTaxNDS struct {
	Type string `json:"type,omitempty"`
}
type TProductCodesAtol struct {
	Undefined    string `json:"undefined,omitempty"` //32 символа только
	Code_EAN_8   string `json:"ean8,omitempty"`
	Code_EAN_13  string `json:"ean13,omitempty"`
	Code_ITF_14  string `json:"itf14,omitempty"`
	Code_GS_1    string `json:"gs10,omitempty"`
	Tag1305      string `json:"gs1m,omitempty"`
	Code_KMK     string `json:"short,omitempty"`
	Code_MI      string `json:"furs,omitempty"`
	Code_EGAIS_2 string `json:"egais20,omitempty"`
	Code_EGAIS_3 string `json:"egais30,omitempty"`
	Code_F_1     string `json:"f1,omitempty"`
	Code_F_2     string `json:"f2,omitempty"`
	Code_F_3     string `json:"f3,omitempty"`
	Code_F_4     string `json:"f4,omitempty"`
	Code_F_5     string `json:"f5,omitempty"`
	Code_F_6     string `json:"f6,omitempty"`
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
	Value        string             `json:"value,omitempty"`
	Print        bool               `json:"print,omitempty"`
	ProductCodes *TProductCodesAtol `json:"productCodes,omitempty"`
	ImcParams    *TImcParams        `json:"imcParams,omitempty"`
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

// json чека в ОФД.RU
type TReceipt struct {
	Version  int `json:"Version"`
	Document struct {
		// document fields
		Items []TItem `json:"Items"`
		// other fields
	} `json:"Document"`
}
type TItem struct {
	Name                      string
	Price                     float64
	Quantity                  float64
	Nds18_TotalSumm           float64
	Nds10_TotalSumm           float64
	Nds00_TotalSumm           float64
	NdsNA_TotalSumm           float64
	Nds18_CalculatedTotalSumm float64
	Nds10_CalculatedTotalSumm float64
	Total                     float64
	CalculationMethod         int
	SubjectType               int
	ProductCode               TProductCodeOFD
	NDS_PieceSumm             float64
	NDS_Rate                  int
	NDS_Summ                  float64
	ProductUnitOfMeasure      int
}
type TProductCodeOFD struct {
	Code_Undefined string
	Code_EAN_8     string
	Code_EAN_13    string
	Code_ITF_14    string
	Code_GS_1      string
	Code_GS_1M     string
	Code_KMK       string
	Code_MI        string
	Code_EGAIS_2   string
	Code_EGAIS_3   string
	Code_F_1       string
	Code_F_2       string
	Code_F_3       string
	Code_F_4       string
	Code_F_5       string
	Code_F_6       string
}

type TItemInfoCheckResult struct {
	ImcCheckFlag              bool `json:"imcCheckFlag"`
	ImcCheckResult            bool `json:"imcCheckResult"`
	ImcStatusInfo             bool `json:"imcStatusInfo"`
	ImcEstimatedStatusCorrect bool `json:"imcEstimatedStatusCorrect"`
	EcrStandAloneFlag         bool `json:"ecrStandAloneFlag"`
}

type TImcParams struct {
	ImcType             string                `json:"imcType"`
	Imc                 string                `json:"ims"`
	ItemEstimatedStatus string                `json:"itemEstimatedStatus"`
	ImcModeProcessing   int                   `json:"imcModeProcessing"`
	ImcBarcode          string                `json:"imcBarcode,omitempty"`
	ItemInfoCheckResult *TItemInfoCheckResult `json:"itemInfoCheckResult,omitempty"`
	ItemQuantity        float64               `json:"itemQuantity,omitempty"`
	ItemUnits           string                `json:"itemUnits,omitempty"`
}

var clearLogsProgramm = flag.Bool("clearlogs", true, "очистить логи программы")

var ofdtemplate = flag.String("ofdtemplate", "ofd.ru", "шаблон ОФД: ofd.ru, 1-ofd.ru, kontur.ru")
var email = flag.String("email", "", "email, на которое будут отсылаться все чеки")
var noprint = flag.Bool("noprint", false, "печатать на бумагу (false) или не печатать (true) чек коорекции")
var GradeDetailedMark = flag.Int("gradesimplymark", 4, "уровень простоты вставки марки в чек коррекции. 0-не вставлять, 1 - в тег 1300, 2 - в тег 1305, 3 - вставка полной структура марки со всеми сопутсвующими тегами, 4 - всатвка со всеми полями единиц")

var FieldsNums map[string]int
var FieldsNames map[string]string
var OFD string
var AllFieldsHeadOfCheck []string
var AllFieldPositionsOfCheck []string

// var emulation = flag.Bool("emul", false, "эмуляция")
func main() {
	var data map[string]interface{}
	var ofdmap interface{}
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
	//читаем файл настроек
	if _, err := toml.DecodeFile("init.toml", &data); err != nil {
		fmt.Println(err)
		panic(err)
	}
	ofdmap = data["template"].(map[string]interface{})["ofd"]
	for k, v := range ofdmap.(map[string]interface{}) {
		if *ofdtemplate == fmt.Sprint(v) || *ofdtemplate == k {
			OFD = k
			break
		}
	}
	if OFD == "" {
		log.Fatal("не найден шаблон ОФД")
	}
	fmt.Println("шаблон ОФД", OFD)
	FieldsNums = make(map[string]int)
	FieldsNames = make(map[string]string)
	for k, v := range data[OFD].(map[string]interface{}) {
		FieldsNames[k] = fmt.Sprint(v)
	}
	for k := range data["fields"].(map[string]interface{})["kkt"].(map[string]interface{}) {
		AllFieldsHeadOfCheck = append(AllFieldsHeadOfCheck, k)
	}
	for k := range data["fields"].(map[string]interface{})["check"].(map[string]interface{}) {
		AllFieldsHeadOfCheck = append(AllFieldsHeadOfCheck, k)
	}
	for k := range data["fields"].(map[string]interface{})["positions"].(map[string]interface{}) {
		AllFieldPositionsOfCheck = append(AllFieldPositionsOfCheck, k)
	}
	//инициализация директории результатов
	if foundedLogDir, _ := doesFileExist(JSONRES); !foundedLogDir {
		os.Mkdir(JSONRES, 0777)
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
		panic(descrError)
	}
	//инициализация номеров колонок
	if len(lines) > 0 {
		FieldsNums = getNumberOfFieldsInCSV(lines[0], FieldsNames, FieldsNums, true)
	}
	fillFieldsNumByPositionTable(FieldsNames, FieldsNums)
	//fmt.Println("FieldsNames", FieldsNames)
	//fmt.Println("-------------------")
	//fmt.Println("FieldsNums", FieldsNums)
	//panic("ok")
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
		//заполняема поля шапки
		HeadOfCheck := make(map[string]string)
		for _, field := range AllFieldsHeadOfCheck {
			if !isInvField(FieldsNames[field]) {
				HeadOfCheck[field] = getfieldval(line, FieldsNums, field)
			}
		}
		//заполняем поля шапки с префиксом inv - те эти поля будут - это значения полей позиций
		for _, field := range AllFieldPositionsOfCheck {
			if isInvField(FieldsNames[field]) {
				HeadOfCheck["inv$"+field] = getfieldval(line, FieldsNums, field)
			}
		}
		if HeadOfCheck[COLBINDHEADFIELDKASSA] == "" {
			logsmap[LOGERROR].Printf("строка %v пропущена, так как в ней не опредлена касса", line)
			continue
		}
		HeadOfCheck[EMAILFIELD] = *email
		HeadOfCheck[NOPRINTFIELD] = fmt.Sprint(*noprint)
		valbindkassa := HeadOfCheck[COLBINDHEADFIELDKASSA]
		valbindcheck := HeadOfCheck[COLBINDHEADDIELDCHECK]
		//ищем позиции в файле позиций чека, которые бы соответсвовали бы текущеё строке чека //по номеру ФН и названию кассы
		checkDescrInfo := fmt.Sprintf("(ФД %v (ФП %v) от %v)", HeadOfCheck[COLFD], HeadOfCheck[COLFP], HeadOfCheck[COLDATE])
		descrInfo = fmt.Sprintf("для чека %v ищем позиции", checkDescrInfo)
		logsmap[LOGINFO].Println(descrInfo)
		findedPositions, summsOfPayment := findPositions(valbindkassa, valbindcheck, FieldsNames, FieldsNums)
		//fmt.Println("------------------------------")
		//fmt.Println("HeadOfCheck", HeadOfCheck)
		//fmt.Println("------------------------------")
		//fmt.Println("findedPositions", findedPositions)
		//fmt.Println("------------------------------")
		//panic("ok1")
		countOfPositions := len(findedPositions)
		descrInfo = fmt.Sprintf("для чека %v найдено %v позиций", checkDescrInfo, countOfPositions)
		//декопзируем head and postions
		//invDecopostions(HeadOfCheck, findedPositions, FieldsNames, FieldsNums)
		for fieldHead, valFieldHead := range HeadOfCheck {
			if isInvField(fieldHead) { //переносим его в findedPositions
				for _, pos := range findedPositions {
					fieldnameclear, _ := strings.CutPrefix(fieldHead, "inv$")
					pos[fieldnameclear] = valFieldHead
				}
			}
		}
		for _, pos := range findedPositions {
			for fieldPos, valFieldPos := range pos {
				if isInvField(fieldPos) { //переносим его в HeadOfCheck
					fieldnameclear, _ := strings.CutPrefix(fieldPos, "inv$")
					HeadOfCheck[fieldnameclear] = valFieldPos
				}
			}
			break
		}
		//fmt.Println("------------------------------")
		//fmt.Println("summsOfPayment", summsOfPayment)
		//fmt.Println("HeadOfCheck", HeadOfCheck)
		//fmt.Println("------------------------------")
		//перенгоси суммы оплат из позиций, если сумма оплат была указана у позиций
		for k, v := range summsOfPayment {
			HeadOfCheck[k] = strconv.FormatFloat(v, 'f', -1, 64)
		}
		//fmt.Println("*****************************")
		//fmt.Println("HeadOfCheck", HeadOfCheck)
		//fmt.Println("------------------------------")
		//fmt.Println("findedPositions", findedPositions)
		//fmt.Println("------------------------------")
		//panic("ok")
		logsmap[LOGINFO].Println(descrInfo)
		//производим сложный анализ
		analyzeComlite := true
		if countOfPositions > 0 { //если для чека были найдены позиции
			neededGetMarks := false
			for _, pos := range findedPositions {
				if (pos[COLPREDMET] == "ТМ") || (pos[COLPREDMET] == "АТМ") {
					logsmap[LOGINFO].Printf("для позицции %v требуется получить марку", pos)
					neededGetMarks = true
					break
				}
			}
			logsmap[LOGINFO].Printf("neededGetMarks = %v", neededGetMarks)
			if neededGetMarks {
				logsmap[LOGINFO].Println("будем получать/читать json с марками")
				//fmt.Println(checkDescrInfo, "полчаем json для марки")
				var receipt TReceipt
				var descrErr string
				var err error
				//receiptGetted := false
				//if alredyGettedFetch(HeadOfCheck[COLFD], HeadOfCheck[COLFP]) {
				//	//fmt.Println("уже был запрос")
				//	logsmap[LOGINFO].Println("полученный файл уже обработан")
				//	receipt, descrErr, err = getRecitpFromDisk(HeadOfCheck[COLFD], HeadOfCheck[COLFP])
				//	if err == nil {
				//		receiptGetted = true
				//		//fmt.Println("прочитали сохранённый запрос")
				//	} else {
				//		//fmt.Println("не удалось прочиать сохранённый запрос")
				//		logsmap[LOGERROR].Println(descrErr)
				//	}
				//}
				//if !receiptGetted {
				//fmt.Println("отрпавим сейчас запрос на сервер ОФД")
				logsmap[LOGINFO].Println("анализируем поле", FieldsNames[COLLINK])
				hypperlinkjson := replacefieldbyjsonhrep(HeadOfCheck[COLLINK])
				//fmt.Println("hypperlinkjson", hypperlinkjson)
				receipt, descrErr, err = fetchcheck(HeadOfCheck[COLFD], HeadOfCheck[COLFP], hypperlinkjson)
				if err != nil {
					logsmap[LOGERROR].Println(descrErr)
					analyzeComlite = false
					break
				}
				//saveReceiptToDisk(HeadOfCheck[COLFD], HeadOfCheck[COLFP], receipt)
				//fmt.Println("receipt", receipt)
				//fmt.Println("сохраняем запрос в файл")
				//}
				//записваем значение марки
				//fmt.Println("записваем значение марки")
				//fmt.Println("findedPositions", findedPositions)
				//fmt.Println("------------------")
				for _, itemPos := range receipt.Document.Items {
					//fmt.Printf("itemPos =%v.\n", itemPos.Name)
					//fmt.Printf("Code_GS_1M=%v.\n", itemPos.ProductCode.Code_GS_1M)
					markOfField := ""
					nameTypeOfMark := ""
					if itemPos.ProductCode.Code_Undefined != "" {
						nameTypeOfMark = "Undefined"
						markOfField = itemPos.ProductCode.Code_Undefined
					}
					if itemPos.ProductCode.Code_EAN_8 != "" {
						nameTypeOfMark = "EAN_8"
						markOfField = itemPos.ProductCode.Code_EAN_8
					}
					if itemPos.ProductCode.Code_EAN_13 != "" {
						nameTypeOfMark = "EAN_13"
						markOfField = itemPos.ProductCode.Code_EAN_13
					}
					if itemPos.ProductCode.Code_ITF_14 != "" {
						nameTypeOfMark = "ITF_14"
						markOfField = itemPos.ProductCode.Code_ITF_14
					}
					if itemPos.ProductCode.Code_GS_1 != "" {
						nameTypeOfMark = "GS_1"
						markOfField = itemPos.ProductCode.Code_GS_1
					}
					if itemPos.ProductCode.Code_GS_1M != "" {
						nameTypeOfMark = "GS_1M"
						markOfField = itemPos.ProductCode.Code_GS_1M
					}
					if itemPos.ProductCode.Code_KMK != "" {
						nameTypeOfMark = "KMK"
						markOfField = itemPos.ProductCode.Code_KMK
					}
					if itemPos.ProductCode.Code_MI != "" {
						nameTypeOfMark = "MI"
						markOfField = itemPos.ProductCode.Code_MI
					}
					if itemPos.ProductCode.Code_EGAIS_2 != "" {
						nameTypeOfMark = "EGAIS_2"
						markOfField = itemPos.ProductCode.Code_EGAIS_2
					}
					if itemPos.ProductCode.Code_EGAIS_3 != "" {
						nameTypeOfMark = "EGAIS_3"
						markOfField = itemPos.ProductCode.Code_EGAIS_3
					}
					if itemPos.ProductCode.Code_F_1 != "" {
						nameTypeOfMark = "F_1"
						markOfField = itemPos.ProductCode.Code_F_1
					}
					if itemPos.ProductCode.Code_F_2 != "" {
						nameTypeOfMark = "F_2"
						markOfField = itemPos.ProductCode.Code_F_2
					}
					if itemPos.ProductCode.Code_F_3 != "" {
						nameTypeOfMark = "F_3"
						markOfField = itemPos.ProductCode.Code_F_3
					}
					if itemPos.ProductCode.Code_F_4 != "" {
						nameTypeOfMark = "F_4"
						markOfField = itemPos.ProductCode.Code_F_4
					}
					if itemPos.ProductCode.Code_F_5 != "" {
						nameTypeOfMark = "F_5"
						markOfField = itemPos.ProductCode.Code_F_5
					}
					if itemPos.ProductCode.Code_F_6 != "" {
						nameTypeOfMark = "F_6"
						markOfField = itemPos.ProductCode.Code_F_6
					}
					if markOfField == "" {
						continue
					}
					for _, posFined := range findedPositions {
						//fmt.Printf("posFined=%v.\n", posFined[COLNAME])
						if strings.EqualFold(strings.ToLower(itemPos.Name), strings.ToLower(posFined[COLNAME])) {
							//fmt.Println("нашли позицию", itemPos.Name)
							//logsmap[LOG]
							//posFined[COLMARK] = itemPos.ProductCode.Code_GS_1M
							posFined[COLMARK] = markOfField
							posFined[NAMETYPEOFMARK] = nameTypeOfMark
							break
						}
					}
					analyzeComlite = true
				}
				//fmt.Println("------------------")
				//fmt.Println("findedPositions", findedPositions)
				//fmt.Println("------------------****************")
			}
		}
		if (countOfPositions > 0) && analyzeComlite { //если для чека были найдены позиции
			logsmap[LOGINFO].Println("генерируем json файл")
			//jsonres, descError, err := generateCheckCorrection(headOfCheckkassir, innkassir, dateCh, fd, fp, typeCheck, *email, nal, bez, avance, kred, obmen, findedPositions)
			jsonres, descError, err := generateCheckCorrection(HeadOfCheck, findedPositions)
			//for k, v := range jsonres.Items {
			//fmt.Println("name", v.Name)
			//fmt.Println("k", k)
			//fmt.Println("v", v)
			//fmt.Println(v.ImcParams)
			//fmt.Println(v.ImcParams.Imc)
			//}
			//fmt.Println("jsonres", jsonres)
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
			dir_file_name := fmt.Sprintf("%v%v/", JSONRES, HeadOfCheck[COLFNKKT])
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
			file_name := fmt.Sprintf("%v%v/%v_%v.json", JSONRES, HeadOfCheck[COLFNKKT], HeadOfCheck[COLFNKKT], HeadOfCheck[COLFD])
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
			//panic("ok2")
		} else {
			if analyzeComlite {
				descrError := fmt.Sprintf("для чека %v не найдены позиции", checkDescrInfo)
				logsmap[LOGERROR].Println(descrError)
			} else {
				descrError := fmt.Sprintf("для чека %v не получилось произвести анализ (получить марку)", checkDescrInfo)
				logsmap[LOGERROR].Println(descrError)
			}
		} //если для чека были найдены позиции
	} //перебор чеков
	logsmap[LOGINFO_WITHSTD].Println("формирование json заданий завершено")
	logsmap[LOGINFO_WITHSTD].Printf("обработано %v из %v чеков", countWritedChecks, countAllChecks)
	logsmap[LOGINFO_WITHSTD].Println("проверка завершена")
}

func fillFieldsNumByPositionTable(fieldsnames map[string]string, fieldsnums map[string]int) error {
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
	if len(lines) > 0 {
		getNumberOfFieldsInCSV(lines[0], fieldsnames, fieldsnums, false)
	}
	return nil
}

func findPositions(valbindkassainhead, valbindcheckinhead string, fieldsnames map[string]string, fieldsnums map[string]int) (map[int]map[string]string, map[string]float64) {
	//fmt.Println("valbindkassainhead", valbindkassainhead)
	//fmt.Println("valbindcheckinhead", valbindcheckinhead)
	res := make(map[int]map[string]string)
	summsPayment := make(map[string]float64)
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
		//fmt.Println(line)
		if currLine == 1 {
			continue
		}
		valbindkassainpos := getfieldval(line, fieldsnums, COLBINDPOSFIELDKASSA)
		valbindcheckpos := getfieldval(line, fieldsnums, COLBINDPOSFIELDCHECK)
		if (valbindkassainhead != valbindkassainpos) || (valbindcheckinhead != valbindcheckpos) {
			continue
		}
		logsmap[LOGINFO].Println("Найдена строка", line)
		currPos++
		res[currPos] = make(map[string]string)
		for _, field := range AllFieldsHeadOfCheck {
			if isInvField(fieldsnames[field]) {
				curValOfField := getfieldval(line, fieldsnums, field)
				curValOfField = strings.TrimSpace(curValOfField)
				res[currPos]["inv$"+field] = curValOfField
				//получаем суммы оплат
				if field == COLNAL || field == COLBEZ || field == COLAVANCE || field == COLCREDIT ||
					field == COLVSTRECHPREDST {
					if notEmptyFloatField(curValOfField) {
						currSumm, errDescr, err := getFloatFromStr(getfieldval(line, fieldsnums, COLAMOUNTPOS))
						if err != nil {
							logsmap[LOGERROR].Println(errDescr, line)
							continue
						}
						summsPayment[field] = summsPayment[field] + currSumm
					}
				}
			}
		}
		for _, field := range AllFieldPositionsOfCheck {
			if !isInvField(fieldsnames[field]) {
				res[currPos][field] = getfieldval(line, fieldsnums, field)
			}
		}
		//for k := range FieldsNums {
		//	res[currPos][k] = getfieldval(line, fieldsnums, k)
		//}
	} //перебор всех строк в файле позиций чека
	//if len(res) > 0 {
	//fmt.Println(res)
	//}
	return res, summsPayment
} //findPositions

func generateCheckCorrection(headofcheck map[string]string, poss map[int]map[string]string) (TCorrectionCheck, string, error) {
	var checkCorr TCorrectionCheck
	strInfoAboutCheck := fmt.Sprintf("(ФД %v, ФП %v %v)", headofcheck[COLFD], headofcheck[COLFP], headofcheck[COLDATE])
	chekcCorrTypeLoc := ""
	typeCheck := strings.ToLower(headofcheck[COLTAG1054])
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
		descError := fmt.Sprintf("ошибка (для типа %v не определён тип чека коррекциии) %v", typeCheck, strInfoAboutCheck)
		logsmap[LOGERROR].Println(descError)
		return checkCorr, descError, errors.New("ошибка определения типа чека коррекции")
	}
	//strconv.ParseBool
	checkCorr.Electronically, _ = strconv.ParseBool(headofcheck[NOPRINTFIELD])
	if headofcheck[EMAILFIELD] == "" {
		checkCorr.Electronically = false
	} else {
		checkCorr.Electronically = true
	}
	checkCorr.CorrectionType = "self"
	checkCorr.CorrectionBaseDate = headofcheck[COLDATE]
	checkCorr.CorrectionBaseNumber = ""
	checkCorr.ClientInfo.EmailOrPhone = headofcheck[EMAILFIELD]
	checkCorr.Operator.Name = headofcheck[COLKASSIR]
	checkCorr.Operator.Vatin = headofcheck[COLINNKASSIR]
	nal := headofcheck[COLNAL]
	if notEmptyFloatField(nal) {
		nalch, err := strconv.ParseFloat(nal, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для налчиного расчёта %v", err, nal, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "cash", Sum: nalch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	bez := headofcheck[COLBEZ]
	if notEmptyFloatField(bez) {
		bezch, err := strconv.ParseFloat(bez, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для безналичного расчёта %v", err, bez, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "electronically", Sum: bezch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	avance := headofcheck[COLAVANCE]
	if notEmptyFloatField(avance) {
		avancech, err := strconv.ParseFloat(avance, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для суммы зачета аванса %v", err, avance, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "prepaid", Sum: avancech}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	kred := headofcheck[COLCREDIT]
	if notEmptyFloatField(kred) {
		kredch, err := strconv.ParseFloat(kred, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для суммы оплаты в рассрочку %v", err, kred, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "credit", Sum: kredch}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	obmen := headofcheck[COLVSTRECHPREDST]
	if notEmptyFloatField(obmen) {
		obmench, err := strconv.ParseFloat(obmen, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для суммы оплаты встречным представлением %v", err, obmen, strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		pay := TPayment{Type: "other", Sum: obmench}
		checkCorr.Payments = append(checkCorr.Payments, pay)
	}
	currFP := headofcheck[COLFP]
	if currFP == "" {
		currFP = headofcheck[COLFD]
	}
	//в тег 1192 - записываем ФП (если нет ФП, то записываем ФД)
	if currFP != "" {
		newAdditionalAttribute := TPosition{Type: "additionalAttribute"}
		newAdditionalAttribute.Value = currFP
		newAdditionalAttribute.Print = true
		checkCorr.Items = append(checkCorr.Items, newAdditionalAttribute)
	}
	if headofcheck[COLFD] != "" {
		newAdditionalAttribute := TPosition{Type: "userAttribute"}
		newAdditionalAttribute.Name = "ФД"
		newAdditionalAttribute.Value = headofcheck[COLFD]
		newAdditionalAttribute.Print = true
		checkCorr.Items = append(checkCorr.Items, newAdditionalAttribute)
	}
	for _, pos := range poss {
		newPos := TPosition{Type: "position"}
		newPos.Name = pos[COLNAME]
		qch, err := strconv.ParseFloat(pos[COLQUANTITY], 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v количества %v", err, pos["Quantity"], strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		newPos.Quantity = qch
		prch, err := strconv.ParseFloat(pos[COLPRICE], 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v цены %v", err, pos[COLPRICE], strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		newPos.Price = prch
		sch, err := strconv.ParseFloat(pos[COLAMOUNTPOS], 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v суммы %v", err, pos[COLAMOUNTPOS], strInfoAboutCheck)
			logsmap[LOGERROR].Println(descrErr)
			return checkCorr, descrErr, err
		}
		newPos.Amount = sch
		newPos.MeasurementUnit = "piece"
		newPos.PaymentMethod = getSposobRash(pos[COLSPOSOB])
		//commodityWithMarking
		newPos.PaymentObject = getPredmRasch(pos[COLPREDMET])
		newPos.Tax = new(TTaxNDS)
		newPos.Tax.Type = STAVKANDSNONE
		if pos[COLSTAVKANDS20] != "" {
			newPos.Tax.Type = STAVKANDS20
		} else if pos[COLSTAVKANDS10] != "" {
			newPos.Tax.Type = STAVKANDS10
		} else if pos[COLSTAVKANDS0] != "" {
			newPos.Tax.Type = STAVKANDS0
		} else if pos[COLSTAVKANDS120] != "" {
			newPos.Tax.Type = STAVKANDS120
		} else if pos[COLSTAVKANDS110] != "" {
			newPos.Tax.Type = STAVKANDS110
		}
		if pos[COLMARK] != "" {
			currMark := pos[COLMARK]
			if *GradeDetailedMark == 1 {
				byte32onlyCut := min(32, len(currMark))
				newPos.ProductCodes = new(TProductCodesAtol)
				newPos.ProductCodes.Undefined = currMark[:byte32onlyCut]
			} else if *GradeDetailedMark == 2 || (pos[NAMETYPEOFMARK] == "Undefined") || (pos[NAMETYPEOFMARK] == "EAN_8") ||
				(pos[NAMETYPEOFMARK] == "EAN_13") || (pos[NAMETYPEOFMARK] == "ITF_14") {
				newPos.ProductCodes = new(TProductCodesAtol)
				setMarkInArolDriverCorrenspOFDMark(newPos.ProductCodes, currMark, pos[NAMETYPEOFMARK])
			} else if *GradeDetailedMark == 3 || *GradeDetailedMark == 4 {
				currMarkInBase64 := base64.StdEncoding.EncodeToString([]byte(currMark))
				newPos.ImcParams = new(TImcParams)
				newPos.ImcParams.ImcType = "auto"
				newPos.ImcParams.Imc = currMarkInBase64
				itemPieceSold := "itemPieceSold"
				if headofcheck[COLTAG1054] == "sellReturnCorrection" {
					itemPieceSold = "itemPieceReturn"
				}
				newPos.ImcParams.ItemEstimatedStatus = itemPieceSold
				newPos.ImcParams.ItemUnits = "piece"
				newPos.ImcParams.ImcModeProcessing = 0
				//newPos.ImcParams.ImcBarcode
				if *GradeDetailedMark == 4 {
					newPos.ImcParams.ItemQuantity = newPos.Quantity
					newPos.ImcParams.ItemInfoCheckResult = new(TItemInfoCheckResult)
					newPos.ImcParams.ItemInfoCheckResult.ImcCheckFlag = true
					newPos.ImcParams.ItemInfoCheckResult.ImcCheckResult = true
					newPos.ImcParams.ItemInfoCheckResult.ImcStatusInfo = true
					newPos.ImcParams.ItemInfoCheckResult.ImcEstimatedStatusCorrect = true
					newPos.ImcParams.ItemInfoCheckResult.EcrStandAloneFlag = false
				}
			}
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
	//2023-11-09 - офд.ru
	if OFD == "ofdru" {
		res := strings.ReplaceAll(dt, "-", ".")
		return res
	}
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
	res = strings.ReplaceAll(res, " р", "")
	return res
}

func getNumberOfFieldsInCSVloc(line []string, fieldsnames map[string]string, fieldsnums map[string]int, fieldsOfBlock []string, notinv bool) map[string]int {
	for _, name := range fieldsOfBlock {
		colname := fieldsnames[name]
		if notinv && isInvField(colname) {
			continue
		}
		if !notinv && !isInvField(colname) {
			continue
		}
		colnamefinding := colname
		if len(colname) == 0 {
			continue
		}
		if strings.Contains(colname, "$") && colname[:1] == "#" && len(colname) > 2 { //есть какие-то служебные
			posEndServiceWords := strings.Index(colname, "$")
			colnamefinding = colname[posEndServiceWords+1:]
		}
		if (name == "bindheadfieldkassa") || (name == "bindheadfieldcheck") {
			colnamefinding = fieldsnames[fieldsnames[name]]
		}
		for i, val := range line {
			valformated := formatfieldname(val)
			if valformated == colnamefinding {
				fieldsnums[name] = i
				break
			}
		}
	}
	return fieldsnums
}

func getNumberOfFieldsInCSV(line []string, fieldsnames map[string]string, fieldsnums map[string]int, headAndNotOfPositions bool) map[string]int {
	var fieldsOfBlock []string
	if headAndNotOfPositions {
		fieldsOfBlock = AllFieldsHeadOfCheck
	} else {
		fieldsOfBlock = AllFieldPositionsOfCheck
	}
	fieldsnums = getNumberOfFieldsInCSVloc(line, fieldsnames, fieldsnums, fieldsOfBlock, true)
	if headAndNotOfPositions {
		fieldsOfBlock = AllFieldPositionsOfCheck
	} else {
		fieldsOfBlock = AllFieldsHeadOfCheck
	}
	fieldsnums = getNumberOfFieldsInCSVloc(line, fieldsnames, fieldsnums, fieldsOfBlock, false)
	return fieldsnums
}

func formatfieldname(name string) string {
	res := strings.ReplaceAll(name, "\r\n", " ")
	res = strings.ReplaceAll(res, "\n", " ")
	res = strings.ReplaceAll(res, "\r", " ")
	res = strings.TrimSpace(res)
	return res
}

func getfieldval(line []string, fieldsnum map[string]int, name string) string {
	var num int
	var ok bool
	if num, ok = fieldsnum[name]; !ok {
		return ""
	}
	//pref := ""
	//if strings.Contains(FieldsNames[name], "inv") { //значение этого столбца находится в другой таблице
	//	pref = "inv$"
	//}
	resVal := line[num]
	if name == "date" {
		resVal = formatMyDate(resVal)
	}
	if name == "amountCheck" || name == "nal" || name == "bez" || name == "credit" ||
		name == "avance" || name == "vstrechpredst" || name == "quantity" ||
		name == "price" || name == "amountpos" {
		resVal = formatMyNumber(resVal) //преобразование к формату для чисел
	}
	if strings.Contains(name, "stavkaNDS") {
		if notEmptyFloatField(resVal) {
			switch name {
			case "stavkaNDS0":
				resVal = STAVKANDS0
			case "stavkaNDS10":
				resVal = STAVKANDS10
			case "stavkaNDS20":
				resVal = STAVKANDS20
			case "stavkaNDS110":
				resVal = STAVKANDS110
			case "stavkaNDS120":
				resVal = STAVKANDS120
			default:
				resVal = STAVKANDSNONE
			}
		} else {
			resVal = ""
		}
	}
	if strings.Contains(FieldsNames[name], "analyse") {
		//делаем анализ поля
		switch OFD {
		case "firstofd":
			if name == "bindposfieldkassa" || name == "bindposfieldcheck" {
				reg, fn, fd, descrErr, err := getRegFnFdFromName(resVal)
				if err != nil {
					logsmap[LOGERROR].Println(descrErr)
					erroFatal := fmt.Sprintf("ошбка: %v. Не удалось получить регистрационный номер, номер ФД, ФН из имени кассы %v", err, resVal)
					log.Fatal(erroFatal)
				}
				if name == "bindposfieldkassa" {
					resVal = reg
					if FieldsNames[COLBINDHEADFIELDKASSA] == "fnkkt" {
						resVal = fn
					}
				} else if name == "bindposfieldcheck" {
					resVal = fd
				}
			}
		}
	}
	return resVal
}

func fetchcheck(fd, fp, hyperlinkonjson string) (TReceipt, string, error) {
	var receipt TReceipt
	var resp *http.Response
	var err error
	var body []byte
	nameoffile := fd + "_" + fp + ".resp"
	fullFileName := DIROFREQUEST + nameoffile
	if !alredyGettedFetch(fd, fp) {
		logsmap[LOGINFO].Printf("получение данных о чеке по ссылке %v", hyperlinkonjson)
		resp, err = http.Get(hyperlinkonjson)
		if err != nil {
			errDescr := fmt.Sprintf("ошибка(не удалось получить ответ от сервера ОФД): %v. Не удалось получить данные о чеке по ссылке %v", err, hyperlinkonjson)
			logsmap[LOGERROR].Println(errDescr)
			return receipt, errDescr, err
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			errDescr := fmt.Sprintf("ошибка(прочитать данные от сервера ОФД): %v. Не удалось получить данные о чеке по ссылке %v", err, hyperlinkonjson)
			logsmap[LOGERROR].Println(errDescr)
			return receipt, errDescr, err
		}
		ioutil.WriteFile(fullFileName, body, 0644)
	} else {
		logsmap[LOGINFO].Printf("получение данных из файла %v ", fullFileName)
		body, err = ioutil.ReadFile(fullFileName)
		if err != nil {
			errDescr := fmt.Sprintf("ошибка(чтения данные с диска): %v. Не удалось получить данные с диска файла %v", err, fullFileName)
			logsmap[LOGERROR].Println(errDescr)
			return receipt, errDescr, err
		}
	}
	//"https://ofd.ru/Document/ReceiptJsonDownload?DocId=289f8926-74f2-b25b-f34a-6edf933b9999"
	err = json.Unmarshal(body, &receipt)
	if err != nil {
		errDescr := fmt.Sprintf("ошибка(парсинг данных от сервера ОФД): %v. Не удалось получить данные о чеке по ссылке %v", err, hyperlinkonjson)
		logsmap[LOGERROR].Println(errDescr)
		return receipt, errDescr, err
	}
	//for _, item := range receipt.Document.Items {
	//	fmt.Println(item.Name)
	//	fmt.Println(item.Price)
	//	fmt.Println(item.Quantity)
	//	fmt.Println(item.ProductCode.Code_GS_1M)
	//}
	return receipt, "", nil
}

func notEmptyFloatField(val string) bool {
	res := true
	if val == "" || val == "0.00" || val == "0.00 ₽" || val == "0,00 ₽" || val == "0,00" ||
		val == "0,00 р" {
		res = false
	}
	return res
}

func isInvField(fieldname string) bool {
	res := false
	if strings.Contains(fieldname, "inv") {
		res = true
	}
	return res
}

func getSposobRash(sposob string) string {
	res := "fullPayment"
	return res
}

func getPredmRasch(predm string) string {
	res := "commodity"
	switch predm {
	case "ТМ":
		res = "commodityWithMarking"
	case "ПОДАКЦИЗНЫЙ ТОВАР":
		res = "excise"
	case "ТНМ":
		res = "commodityWithoutMarking"
	case "АТМ":
		res = "exciseWithMarking"
	}
	return res
}

func getFloatFromStr(val string) (float64, string, error) {
	var err error
	res := 0.0
	if val != "" {
		res, err = strconv.ParseFloat(val, 64)
		if err != nil {
			descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для суммы оплаты", err, val)
			logsmap[LOGERROR].Println(descrErr)
			return res, descrErr, err
		}
	}
	return res, "", nil
	//nalch, err := strconv.ParseFloat(nal, 64)
}

func replacefieldbyjsonhrep(hyperlhtml string) string {
	//https://ofd.ru/Document/RenderDoc?RawId=a1c0fddc-917c-0b93-6b30-25eb0ee91259
	//to
	//https://ofd.ru/Document/ReceiptJsonDownload?DocId=a1c0fddc-917c-0b93-6b30-25eb0ee91259
	return strings.ReplaceAll(hyperlhtml, "RenderDoc?RawId=", "ReceiptJsonDownload?DocId=")
}

func alredyGettedFetch(fd, fp string) bool {
	res := true
	nameoffile := fd + "_" + fp + ".resp"
	fullname := DIROFREQUEST + nameoffile
	if foundedRespFile, _ := doesFileExist(fullname); !foundedRespFile {
		res = false
	}
	return res
}

func setMarkInArolDriverCorrenspOFDMark(prcode *TProductCodesAtol, mark, typeCode string) {
	switch typeCode {
	case "Undefined":
		prcode.Undefined = mark
	case "EAN_8":
		prcode.Code_EAN_8 = mark
	case "EAN_13":
		prcode.Code_EAN_13 = mark
	case "ITF_14":
		prcode.Code_ITF_14 = mark
	case "GS_1":
		prcode.Code_GS_1 = mark
	case "GS_1M":
		prcode.Tag1305 = mark
	case "KMK":
		prcode.Code_KMK = mark
	case "MI":
		prcode.Code_MI = mark
	case "EGAIS_2":
		prcode.Code_EGAIS_2 = mark
	case "EGAIS_3":
		prcode.Code_EGAIS_3 = mark
	case "F_1":
		prcode.Code_F_1 = mark
	case "F_2":
		prcode.Code_F_2 = mark
	case "F_3":
		prcode.Code_F_3 = mark
	case "F_4":
		prcode.Code_F_4 = mark
	case "F_5":
		prcode.Code_F_5 = mark
	case "F_6":
		prcode.Code_F_6 = mark
	default:
		prcode.Tag1305 = mark
	}
}
