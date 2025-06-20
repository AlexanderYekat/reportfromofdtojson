import json
import os
import argparse
import csv
from functools import lru_cache
from datetime import datetime
import logging
import shutil
import re

# ————— Настройка логирования —————
LOG_FILE = "convert_taxcom_to_atol_errors.log"
logging.basicConfig(
    filename=LOG_FILE,
    filemode="w",
    encoding="utf-8",      # теперь лог будет в UTF-8
    level=logging.ERROR,
    format="%(asctime)s %(levelname)s %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

error_folder = ("./errors")
undefined_nds_file = "undefined_nds_goods.csv"

def normalize_name(name: str) -> str:
    """
    Нормализует название товара для сопоставления:
    - убирает внешние кавычки и двойные кавычки
    - убирает завершающие точки с запятой
    - сводит несколько пробелов к одному
    - обрезает пробелы по краям
    """
    if name is None:
        return ''
    # trim
    norm = name.strip()
    if norm.find('Контейнер алюминевый') >= 0:
       norm = 'Контейнер алюминевый' 
    # Удаляем шаблоны тысяч через rstrip
    norm = norm.replace('37 г.', '31 г.')
    norm = norm.replace('газ ТМ', '')
    norm = norm.replace('п/а', '')
    norm = norm.replace('ПАНКО', '')
    norm = norm.replace('филе', '')
    norm = norm.replace('(Египет)', '')
    norm = norm.replace('(Сербия)', '')
    norm = norm.replace('(Китай)', '')
    #norm = norm.replace('500 г', '')
    norm = norm.replace('"Засолыч" 500 г', '"Засолыч"')
    norm = norm.replace('б/г 31/40', '16/20 б/г')
    norm = norm.replace('Говяжья вырезка', 'Говяжий рубец')
    norm = norm.replace('500 г.', '450 г.')
    norm = norm.replace('Санарем', 'Санрем')
    norm = norm.rstrip('1,000').strip()
    norm = norm.rstrip('1,000 ;').strip()    
    norm = norm.replace(' ', '')
    norm = norm.replace('(лойны)', '')
    norm = norm.replace('б/к', '')
    norm = norm.replace('г/к', '')
    norm = norm.replace('"Рибай"', '')
    norm = norm.replace('из грудки', '')
    norm = norm.replace('Вьетнам', '')
    norm = norm.replace('0,44', '0,45')
    # remove surrounding quotes
    if norm.startswith('"') and norm.endswith('"'):
        norm = norm[1:-1]
    # replace double double-quotes with single
    norm = norm.replace('""', '"')
    # remove trailing semicolon
    norm = norm.rstrip(';').strip()
    # collapse whitespace
    norm = re.sub(r'\s+', ' ', norm)
    norm = norm.replace('’', '')
    norm = norm.replace(' ', '')
    norm = norm.replace('"', '')
    return norm

@lru_cache(maxsize=1)
def _load_nds_map(csv_path):
    """
    Загружает и кэширует отображение 'название товара' -> ставка НДС (int).
    """
    nds_map = {}
    with open(csv_path, newline='', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter=';', quotechar='"')
        # Проверяем наличие необходимых столбцов
        if 'name' not in reader.fieldnames or 'nds' not in reader.fieldnames:
            raise ValueError(f"Ожидал заголовки 'name' и 'nds' в файле {csv_path}")
        for row in reader:
            raw_name = row['name']
            key = normalize_name(raw_name)
            raw_nds = row['nds'].strip()
            # Убираем символ '%' при наличии
            if raw_nds.endswith('%'):
                raw_nds = raw_nds[:-1].strip()
            try:
                nds_val = int(raw_nds)
            except ValueError:
                raise ValueError(f"Невалидное значение НДС '{row['nds']}' для товара '{key}'")
            nds_map[key] = nds_val
    return nds_map

def convert_tax_type(tax_type):
    """Конвертирует тип налогообложения"""
    tax_mapping = {
        "Патент": "patent",
        "УСН доход": "usnIncome",
        "УСН доход - расход": "usnIncomeOutcome",
    }
    return tax_mapping.get(tax_type, "usnIncomeOutcome")

def convert_measurement(mesure):
    mesure_mapping = {
        "шт.": "piece",
        "шт": "piece",
        "кг": "kilogram",
        "кг.": "kilogram",
    }
    return mesure_mapping.get(mesure, "piece")

def convert_payment_method(calculation_method):
    """Конвертирует метод оплаты"""
    method_mapping = {
        "ПОЛНЫЙ РАСЧЕТ": "fullPayment",
        "ЧАСТИЧНАЯ ПРЕДОПЛАТА": "partialPayment",
        # Добавьте другие соответствия по необходимости
    }
    return method_mapping.get(calculation_method, "fullPayment")

def convert_payment_subject(item_type):
    """Конвертирует предмет расчета"""
    subject_mapping = {
        "ТОВАР": "commodity",
        "УСЛУГА": "service",
        "РАБОТА": "job",
        "ПРЕМИЯ": "gift",
        "ПРОЧИЕ": "other",
        "ТМ": "commodityWithMarking",
    }
    return subject_mapping.get(item_type, "commodity")

def convert_nds_rate_key(nds_rate: int) -> str:
    mapping = {
        None: "none",
        20: "vat20",
        10: "vat10",
        5: "vat5",
        7: "vat7",
        0: "vat0",
    }
    return mapping.get(nds_rate)

def itIsGoodWithNDS10(csv_path, nameofgood):
    try:
        return getNDSOfGood(csv_path, nameofgood) == 10
    except KeyError:
        # Логируем отсутствие товара в справочнике НДС
        logging.error(f"НДС не найден для товара '{nameofgood}'")
        raise

def getNDSOfGood(csv_path, nameofgood):
    """
    Возвращает ставку НДС (int) для товара nameofgood.
    Файл загружается и кэшируется при первом вызове.

    :param csv_path: путь к CSV-файлу с разделителем ';' и кавычками '"'
    :param nameofgood: точное название товара для поиска
    :return: ставка НДС в процентах (int)
    :raises KeyError: если товар не найден в файле
    :raises ValueError: при ошибке формата файла или значения НДС
    """
    key = normalize_name(nameofgood)
    nds_map = _load_nds_map(csv_path)
    if key in nds_map:
        return nds_map[key]
    # Логируем и пробрасываем, если товар не найден
    logging.error(f"Товар '{nameofgood}' (normalized '{key}') не найден в файле НДС {csv_path}")
    raise KeyError(f"Товар '{nameofgood}' не найден")

def convert_items(items, goodsFilesWithNDS = "", changeNDS = False):
    """Конвертирует позиции чека"""
    converted = []
    
    for item in items:
        if item.get("unit_price", 0) == 0:
            logging.error(f"Товар {item.get("description")} с нулевой ценой в файле")
            continue

        nds_rate = convert_nds_rate_key(item.get("vat_percent"))
        if changeNDS:
            try:
                nds_val = getNDSOfGood(goodsFilesWithNDS, item.get("description", ""))
                nds_rate = convert_nds_rate_key(nds_val)
            except KeyError:
                continue
        
        converted.append({
            "type": "position",
            "name": item.get("description"),
            "price": item.get("unit_price"),
            "quantity": item.get("quantity"),
            "amount": item.get("sum"),
            "measurementUnit": convert_measurement(item.get("unit")),
            "paymentMethod": convert_payment_method(item.get("payment_method")),
            "paymentObject": convert_payment_subject(item.get("item_type")), #"commodity",
            "tax": {"type": nds_rate}
        })
    return converted

def convert_fetch_to_atol(doc, storno=False, changeNDS=False, goodsFilesWithNDS = ""):
    """Конвертирует JSON из формата fetch в формат atol"""

    issues = []
    # Проверка товаров на наличие ставки 10%
    has10 = False
    for it in doc.get("items", []):
        name = it.get("description", "")
        try:
            if itIsGoodWithNDS10(goodsFilesWithNDS, name):
                has10 = True
        except KeyError:
            issues.append(name)
    if not has10:
        # Пропускаем документы без товаров с НДС‑10%
        return None, issues

    cash = doc.get("paid_cash", 0)
    card = doc.get("paid_card", 0)
       
    # Определяем тип оплаты
    if card == None:
        payments = [{"type": "cash", "sum": cash}]
    elif cash == 0:
        payments = [{"type": "electronically", "sum": card}]
    else:
        payments = [
            {"type": "electronically", "sum": card},
            {"type": "cash", "sum": cash}
        ]

    # Конвертируем обычные позиции
    op_type = {"sale": "sellCorrection", "return": "sellReturnCorrection"}.get(doc.get("receipt_type"))

    if storno:
        op_type = {"sellCorrection": "sellReturnCorrection",
                   "sellReturnCorrection": "sellCorrection",
                   "buyCorrection": "buyReturnCorrection",
                   "buyReturnCorrection": "buyCorrection"}.get(op_type, op_type)

    # Системы налогообложения
    taxation = {"Патент": "patent",
                 "УСН доход": "usnIncome",
                 "УСН доход - расход": "usnIncomeOutcome"}.get(doc.get("tax_system"))


    items = convert_items(doc.get("items", []), goodsFilesWithNDS, changeNDS)
    
    # Системы налогообложения
    taxation = {"Патент": "patent",
                 "УСН доход": "usnIncome",
                 "УСН доход - расход": "usnIncomeOutcome"}.get(doc.get("tax_system"))    
    # Добавляем дополнительные атрибуты с реальными значениями из документа
    # Доп. атрибуты
    extra = [
        {"type": "userAttribute", "name": "ФД", "value": str(doc.get("fd_number")), "print": True},
        {"type": "additionalAttribute", "value": str(doc.get("fp")), "print": True}
    ]
    
    dt = datetime.fromisoformat(doc.get('datetime'))
    base_date = dt.strftime('%Y.%m.%d')

    result = {
        "type": op_type,
        "electronically": True,
        "taxationType": taxation,
        "correctionType": "self",
        "correctionBaseDate": base_date,
        "operator": {"name": doc.get("cashier")},
        "items": extra + items,
        "payments": payments,
        "total": doc.get("total")
    }
    
    return result, issues

def process_directory(storno=False, changeNDS=False, goodsFilesWithNDS = ""):
    """Обрабатывает все JSON файлы в директории receipts"""
    receipts_dir = "./taxcomjsons"
    output_dir = "./atoljsonstaxcom"
    # Множество для накопления товаров с неопределённым НДС
    undefined_goods = set()
    
    # Создаем директорию для сконвертированных файлов, если её нет
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)
    
    # Перебираем все файлы в директории
    for filename in os.listdir(receipts_dir):
        if not filename.endswith(".json"):
            continue

        inp = os.path.join(receipts_dir, filename)
        out = os.path.join(output_dir, f"{filename}")
        #print(filename)
            
        try:
            # Читаем входной файл
            with open(inp, 'r', encoding='utf-8') as f:
                data = json.load(f)
               
            # Конвертируем данные
            conv, issues = convert_fetch_to_atol(data, storno, changeNDS, goodsFilesWithNDS)
            undefined_goods.update(issues)    
            #print(conv)
            if conv is None:
                continue
            #print('Сохраняем результат')
                
            # Сохраняем результат
            with open(out, 'w', encoding='utf-8') as f:
                json.dump(conv, f, ensure_ascii=False, indent=2)

            # если были ошибки — копируем PDF в errors/
            if issues:
                shutil.copy(inp, os.path.join(error_folder, filename))
                logging.error(f"Файл {filename} скопирован в {error_folder}, отсутствуют НДС: {issues}")
            else:
                print(f"Успешно сконвертирован файл: {filename}")
                
        except Exception:
            logging.error(f"Ошибка при обработке файла {filename}", exc_info=True)
            print(f"Ошибка при обработке файла {filename}, см. {LOG_FILE}")

    # После обработки всех файлов — сохраняем список товаров с неопределённым НДС
    if undefined_goods:
        with open(undefined_nds_file, 'w', encoding='utf-8') as uf:
            for good in sorted(undefined_goods):
                uf.write(good + "\n")
        print(f"Список товаров с неопределённым НДС сохранен в {undefined_nds_file}")
    else:
        print("Все товары имеют определённый НДС.")

if __name__ == "__main__":
    os.makedirs(error_folder, exist_ok=True)    
    # Создаем парсер аргументов командной строки
    parser = argparse.ArgumentParser(description='Конвертер чеков из формата fetch в формат atol')
    parser.add_argument('--storno', action='store_true', help='Флаг сторно')
    parser.add_argument('--changeNDS', action='store_true', help='Флаг изменения НДС')
    
    # Парсим аргументы
    args = parser.parse_args()

    # Запускаем обработку директории
    process_directory(args.storno, args.changeNDS, "./goodsnds.csv")    
    
    # Флаг сторно доступен как args.storno
    # Пример использования:
    if args.storno:
        print("Режим сторно активирован")

    if args.changeNDS:
        print("Режим изменения НДС активирован")