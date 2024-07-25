//go:generate ./resource/goversioninfo.exe -icon=resource/icon.ico -manifest=resource/goversioninfo.exe.manifest
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const VERSION_OF_PROGRAM = "2024_07_25_01"
const NAME_OF_PROGRAM = "формирование json заданий чеков коррекции на основании отчетов из ОФД (xsl-csv)"

const EMAILFIELD = "email"
const NOPRINTFIELD = "electronically"
const NAMETYPEOFMARK = "TYPEMARK"

const COLREGNUMKKT = "regnumkkt"
const COLFNKKT = "fnkkt"
const COLNAMEOFKKT = "nameofkkt"

const COLFD = "fd"
const COLFP = "fp"
const COLSTATUSINFNS = "statusofcheck"
const COLAMOUNTCHECK = "amountCheck"
const COLNAL = "nal"
const COLBEZ = "bez"
const COLCREDIT = "credit"
const COLAVANCE = "avance"
const COLVSTRECHPREDST = "vstrechpredst"
const COLKASSIR = "kassir"
const COLINNKASSIR = "innkassir"
const COLNAMECLIENT = "nameclient"
const COLINNCLIENT = "innclient"
const COLTELKASSIR = "telkassir"
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
const COLPRIZAGENTA = "prizagenta"
const COLNAMEOFSUPPLIER = "nameofsupl"
const COLINNOFSUPPLIER = "innofsupl"
const COLTELOFSUPPLIER = "telofsupl"

const COLSTAVKANDS = "stavkaNDS"
const COLSTAVKANDS0 = "stavkaNDS0"
const COLSTAVKANDS10 = "stavkaNDS10"
const COLSTAVKANDS20 = "stavkaNDS20"
const COLSTAVKANDS110 = "stavkaNDS110"
const COLSTAVKANDS120 = "stavkaNDS120"
const COLMARK = "mark"
const COLBINDPOSFIELDKASSA = "bindposfieldkassa"
const COLBINDPOSFIELDCHECK = "bindposfieldcheck"
const COLBINDPOSPOSFIELDCHECK = "bindposposfieldcheck"

const COLMARKOTHER = "markother"
const COLMARKOTHER2 = "markother2"
const COLBINDOTHERKASSS = "bindotherfieldkassa"
const COLBINDOTHERCHECK = "bindotherfieldcheck"
const COLBINDOTHERPOS = "bindotherposfieldcheck"

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
const DIROFREQUESTASTRAL = "./request/astral/"

type TClientInfo struct {
	EmailOrPhone string `json:"emailOrPhone,omitempty"`
	Vatin        string `json:"vatin,omitempty"`
	Name         string `json:"name,omitempty"`
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
	Name            string   `json:"name"`
	Price           float64  `json:"price"`
	Quantity        float64  `json:"quantity"`
	Amount          float64  `json:"amount"`
	MeasurementUnit string   `json:"measurementUnit"`
	PaymentMethod   string   `json:"paymentMethod"`
	PaymentObject   string   `json:"paymentObject"`
	Tax             *TTaxNDS `json:"tax,omitempty"`
	//fot type tag1192 //AdditionalAttribute
	Value        string             `json:"value,omitempty"`
	Print        bool               `json:"print,omitempty"`
	ProductCodes *TProductCodesAtol `json:"productCodes,omitempty"`
	ImcParams    *TImcParams        `json:"imcParams,omitempty"`
	//Mark         string             `json:"mark,omitempty"`
	AgentInfo    *TAgentInfo    `json:"agentInfo,omitempty"`
	SupplierInfo *TSupplierInfo `json:"supplierInfo,omitempty"`
}

type TTag1192_91 struct {
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
	Print bool   `json:"print,omitempty"`
}

type TOperator struct {
	Name  string `json:"name"`
	Vatin string `json:"vatin,omitempty"`
}

type TAgentInfo struct {
	Agents []string `json:"agents"`
}
type TSupplierInfo struct {
	Vatin  string   `json:"vatin"`
	Name   string   `json:"name,omitempty"`
	Phones []string `json:"phones,omitempty"`
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
	TaxationType         string      `json:"taxationType,omitempty"`
	ClientInfo           TClientInfo `json:"clientInfo"`
	CorrectionType       string      `json:"correctionType"` //
	CorrectionBaseDate   string      `json:"correctionBaseDate"`
	CorrectionBaseNumber string      `json:"correctionBaseNumber"`
	Operator             TOperator   `json:"operator"`
	//Items                []TPosition `json:"items"`
	Items    []interface{} `json:"items"`
	Payments []TPayment    `json:"payments"`
	Total    float64       `json:"total,omitempty"`
}

// json чека в ОФД.RU
type TReceiptOFD struct {
	Version  int `json:"Version"`
	Document struct {
		// document fields
		Amount_Total    int64      `json:"Amount_Total"`
		Amount_Cash     int64      `json:"Amount_Cash"`
		Amount_ECash    int64      `json:"Amount_ECash"`
		Amount_Advance  int64      `json:"Amount_Advance"`
		Amount_Loan     int64      `json:"Amount_Loan"`
		Amount_Granting int64      `json:"Amount_Granting"`
		Items           []TItemOFD `json:"Items"`
		// other fields
	} `json:"Document"`
}
type TItemOFD struct {
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
	Imc                 string                `json:"imc"`
	ItemEstimatedStatus string                `json:"itemEstimatedStatus,omitempty"`
	ImcModeProcessing   int                   `json:"imcModeProcessing"`
	ImcBarcode          string                `json:"imcBarcode,omitempty"`
	ItemInfoCheckResult *TItemInfoCheckResult `json:"itemInfoCheckResult,omitempty"`
	ItemQuantity        float64               `json:"itemQuantity,omitempty"`
	ItemUnits           string                `json:"itemUnits,omitempty"`
}

var clearLogsProgramm = flag.Bool("clearlogs", true, "очистить логи программы")

var ofdchoice = flag.Int("ofd", 0, "Порядковый номер ОФД в файле настроек init.toml раздел [template.ofd]")
var email = flag.String("email", "", "email, на которое будут отсылаться все чеки")
var printonpaper = flag.Bool("print", true, "печатать на бумагу (true) или не печатать (false) чек коорекции")
var debug = flag.Bool("debug", false, "режим отладки")
var fetchalways = flag.Bool("fetchalways", true, "всегда посылать запросы по ссылке, не зависимо от предмета расчета")
var byPrescription = flag.Bool("prescription", false, "по предписанию (true) или самостоятельно (false)")
var docNumbOfPrescription = flag.String("docnumbprescr", "", "номер документа предписания налоговой")
var measurementUnitOfFracQuantSimple = flag.String("fracquantunitsimple", "кг", "мера измерения дробного количества товара без макри (кг, л, грамм, иная)")
var measurementUnitOfFracQuantMark = flag.String("fracquantunitmark", "кг", "мера измерения дробного количества товара с маркой (кг, л, грамм, иная)")
var checkdoublepos = flag.Bool("checkdoule", false, "проверять на задвоение позиции")

var FieldsNums map[string]int
var FieldsNames map[string]string
var OFD string
var AllFieldsUnionOfCheck []string
var AllFieldsHeadOfCheck []string
var AllFieldPositionsOfCheck []string
var AllFieldOtherOfCheck []string

// var emulation = flag.Bool("emul", false, "эмуляция")
func main() {
	var data map[string]interface{}
	var ofdmap interface{}
	var ofdsinit map[string]string
	var ofdarray map[int]string
	var numOFDSorted []int
	RowOfHeadInHeaderChecks := 1
	//RowOfHeadInPositionsChecks := 1
	//RowOfHeadInOtherChecks := 1
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
		fmt.Println(descrError)
		fmt.Println("Нажмите любую клавишу...")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		log.Panic(descrError)
	}
	logginInFile(runDescription)
	fmt.Println("debug: ", *debug)
	//определение параметров запуска
	//читаем файл настроек
	if _, err := toml.DecodeFile("init.toml", &data); err != nil {
		fmt.Println(err)
		fmt.Println("Нажмите любую клавишу...")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		log.Panic(err)
	}
	//читаем все доступные ОФД
	ofdsinit = make(map[string]string)
	ofdarray = make(map[int]string)
	ofdmap = data["template"].(map[string]interface{})["ofd"]
	//for k, v := range ofdmap.(map[string]interface{}) {
	for _, v := range ofdmap.([]map[string]interface{}) {
		ofdsinit[v["name"].(string)] = v["descr"].(string)
		strii := int(v["num"].(int64))
		ofdarray[strii] = v["name"].(string)
		numOFDSorted = append(numOFDSorted, strii)
	}
	//сортируем по номерам ОФД
	sort.Ints(numOFDSorted)
	if *ofdchoice == 0 {
		sQuestOFD := "Выберите ОФД. "
		for _, currNumOfd := range numOFDSorted {
			v := ofdarray[currNumOfd]
			//for currNumOfd, v := range ofdarray {
			if sQuestOFD != "Выберите ОФД. " {
				sQuestOFD = sQuestOFD + ", "
			}
			sQuestOFD = sQuestOFD + strconv.Itoa(currNumOfd) + ". " + ofdsinit[v]
		}
		sQuestOFD = sQuestOFD + ": "
		fmt.Print(sQuestOFD)
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		*ofdchoice, _ = strconv.Atoi(input.Text())
	}
	if *ofdchoice <= 0 {
		descrError = fmt.Sprintf("неверное значение флага -ofd %v", *ofdchoice)
		logsmap[LOGERROR].Println(descrError)
		input := bufio.NewScanner(os.Stdin)
		println("Нажмите любую клавишу...")
		input.Scan()
		log.Panic(descrError)
	}
	OFD = ofdarray[*ofdchoice]
	if OFD == "" {
		descrError = fmt.Sprintf("не найден %v шаблон ОФД", *ofdchoice)
		logsmap[LOGERROR].Println(descrError)
		input := bufio.NewScanner(os.Stdin)
		println("Нажмите любую клавишу...")
		input.Scan()
		log.Panic(descrError)
	}
	fmt.Println(ofdsinit[OFD])
	if *email == "" {
		fmt.Print("Введите email, на которое будут отсылаться все чеки: ")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		*email = input.Text()
	}
	if (*email != "") && (*printonpaper) {
		fmt.Println("printonpaper", *printonpaper)
		fmt.Print("Печать чеки на бумаге (да/нет, по умолчание да) :")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		*printonpaper, _ = getBoolFromString(input.Text(), *printonpaper)
	}
	if *email == "" {
		*printonpaper = true
	}
	if OFD == "ofdru" {
		fmt.Print("Всегда посылать запросы по ссылке, не зависимо от предмета расчета (да/нет, по умолчанию (да)):")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		*fetchalways, _ = getBoolFromString(input.Text(), *fetchalways)
	}
	fmt.Print("Чек коррекции по предписанию? (да/нет, по умолчанию: нет):")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	*byPrescription, _ = getBoolFromString(input.Text(), *byPrescription)
	if *byPrescription {
		fmt.Print("Введите номер предписания налоговой: ")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		*docNumbOfPrescription = input.Text()
	}
	fmt.Print("Мера измерения дробного количества товара без марки (кг, л, грамм, иная, по умолчанию кг):")
	input = bufio.NewScanner(os.Stdin)
	input.Scan()
	*measurementUnitOfFracQuantSimple = input.Text()
	if *measurementUnitOfFracQuantSimple == "" {
		*measurementUnitOfFracQuantSimple = "кг"
	}
	fmt.Print("Мера измерения дробного количества товара с маркой (кг, л, грамм, иная, по умолчанию кг):")
	input = bufio.NewScanner(os.Stdin)
	input.Scan()
	*measurementUnitOfFracQuantMark = input.Text()
	if *measurementUnitOfFracQuantMark == "" {
		*measurementUnitOfFracQuantMark = "кг"
	}
	//
	fmt.Println("**********************")
	fmt.Println("ОФД: ", ofdsinit[OFD])
	fmt.Println("email: ", *email)
	fmt.Println("печать чеки на бумаге: ", *printonpaper)
	fmt.Println("Всегда посылать запросы по ссылке, не зависимо от предмета расчета ", *fetchalways)
	fmt.Println("Чек коррекции по предписанию", *byPrescription)
	if *byPrescription {
		fmt.Println("Номер документа предписания налоговой", *docNumbOfPrescription)
	}
	fmt.Println("Мера измерения дробного количества товара без марки: ", *measurementUnitOfFracQuantSimple)
	fmt.Println("Мера измерения дробного количества товара с маркой: ", *measurementUnitOfFracQuantMark)
	fmt.Print("Настройки верны? Продолжить? (да/нет, по умолчанию: да): ")
	//input = bufio.NewScanner(os.Stdin)
	input.Scan()
	contin := true
	contin, _ = getBoolFromString(input.Text(), contin)
	if !contin {
		descrError = "Настройки не верны. Завершение работы программы"
		logginInFile(descrError)
		fmt.Println(descrError)
		println("Нажмите любую клавишу...")
		input.Scan()
		log.Panic(descrError)
	}
	if OFD == "platforma" {
		*checkdoublepos = true
	}
	//инициализация колонок файлов
	logginInFile("инициализация номеров колонок")
	FieldsNums = make(map[string]int)
	FieldsNames = make(map[string]string)
	for k, v := range data[OFD].(map[string]interface{}) {
		FieldsNames[k] = fmt.Sprint(v)
		AllFieldsUnionOfCheck = append(AllFieldsUnionOfCheck, k)
	}
	if OFD != "astral_union" {
		for k := range data["fields"].(map[string]interface{})["kkt"].(map[string]interface{}) {
			AllFieldsHeadOfCheck = append(AllFieldsHeadOfCheck, k)
		}
		for k := range data["fields"].(map[string]interface{})["check"].(map[string]interface{}) {
			AllFieldsHeadOfCheck = append(AllFieldsHeadOfCheck, k)
		}
		for k := range data["fields"].(map[string]interface{})["positions"].(map[string]interface{}) {
			AllFieldPositionsOfCheck = append(AllFieldPositionsOfCheck, k)
		}
		for k := range data["fields"].(map[string]interface{})["others"].(map[string]interface{}) {
			AllFieldOtherOfCheck = append(AllFieldOtherOfCheck, k)
		}
	}
	//инициализация директории результатов
	if foundedLogDir, _ := doesFileExist(JSONRES); !foundedLogDir {
		os.Mkdir(JSONRES, 0777)
	}
	logsmap[LOGINFO_WITHSTD].Println("формирование json заданий начато")
	//инициализация входных данных
	logginInFile("открытие файла списка чеков")
	fileofheadername := "checks_header"
	if OFD == "astral_union" {
		fileofheadername = "union"
	}
	f, err := os.Open(DIRINFILES + fileofheadername + ".csv")
	if err != nil {
		descrError := fmt.Sprintf("не удлаось (%v) открыть файл (%v.csv) входных данных (шапки чека)", err, fileofheadername)
		logsmap[LOGERROR].Println(descrError)
		fmt.Println("Нажмите любую клавишу...")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		log.Panic(descrError)
	}
	defer f.Close()
	csv_red := csv.NewReader(f)
	csv_red.FieldsPerRecord = -1
	csv_red.LazyQuotes = true
	csv_red.Comma = ';'
	logginInFile("чтение списка чеков")
	lines, err := csv_red.ReadAll()
	if err != nil {
		descrError := fmt.Sprintf("не удлаось (%v) прочитать файл (%v.csv) входных данных (шапки чека)", err, fileofheadername)
		logsmap[LOGERROR].Println(descrError)
		fmt.Println("Нажмите любую клавишу...")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		log.Panic(descrError)
	}
	//инициализация номеров колонок
	//fmt.Printf("dd=%v\n", lines)
	if len(lines) > 0 {
		typetanletemp := "head"
		if OFD == "astral_union" {
			typetanletemp = "union"
		}
		//проверка на пустоту первой строки для такском
		currNumbLineOfHead := 0
		if strings.Join(lines[0], "") == "" {
			currNumbLineOfHead = currNumbLineOfHead + 1
			RowOfHeadInHeaderChecks = RowOfHeadInHeaderChecks + 1
		}
		lineoffields := strings.Join(lines[currNumbLineOfHead], "")
		if lineoffields[:5] == ";;;;;" {
			currNumbLineOfHead = currNumbLineOfHead + 1
			RowOfHeadInHeaderChecks = RowOfHeadInHeaderChecks + 1
		}
		FieldsNums = getNumberOfFieldsInCSV(lines[currNumbLineOfHead], FieldsNames, FieldsNums, typetanletemp)
		logginInFile(fmt.Sprintln("FieldsNums0", FieldsNums))
	}
	//fillFieldsNumByPositionTable(FieldsNames, FieldsNums, "checks_header.csv", "head")
	err = fillFieldsNumByPositionTable(FieldsNames, FieldsNums, "checks_poss.csv", "positions")
	//logginInFile(fmt.Sprintln("FieldsNums01", FieldsNums))
	if (err != nil) && (OFD != "astral_link") && (OFD != "astral_union") {
		descrError := fmt.Sprintf("не удлаось (%v) прочитать файл (checks_poss.csv) входных данных (позиции чека)", err)
		logsmap[LOGERROR].Println(descrError)
		fmt.Println("Нажмите любую клавишу...")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		log.Panic(descrError)
	}
	err = fillFieldsNumByPositionTable(FieldsNames, FieldsNums, "checks_other.csv", "other")
	if (err != nil) && (OFD != "astral_link") && (OFD != "astral_union") {
		logstr := fmt.Sprintf("не удлаось (%v) прочитать файл (checks_other.csv) входных данных (прочие данные чека(например марик))", err)
		logginInFile(logstr)
	}
	//fmt.Println("FieldsNames", FieldsNames)
	//fmt.Println("-------------------")
	//fmt.Println("FieldsNums", FieldsNums)
	//panic("ok")
	//перебор всех строчек файла с шапкоми чеков
	countWritedChecks := 0
	countAllChecks := len(lines) - 1
	logsmap[LOGINFO_WITHSTD].Printf("перебор %v чеков", countAllChecks)
	currLine := 0

	currNumbPos := 0
	PrevAllFieldsOfCheck := make(map[string]string)
	resultFindedPositions := make(map[int]map[string]string)
	if OFD == "astral_union" {
		lines = append(lines, []string{""})
	}
	for _, line := range lines {
		var summsOfPayment map[string]float64
		var findedPositions map[int]map[string]string
		var passedPositions []int
		fictivnaystr := false
		currNewCheck := false
		currLine++
		if currLine <= RowOfHeadInHeaderChecks {
			continue //пропускаем настройку названий столбцов
		}
		if currLine == RowOfHeadInHeaderChecks {
			currNewCheck = true
		}
		regKKT := line[FieldsNums[FieldsNames[COLBINDHEADFIELDKASSA]]]
		if regKKT == "" {
			logsmap[LOGERROR].Printf("строка №%v \"%v\" пропущена, так как в ней не опредлена касса", currLine, line)
			continue
		}
		//проверяем статус чека в ФНС
		if num, ok := FieldsNums[COLSTATUSINFNS]; ok {
			if !strings.Contains(strings.ToUpper(line[num]), strings.ToUpper("Ошибка")) {
				logsmap[LOGERROR].Printf("строка №%v \"%v\" пропущена, так как чек принят ФНС", currLine, line)
				continue
			}
		}
		if OFD == "astral_union" {
			if currLine == len(lines) {
				fictivnaystr = true
			}
		}
		CurrAllFieldsOfCheck := make(map[string]string)
		//for k := range CurrAllFieldsOfCheck {
		//	delete(CurrAllFieldsOfCheck, k)
		//}
		descrInfo := fmt.Sprintf("обработка строки %v из %v", currLine-1, countAllChecks)
		logginInFile(descrInfo)
		strlog := fmt.Sprintln(line)
		logginInFile(strlog)
		//заполняема поля шапки
		HeadOfCheck := make(map[string]string)
		HeadOfCheck[EMAILFIELD] = *email
		HeadOfCheck[NOPRINTFIELD] = fmt.Sprint(!*printonpaper)
		if OFD == "astral_union" {
			needGererationJson := false
			if !fictivnaystr {
				for _, field := range AllFieldsUnionOfCheck {
					CurrAllFieldsOfCheck[field] = getfieldval(line, FieldsNums, field)
				}
				if CurrAllFieldsOfCheck[COLFD] != PrevAllFieldsOfCheck[COLFD] {
					currNewCheck = true
				} else {
					currNumbPos++
				}
			} else {
				currNewCheck = true
			}
			if currNewCheck {
				if len(resultFindedPositions) > 0 {
					HeadOfCheck[COLFD] = PrevAllFieldsOfCheck[COLFD]
					HeadOfCheck[COLFP] = PrevAllFieldsOfCheck[COLFP]
					HeadOfCheck[COLDATE] = PrevAllFieldsOfCheck[COLDATE]
					HeadOfCheck[COLTAG1054] = PrevAllFieldsOfCheck[COLTAG1054]
					HeadOfCheck[COLKASSIR] = PrevAllFieldsOfCheck[COLKASSIR]
					HeadOfCheck[COLNAMECLIENT] = PrevAllFieldsOfCheck[COLNAMECLIENT]
					HeadOfCheck[COLINNCLIENT] = PrevAllFieldsOfCheck[COLINNCLIENT]
					HeadOfCheck[COLAMOUNTCHECK] = PrevAllFieldsOfCheck[COLAMOUNTCHECK]
					HeadOfCheck[COLNAL] = PrevAllFieldsOfCheck[COLNAL]
					HeadOfCheck[COLBEZ] = PrevAllFieldsOfCheck[COLBEZ]
					//for k := range findedPositions {
					//	for v := range findedPositions[k] {
					//		delete(findedPositions[k], v)
					//	}
					//	delete(findedPositions, k)
					//}
					//logsmap[LOGINFO].Println("resultFindedPositions=", resultFindedPositions)
					findedPositions = make(map[int]map[string]string)
					for k, v := range resultFindedPositions {
						findedPositions[k] = make(map[string]string)
						for kk, vv := range v {
							findedPositions[k][kk] = vv
						}
					}
					needGererationJson = true
				}
				currNumbPos = 1
				for k := range resultFindedPositions {
					for v := range resultFindedPositions[k] {
						delete(resultFindedPositions[k], v)
					}
					delete(resultFindedPositions, k)
				}
			}
			//logsmap[LOGINFO].Println("resultFindedPositions0", resultFindedPositions)
			resultFindedPositions[currNumbPos] = make(map[string]string)
			for k, v := range CurrAllFieldsOfCheck {
				//logsmap[LOGINFO].Println("k", k, "v", v)
				resultFindedPositions[currNumbPos][k] = v
			}
			//logsmap[LOGINFO].Println("resultFindedPositions1", resultFindedPositions)
			//findedPositions[currNumbPos] = AllFieldsOfCheck
			for k := range PrevAllFieldsOfCheck {
				delete(PrevAllFieldsOfCheck, k)
			}
			for k := range CurrAllFieldsOfCheck {
				PrevAllFieldsOfCheck[k] = CurrAllFieldsOfCheck[k]
			}
			if !needGererationJson {
				continue
			}
		}
		for _, field := range AllFieldsHeadOfCheck {
			//println(FieldsNames[field])
			//FieldsNames[COLTYPECHECK]
			if !isInvField(FieldsNames[field]) {
				HeadOfCheck[field] = getfieldval(line, FieldsNums, field)
				//strloggin := fmt.Sprintf("заполнение поля %v значением %v\n", field, HeadOfCheck[field])
				//logginInFile(strloggin)
			}
		}
		//заполняем поля шапки с префиксом inv - те эти поля будут - это значения полей позиций
		for _, field := range AllFieldPositionsOfCheck {
			if isInvField(FieldsNames[field]) {
				HeadOfCheck["inv$"+field] = getfieldval(line, FieldsNums, field)
			}
		}
		if (HeadOfCheck[COLBINDHEADFIELDKASSA] == "") && (OFD != "astral_json") && (OFD != "astral_union") {
			logsmap[LOGERROR].Printf("строка №%v \"%v\" пропущена, так как в ней не опредлена касса", currLine, line)
			continue
		}
		//проверяем тип чека
		if strings.Contains(HeadOfCheck[COLTYPECHECK], "Отчет об открытии смены") ||
			strings.Contains(HeadOfCheck[COLTYPECHECK], "Отчет о закрытии смены") {
			logginInFile("пропускаем строку, так как она является отчетом о закрытии или открытии смены")
			continue
		}
		valbindkassa := HeadOfCheck[COLBINDHEADFIELDKASSA]
		valbindcheck := HeadOfCheck[COLBINDHEADDIELDCHECK]
		//ищем позиции в файле позиций чека, которые бы соответсвовали бы текущеё строке чека //по номеру ФН и названию кассы
		checkDescrInfo := fmt.Sprintf("(ФД %v (ФП %v) от %v)", HeadOfCheck[COLFD], HeadOfCheck[COLFP], HeadOfCheck[COLDATE])
		if (OFD != "astral_link") && (OFD != "astral_union") {
			descrInfo = fmt.Sprintf("для чека %v ищем позиции", checkDescrInfo)
			logginInFile(descrInfo)
			findedPositions, summsOfPayment = findPositions(valbindkassa, valbindcheck, FieldsNames, FieldsNums, &passedPositions)
		} else if OFD != "astral_union" {
			//findedPositions, summsOfPayment = fillpossitonsbyref(HeadOfCheck, FieldsNames, FieldsNums)
			descrInfo = fmt.Sprintf("для чека %v получаем позиции get запросом", checkDescrInfo)
			logginInFile(descrInfo)
			//findedPositions, summsOfPayment, err = fillpossitonsbyrefastral(HeadOfCheck[COLFD], HeadOfCheck[COLFP], HeadOfCheck[COLFNKKT])
			_, _, err = fillpossitonsbyrefastral(HeadOfCheck[COLFD], HeadOfCheck[COLFP], HeadOfCheck[COLFNKKT])
			if err != nil {
				logerrordescr := fmt.Sprintf("ошибка1 (%v) получение данных позиций для чека %v", err, checkDescrInfo)
				logsmap[LOGERROR].Println(logerrordescr)
				_, _, err = fillpossitonsbyrefastral(HeadOfCheck[COLFD], HeadOfCheck[COLFP], HeadOfCheck[COLFNKKT])
			}
			if err != nil {
				logerrordescr := fmt.Sprintf("ошибка2 (%v) получение данных позиций для чека %v", err, checkDescrInfo)
				logsmap[LOGERROR].Println(logerrordescr)
				log.Panic(logerrordescr)
				//continue
			}
			//fmt.Println(findedPositions)
			//fmt.Println(summsOfPayment)
			continue
		}
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
		amountOfCheck := 0.0
		//logsmap[LOGINFO_WITHSTD].Println("findedPositions=", findedPositions)
		for _, pos := range findedPositions {
			spos, errgen := strconv.ParseFloat(pos[COLAMOUNTPOS], 64)
			if errgen != nil {
				prloc, errlocpr := strconv.ParseFloat(pos[COLPRICE], 64)
				quloc, errlocqt := strconv.ParseFloat(pos[COLQUANTITY], 64)
				if (errlocpr != nil) || (errlocqt != nil) {
					descrErr := fmt.Sprintf("ошибка (%v, %v) парсинга строки (%v, %v) суммы для чека %v", errlocpr, errlocqt, pos[COLPRICE], pos[COLQUANTITY], checkDescrInfo)
					logsmap[LOGERROR].Println(descrErr)
				} else {
					spos = prloc * quloc
					errgen = nil
					pos[COLAMOUNTPOS] = fmt.Sprint(spos)
				}
			}
			if errgen != nil {
				descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v суммы для чека %v", err, pos[COLAMOUNTPOS], checkDescrInfo)
				logsmap[LOGERROR].Println(descrErr)
				continue
			}
			amountOfCheck += spos
		}
		mistakesInPayment := false
		if (OFD == "astral_json") || (OFD == "astral_union") {
			amountOfCheckinHead, errparseam := strconv.ParseFloat(HeadOfCheck[COLAMOUNTCHECK], 64)
			if errparseam != nil {
				descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v суммы для всего чека %v", errparseam, HeadOfCheck[COLAMOUNTCHECK], checkDescrInfo)
				logsmap[LOGERROR].Println(descrErr)
				continue
			}
			if amountOfCheckinHead != amountOfCheck {
				//mistakesInPayment = checkMistakeInPayments(amountOfCheck, summsOfPayment)
				descrErr := fmt.Sprintf("ошибка: сумма итого по чеку %v не совпадает с суммой %v по позициям для чека %v", amountOfCheckinHead, amountOfCheck, checkDescrInfo)
				logsmap[LOGERROR].Println(descrErr)
				continue
			}
		}
		if OFD == "ofdru" {
			mistakesInPayment = checkMistakeInPayments(amountOfCheck, summsOfPayment)
			if mistakesInPayment {
				logginInFile("ошибка в суммах оплат, пытаемся получить данные из ссылки чека")
				var receipt TReceiptOFD
				var descrErr string
				var err error
				hypperlinkjson := replacefieldbyjsonhrep(HeadOfCheck[COLLINK])
				//fmt.Println("hypperlinkjson", hypperlinkjson)
				receipt, descrErr, err = fetchcheck(HeadOfCheck[COLFD], HeadOfCheck[COLFP], hypperlinkjson)
				analyzeComlite := true
				if err != nil {
					logsmap[LOGERROR].Println(descrErr)
					analyzeComlite = false
				}
				if analyzeComlite {
					summsOfPayment[COLNAL] = float64(receipt.Document.Amount_Cash) / 100
					summsOfPayment[COLBEZ] = float64(receipt.Document.Amount_ECash) / 100
					summsOfPayment[COLCREDIT] = float64(receipt.Document.Amount_Loan) / 100
					summsOfPayment[COLAVANCE] = float64(receipt.Document.Amount_Advance) / 100
					summsOfPayment[COLVSTRECHPREDST] = float64(receipt.Document.Amount_Granting) / 100
					mistakesInPayment = checkMistakeInPayments(amountOfCheck, summsOfPayment)
				}
			}

		}
		if mistakesInPayment {
			deskMistPaym := fmt.Sprintf("Для чека %v не возможно определить сумму оплат. Сделаёте это вручную. И укажите суммы оплат далее...", checkDescrInfo)
			summPaymentsCurrDescr := fmt.Sprintf("Сейчас суммы оплат такие: наличными %v", summsOfPayment[COLNAL])
			summPaymentsCurrDescr += fmt.Sprintf(", картой %v", summsOfPayment[COLBEZ])
			summPaymentsCurrDescr += fmt.Sprintf(", кредитом %v", summsOfPayment[COLCREDIT])
			summPaymentsCurrDescr += fmt.Sprintf(", дебетом %v", summsOfPayment[COLAVANCE])
			summPaymentsCurrDescr += fmt.Sprintf(", встречным представлением %v", summsOfPayment[COLVSTRECHPREDST])
			logginInFile(deskMistPaym)
			logginInFile(summPaymentsCurrDescr)
			fmt.Println(deskMistPaym)
			fmt.Println(summPaymentsCurrDescr)
			fmt.Printf("Сумма чека %v\n", amountOfCheck)

			fmt.Printf("Введите сумму оплаты наличными (%v):\n", summsOfPayment[COLNAL])
			input := bufio.NewScanner(os.Stdin)
			input.Scan()
			nalch := summsOfPayment[COLNAL]
			nalstr := input.Text()
			if notEmptyFloatField(nalstr) {
				nalch, err = strconv.ParseFloat(nalstr, 64)
				if err != nil {
					descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для налчиного расчёта", err, nalstr)
					logsmap[LOGERROR].Println(descrErr)
				}
			}
			fmt.Printf("Введите сумму оплаты безналичными (%v):\n", summsOfPayment[COLBEZ])
			input.Scan()
			bezch := summsOfPayment[COLBEZ]
			bezstr := input.Text()
			if notEmptyFloatField(bezstr) {
				bezch, err = strconv.ParseFloat(bezstr, 64)
				if err != nil {
					descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для безналичного расчёта", err, bezstr)
					logsmap[LOGERROR].Println(descrErr)
				}
			}
			fmt.Printf("Введите сумму оплаты дебетом (%v):\n", summsOfPayment[COLAVANCE])
			input.Scan()
			avnch := summsOfPayment[COLAVANCE]
			avnstr := input.Text()
			if notEmptyFloatField(avnstr) {
				avnch, err = strconv.ParseFloat(avnstr, 64)
				if err != nil {
					descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для аванса", err, avnstr)
					logsmap[LOGERROR].Println(descrErr)
				}
			}
			fmt.Printf("Введите сумму оплаты кредитом (%v):\n", summsOfPayment[COLCREDIT])
			input.Scan()
			crdch := summsOfPayment[COLCREDIT]
			crdstr := input.Text()
			if notEmptyFloatField(crdstr) {
				crdch, err = strconv.ParseFloat(crdstr, 64)
				if err != nil {
					descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для кредита расчёта", err, crdstr)
					logsmap[LOGERROR].Println(descrErr)
				}
			}
			fmt.Printf("Введите сумму оплаты кредитом (%v):\n", summsOfPayment[COLVSTRECHPREDST])
			input.Scan()
			vstrch := summsOfPayment[COLVSTRECHPREDST]
			vstrstr := input.Text()
			if notEmptyFloatField(vstrstr) {
				vstrch, err = strconv.ParseFloat(vstrstr, 64)
				if err != nil {
					descrErr := fmt.Sprintf("ошибка (%v) парсинга строки %v для встречного представления", err, vstrstr)
					logsmap[LOGERROR].Println(descrErr)
				}
			}
			summsOfPayment[COLNAL] = nalch
			summsOfPayment[COLBEZ] = bezch
			summsOfPayment[COLAVANCE] = avnch
			summsOfPayment[COLCREDIT] = crdch
			summsOfPayment[COLVSTRECHPREDST] = vstrch
			summPaymentsCurrDescr = fmt.Sprintf("Сейчас суммы оплат такие: наличными %v", summsOfPayment[COLNAL])
			summPaymentsCurrDescr += fmt.Sprintf(", картой %v", summsOfPayment[COLBEZ])
			summPaymentsCurrDescr += fmt.Sprintf(", кредитом %v", summsOfPayment[COLCREDIT])
			summPaymentsCurrDescr += fmt.Sprintf(", дебетом %v", summsOfPayment[COLAVANCE])
			summPaymentsCurrDescr += fmt.Sprintf(", встречным представлением %v", summsOfPayment[COLVSTRECHPREDST])
			fmt.Println(summPaymentsCurrDescr)
			logginInFile("суммы оплат были изменены")
			logginInFile(deskMistPaym)
			logginInFile(summPaymentsCurrDescr)
			fmt.Printf("Нажмите любую клавишу для продолжения формирования json-заданий...")
			input.Scan()
		}
		//fmt.Println("------------------------------")
		//fmt.Println("summsOfPayment", summsOfPayment)
		//fmt.Println("HeadOfCheck", HeadOfCheck)
		//fmt.Println("------------------------------")
		//переносим суммы оплат из позиций, если сумма оплат была указана у позиций
		for k, v := range summsOfPayment {
			HeadOfCheck[k] = strconv.FormatFloat(v, 'f', -1, 64)
		}
		//fmt.Println("*****************************")
		//fmt.Println("HeadOfCheck", HeadOfCheck)
		//fmt.Println("------------------------------")
		//fmt.Println("findedPositions", findedPositions)
		//fmt.Println("------------------------------")
		//panic("ok")
		logginInFile(descrInfo)
		//производим сложный анализ
		analyzeComlite := true
		strloggin := fmt.Sprintln("countOfPositions=", countOfPositions, "OFD=", OFD)
		logginInFile(strloggin)
		if (countOfPositions > 0) && (OFD == "ofdru") { //если для чека были найдены позиции
			logginInFile("проверка требований к марке")
			neededGetMarks := false
			for _, pos := range findedPositions {
				if (pos[COLPREDMET] == "ТМ") || (pos[COLPREDMET] == "АТМ") {
					logginstr := fmt.Sprintf("для позицции %v требуется получить марку", pos)
					logginInFile(logginstr)
					neededGetMarks = true
					break
				}
			}
			if *fetchalways {
				neededGetMarks = true
			}
			logginstr := fmt.Sprintf("neededGetMarks = %v", neededGetMarks)
			logginInFile(logginstr)
			if neededGetMarks {
				logginInFile("будем получать/читать json с марками")
				//fmt.Println(checkDescrInfo, "полчаем json для марки")
				var receipt TReceiptOFD
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
				strloggin := fmt.Sprintln("анализируем поле", FieldsNames[COLLINK])
				logginInFile(strloggin)
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
					//"Code_Undefined":null,"Code_EAN_8":null,"Code_EAN_13":"4603739334345","Code_ITF_14":null,"Code_GS_1":null,"Code_GS_1M":null,"Code_KMK":null,"Code_MI":null,"Code_EGAIS_2":null,"Code_EGAIS_3":null,"Code_F_1":null,"Code_F_2":null,"Code_F_3":null,"Code_F_4":null,"Code_F_5":null,"Code_F_6":null
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
						if strings.EqualFold(strings.ToLower(strings.TrimSpace(itemPos.Name)), strings.ToLower(strings.TrimSpace(posFined[COLNAME]))) {
							//fmt.Println("нашли позицию", itemPos.Name)
							//logsmap[LOG]
							//posFined[COLMARK] = itemPos.ProductCode.Code_GS_1M
							if posFined[COLMARK] != "" {
								//fmt.Println("позиция уже имеет марку", posFined[COLMARK])
								continue
							}
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
			logginInFile("генерируем json файл")
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
			loggstr := fmt.Sprintln(jsonres)
			logginInFile(loggstr)
			as_json, err := json.MarshalIndent(jsonres, "", "\t")
			if err != nil {
				descrError := fmt.Sprintf("ошибка (%v) преобразвания объекта в json для чека %v", err, checkDescrInfo)
				logsmap[LOGERROR].Println(descrError)
				continue //пропускаем чек
			}
			dir_file_name := fmt.Sprintf("%v%v/", JSONRES, HeadOfCheck[COLFNKKT])
			if foundedLogDir, _ := doesFileExist(dir_file_name); !foundedLogDir {
				logginInFile("генерируем папку результатов, если раньше она не была сгенерирована")
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
	println("Нажмите любую клавишу...")
	input.Scan()
}

func fillFieldsNumByPositionTable(fieldsnames map[string]string, fieldsnums map[string]int, filename, partOfCheck string) error {
	fullnameoffile := DIRINFILES + filename
	existfile, _ := doesFileExist(fullnameoffile)
	if !existfile {
		return fmt.Errorf("файл %v не найден", fullnameoffile)
	}
	f, err := os.Open(fullnameoffile)
	if err != nil {
		log.Panic("не удлась открыть файл позиций чека", err)
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
		//logginInFile(fmt.Sprintln("lines[0]", lines[0]))
		//logginInFile(fmt.Sprintln("lines[0]", lines[0]))
		//проверка на пустоту первой строки для такском
		currNumbLineOfHead := 0
		if strings.Join(lines[0], "") == "" {
			currNumbLineOfHead = currNumbLineOfHead + 1
		}
		lineoffields := strings.Join(lines[currNumbLineOfHead], "")
		if lineoffields[:5] == ";;;;;" {
			currNumbLineOfHead = currNumbLineOfHead + 1
		}
		getNumberOfFieldsInCSV(lines[currNumbLineOfHead], fieldsnames, fieldsnums, partOfCheck)
	}
	return nil
}

func findPositions(valbindkassainhead, valbindcheckinhead string, fieldsnames map[string]string, fieldsnums map[string]int, passedPositions *[]int) (map[int]map[string]string, map[string]float64) {
	//fmt.Println("valbindkassainhead", valbindkassainhead)
	//fmt.Println("valbindcheckinhead", valbindcheckinhead)
	//logsmap[LOGINFO].Println("*****************************************************")
	//logsmap[LOGINFO].Println("valbindkassainhead", valbindkassainhead)
	//logsmap[LOGINFO].Println("valbindcheckinhead", valbindcheckinhead)
	res := make(map[int]map[string]string)
	summsPayment := make(map[string]float64)
	f, err := os.Open(DIRINFILES + "checks_poss.csv")
	if err != nil {
		errDescr := fmt.Sprintf("ошибка (%v), не удлась открыть файл позиций чека", err)
		logsmap[LOGERROR].Println(errDescr)
		log.Panic(errDescr)
	}
	defer f.Close()
	csv_red := csv.NewReader(f)
	csv_red.FieldsPerRecord = -1
	csv_red.LazyQuotes = true
	csv_red.Comma = ';'
	lines, err := csv_red.ReadAll()
	if err != nil {
		errDescr := fmt.Sprintf("ошибка (%v). не удлась прочитать csv файл позиций чека", err)
		logsmap[LOGERROR].Println(errDescr)
		log.Panic(errDescr)
	}
	currPos := 0
	currLine := 0
	valbindkassainpos := ""
	valbindcheckpos := ""
	wasfindedpositions := false
	//wasfindedandlosspositions := false
	for _, line := range lines { //перебор всех строк в файле позиций чека
		currLine++
		//fmt.Println(line)
		if currLine == 1 {
			continue
		}
		if OFD == "sbis" {
			kassaname := getfieldval(line, fieldsnums, COLBINDPOSFIELDKASSA)
			docnum := getfieldval(line, fieldsnums, COLBINDPOSFIELDCHECK)
			if kassaname != "" || docnum != "" {
				valbindkassainpos = kassaname
				valbindcheckpos = docnum
				continue
			}
		} else {
			valbindkassainpos = getfieldval(line, fieldsnums, COLBINDPOSFIELDKASSA)
			valbindcheckpos = getfieldval(line, fieldsnums, COLBINDPOSFIELDCHECK)
		}
		valbindkassainhead = strings.TrimLeft(valbindkassainhead, "0")
		valbindkassainpos = strings.TrimLeft(valbindkassainpos, "0")
		valbindcheckinhead = strings.TrimLeft(valbindcheckinhead, "0")
		valbindcheckpos = strings.TrimLeft(valbindcheckpos, "0")
		if (valbindkassainhead != valbindkassainpos) || (valbindcheckinhead != valbindcheckpos) {
			if *checkdoublepos {
				if wasfindedpositions {
					//wasfindedandlosspositions = true
					break
				}
			}
			continue
		}
		//if !notEmptyFloatField(getfieldval(line, fieldsnums, COLQUANTITY)) {
		//	continue //пропускаем строки с пустым или нулевым количесве
		//}
		logstr := fmt.Sprintln("Найдена строка", line)
		wasfindedpositions = true
		logginInFile(logstr)
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
						//if OFD == "ofdru" {
						//} else {
						summsPayment[field] = summsPayment[field] + currSumm
						//}
					}
				}
			}
		}
		//logginInFile(fmt.Sprintln("AllFieldPositionsOfCheck", AllFieldPositionsOfCheck))
		//logginInFile(fmt.Sprintln("fieldsnames", fieldsnames))
		//logginInFile(fmt.Sprintln("fieldsnums", fieldsnums))
		for _, field := range AllFieldPositionsOfCheck {
			logginInFile(fmt.Sprintln("field", field))
			//logginInFile(fmt.Sprintln("fieldsnames[field]", fieldsnames[field]))
			if !isInvField(fieldsnames[field]) {
				//logginInFile(fmt.Sprintln("notinv"))
				//logginInFile(fmt.Sprintln("fieldsnums[field]", fieldsnums[field]))
				//logginInFile(fmt.Sprintln("val=", getfieldval(line, fieldsnums, field)))
				res[currPos][field] = getfieldval(line, fieldsnums, field)
				//logsmap[LOGINFO_WITHSTD].Println(field)
			}
		}
		if (OFD == "platforma") || (OFD == "firstofd") {
			//ищем марки в таблице марок
			logginInFile("ищем марки в дполнительном файле платформы ОФД")
			logginInFile(fmt.Sprintln("passedPositions before", passedPositions))
			marka, err := findMarkInOtherFile(res[currPos][COLBINDPOSFIELDKASSA], res[currPos][COLBINDPOSFIELDCHECK],
				res[currPos][COLBINDPOSPOSFIELDCHECK], fieldsnums, passedPositions)
			logginInFile(fmt.Sprintln("passedPositions after", passedPositions))
			if err != nil {
				errDescr := fmt.Sprintf("ошибка (%v), не удалось прочитать файл марок", err)
				logsmap[LOGERROR].Println(errDescr)
				log.Panic(errDescr)
			}
			if marka != "" {
				res[currPos][COLMARK] = marka
			} else {
				logginInFile("марка в файле марок не найдена")
			}
		}
	} //перебор всех строк в файле позиций чека
	return res, summsPayment
} //findPositions

// func fillpossitonsbyref(headofcheck, fieldsnames map[string]string, fieldsnums map[string]int) (map[int]map[string]string, map[string]float64) {
func fillpossitonsbyrefastral(fd, fp, fn string) (map[int]map[string]string, map[string]float64, error) {
	var resp *http.Response
	var err error
	var body []byte
	res := make(map[int]map[string]string)
	summsPayment := make(map[string]float64)
	//https://ofd.astralnalog.ru/api/v4.2/landing.pdfNew?fiscalSign=<Фискальный признак>&fiscalDocumentNumber=<Номер документа>&fiscalDriveNumber=<Номер ФН>
	hyperlinkonjson := fmt.Sprintf("https://ofd.astralnalog.ru/api/v4.2/landing.pdfNew?fiscalSign=%v&fiscalDocumentNumber=%v&fiscalDriveNumber=%v", fp, fd, fn)
	nameoffile := fd + "_" + fp + ".pdf"
	fullFileName := DIROFREQUESTASTRAL + nameoffile
	if !alredyGettedFetchAstral(fullFileName) {
		logginInFile("перед get запросом делаем паузу...")
		duration := time.Millisecond * 10
		time.Sleep(duration)
		strlog := fmt.Sprintf("получение данных о чеке по ссылке %v", hyperlinkonjson)
		logginInFile(strlog)
		resp, err = http.Get(hyperlinkonjson)
		if err != nil {
			errDescr := fmt.Sprintf("ошибка(не удалось получить ответ от сервера ОФД): %v. Не удалось получить данные о чеке по ссылке %v", err, hyperlinkonjson)
			logsmap[LOGERROR].Println(errDescr)
			return res, summsPayment, err
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			errDescr := fmt.Sprintf("ошибка(прочитать данные от сервера ОФД): %v. Не удалось получить данные о чеке по ссылке %v", err, hyperlinkonjson)
			logsmap[LOGERROR].Println(errDescr)
			return res, summsPayment, err
		}
		//logginInFile("------------------")
		//logginInFile(string(fullFileName))
		//logginInFile(string(body))
		//logginInFile("------------------------------------")
		ioutil.WriteFile(fullFileName, body, 0644)
	} else {
		logginInFile(fmt.Sprintf("запрос %v был уже выполнен ранее", hyperlinkonjson))
	}
	return res, summsPayment, err
}

func findMarkInOtherFile(kassa, doc, posnum string, fieldsnums map[string]int, passedPositions *[]int) (string, error) {
	var marka string
	//проверяем существует ли файл
	logstr := fmt.Sprintln("ищме в файле марок:", "kassa", kassa, "doc", doc, "posnum", posnum)
	logginInFile(logstr)
	if fileofmarksexist, _ := doesFileExist(DIRINFILES + "checks_other.csv"); !fileofmarksexist {
		//logginInFile("файл марок не существует")
		return "", nil //если не существует, то просто нет марок
	}
	f, err := os.Open(DIRINFILES + "checks_other.csv")
	if err != nil {
		errDescr := fmt.Sprintf("ошибка (%v), не удлась открыть файл марок чека", err)
		logsmap[LOGERROR].Println(errDescr)
		return "", errors.New(errDescr)
	}
	defer f.Close()
	csv_red := csv.NewReader(f)
	csv_red.FieldsPerRecord = -1
	csv_red.LazyQuotes = true
	csv_red.Comma = ';'
	lines, err := csv_red.ReadAll()
	if err != nil {
		errDescr := fmt.Sprintf("ошибка (%v), не удлась открыть csv файл марок чека", err)
		logsmap[LOGERROR].Println(errDescr)
		return "", errors.New(errDescr)
	}
	currLine := 0
	for _, line := range lines { //перебор всех строк в файле позиций чека
		currLine++
		if currLine == 1 {
			continue
		}
		logginInFile(fmt.Sprintln("passedPositions=", *passedPositions))
		logginInFile(fmt.Sprintln("currLine=", currLine))
		found := slices.Contains(*passedPositions, currLine)
		logginInFile(fmt.Sprintln("found", found))
		if found {
			continue
		}
		//logstr := fmt.Sprintln(line)
		//logginInFile(logstr)
		kassaother := getfieldval(line, fieldsnums, COLBINDOTHERKASSS)
		docother := getfieldval(line, fieldsnums, COLBINDOTHERCHECK)
		posnumother := getfieldval(line, fieldsnums, COLBINDOTHERPOS)

		//logsmap[LOGINFO_WITHSTD].Println("kassa", kassa)
		//logsmap[LOGINFO_WITHSTD].Println("doc", doc)
		//logsmap[LOGINFO_WITHSTD].Println("posnum", posnum)
		//logsmap[LOGINFO_WITHSTD].Println("kassaother", kassaother)
		//logsmap[LOGINFO_WITHSTD].Println("docother", docother)
		//logsmap[LOGINFO_WITHSTD].Println("posnumother", posnumother)

		//logsmap[LOGINFO_WITHSTD].Println("kassa=kassaother", kassa == kassaother)
		//logsmap[LOGINFO_WITHSTD].Println("doc=docother", doc == docother)
		//logsmap[LOGINFO_WITHSTD].Println("posnum=posnumother", posnum == posnumother)

		//logsmap[LOGINFO_WITHSTD].Println("(kassaother != kassa) || (docother != doc) || (posnumother != posnum)", (kassaother != kassa) || (docother != doc) || (posnumother != posnum))
		//logstr = fmt.Sprintln("строка в файле марок:", "kassaother", kassaother, "docother", docother, "posnumother", posnumother)
		//logginInFile(logstr)
		if (kassaother != kassa) || (docother != doc) || (posnumother != posnum) {
			//logginInFile("не подходит строка")
			continue
		}
		(*passedPositions) = append(*passedPositions, currLine)
		//logginInFile("подходит строка")
		marka = getfieldval(line, fieldsnums, COLMARKOTHER)
		if marka == "" {
			marka = getfieldval(line, fieldsnums, COLMARKOTHER2)
		}
		//logstr = fmt.Sprintln("марка", marka)
		//logginInFile(logstr)
		break
	}
	//logsmap[LOGINFO_WITHSTD].Println("marka", marka)
	return marka, nil
}

func generateCheckCorrection(headofcheck map[string]string, poss map[int]map[string]string) (TCorrectionCheck, string, error) {
	var checkCorr TCorrectionCheck
	strInfoAboutCheck := fmt.Sprintf("(ФД %v, ФП %v %v)", headofcheck[COLFD], headofcheck[COLFP], headofcheck[COLDATE])
	chekcCorrTypeLoc := ""
	typeCheck := strings.ToLower(headofcheck[COLTAG1054])
	if typeCheck == "приход" {
		chekcCorrTypeLoc = "sellCorrection"
	}
	if (typeCheck == "возврат прихода") || (typeCheck == "возврат") {
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
	osnLoc := getOsnFromChernovVal(headofcheck[COLOSN])
	if osnLoc != "" {
		checkCorr.TaxationType = osnLoc
	}
	//strconv.ParseBool
	checkCorr.Electronically, _ = strconv.ParseBool(headofcheck[NOPRINTFIELD])
	if headofcheck[EMAILFIELD] == "" {
		checkCorr.Electronically = false
	} else {
		checkCorr.Electronically = true
	}
	correctionType := "self"
	correctionBaseNumber := ""
	if *byPrescription {
		correctionType = "instruction"
		correctionBaseNumber = *docNumbOfPrescription
	}
	checkCorr.CorrectionType = correctionType
	checkCorr.CorrectionBaseDate = headofcheck[COLDATE]
	checkCorr.CorrectionBaseNumber = correctionBaseNumber
	checkCorr.ClientInfo.EmailOrPhone = headofcheck[EMAILFIELD]
	checkCorr.Operator.Name = headofcheck[COLKASSIR]
	if headofcheck[COLINNCLIENT] != "" {
		checkCorr.ClientInfo.Vatin = headofcheck[COLINNCLIENT]
	}
	if headofcheck[COLNAMECLIENT] != "" {
		checkCorr.ClientInfo.Name = headofcheck[COLNAMECLIENT]
	}
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
	//logsmap[LOGINFO].Println("avance", avance)
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
	currFD := headofcheck[COLFD]
	//if currFP == "" {
	//	currFP = headofcheck[COLFD]
	//}
	//в тег 1192 - записываем ФП //(если нет ФП, то записываем ФД) - отменил
	if (currFP != "") || (currFD != "") {
		newAdditionalAttribute := TTag1192_91{Type: "additionalAttribute"}
		if currFP != "" {
			newAdditionalAttribute.Value = currFP
		} else {
			newAdditionalAttribute.Value = currFD
		}
		newAdditionalAttribute.Print = true
		checkCorr.Items = append(checkCorr.Items, newAdditionalAttribute)
	}
	if (currFD != "") && (currFP != "") {
		newAdditionalAttribute := TTag1192_91{Type: "userAttribute"}
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
		if prch != 0 {
			qchRight := math.Round(sch*1000/prch) / 1000
			if qchRight != qch {
				qch = qchRight
				newPos.Quantity = qch
			}
		}
		if qch == 0 {
			if prch != 0 {
				qchTemp := (sch * 1000 / prch)
				qch = math.Round(qchTemp) / 1000
			} else {
				qch = 1
			}
			newPos.Quantity = qch
		}
		measunit := "piece"
		if math.Round(qch) != qch {
			measunit = getMeasUnitFromStr(*measurementUnitOfFracQuantSimple)
		}
		newPos.MeasurementUnit = measunit //liter
		newPos.PaymentMethod = getSposobRash(pos[COLSPOSOB])
		//commodityWithMarking
		newPos.PaymentObject = getPredmRasch(pos[COLPREDMET])
		newPos.Tax = new(TTaxNDS)
		stavkaNDSStr := STAVKANDSNONE
		if pos[COLSTAVKANDS20] != "" {
			stavkaNDSStr = STAVKANDS20
		} else if pos[COLSTAVKANDS10] != "" {
			stavkaNDSStr = STAVKANDS10
		} else if pos[COLSTAVKANDS0] != "" {
			stavkaNDSStr = STAVKANDS0
		} else if pos[COLSTAVKANDS120] != "" {
			stavkaNDSStr = STAVKANDS120
		} else if pos[COLSTAVKANDS110] != "" {
			stavkaNDSStr = STAVKANDS110
		}
		logginInFile(fmt.Sprintln("pos[COLSTAVKANDS]=", pos[COLSTAVKANDS]))
		if pos[COLSTAVKANDS] != "" {
			if strings.Contains(pos[COLSTAVKANDS], "20") {
				stavkaNDSStr = STAVKANDS20
			} else if strings.Contains(pos[COLSTAVKANDS], "10") {
				stavkaNDSStr = STAVKANDS10
			} else if strings.Contains(pos[COLSTAVKANDS], "0%") || strings.Contains(pos[COLSTAVKANDS], "0 %") {
				stavkaNDSStr = STAVKANDS0
			}
			//if pos[COLSTAVKANDS] != "НДС не облагается" {
			//	logsmap[LOGERROR].Printf("требуется доработка программы для учёта ставки НДС %v в чеке", pos[COLSTAVKANDS])
			//}
		}
		newPos.Tax.Type = stavkaNDSStr
		postDataExist := false
		if pos[COLPRIZAGENTA] != "" {
			if strings.ToUpper(pos[COLPRIZAGENTA]) == "КОМИССИОНЕР" {
				postDataExist = true
			}
		} else {
			if pos[COLINNOFSUPPLIER] != "" {
				postDataExist = true
			}
		}
		if postDataExist {
			newPos.AgentInfo = new(TAgentInfo)
			newPos.AgentInfo.Agents = append(newPos.AgentInfo.Agents, "commissionAgent")
			newPos.SupplierInfo = new(TSupplierInfo)
			newPos.SupplierInfo.Vatin = pos[COLINNOFSUPPLIER]
			//teststrloc1 := fmt.Sprintf("Номер колнки ИНН поставщика %v", COLINNOFSUPPLIER)
			//teststrloc2 := fmt.Sprintf("ИНН поставщика %v", pos[COLINNOFSUPPLIER])
			//logginInFile(teststrloc1)
			//logginInFile(teststrloc2)
			newPos.SupplierInfo.Name = pos[COLNAMEOFSUPPLIER]
			if pos[COLTELOFSUPPLIER] != "" {
				newPos.SupplierInfo.Phones = append(newPos.SupplierInfo.Phones, pos[COLTELOFSUPPLIER])
			}
		}
		//chanePredmetRascheta := false
		if pos[COLMARK] != "" {
			if newPos.PaymentObject == "commodity" {
				newPos.PaymentObject = "commodityWithMarking"
			}
			measunit = "piece"
			if qch != 1 {
				measunit = getMeasUnitFromStr(*measurementUnitOfFracQuantMark)
				newPos.MeasurementUnit = measunit
			}
			currMark := pos[COLMARK]
			if (pos[NAMETYPEOFMARK] == "Undefined") || (pos[NAMETYPEOFMARK] == "EAN_8") ||
				(pos[NAMETYPEOFMARK] == "EAN_13") || (pos[NAMETYPEOFMARK] == "ITF_14") {
				newPos.ProductCodes = new(TProductCodesAtol)
				setMarkInArolDriverCorrenspOFDMark(newPos.ProductCodes, currMark, pos[NAMETYPEOFMARK])
			} else {
				currMarkInBase64 := base64.StdEncoding.EncodeToString([]byte(currMark))
				//newPos.Mark = currMarkInBase64
				newPos.ImcParams = new(TImcParams)
				//newPos.ImcParams.ImcType = "auto"
				newPos.ImcParams.Imc = currMarkInBase64
				//newPos.ImcParams.ItemEstimatedStatus = "itemStatusUnchanged" //статус товара не изменился
				//newPos.ImcParams.ItemUnits = measunit
				//newPos.ImcParams.ImcModeProcessing = 0
				////newPos.ImcParams.ImcBarcode
				//newPos.ImcParams.ItemQuantity = newPos.Quantity
				//newPos.ImcParams.ItemInfoCheckResult = new(TItemInfoCheckResult)
				//newPos.ImcParams.ItemInfoCheckResult.ImcCheckFlag = true
				//newPos.ImcParams.ItemInfoCheckResult.ImcCheckResult = true
				//newPos.ImcParams.ItemInfoCheckResult.ImcStatusInfo = true
				//newPos.ImcParams.ItemInfoCheckResult.ImcEstimatedStatusCorrect = true
				//newPos.ImcParams.ItemInfoCheckResult.EcrStandAloneFlag = false
				////chanePredmetRascheta = true
			}
			//if chanePredmetRascheta {
			//	newPos.PaymentObject = addMarkToPredmetRasheta(newPos.PaymentObject)
			//}
		}
		checkCorr.Items = append(checkCorr.Items, newPos)
	} //запись всех позиций чека
	return checkCorr, "", nil
}

//func addMarkToPredmetRasheta(predmet string) string {
//	if predmet == "excise" {
//		return "exciseWithMarking"
//	}
//	if predmet == "commodity" {
//		return "commodityWithMarking"
//	}
//	return predmet
//}

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
	//10.11.23 19:40 - сбис
	//2023-01-01T15:50
	if OFD == "ofdru" {
		res := strings.ReplaceAll(dt, "-", ".")
		return res
	}
	indOfT := strings.Index(dt, "T")
	if indOfT > 0 {
		//res := strings.ReplaceAll(dt, "T", " ")
		res := dt[:10]
		res = strings.ReplaceAll(res, "-", ".")
		return res
	}
	indOfPoint := strings.Index(dt, ".")
	if indOfPoint == 4 {
		return dt
	}
	y := ""
	if OFD == "sbis" {
		y = "20" + dt[6:8]
	} else {
		y = dt[6:10]
	}
	if strings.Contains(y, " ") {
		y = "20" + dt[6:8]
	}
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
	logginInFile(clearLogsDescr)
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
	res = strings.ReplaceAll(res, "₽", "")
	res = strings.ReplaceAll(res, "р", "")
	return res
}

func getNumberOfFieldsInCSVloc(line []string, fieldsnames map[string]string, fieldsnums map[string]int, fieldsOfBlock []string, notinv bool) map[string]int {
	for _, name := range fieldsOfBlock {
		//logginInFile(fmt.Sprintf("поиск поля %v", name))
		colname := fieldsnames[name]
		//logginInFile(fmt.Sprintln("colname", colname))
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
		if (name == "bindheadfieldkassa") || (name == "bindheadfieldcheck") || (name == "bindposposfieldcheck") {
			_, ok := fieldsnames[colname]
			if ok {
				colnamefinding = fieldsnames[fieldsnames[name]]
			}
		}
		//logginInFile(fmt.Sprintln("colnamefinding", colnamefinding))
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

func getNumberOfFieldsInCSV(line []string, fieldsnames map[string]string, fieldsnums map[string]int, partOfCheck string) map[string]int {
	var fieldsOfBlock []string
	if partOfCheck == "other" {
		fieldsOfBlock = AllFieldOtherOfCheck
	} else if partOfCheck == "positions" {
		fieldsOfBlock = AllFieldPositionsOfCheck
	} else if partOfCheck == "union" {
		fieldsOfBlock = AllFieldsUnionOfCheck
	} else {
		fieldsOfBlock = AllFieldsHeadOfCheck
	}
	//headAndNotOfPositions = AllFieldOtherOfCheck
	fieldsnums = getNumberOfFieldsInCSVloc(line, fieldsnames, fieldsnums, fieldsOfBlock, true)
	if (partOfCheck == "other") || (partOfCheck == "union") {
		return fieldsnums
	}
	if partOfCheck == "head" {
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
			if name != "stavkaNDS" {
				switch name {
				//case "stavkaNDS":
				//	resVal = resVal
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
			}
		} else {
			resVal = ""
		}
	}
	if strings.Contains(FieldsNames[name], "analyse") {
		//делаем анализ поля
		switch OFD {
		case "firstofd":
			if name == "bindposfieldkassa" || name == "bindposfieldcheck" || name == "bindotherfieldcheck" || name == "bindotherfieldkassa" {
				reg, fn, fd, descrErr, err := getRegFnFdFromName(resVal)
				if err != nil {
					logsmap[LOGERROR].Println(descrErr)
					erroFatal := fmt.Sprintf("ошбка: %v. Не удалось получить регистрационный номер, номер ФД, ФН из имени кассы %v", err, resVal)
					log.Fatal(erroFatal)
				}
				if name == "bindposfieldkassa" || name == "bindotherfieldkassa" {
					resVal = reg
					if FieldsNames[COLBINDHEADFIELDKASSA] == "fnkkt" {
						resVal = fn
					}
				} else if name == "bindposfieldcheck" || name == "bindotherfieldcheck" {
					resVal = fd
				}
			}
		}
	}
	return resVal
}

func fetchcheck(fd, fp, hyperlinkonjson string) (TReceiptOFD, string, error) {
	var receipt TReceiptOFD
	var resp *http.Response
	var err error
	var body []byte
	nameoffile := fd + "_" + fp + ".resp"
	fullFileName := DIROFREQUEST + nameoffile
	if !alredyGettedFetch(fd, fp) {
		strlog := fmt.Sprintf("получение данных о чеке по ссылке %v", hyperlinkonjson)
		logginInFile(strlog)
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
		strlog := fmt.Sprintf("получение данных из файла %v ", fullFileName)
		logginInFile(strlog)
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
		val == "0,00 р" || val == "-" || val == "0" {
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
	if strings.Contains(strings.ToUpper(sposob), "ПРЕДОПЛАТА 100%") {
		res = "fullPrepayment"
	}
	if strings.ToUpper(sposob) == "ПРЕДОПЛАТА" {
		res = "prepayment"
	}
	if strings.EqualFold(sposob, "Аванс") {
		res = "advance"
	}
	return res
}

func getOsnFromChernovVal(osnChernvVal string) string {
	res := ""
	switch strings.ToLower(osnChernvVal) {
	case "осн":
		res = "osn"
	case "усн доход":
		res = "usnIncome"
	case "усн доход-расход":
		res = "usnIncomeOutcome"
	case "усн доход - расход":
		res = "usnIncomeOutcome"
	case "есн":
		res = "esn"
	case "патент":
		res = "patent"
	}
	return res
}

func getPredmRasch(predm string) string {
	res := "commodity"
	switch strings.ToUpper(predm) {
	case "ТМ":
		res = "commodityWithMarking"
	case "ПОДАКЦИЗНЫЙ ТОВАР":
		res = "excise"
	case "ТНМ":
		res = "commodityWithoutMarking"
	case "АТМ":
		res = "exciseWithMarking"
	case "ПЛАТЕЖ":
		res = "payment"
	case "УСЛУГА":
		res = "service"
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
	//"=ГИПЕРССЫЛКА(""https://ofd.ru/Document/RenderDoc?RawId=5c581117-0587-d6ba-98fd-b404c6da4627"";""Перейти"")"
	s, _ := strings.CutPrefix(hyperlhtml, "=ГИПЕРССЫЛКА(\"")
	s, _ = strings.CutSuffix(s, "\";\"Перейти\")")
	return strings.ReplaceAll(s, "RenderDoc?RawId=", "ReceiptJsonDownload?DocId=")
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

func alredyGettedFetchAstral(fullname string) bool {
	res := true
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

func logginInFile(loggin string) {
	if *debug {
		logsmap[LOGINFO].Println(loggin)
	}
}

func getBoolFromString(val string, onErrorDefault bool) (bool, error) {
	var err error
	res := onErrorDefault
	if (val == "да") || (val == "ДА") || (val == "Да") || (val == "yes") || (val == "Yes") || (val == "YES") {
		res = true
	} else if (val == "НЕТ") || (val == "нет") || (val == "Нет") || (val == "no") || (val == "No") || (val == "NO") {
		res = false
	} else {
		res, err = strconv.ParseBool(val)
		if err != nil {
			res = onErrorDefault
		}
	}
	return res, err
}

func checkMistakeInPayments(amountcheck float64, payments map[string]float64) bool {
	resmist := false
	nal := payments[COLNAL]
	bez := payments[COLBEZ]
	ava := payments[COLAVANCE]
	crd := payments[COLCREDIT]
	vst := payments[COLVSTRECHPREDST]
	allnotnalsumm := bez + ava + crd + vst
	allsumms := allnotnalsumm + nal
	if allsumms > amountcheck {
		resmist = true
	}
	return resmist
}

func getMeasUnitFromStr(s string) string {
	res := "kilogram"
	switch s {
	case "л":
		res = "liter"
	case "грамм":
		res = "gram"
	case "иная":
		res = "otherUnits"
	case "шт":
		res = "piece"
	}
	return res
}
